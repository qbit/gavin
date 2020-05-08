# gavin

[![builds.sr.ht status](https://builds.sr.ht/~qbit/gavin.svg)](https://builds.sr.ht/~qbit/gavin?)

Simple utility to serve password protected WebDAV.

## Installation

```
go git -u suah.dev/gavin
```

## Example usage

`gavin` was built as a simple WebDAV server specifically to run
[organice](https://github.com/200ok-ch/organice). Here is an example showing how
to host organice via WebDAV.

| Flag      | Value               | Description                                                                             |
|-----------|---------------------|-----------------------------------------------------------------------------------------|
| `-davdir` | /tmp/org            | The directory we have our .org files in.                                                |
| `-htpass` | /tmp/.htpasswd      | Standard `htpasswd` file generated with `htpasswd`. Currently only bcrypt is supported. |
| `-static` | /tmp/organice/build | The directory that contains the built organice files.                                   |

```
gavin -davdir /tmp/org -htpass /tmp/.htpasswd -static /tmp/organice/build/
```

Now you can open your browser to
[http://localhost:8080/](http://localhost:8080/), sign in using the credentials
in the `.htpasswd` file, and org away!
