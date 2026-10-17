package cpkeys

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:BRG-P0-002] The control plane has a stable, persisted Ed25519 identity:
// a second load returns the SAME key (so paired nodes' pinned key keeps
// verifying across restarts), and the private seed is written owner-only.
func TestLoadOrCreate_StableAcrossLoads(t *testing.T) {
	dir := t.TempDir()

	first, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	second, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if first.PublicKeyBase64() != second.PublicKeyBase64() {
		t.Fatalf("public key changed across loads: %s != %s", first.PublicKeyBase64(), second.PublicKeyBase64())
	}

	info, err := os.Stat(filepath.Join(dir, keyFileName))
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("key file perm = %o, want 0600 (owner-only secret)", perm)
	}
}

// [REQ:BRG-P0-002] A node that pins the control plane's published public key
// can verify the control plane's signatures — the basis for rejecting an
// impostor coordinator.
func TestSign_VerifiableWithPublishedPublicKey(t *testing.T) {
	kp, err := LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	msg := []byte("control-plane push frame")
	sig := kp.Sign(msg)

	pubBytes, err := base64.StdEncoding.DecodeString(kp.PublicKeyBase64())
	if err != nil {
		t.Fatalf("decode published key: %v", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), msg, sig) {
		t.Fatal("signature did not verify against the published public key")
	}
	if ed25519.Verify(ed25519.PublicKey(pubBytes), []byte("tampered"), sig) {
		t.Fatal("signature verified against a tampered message")
	}
}

func TestLoadOrCreate_RejectsMalformedKeyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, keyFileName), []byte("too short"), 0o600); err != nil {
		t.Fatalf("seed bad file: %v", err)
	}
	if _, err := LoadOrCreate(dir); err == nil {
		t.Fatal("expected error loading a malformed key file")
	}
}
