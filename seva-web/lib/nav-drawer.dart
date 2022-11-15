import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'settings_page.dart';

class NavDrawer extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Drawer(
      child: ListView(
        padding: EdgeInsets.zero,
        children: <Widget>[
          DrawerHeader(
            child: Text(
              'Side menu',
              style: TextStyle(color: Colors.white, fontSize: 25),
            ),
            decoration: BoxDecoration(
              color: Colors.red,
            ),
          ),
          ListTile(
            leading: Icon(Icons.input),
            title: Text('Home'),
            onTap: () => {Navigator.of(context).pop()},
          ),
          ListTile(
            leading: Icon(Icons.settings),
            title: Text('Web Proxy Settings'),
            onTap: () {
              Navigator.push(
                context,
                MaterialPageRoute(
                  builder: (context) => WebProxy(),
                ),
              );
            },
          ),
          ListTile(
            leading: Icon(Icons.border_color),
            title: Text('Feedback'),
            onTap: () => {}, // Maybe add something later
          ),
          ListTile(
            leading: Icon(Icons.close),
            title: Text('Close'),
            onTap: () =>
                {}, // Probably use exit(0) or SystemNavigator.pop() if needed
          ),
        ],
      ),
    );
  }
}
