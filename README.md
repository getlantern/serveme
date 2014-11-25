serveme [![Travis CI Status](https://travis-ci.org/getlantern/serveme.svg?branch=master)](https://travis-ci.org/getlantern/serveme)&nbsp;[![Coverage Status](https://coveralls.io/repos/getlantern/serveme/badge.png)](https://coveralls.io/r/getlantern/serveme)&nbsp;[![GoDoc](https://godoc.org/github.com/getlantern/serveme?status.png)](http://godoc.org/github.com/getlantern/serveme)
==========
```golang
// package serveme implements a Dialer and a net.Listener that allow a client
// and server to communicate by having the server contact the client rather than
// the other way around. This is handy in situations where the client is
// reachable using an IP and port, but the server isn't (for example if the
// server is behind an impenetrable NAT).
//
// For this to work, the client and server must be able to communicate with each
// other via a signaling channel. When the client wants to connect to the
// server, it sends a message with the necessary connection information to the
// server, which then connects to the client.
//
// Right now, serveme only supports TCP.
```

To install:

`go get github.com/getlantern/serveme`

For docs:

`godoc github.com/getlantern/serveme`
