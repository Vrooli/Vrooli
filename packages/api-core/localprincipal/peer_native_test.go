//go:build linux || darwin || windows

package localprincipal

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

// This native integration test deliberately fails when the OS cannot supply
// kernel peer identity. A cross-build of this test is source proof only.
func TestKernelPeerIdentityMatchesBothConnectedProcesses(t *testing.T) {
	directory, err := os.MkdirTemp("", "peer-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(directory, "p.sock"), Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	connected := make(chan *net.UnixConn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := listener.AcceptUnix()
		if err != nil {
			acceptErr <- err
			return
		}
		connected <- conn
	}()
	client, err := net.DialUnix("unix", nil, listener.Addr().(*net.UnixAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var server *net.UnixConn
	select {
	case server = <-connected:
	case err := <-acceptErr:
		t.Fatal(err)
	}
	defer server.Close()
	expected, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	for _, conn := range []*net.UnixConn{client, server} {
		pid, err := PeerPID(conn)
		if err != nil || int(pid) != os.Getpid() {
			t.Fatalf("kernel peer identity: pid=%d err=%v", pid, err)
		}
		principal, err := Peer(conn)
		if err != nil || principal != expected {
			t.Fatalf("kernel peer principal mismatch: %v", err)
		}
	}
}
