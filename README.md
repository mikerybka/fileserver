# fileserver

`fileserver` is a simple linux utility that serves local files over HTTPS.
Let's Encrypt is used to automatically generate SSL certificates.

## Install

```bash
go install github.com/webmachine-dev/fileserver@latest
```

## Usage
Simply arrange a directory so that the top level only filled with other directorys; each named after the host you want to serve.
For example:

```
example.com
  index.html
  blog
    post-1.html
api.example.com
  blog
    post-1.json
example.org
  index.html
```

You can add/edit/rename/move any file or host directory without needing to reload the server.

To start the server:
```bash
fileserver <file_directory> <certificate_directory> <email>
```

- `file_directory` is the directory of files you want to serve.
- `certificate_directory` is the directory used to store your generated SSL certificates.
- `email` is a contact email that Let's Encrypt can use to notify you about problems with issued certificates.

## Example systemd service

/etc/systemd/system/fileserver.service
```
[Unit]
Description=fileserver

[Service]
ExecStart=/root/go/bin/fileserver /root/public /root/certs you@example.com

[Install]
WantedBy=multi-user.target
```

```
systemctl daemon-reload
systemctl start fileserver
systemctl status fileserver
```
