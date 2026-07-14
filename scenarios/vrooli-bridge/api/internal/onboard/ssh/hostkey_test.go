package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

func newHostPub(t *testing.T) gossh.PublicKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	signer, err := gossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return signer.PublicKey()
}

func TestTOFUAcceptsUnknownThenRejectsChangedHostKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts")

	keyA := newHostPub(t)
	keyB := newHostPub(t)

	const host = "127.0.0.1"
	const port = 22
	hostname := net.JoinHostPort(host, "22")
	remote := &net.TCPAddr{IP: net.ParseIP(host), Port: port}

	// First contact with an unknown host: trust on first use and persist.
	cb1, err := newTOFUHostKeyCallback(host, port, path)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if err := cb1(hostname, remote, keyA); err != nil {
		t.Fatalf("first contact should be accepted (TOFU), got %v", err)
	}

	// Known-hosts file now exists 0600 and holds the key.
	assertMode(t, path, 0o600)
	data, _ := os.ReadFile(path)
	if len(data) == 0 {
		t.Fatal("known_hosts should have been persisted")
	}

	// A fresh callback (re-loaded state) accepts the same key...
	cb2, err := newTOFUHostKeyCallback(host, port, path)
	if err != nil {
		t.Fatalf("callback reload: %v", err)
	}
	if err := cb2(hostname, remote, keyA); err != nil {
		t.Errorf("known matching key should be accepted, got %v", err)
	}

	// ...but rejects a CHANGED host key for the same host.
	if err := cb2(hostname, remote, keyB); err == nil {
		t.Error("changed host key must be rejected")
	}

	// The recorded fingerprint is discoverable for the stored host.
	if fp := hostFingerprint(path, host, port); fp != gossh.FingerprintSHA256(keyA) {
		t.Errorf("hostFingerprint = %q, want %q", fp, gossh.FingerprintSHA256(keyA))
	}
}

func TestEnsureKnownHostsFileCreatesDirAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "known_hosts")
	if err := ensureKnownHostsFile(path); err != nil {
		t.Fatalf("ensureKnownHostsFile: %v", err)
	}
	assertMode(t, filepath.Dir(path), 0o700)
	assertMode(t, path, 0o600)
}
