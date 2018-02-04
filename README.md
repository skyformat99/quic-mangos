# QUIC-mangos

A QUIC transport for [mangos](https://github.com/go-mangos/mangos) written in pure Go

[![Godoc Reference](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/lthibault/portal)

## Motivation

QUIC-mangos brings the low latency and multiplexed streaming of the [QUIC](https://en.wikipedia.org/wiki/QUIC#Details) protocol to mangos.

URL paths passed to `sock.Listen` and `sock.Dial` are mapped to a separate QUIC
stream, allowing several `mangos.Socket`s to share a single port mapping.

Thus two sockets can dial/listen to `quic://127.0.0.1:9001/foo/bar` and
`quic://127.0.0.1:9001/foo/qux`, respectively, and they will share both an
underlying QUIC session (roughly equivalent to a TCP connection) and an OS port.

Moreover, QUIC is designed with the modern web in mind and performs significantly
better than TCP over lossy connections.  It also features mandatory TLS
encryption, which is configruable via socket options.

## Usage

QUIC-mangos can be installed via the standard go toolchain:

```bash
go get -u github.com/lthibault/quic-mangos
```

The QUIC transport adheres to the public API for mangos transports.

```go
import (
    // ...
    "github.com/lthibault/quic-mangos"
)

// set up a mangos.Socket the usual way

sock.AddTransport(quic.NewTransport())

_ = sock.Listen("quic://127.0.0.1:9001/foo/bar")

```


