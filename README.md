# fileserver

`fileserver` is a simple linux utility that serves local files over HTTPS.
Let's Encrypt is used to automatically generate SSL certificates.

## Install

```bash
go install github.com/mikerybka/fileserver@latest
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
fileserver <public_directory> <private_directory> <certificate_directory> <logs_directory> <auth_directory> <email>
```

- `public_directory` is the directory of public files you want to serve.
- `private_directory` is the directory of private files you want to serve.
- `certificate_directory` is the directory used to store your generated SSL certificates.
- `logs_directory` is the directory used to store a log entry for each incoming request.
- `auth_directory` is the directory use to store auth data like users and sessions.
- `email` is a contact email that Let's Encrypt can use to notify you about problems with issued certificates.

## Example systemd service

/etc/systemd/system/fileserver.service
```
[Unit]
Description=fileserver

[Service]
ExecStart=/root/go/bin/fileserver /root/public /root/private /root/certs /root/logs you@example.com

[Install]
WantedBy=multi-user.target
```

```
systemctl daemon-reload
systemctl start fileserver
systemctl status fileserver
```
