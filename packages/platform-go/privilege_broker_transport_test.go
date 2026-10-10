//go:build linux

package platform

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func brokerFixture(t *testing.T, handle func(net.Conn)) string {
	t.Helper()
	// Unix socket names have a small kernel limit. t.TempDir includes the full
	// test name and can exceed it when TMPDIR is the governed Go work root.
	directory, err := os.MkdirTemp("", "pb-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	socket := filepath.Join(directory, "broker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		handle(conn)
	}()
	return socket
}

func TestBrokerTransportReadsOneBoundedJSONResponse(t *testing.T) {
	request := map[string]string{"version": "v1", "action": TemporaryActivityObserveAction}
	socket := brokerFixture(t, func(conn net.Conn) {
		var received map[string]string
		if err := json.NewDecoder(conn).Decode(&received); err != nil {
			return
		}
		_ = json.NewEncoder(conn).Encode(map[string]string{"action": received["action"], "status": "completed"})
	})
	transport := PrivilegeBrokerTransport{SocketPath: socket, verifyPeer: func(net.Conn) error { return nil }}
	var response map[string]string
	if err := transport.Exchange(context.Background(), request, &response); err != nil || response["action"] != TemporaryActivityObserveAction {
		t.Fatalf("response=%v err=%v", response, err)
	}
}

func TestBrokerTransportCancellationClosesSilentPeer(t *testing.T) {
	entered := make(chan struct{})
	socket := brokerFixture(t, func(conn net.Conn) {
		var request any
		_ = json.NewDecoder(conn).Decode(&request)
		close(entered)
		var buf [1]byte
		_, _ = conn.Read(buf[:])
	})
	transport := PrivilegeBrokerTransport{SocketPath: socket, Timeout: time.Minute, verifyPeer: func(net.Conn) error { return nil }}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		var response any
		done <- transport.Exchange(ctx, map[string]string{"action": "inspect"}, &response)
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("peer not reached")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancellation accepted as successful response")
		}
	case <-time.After(time.Second):
		t.Fatal("canceled transport remained blocked")
	}
}

func TestBrokerTransportRefusesOversizedOrTrailingResponse(t *testing.T) {
	for _, data := range []string{strings.Repeat("x", 65), `{"status":"completed"} {"status":"failed"}`} {
		socket := brokerFixture(t, func(conn net.Conn) {
			var req any
			_ = json.NewDecoder(conn).Decode(&req)
			_, _ = conn.Write([]byte(data))
		})
		transport := PrivilegeBrokerTransport{SocketPath: socket, MaxResponseBytes: 64, verifyPeer: func(net.Conn) error { return nil }}
		var response any
		if err := transport.Exchange(context.Background(), map[string]string{"action": "inspect"}, &response); err == nil {
			t.Fatal("accepted oversized or extra response")
		}
	}
}

func TestBrokerTransportVerifiesRootPeer(t *testing.T) {
	socket := brokerFixture(t, func(conn net.Conn) {
		var request any
		_ = json.NewDecoder(conn).Decode(&request)
		_ = json.NewEncoder(conn).Encode(map[string]string{"status": "completed"})
	})
	transport := PrivilegeBrokerTransport{SocketPath: socket}
	var response any
	err := transport.Exchange(context.Background(), map[string]string{"action": "inspect"}, &response)
	// Fixtures run as the operator. A fake same-uid broker must not be trusted
	// for privileged process evidence; root CI can qualify the positive branch.
	if os.Geteuid() == 0 {
		if err != nil {
			t.Fatal(err)
		}
	} else if err == nil || !strings.Contains(err.Error(), "not root") {
		t.Fatalf("untrusted peer: %v", err)
	}
}
