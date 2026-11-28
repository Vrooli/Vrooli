package sshcore

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

func TestTOFUAcceptsUnknownAndRejectsChangedKey(t *testing.T) {
	_, privateA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, privateB, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signerA, err := gossh.NewSignerFromKey(privateA)
	if err != nil {
		t.Fatal(err)
	}
	signerB, err := gossh.NewSignerFromKey(privateB)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "known_hosts")
	callback, err := NewTOFUHostKeyCallback("example.com", 22, path)
	if err != nil {
		t.Fatal(err)
	}
	remote := &net.TCPAddr{IP: net.ParseIP("203.0.113.10"), Port: 22}
	if err := callback("example.com:22", remote, signerA.PublicKey()); err != nil {
		t.Fatal(err)
	}
	if err := callback("example.com:22", remote, signerA.PublicKey()); err != nil {
		t.Fatal(err)
	}
	if err := callback("example.com:22", remote, signerB.PublicKey()); err == nil {
		t.Fatal("changed host key was accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("TOFU did not persist a host key")
	}
}
