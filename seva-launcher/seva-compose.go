/*
Wrapper for docker compose functions
*/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/melbahja/got"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type proxyContainer struct {
	command   string
	arguments []string
	exit_code int
	response  []string
}

type Containers []struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	Command    string `json:"Command"`
	Project    string `json:"Project"`
	Service    string `json:"Service"`
	State      string `json:"State"`
	Health     string `json:"Health"`
	ExitCode   int    `json:"ExitCode"`
	Publishers []struct {
		URL           string `json:"URL"`
		TargetPort    int    `json:"TargetPort"`
		PublishedPort int    `json:"PublishedPort"`
		Protocol      string `json:"Protocol"`
	} `json:"Publishers"`
}

func start_app(command WebSocketCommand) WebSocketCommand {
	log.Println("Starting selected app")
	cmd := exec.Command(docker_compose, "-p", "seva-launcher", "up", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Failed to start selected app!")
		exit(1)
	}
	output_s := strings.TrimSpace(string(output))
	command.Response = append(command.Response, strings.Split(output_s, "\n")...)
	log.Printf("|\n%s\n", output_s)
	return command
}

func update_sysconfig(proxy_containers proxyContainer) {
	var http_ string = proxy_containers.arguments[0]["http_proxy"]
	var no_proxy_ string = proxy_containers.arguments[0]["no_proxy"]

	final_sysconfig_proxy := fmt.Sprintf("export HTTPS_PROXY=\"%s\"\nexport HTTPS_PROXY=\"%s\"\nexport NO_PROXY=\"%s\"", http_, http_, no_proxy_)

	fmt.Println(final_sysconfig_proxy)

	// Write the proxy setting
	err := ioutil.WriteFile("/etc/sysconfig/docker", []byte(final_sysconfig_proxy), 0644)
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

func update_systemd(proxy_containers proxyContainer) {

	// Create /etc/systemd/system/docker.service.d directory structure
	if err := os.MkdirAll("/etc/systemd/system/docker.service.d", os.ModePerm); err != nil {
		log.Fatal(err)
	}

	// Create a file http-proxy.conf in /etc/systemd/system/docker.service.d
	myfile, e := os.Create("/etc/systemd/system/docker.service.d/http-proxy.conf")
	if e != nil {
		log.Fatal(e)
	}

	var http_ string = proxy_containers.arguments[0]["http_proxy"]
	var no_proxy_ string = proxy_containers.arguments[0]["no_proxy"]

	final_systemd_proxy := fmt.Sprintf("[Service]\nEnvironment=\"HTTP_PROXY=%s\"\nEnvironment=\"HTTPS_PROXY=%s\"\nEnvironment=\"NO_PROXY=%s\"\n", http_, http_, no_proxy_)

	fmt.Println(final_systemd_proxy)

	// Write the proxy setting
	err_ := ioutil.WriteFile("/etc/systemd/system/docker.service.d/http-proxy.conf", []byte(final_systemd_proxy), 0644)
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

func save_settings(command WebSocketCommand) WebSocketCommand {
	log.Println("Started Applying Docker Proxy Settings")

	var proxy_containers proxyContainer
	err = json.Unmarshal([]byte(command), &proxy_containers)
	if err != nil {
		log.Println("Failed to de-serialize the JSON String!")
		exit(1)
	}

	// Checks if File /etc/sysconfig/docker exists
	if _, err := os.Stat("/etc/sysconfig/docker"); err == nil {
		update_sysconfig(proxy_containers)
	} else {
		update_systemd(proxy_containers)
	}

	// Setting up Environment Variables
	var http_ string = proxy_containers.arguments[0]["http_proxy"]
	var no_proxy_ string = proxy_containers.arguments[0]["no_proxy"]

	os.Setenv("HTTP_PROXY", http_)
	os.Setenv("http_proxy", http_)
	os.Setenv("HTTPs_PROXY", http_)
	os.Setenv("https_proxy", http_)
	os.Setenv("no_proxy", no_proxy_)

	command.Response = append(command.Response, "write-done")
	log.Println("Applied Docker Proxy Settings")
	return command
}

func stop_app(command WebSocketCommand) WebSocketCommand {
	log.Println("Stopping selected app")
	cmd := exec.Command(docker_compose, "-p", "seva-launcher", "down", "--remove-orphans")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Failed to stop selected app! (It may not be running!)")
	}
	output_s := strings.TrimSpace(string(output))
	command.Response = append(command.Response, strings.Split(output_s, "\n")...)
	log.Printf("|\n%s\n", output_s)
	return command
}

func get_app(command WebSocketCommand) WebSocketCommand {
	if _, err := os.Stat("metadata.json"); errors.Is(err, os.ErrNotExist) {
		command.Response = append(command.Response, "{}")
		return command
	}
	content, err := os.ReadFile("metadata.json")
	if err != nil {
		log.Println(err)
		exit(1)
	}
	command.Response = []string{string(content)}
	return command
}

func load_app(command WebSocketCommand) WebSocketCommand {
	name := command.Arguments[0]
	log.Println("Loading " + name + " from store")
	command = stop_app(command)

	files := []string{"metadata.json", "docker-compose.yml"}
	for _, element := range files {
		if _, err := os.Stat(element); errors.Is(err, os.ErrNotExist) {
			continue
		}
		err := os.Remove(element)
		if err != nil {
			log.Println("Failed to remove old files")
			exit(1)
		}
	}
	g := got.New()
	for _, element := range files {
		url := store_url + "/" + name + "/" + element
		log.Println("Fetching " + element + " from: " + url)
		err := g.Download(url, element)
		if err != nil {
			log.Println(err)
			exit(1)
		}
	}
	command.Response = append(command.Response, "0")
	return command
}

func is_running(command WebSocketCommand) WebSocketCommand {
	name := command.Arguments[0]
	log.Println("Checking if " + name + " is running")
	cmd := exec.Command(docker_compose, "-p", "seva-launcher", "ps", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		log.Println("Failed to check if app is running!")
		exit(1)
	}
	var containers Containers
	err = json.Unmarshal([]byte(output), &containers)
	if err != nil {
		log.Println("Failed to parse JSON from docker-compose!")
		exit(1)
	}
	for _, element := range containers {
		if element.Name == name {
			command.Response = append(command.Response, "1")
			return command
		}
	}
	command.Response = append(command.Response, "0")
	return command
}
