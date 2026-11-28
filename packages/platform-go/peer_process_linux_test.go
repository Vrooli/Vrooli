//go:build linux

package platform

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestPeerProcessIDComesFromUnixSocketCredentials(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "peer-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "peer.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverResult := make(chan int, 1)
	serverErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		pid, err := PeerProcessID(conn)
		if err != nil {
			serverErr <- err
			return
		}
		serverResult <- pid
	}()

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
	select {
	case err := <-serverErr:
		t.Fatal(err)
	case got := <-serverResult:
		if got != os.Getpid() {
			t.Fatalf("peer PID=%d, want kernel-authenticated client PID %d", got, os.Getpid())
		}
	}
}

func TestPeerProcessIDRejectsTCPConnection(t *testing.T) {
	server, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	go func() {
		conn, err := server.Accept()
		if err == nil {
			defer conn.Close()
			_, _ = PeerProcessID(conn)
		}
	}()
	client, err := net.Dial("tcp", server.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := PeerProcessID(client); err == nil {
		t.Fatal("TCP connection supplied a peer process identity")
	}
}
