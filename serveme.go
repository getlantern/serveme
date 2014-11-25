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
package serveme

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/getlantern/buuid"
	"github.com/getlantern/framed"
	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("serveme")
)

// ServerId is an opaque identifier for a server used by the signaling channel.
type ServerId interface{}

// Request is a request from a Client to a Server to establish a connection,
// including a GUID identifying that connection.
type Request struct {
	Server  ServerId
	ID      buuid.ID
	Network string
	Address string
}

// Dialer provides a mechanism for dialing servers by ServerId.
type Dialer struct {
	// Requests is a channel on which requests to dial this dialer are posted.
	// The signaling channel should forward these to the server identified by
	// the Server parameter.
	Requests     <-chan *Request
	requests     chan *Request
	network      string
	connChs      map[buuid.ID]chan<- net.Conn
	connChsMutex sync.RWMutex
	l            net.Listener
}

// At constructs a Dialer that listens at the given network and address.
func At(network, address string) (*Dialer, error) {
	if !strings.Contains(network, "tcp") {
		panic("Only tcp is currently supported")
	}
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	requests := make(chan *Request, 1000)
	d := &Dialer{
		Requests: requests,
		requests: requests,
		network:  network,
		connChs:  make(map[buuid.ID]chan<- net.Conn),
		l:        l,
	}
	go d.run()
	return d, nil
}

// Addr returns the address at which this Dialer is listening for connections
// from servers.
func (d *Dialer) Addr() net.Addr {
	return d.l.Addr()
}

// Dial dials the server at the given ServerId.
func (d *Dialer) Dial(server ServerId) net.Conn {
	id := buuid.Random()
	connCh := make(chan net.Conn)

	log.Tracef("Registering to receive connection for id: %s", id)
	d.connChsMutex.Lock()
	d.connChs[id] = connCh
	d.connChsMutex.Unlock()

	log.Tracef("Sending request for id: %s", id)
	d.requests <- &Request{
		Server:  server,
		ID:      id,
		Network: d.network,
		Address: d.l.Addr().String(),
	}

	log.Tracef("Waiting for connection for id: %s", id)
	conn := <-connCh
	d.connChsMutex.Lock()
	delete(d.connChs, id)
	d.connChsMutex.Unlock()
	return conn
}

// Close closes this dialer.
func (d *Dialer) Close() error {
	close(d.requests)
	return d.l.Close()
}

func (d *Dialer) run() {
	b := make([]byte, 16)
	for {
		conn, err := d.l.Accept()
		if err != nil {
			log.Tracef("Unable to accept: %s", err)
			return
		}
		fr := framed.NewReader(conn)
		_, err = fr.Read(b)
		if err != nil {
			log.Tracef("Unable to read conn id bytes: %s", err)
			conn.Close()
			continue
		}
		id, err := buuid.Read(b)
		if err != nil {
			log.Tracef("Unable to read conn id: %s", err)
			conn.Close()
			continue
		}
		log.Tracef("Making conn %s available", id)
		d.connChs[id] <- conn
	}
}

// Listener implements the net.Listener interface.
type Listener struct {
	Requests chan<- *Request
	requests chan *Request
}

// Listen constructs a Listener that listens by responding to connection
// requests sent to its Requests channel.
//
// Note - the Addr() method on the returned Listener is unimplemented and will
// panic if called.
func Listen() *Listener {
	requests := make(chan *Request, 1000)
	return &Listener{
		Requests: requests,
		requests: requests,
	}
}

// Accept implements the method from net.Listener.
func (l *Listener) Accept() (net.Conn, error) {
	req := <-l.requests
	log.Tracef("Dialing %s %s", req.Network, req.Address)
	conn, err := net.Dial(req.Network, req.Address)
	if err != nil {
		return nil, err
	}
	fw := framed.NewWriter(conn)

	log.Trace("Writing connection id to identify connection")
	_, err = fw.Write(req.ID.ToBytes())
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("Unable to write connection id: %s", err)
	}
	return conn, nil
}

// Close implements the method from net.Listener.
func (l *Listener) Close() error {
	return nil
}

// Addr implements the method from net.Listener but panics, since there's no
// implementation actually available.
func (l *Listener) Addr() net.Addr {
	panic("Method Addr() not implemented!")
}
