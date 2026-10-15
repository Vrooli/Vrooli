//go:build linux

package finiteinstallation

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

func TestDisposableTrustedAnchorReadAndRefusals(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pm.json")
	if e := os.WriteFile(path, []byte(`{"disposable":"only"}`), 0644); e != nil {
		t.Fatal(e)
	}
	fd, e := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer unix.Close(fd)
	raw, e := readFromTrustedAnchor(fd, "pm.json", uint32(os.Geteuid()))
	if e != nil || string(raw) != `{"disposable":"only"}` {
		t.Fatalf("read: %v", e)
	}
	if e = os.Chmod(path, 0664); e != nil {
		t.Fatal(e)
	}
	if _, e = readFromTrustedAnchor(fd, "pm.json", uint32(os.Geteuid())); e == nil {
		t.Fatal("group-writable mode accepted")
	}
	if e = os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	if e = os.Symlink(path, filepath.Join(dir, "am.json")); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"missing.json", "am.json", "../pm.json"} {
		if _, e = readFromTrustedAnchor(fd, name, uint32(os.Geteuid())); e == nil {
			t.Fatalf("invalid %s accepted", name)
		}
	}
	if _, e = readFromTrustedAnchor(fd, "pm.json", uint32(os.Geteuid())+1); e == nil {
		t.Fatal("wrong owner accepted")
	}
}
