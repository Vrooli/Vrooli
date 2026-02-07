package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestTOFUHostKeyCallback_AcceptsUnknownAndPersists(t *testing.T) {
	t.Parallel()

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(knownHostsPath, []byte{}, 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}

	callback, err := NewTOFUHostKeyCallbackForPath("example.com", 22, knownHostsPath)
	if err != nil {
		t.Fatalf("NewTOFUHostKeyCallbackForPath: %v", err)
	}

	pub := testPublicKey(t)
	remote := &net.TCPAddr{IP: net.ParseIP("203.0.113.10"), Port: 22}
	if err := callback("example.com:22", remote, pub); err != nil {
		t.Fatalf("callback should accept unknown host: %v", err)
	}

	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if !strings.Contains(string(content), "example.com") {
		t.Fatalf("known_hosts should contain example.com entry, got: %q", string(content))
	}
}

func TestTOFUHostKeyCallback_RejectsChangedHostKey(t *testing.T) {
	t.Parallel()

	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	key1 := testPublicKey(t)
	entry := knownhosts.Line([]string{knownhosts.Normalize("example.com:22")}, key1) + "\n"
	if err := os.WriteFile(knownHostsPath, []byte(entry), 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}

	callback, err := NewTOFUHostKeyCallbackForPath("example.com", 22, knownHostsPath)
	if err != nil {
		t.Fatalf("NewTOFUHostKeyCallbackForPath: %v", err)
	}

	key2 := testPublicKey(t)
	remote := &net.TCPAddr{IP: net.ParseIP("203.0.113.10"), Port: 22}
	if err := callback("example.com:22", remote, key2); err == nil {
		t.Fatal("callback should reject changed host key")
	}
}

func testPublicKey(t *testing.T) gossh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	sshPub, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("ssh.NewPublicKey: %v", err)
	}
	return sshPub
}
