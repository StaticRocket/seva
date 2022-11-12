import 'package:flutter/material.dart';
import 'websocket.dart';
import 'dart:async';
import 'dart:html';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';

void main() {
  runApp( MaterialApp(
       home: WebProxy()
  ));
}

class WebProxy extends  StatefulWidget {
  @override
  State<WebProxy> createState() => _WebProxyState();
}

class _WebProxyState extends State<WebProxy> {

  TextEditingController textarea = TextEditingController();
  bool waiting_on_response1 = false;
  
  Future<WebSocketCommand> response_handler() async {
    // catch the response code and update state accordingly
    setState(() {
      waiting_on_response1 = true;
    });
    String response = await stream.first;
    setState(() {
      waiting_on_response1 = false;
    });
    return WebSocketCommand.from_json(jsonDecode(response));
  }

  Future<void> write_proxy(String myProxy) async {
    // writes proxy 
    print(myProxy);
    WebSocketCommand.outbound(myProxy, []).send();
    WebSocketCommand command = await response_handler();
    print('Done..');
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
         appBar: AppBar(
            title: const Text("Web-Proxy Settings"),
            backgroundColor: Colors.redAccent,
         ),
          body: Container(
             alignment: Alignment.center,
             padding: const EdgeInsets.all(20),
             child: Column(
               children: [
                   TextField(
                      controller: textarea,
                      keyboardType: TextInputType.multiline,
                      maxLines: 10,
                      decoration: const InputDecoration( 
                         hintText: "Enter your Webproxy Settings",
                         focusedBorder: OutlineInputBorder(
                            borderSide: BorderSide(width: 1, color: Colors.redAccent)
                         )
                      ),
                       
                   ),
                   const SizedBox(height: 50),
                 
                   ElevatedButton(
                     onPressed: (){
                         write_proxy(textarea.text);
                     }, 
                     child: const Text("Apply")
                    )
               ],
             ),
          )
      );
  }
}
