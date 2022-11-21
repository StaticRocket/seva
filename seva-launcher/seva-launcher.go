package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strings"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/skratchdot/open-golang/open"
)

var store_url = "https://raw.githubusercontent.com/StaticRocket/seva-apps/main"
var addr = flag.String("addr", "0.0.0.0:8000", "http service address")
var no_browser = flag.Bool("no-browser", false, "do not launch browser")
var docker_browser = flag.Bool("docker-browser", false, "force use of docker browser")
var http_proxy = flag.String("http_proxy", "", "use to set http proxy")
var no_proxy = flag.String("no_proxy", "", "use to set no-proxy")

var container_id_list [2]string
var docker_compose string

//go:embed web/*
var content embed.FS

//go:embed docker-compose
var docker_compose_bin []byte

// Flags to check proxy validation
var is_http_proxy bool = true

func is_docker_compose_installed() bool {
	cmd := exec.Command("docker-compose", "-v")
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Docker-compose is either not installed or cannot be executed")
		log.Println(err)
		log.Println("Using local install for now")
		return false
	}
	return true
}

func prepare_compose() string {
	if !is_docker_compose_installed() {
		ioutil.WriteFile("docker-compose", docker_compose_bin, 0755)
		return "./docker-compose"
	}
	return "docker-compose"
}

func setup_working_directory() {
	err := os.MkdirAll("/tmp/seva-launcher", os.ModePerm)
	if err != nil {
		log.Println(err)
		exit(1)
	}
	err = os.Chdir("/tmp/seva-launcher")
	if err != nil {
		log.Println(err)
		exit(1)
	}
}

func launch_browser() {
	if *docker_browser {
		go launch_docker_browser()
	} else {
		err := open.Start("http://localhost:8000/#/")
		if err != nil {
			log.Println("Host browser not detected, fetching one through docker")
			go launch_docker_browser()
		}
	}
}

func launch_docker_browser() {
	xdg_runtime_dir := os.Getenv("XDG_RUNTIME_DIR")
	user, _ := user.Current()
	output := docker_run("--rm", "--privileged", "--network", "host",
		"-v", "/tmp/.X11-unix",
		"-e", "XAUTHORITY",
		"-e", "XDG_RUNTIME_DIR=/tmp",
		"-e", "DISPLAY",
		"-e", "WAYLAND_DISPLAY",
		"-e", "https_proxy",
		"-e", "http_proxy",
		"-e", "no_proxy",
		"-v", xdg_runtime_dir+":/tmp",
		"--user="+user.Uid+":"+user.Gid,
		"ghcr.io/staticrocket/seva-browser:latest",
		"http://localhost:8000/#/",
	)
	output_strings := strings.Split(strings.TrimSpace(string(output)), "\n")
	container_id_list[1] = output_strings[len(output_strings)-1]
}

func docker_run(args ...string) []byte {
	args = append([]string{"run", "-d"}, args...)
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	log.Printf("|\n%s\n", output)
	if err != nil {
		log.Println("Failed to start container!")
		log.Println(err)
		exit(1)
	}
	return output
}

func start_design_gallery() {
	log.Println("Starting local design gallery service")
	output := docker_run("--rm", "-p", "8001:80",
		"ghcr.io/staticrocket/seva-design-gallery:latest",
	)
	output_strings := strings.Split(strings.TrimSpace(string(output)), "\n")
	container_id_list[0] = output_strings[len(output_strings)-1]
}

func exit(num int) {
	log.Println("Stopping non-app containers")
	for _, container_id := range container_id_list {
		if len(container_id) > 0 {
			cmd := exec.Command("docker", "stop", container_id)
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("Failed to stop container %s : \n%s", container_id, output)
			}
		}
	}
	os.Exit(num)
}

func setup_exit_handler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		exit(0)
	}()
}

func handle_requests() {
	router := mux.NewRouter()
	router.HandleFunc("/ws", websocket_controller)
	log.Println("Listening for websocket messages at " + *addr + "/ws")
	root_content, err := fs.Sub(content, "web")
	if err != nil {
		log.Println("No files to server for web interface!")
		exit(1)
	}
	router.PathPrefix("/").Handler(http.FileServer(http.FS(root_content)))
	log.Println(http.ListenAndServe(*addr, router))
}

func check_env_vars() {
	for _, element := range []string{"DISPLAY", "WAYLAND_DISPLAY"} {
		env_var := os.Getenv(element)
		if len(env_var) > 0 {
			return
		}
	}
	log.Println("Environment variable DISPLAY or WAYLAND_DISPLAY must be set!")
	exit(1)
}

func update_sysconfig_cli() {

	var final_sysconfig_proxy_cli string

	// if http_proxy is not valid then just apply no_proxy
	if is_http_proxy == false {
		final_sysconfig_proxy_cli := fmt.Sprintf("export NO_PROXY=\"%s\"", *no_proxy)
	} else {
		final_sysconfig_proxy_cli := fmt.Sprintf("export HTTPS_PROXY=\"%s\"\nexport HTTPS_PROXY=\"%s\"\nexport NO_PROXY=\"%s\"", *http_proxy, *http_proxy, *no_proxy)
	}

	fmt.Println(final_sysconfig_proxy_cli)

	// Write the proxy setting
	err := ioutil.WriteFile("/etc/sysconfig/docker", []byte(final_sysconfig_proxy_cli), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Restart the Docker daemon after setting up the proxy
	cmd := exec.Command("service", "docker", "restart")
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

}

func update_systemd_cli() {
	// Create /etc/systemd/system/docker.service.d directory structure
	if err := os.MkdirAll("/etc/systemd/system/docker.service.d", os.ModePerm); err != nil {
		log.Fatal(err)
	}

	// Create a file http-proxy.conf in /etc/systemd/system/docker.service.d
	myfile, e := os.Create("/etc/systemd/system/docker.service.d/http-proxy.conf")
	if e != nil {
		log.Fatal(e)
	}

	var final_systemd_proxy_cli string

	// if http_proxy is not valid then just apply no_proxy
	if is_http_proxy == false {
		final_systemd_proxy_cli := fmt.Sprintf("[Service]\nEnvironment=\"NO_PROXY=%s\"\n", *no_proxy)
	} else {
		final_systemd_proxy_cli := fmt.Sprintf("[Service]\nEnvironment=\"HTTP_PROXY=%s\"\nEnvironment=\"HTTPS_PROXY=%s\"\nEnvironment=\"NO_PROXY=%s\"\n", *http_proxy, *http_proxy, *no_proxy)
	}

	fmt.Println(final_systemd_proxy_cli)

	// Write the proxy setting
	err_ := ioutil.WriteFile("/etc/systemd/system/docker.service.d/http-proxy.conf", []byte(final_systemd_proxy_cli), 0644)
	if err_ != nil {
		log.Fatal(err_)
	}

	// Flush changes and restart Docker
	cmd := exec.Command("systemctl", "daemon-reload")
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	cmd1 := exec.Command("systemctl", "restart", "docker")
	err1 := cmd1.Run()

	if err1 != nil {
		log.Fatal(err1)
	}

	myfile.Close()
}

func validate_proxy() {
	u, err := url.ParseRequestURI(*http_proxy)
	if err != nil {
		log.Println("Invalid http proxy. Hence not applying it.")
		is_http_proxy := false
	}
}

func setup_proxy() {
	// Setting up Environment Variables
	// If http_proxy is valid apply changes to Environment variable
	if is_http_proxy == true {
		os.Setenv("HTTP_PROXY", *http_proxy)
		os.Setenv("http_proxy", *http_proxy)
		os.Setenv("HTTPs_PROXY", *http_proxy)
		os.Setenv("https_proxy", *http_proxy)
	}

	os.Setenv("no_proxy", *no_proxy)

	// Checks if File /etc/sysconfig/docker exists
	if _, err := os.Stat("/etc/sysconfig/docker"); err == nil {
		update_sysconfig_cli()
	} else {
		update_systemd_cli()
	}
}

func main() {
	setup_exit_handler()
	check_env_vars()
	flag.Parse()

	validate_proxy()
	setup_proxy()

	log.Println("Setting up working directory")
	setup_working_directory()
	docker_compose = prepare_compose()

	go start_design_gallery()

	if !*no_browser {
		log.Println("Launching browser")
		launch_browser()
	}

	handle_requests()
}
