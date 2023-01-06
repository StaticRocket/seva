import 'package:flutter/material.dart';
import 'websocket.dart';
import 'dart:async';
import 'dart:html';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';

class WebProxy extends StatefulWidget {
  @override
  State<WebProxy> createState() => _WebProxyState();
}

class _WebProxyState extends State<WebProxy> {
  var http_textarea = TextEditingController();
  var no_proxy_textarea = TextEditingController();

  final _form = GlobalKey<FormState>();

  bool waiting_on_response_ = false;

  Future<WebSocketCommand> response_handler() async {
    // catch the response code and update state accordingly
    setState(() {
      waiting_on_response_ = true;
    });
    String response = await stream.first;
    setState(() {
      waiting_on_response_ = false;
    });
    return WebSocketCommand.from_json(jsonDecode(response));
  }

  Future<void> save_settings(var serialized_settings) async {
    WebSocketCommand.outbound("save_settings", [serialized_settings]).send();
    WebSocketCommand command = await response_handler();
    if (command.response[0] == '1') {
	    // TODO: Error handling
    }
  }

  bool isValidUrl(var proxy_url) {
    var urlPattern =
        r"(http|https|socks4|socks5)://[A-Za-z0-9\-._~:/?#\[\]@!$&'\(\)*+,;%=]+";
    var regExp = new RegExp(urlPattern, caseSensitive: false);
    if (!regExp.hasMatch(proxy_url)) {
      return false;
    }

    return true;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
        appBar: AppBar(
          title: const Text("Settings"),
        ),
        body: Form(
          key: _form, //assigning key to form
          child: Column(
            children: [
              const SizedBox(height: 40),
              TextFormField(
                controller: http_textarea,
                keyboardType: TextInputType.multiline,
                maxLines: 1,
                decoration: const InputDecoration(
                    hintText: "Enter your Http Proxy Settings",
                    focusedBorder: OutlineInputBorder(
                        borderSide:
                            BorderSide(width: 1, color: Colors.redAccent))),
                validator: (value) {
                  if (value != null && isValidUrl(http_textarea.text)) {
                    return 'Please Enter Valid URL';
                  }
                  return null;
                },
              ),
              const SizedBox(height: 60),
              TextFormField(
                controller: no_proxy_textarea,
                keyboardType: TextInputType.multiline,
                maxLines: 1,
                decoration: const InputDecoration(
                    hintText: "Enter your No-Proxy Settings",
                    focusedBorder: OutlineInputBorder(
                        borderSide:
                            BorderSide(width: 1, color: Colors.redAccent))),
              ),
            ],
          ),
        ),
        floatingActionButton: Column(
          mainAxisAlignment: MainAxisAlignment.end,
          children: <Widget>[
            FloatingActionButton(
              onPressed: () {
                if (_form.currentState!.validate()) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(
                        content: Text('Applying your Proxy Settings')),
                  );

                  var settings = {
                    "http_proxy": http_textarea.text,
                    "no_proxy": no_proxy_textarea.text
                  };
                  var serialized_settings = json.encode(settings);
                  save_settings(serialized_settings);
                }
              },
              tooltip: 'Save',
              child: const Icon(Icons.save),
            ),
          ],
        ));
  }
}
