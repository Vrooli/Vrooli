//go:build linux || darwin

package credentialunlock

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func shortSocket(t *testing.T) string {
	t.Helper()
	// unix socket paths are limited to ~104 bytes on Darwin; t.TempDir can be longer.
	dir, err := os.MkdirTemp("", "cu")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "state", SocketName)
}

func serve(t *testing.T, socket string, value []byte) {
	t.Helper()
	listener, err := Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = Serve(ctx, listener, func() ([]byte, bool) { return value, len(value) > 0 }) }()
}

// TestTheAgentHandsTheStorePassphraseToItsOwnUser is the reboot path: the
// agent holds the passphrase the control plane delivered, and a local Vrooli
// process running as the same user receives it.
func TestTheAgentHandsTheStorePassphraseToItsOwnUser(t *testing.T) {
	socket := shortSocket(t)
	serve(t, socket, []byte("held-in-memory"))

	got, err := RequestPassphrase(context.Background(), socket)
	if err != nil || got != "held-in-memory" {
		t.Fatalf("RequestPassphrase() = %q, %v", got, err)
	}
	info, err := os.Stat(socket)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode = %v (%v), want 0600", info.Mode().Perm(), err)
	}
}

func TestAnAgentThatHoldsNothingYetIsUnavailable(t *testing.T) {
	socket := shortSocket(t)
	serve(t, socket, nil)
	_, err := RequestPassphrase(context.Background(), socket)
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "not delivered") {
		t.Fatalf("err = %v, want unavailable naming the missing delivery", err)
	}
}

// A store open on a host with no agent must fail fast, not hang.
func TestAMissingAgentFailsFast(t *testing.T) {
	start := time.Now()
	_, err := RequestPassphrase(context.Background(), shortSocket(t))
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("took %v; a missing agent must not stall a store open", time.Since(start))
	}
}

// Anyone able to write the socket's directory could plant their own socket and
// choose the answer, so the client refuses such a location outright.
func TestTheClientRefusesASocketOthersCouldReplace(t *testing.T) {
	socket := shortSocket(t)
	serve(t, socket, []byte("held-in-memory"))
	if err := os.Chmod(filepath.Dir(socket), 0o777); err != nil { // #nosec G302 -- the test's hostile directory.
		t.Fatal(err)
	}
	if _, err := RequestPassphrase(context.Background(), socket); !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "writable by other users") {
		t.Fatalf("err = %v, want a refusal of the world-writable directory", err)
	}
}

func TestListenRefusesToReplaceARegularFile(t *testing.T) {
	socket := shortSocket(t)
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(socket, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(socket); err == nil {
		t.Fatal("Listen replaced a regular file")
	}
}

func TestAMalformedRequestIsRefused(t *testing.T) {
	socket := shortSocket(t)
	serve(t, socket, []byte("held-in-memory"))
	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("{\"version\":1,\"op\":\"dump-everything\"}\n")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256)
	n, _ := conn.Read(buf)
	if strings.Contains(string(buf[:n]), "held-in-memory") || !strings.Contains(string(buf[:n]), "invalid") {
		t.Fatalf("reply = %q, want a refusal without the passphrase", buf[:n])
	}
}
