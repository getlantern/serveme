package serveme

import (
	"io"
	elog "log"
	"os"

	serveme "."
)

func ExampleAt() {
	dialer, err := serveme.At("tcp", "localhost:0")
	if err != nil {
		elog.Fatalf("Unable to start dialer: %s", err)
	}
	defer dialer.Close()

	// Process outbound requests to signaling channel
	go func() {
		for req := range dialer.Requests {
			// TODO: Send request to server via signaling channel
			elog.Printf("Sent: %s", req)
		}
	}()

	conn := dialer.Dial("server's id on signaling channel")
	defer conn.Close()
	io.Copy(os.Stdout, conn)
}

func ExampleListen() {
	l := serveme.Listen()

	// Process inbound requests from signaling channel
	go func() {
		// TODO: read request from signaling channel
		var req *serveme.Request
		l.Requests <- req
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			elog.Fatalf("Unable to accept: %s", err)
		}
		conn.Write([]byte("Here's a message for you"))
		conn.Close()
	}
}

// func ExampleAnswer() {
// 	t := Answer()
// 	defer t.Close()

// 	// Process outbound messages
// 	go func() {
// 		for {
// 			msg, done := t.NextMsgOut()
// 			if done {
// 				return
// 			}
// 			// TODO: Send message to peer via signaling channel
// 			elog.Printf("Sent: %s", msg)
// 		}
// 	}()

// 	// Process inbound messages
// 	go func() {
// 		for {
// 			// TODO: Get message from signaling channel
// 			var msg string
// 			t.MsgIn(msg)
// 		}
// 	}()

// 	fiveTuple, err := t.FiveTupleTimeout(15 * time.Second)
// 	if err != nil {
// 		elog.Fatal(err)
// 	}

// 	local, _, err := fiveTuple.UDPAddrs()
// 	if err != nil {
// 		elog.Fatal(err)
// 	}

// 	conn, err := net.ListenUDP("udp", local)
// 	if err != nil {
// 		elog.Fatal(err)
// 	}

// 	// TODO: signal to peer that we're ready to receive traffic

// 	b := make([]byte, 1024)
// 	for {
// 		n, addr, err := conn.ReadFrom(b)
// 		if err != nil {
// 			elog.Fatal(err)
// 		}

// 		elog.Printf("Received message '%s' from %s", string(b[:n]), addr)
// 	}
// }
