//go:build linux

package securestore

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnlockLoginKeyringRefusesWithoutADaemon(t *testing.T) {
	// A private runtime directory with a live bus socket but no keyring control
	// socket. The bus must exist, otherwise the session repair would swap in
	// this user's real runtime directory and the test would talk to the host's
	// actual daemon.
	runtime := t.TempDir()
	if err := os.Chmod(runtime, 0o700); err != nil {
		t.Fatal(err)
	}
	bus, err := net.Listen("unix", filepath.Join(runtime, "bus"))
	if err != nil {
		t.Skipf("cannot create unix socket in temp dir: %v", err)
	}
	defer bus.Close()
	t.Setenv("XDG_RUNTIME_DIR", runtime)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(runtime, "bus"))
	if _, repaired := repairSession(); repaired {
		t.Fatal("test runtime directory was not accepted as this user's session; refusing to run against the real daemon")
	}

	err = UnlockLoginKeyring(context.Background(), strings.NewReader("not-a-real-passphrase\n"))
	if !errors.Is(err, ErrNoKeyringDaemon) {
		t.Fatalf("expected ErrNoKeyringDaemon, got %v", err)
	}
	if !strings.Contains(err.Error(), "second daemon") {
		t.Fatalf("error must explain the second-daemon hazard: %v", err)
	}
}

func TestKeyringDaemonListeningRequiresASocket(t *testing.T) {
	runtime := t.TempDir()
	environ := []string{"XDG_RUNTIME_DIR=" + runtime}
	if keyringDaemonListening(environ) {
		t.Fatal("empty runtime dir reported a listening daemon")
	}
	if err := os.MkdirAll(filepath.Join(runtime, "keyring"), 0o700); err != nil {
		t.Fatal(err)
	}
	// A plain file at the control path is not a daemon.
	if err := os.WriteFile(filepath.Join(runtime, keyringControlSocket), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if keyringDaemonListening(environ) {
		t.Fatal("a regular file was mistaken for the control socket")
	}
	if err := os.Remove(filepath.Join(runtime, keyringControlSocket)); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", filepath.Join(runtime, keyringControlSocket))
	if err != nil {
		t.Skipf("cannot create unix socket in temp dir: %v", err)
	}
	defer listener.Close()
	if !keyringDaemonListening(environ) {
		t.Fatal("a unix socket at the control path was not recognised")
	}
}
