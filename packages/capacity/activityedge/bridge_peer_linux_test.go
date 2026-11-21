//go:build linux

package activityedge

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestBridgePeerRequiresExactActivityEdgePIDAndOwnerUID(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "activity-peer-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "peer.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	clientDone := make(chan net.Conn, 1)
	go func() { c, _ := net.Dial("unix", path); clientDone <- c }()
	server, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client := <-clientDone
	if client == nil {
		t.Fatal("client connection was not created")
	}
	defer client.Close()
	if err := authorizeUnixPeer(server, os.Getpid(), os.Getuid()); err != nil {
		t.Fatalf("exact edge identity rejected: %v", err)
	}
	if err := authorizeUnixPeer(server, os.Getpid()+1, os.Getuid()); err == nil {
		t.Fatal("different process was accepted as activity edge")
	}
	if err := authorizeUnixPeer(server, os.Getpid(), os.Getuid()+1); err == nil {
		t.Fatal("different owner UID was accepted as activity edge")
	}
}
