//go:build linux

package hostinventory

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopPeerReadsKernelUnixPeerCredentials(t *testing.T) {
	dir, err := os.MkdirTemp("", "dp-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "peer.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	client, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server := <-accepted
	defer server.Close()
	peer, err := desktopPeerForUnixConn(server)
	if err != nil {
		t.Fatal(err)
	}
	if peer.PID != os.Getpid() || peer.UID != uint32(os.Getuid()) {
		t.Fatalf("peer = %+v, want current process pid=%d uid=%d", peer, os.Getpid(), os.Getuid())
	}
}

func TestDesktopPeerRejectsNonUnixConnections(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	if _, err := desktopPeerForUnixConn(left); err == nil {
		t.Fatal("accepted non-Unix connection")
	}
}
