import 'package:flutter/material.dart';

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
                         print(textarea.text);
                     }, 
                     child: const Text("Save")
                    )
               ],
             ),
          )
      );
  }
}