# gavin

*gavin* is a self contained instance of [Organice](https://github.com/200ok-ch/organice).

It combines a WebDAV server and the Organice assets into a single binary that
can be run from virtually everywhere.

`gavin` can be used as a standalone webserver or placed behind a reverse proxy.

## Features

- Built in WebDAV server with basic http authentication.
- Ability to serve over TLS using automatically managed ACME certificates.
- Reverse proxy friendly.

## Docs

### Generating a .htpasswd file

Please note: `gavin` expects the `.htpasswd` file to use bcrypt as the hashing
algorithm!

#### OpenBSD

```
htpasswd .htpasswd $USER
```

#### Linux/macOS

```
htpasswd -B -c .htpasswd $USER
```

### Example usage on local machine

#### Download gavin

Releases can be downloaded for common OSs here:

https://github.com/qbit/gavin/releases

#### Running

- Generate a `.htpasswd` file.
- Run `gavin` pointing it at your `org` files:
```
gavin -davdir ~/org
```

Now you log into `gavin` with the following settings:

URL: https://localhost:8080/dav
Username: $USER
Password: $YOURPASSWORD

### Running in auto ACME mode

```
gavin -domain gavin.example.com -http $externalIP:443
```

If you would like to specify where `gavin` stores the certificates the `-cache`
flag can be used.

By default `gavin` will listen on port *80* for ACME requests. This can be
changed using the `-alisten` flag, however, note that ACME always sends
requests over port 80, so you will need something that forwards requests onto
`gavin`.

