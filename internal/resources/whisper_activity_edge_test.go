package resources

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveStaleWhisperControlSocketRequiresDeadOwnedSocket(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "whisper-edge-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	deadPID := 1_000_000_000

	t.Run("stale socket", func(t *testing.T) {
		path := filepath.Join(root, "stale.sock")
		listener, err := net.Listen("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		if err := listener.Close(); err != nil {
			t.Fatal(err)
		}
		if err := removeStaleWhisperControlSocket(path, deadPID); err != nil {
			t.Fatalf("remove stale socket: %v", err)
		}
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("stale socket still exists, stat err=%v", err)
		}
	})

	t.Run("live socket", func(t *testing.T) {
		path := filepath.Join(root, "live.sock")
		listener, err := net.Listen("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		if err := removeStaleWhisperControlSocket(path, deadPID); err == nil {
			t.Fatal("live control socket was removed")
		}
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("live socket disappeared: %v", err)
		}
	})

	t.Run("regular file", func(t *testing.T) {
		path := filepath.Join(root, "regular.sock")
		if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := removeStaleWhisperControlSocket(path, deadPID); err == nil {
			t.Fatal("regular file at the socket path was removed")
		}
		if data, err := os.ReadFile(path); err != nil || string(data) != "keep" {
			t.Fatalf("regular path changed: %q, %v", data, err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		target := filepath.Join(root, "target")
		path := filepath.Join(root, "link.sock")
		if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if err := removeStaleWhisperControlSocket(path, deadPID); err == nil {
			t.Fatal("symlink at the socket path was removed")
		}
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("symlink disappeared: %v", err)
		}
	})
}

func TestWhisperEdgeRecoveryRequiresExactExecutableAndArgv(t *testing.T) {
	if whisperEdgeProcessOwned(os.Getpid(), "/bin/false", "/tmp/not-this-process.sock", []string{"activity-edge", "--native"}) {
		t.Fatal("unrelated live process passed Whisper activity-edge ownership proof")
	}
}
