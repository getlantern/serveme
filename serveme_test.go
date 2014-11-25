package serveme

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/testify/assert"
)

const (
	MESSAGE_TEXT = "Hello World"

	WADDELL_ADDR = "localhost:19543"
)

var (
	tlog = golog.LoggerFor("serveme-test")

	dialTimeout = 100 * time.Millisecond
)

func Test(t *testing.T) {
	numFilesAtStart := countTCPFiles()
	stuffToClose := make([]io.Closer, 0)
	defer func() {
		for _, closer := range stuffToClose {
			err := closer.Close()
			assert.NoError(t, err, "Should have been able to close: ", closer)
		}
		numFilesAtEnd := countTCPFiles()
		assert.Equal(t, numFilesAtStart, numFilesAtEnd, "Number of TCP file descriptors should have remained constant between start and end of test")
	}()

	dialer, err := At("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Unable to start dialer: %s", err)
	}
	dialer.Timeout = dialTimeout
	stuffToClose = append(stuffToClose, dialer)

	assert.NotEmpty(t, dialer.Addr().String(), "Dialer should report address")

	l1 := testListener(t, "Message 1")
	l2 := testListener(t, "Message 2")
	l3 := testListener(t, "Message 3")
	startSignaling(t, dialer, l1, l2, l3)

	conn1, err := dialer.Dial(1)
	if err != nil {
		t.Fatalf("Unable to dial server 1")
	}
	stuffToClose = append(stuffToClose, conn1)
	conn2, err := dialer.Dial(2)
	if err != nil {
		t.Fatalf("Unable to dial server 2")
	}
	stuffToClose = append(stuffToClose, conn2)
	_, err = dialer.Dial(3)
	assert.Error(t, err, "Dialing server 3 should have errored")
	if err != nil {
		assert.Contains(t, err.Error(), "timed out", "Error should have been timeout error")
	}

	dialer.connChsMutex.RLock()
	assert.Empty(t, dialer.connChs, "connChs should be empty after dialing")
	dialer.connChsMutex.RUnlock()

	b1 := make([]byte, 9)
	n1, err := io.ReadFull(conn1, b1)
	assert.NoError(t, err, "Reading from conn1 should have succeeded")
	assert.Equal(t, 9, n1, "Reading from conn1 should have resulted in 9 bytes")
	assert.Equal(t, []byte("Message 1"), b1[:n1], "Conn1 should have read 'Message 1'")

	b2 := make([]byte, 9)
	n2, err := io.ReadFull(conn2, b2)
	assert.NoError(t, err, "Reading from conn2 should have succeeded")
	assert.Equal(t, 9, n2, "Reading from conn2 should have resulted in 9 bytes")
	assert.Equal(t, []byte("Message 2"), b2[:n2], "Conn1 should have read 'Message 2'")
}

func TestRapidDialTimeout(t *testing.T) {
	numFilesAtStart := countTCPFiles()
	dialer, err := At("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Unable to start dialer: %s", err)
	}
	defer func() {
		dialer.Close()
		numFilesAtEnd := countTCPFiles()
		assert.Equal(t, numFilesAtStart, numFilesAtEnd, "Number of TCP file descriptors should have remained constant between start and end of test")
	}()

	dialer.Timeout = 1 * time.Nanosecond
	_, err = dialer.Dial(1)
	assert.Error(t, err, "Dialing should have errored")
	if err != nil {
		assert.Contains(t, err.Error(), "timed out", "Error should have been timeout error")
	}
}

func TestRapidListenerTimeout(t *testing.T) {
	numFilesAtStart := countTCPFiles()
	dialer, err := At("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Unable to start dialer: %s", err)
	}
	l := Listen()
	l.Timeout = 1 * time.Nanosecond
	go func() {
		dialer.Dial(nil)
	}()

	defer func() {
		dialer.Close()
		l.Close()
		numFilesAtEnd := countTCPFiles()
		assert.Equal(t, numFilesAtStart, numFilesAtEnd, "Number of TCP file descriptors should have remained constant between start and end of test")
	}()

	l.Requests <- <-dialer.Requests
	_, err = l.Accept()
	assert.Error(t, err, "Accept should have errored")
	if err != nil {
		assert.Contains(t, err.Error(), "timeout", "Error should have been timeout error")
	}
}

func testListener(t *testing.T, msg string) *Listener {
	l := Listen()
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Tracef("Unable to listen: %s", err)
				continue
			}
			log.Tracef("Writing message: %s", msg)
			conn.Write([]byte(msg))
			conn.Close()
		}
	}()
	return l
}

func startSignaling(t *testing.T, dialer *Dialer, l1 *Listener, l2 *Listener, l3 *Listener) {
	go func() {
		for req := range dialer.Requests {
			switch req.Server {
			case 1:
				tlog.Trace("Got request for listener 1")
				l1.Requests <- req
			case 2:
				tlog.Trace("Got request for listener 2")
				l2.Requests <- req
			case 3:
				tlog.Trace("Got request for listener 3, sleeping")
				time.Sleep(dialTimeout + 10*time.Millisecond)
				l2.Requests <- req
			default:
				t.Fatalf("Unknown server id: %d", req.Server)
			}
		}
	}()
}

// see https://groups.google.com/forum/#!topic/golang-nuts/c0AnWXjzNIA
func countTCPFiles() int {
	out, err := exec.Command("lsof", "-p", fmt.Sprintf("%v", os.Getpid())).Output()
	if err != nil {
		log.Fatal(err)
	}
	log.Tracef("lsof result: %s", string(out))
	return bytes.Count(out, []byte("TCP"))
}
