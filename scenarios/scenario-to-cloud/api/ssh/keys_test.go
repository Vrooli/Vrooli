package ssh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakePEMKeyHeader constructs a test-only PEM marker without embedding a
// private-key fixture in the repository or triggering secret scanners.
func fakePEMKeyHeader(parts ...string) string {
	return "-----BEGIN " + strings.Join(parts, " ") + " KEY-----\nfake"
}

// fakeCommandRunner provides controllable command execution for key tests.
type fakeCommandRunner struct {
	responses map[string]struct {
		stdout []byte
		stderr []byte
		err    error
	}
}

func (f *fakeCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	// Build a lookup key from the command name and first arg
	key := name
	if len(args) > 0 {
		key += " " + args[0]
	}
	if resp, ok := f.responses[key]; ok {
		return resp.stdout, resp.stderr, resp.err
	}
	return nil, nil, nil
}

func TestDiscoverKeys_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ks := NewKeyService(&fakeCommandRunner{}, dir)

	keys, err := ks.DiscoverKeys()
	if err != nil {
		t.Fatalf("DiscoverKeys() error = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}

func TestDiscoverKeys_SkipsNonKeyFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create files that should be skipped
	for _, name := range []string{"known_hosts", "known_hosts.old", "config", "authorized_keys", "environment", "id_ed25519.pub"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	ks := NewKeyService(&fakeCommandRunner{}, dir)

	keys, err := ks.DiscoverKeys()
	if err != nil {
		t.Fatalf("DiscoverKeys() error = %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys (all skipped), got %d", len(keys))
	}
}

func TestDiscoverKeys_ParsesKeyTypes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create mock key files with PEM headers
	tests := []struct {
		filename string
		header   string
		wantType KeyType
	}{
		{"id_rsa", fakePEMKeyHeader("RSA", "PRIVATE"), KeyTypeRSA},
		{"id_dsa", fakePEMKeyHeader("DSA", "PRIVATE"), KeyTypeDSA},
		{"id_ecdsa", fakePEMKeyHeader("EC", "PRIVATE"), KeyTypeECDSA},
	}

	for _, tt := range tests {
		if err := os.WriteFile(filepath.Join(dir, tt.filename), []byte(tt.header), 0o600); err != nil {
			t.Fatal(err)
		}
		// Create a corresponding .pub file (needed for fingerprint but won't parse)
		if err := os.WriteFile(filepath.Join(dir, tt.filename+".pub"), []byte("ssh-rsa AAAA... test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	ks := NewKeyService(&fakeCommandRunner{}, dir)
	keys, err := ks.DiscoverKeys()
	if err != nil {
		t.Fatalf("DiscoverKeys() error = %v", err)
	}
	if len(keys) != len(tests) {
		t.Fatalf("expected %d keys, got %d", len(tests), len(keys))
	}

	// Build a map by filename for easy lookup
	keyMap := make(map[string]KeyInfo)
	for _, k := range keys {
		keyMap[filepath.Base(k.Path)] = k
	}

	for _, tt := range tests {
		k, ok := keyMap[tt.filename]
		if !ok {
			t.Errorf("key %s not found", tt.filename)
			continue
		}
		if k.Type != tt.wantType {
			t.Errorf("key %s: type = %q, want %q", tt.filename, k.Type, tt.wantType)
		}
	}
}

func TestDeleteKey_ProtectsSpecialFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ks := NewKeyService(&fakeCommandRunner{}, dir)

	for _, name := range []string{"authorized_keys", "known_hosts", "config"} {
		resp := ks.DeleteKey(DeleteKeyRequest{KeyPath: filepath.Join(dir, name)})
		if resp.OK {
			t.Errorf("DeleteKey(%s) should have failed for special file", name)
		}
		if resp.Status != StatusInvalidInput {
			t.Errorf("DeleteKey(%s) status = %q, want %q", name, resp.Status, StatusInvalidInput)
		}
	}
}

func TestDeleteKey_RejectsPathTraversal(t *testing.T) {
	t.Parallel()

	ks := NewKeyService(&fakeCommandRunner{}, "")

	resp := ks.DeleteKey(DeleteKeyRequest{KeyPath: "/tmp/../etc/passwd"})
	if resp.OK {
		t.Error("DeleteKey with path traversal should have failed")
	}
}

func TestDeleteKey_DeletesPrivateAndPublic(t *testing.T) {
	t.Parallel()

	// Use a subdirectory of ~/.ssh to pass ValidateSSHPath
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	sshDir := filepath.Join(homeDir, ".ssh")
	testDir := filepath.Join(sshDir, "test_delete_keys_"+t.Name())
	if err := os.MkdirAll(testDir, 0o700); err != nil {
		t.Fatalf("create test dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(testDir) })

	keyPath := filepath.Join(testDir, "test_key")
	pubPath := keyPath + ".pub"

	if err := os.WriteFile(keyPath, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pubPath, []byte("public"), 0o600); err != nil {
		t.Fatal(err)
	}

	ks := NewKeyService(&fakeCommandRunner{}, testDir)
	resp := ks.DeleteKey(DeleteKeyRequest{KeyPath: keyPath})
	if !resp.OK {
		t.Fatalf("DeleteKey failed: %s", resp.Message)
	}
	if !resp.PrivateDeleted {
		t.Error("expected PrivateDeleted = true")
	}
	if !resp.PublicDeleted {
		t.Error("expected PublicDeleted = true")
	}

	// Verify files are gone
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Error("private key still exists")
	}
	if _, err := os.Stat(pubPath); !os.IsNotExist(err) {
		t.Error("public key still exists")
	}
}

func TestReadPublicKey_Success(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	sshDir := filepath.Join(homeDir, ".ssh")
	testDir := filepath.Join(sshDir, "test_read_pubkey_"+t.Name())
	if err := os.MkdirAll(testDir, 0o700); err != nil {
		t.Fatalf("create test dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(testDir) })

	keyPath := filepath.Join(testDir, "test_key")
	pubPath := keyPath + ".pub"

	pubContent := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITest test@host"
	if err := os.WriteFile(pubPath, []byte(pubContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte(fakePEMKeyHeader("OPENSSH", "PRIVATE")), 0o600); err != nil {
		t.Fatal(err)
	}

	ks := NewKeyService(&fakeCommandRunner{}, testDir)
	pub, _, err := ks.ReadPublicKey(keyPath)
	if err != nil {
		t.Fatalf("ReadPublicKey() error = %v", err)
	}
	if pub != pubContent {
		t.Errorf("ReadPublicKey() = %q, want %q", pub, pubContent)
	}
}
