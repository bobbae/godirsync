godirsync
=========

A tiny tool to replicate a directory containing bunch of sub directories and files to another machine over http.


example:
```
$ godirsync -server 0.0.0.0:9999
```

This will serve files located in "." where the program runs

on the client machine:
```
$ godirsync -from http://serverip:9999 
```

Each time you run godirsync on client machine it will get files that have 
been added or changed since the last time godirsync ran. All files will be 
fetched from the server via http.  Godirsync running in client 
mode will ask the server, over http, for a list of files that have been added or 
changed. Then it will do GET for each machine and write to "." where the client is running.
