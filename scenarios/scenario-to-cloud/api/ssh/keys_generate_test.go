package ssh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// capturedCommandRunner wraps a CommandRunner and captures the last args.
type capturedCommandRunner struct {
	lastArgs []string
	allCalls [][]string
}

func (c *capturedCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	c.lastArgs = args
	c.allCalls = append(c.allCalls, append([]string{name}, args...))
	return nil, []byte("ssh-keygen not in test env"), fmt.Errorf("ssh-keygen not in test env")
}

func TestGenerateKey_Ed25519Defaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cmdRunner := &capturedCommandRunner{}
	ks := NewKeyService(cmdRunner, dir)

	_, err := ks.GenerateKey(GenerateKeyRequest{
		Type: KeyTypeEd25519,
	})

	// ssh-keygen won't actually succeed in test, but we can verify the args
	if err == nil {
		t.Skip("ssh-keygen unexpectedly succeeded")
	}

	if len(cmdRunner.lastArgs) == 0 {
		t.Fatal("no args captured")
	}

	// Verify -t ed25519 is present
	found := false
	for i, arg := range cmdRunner.lastArgs {
		if arg == "-t" && i+1 < len(cmdRunner.lastArgs) && cmdRunner.lastArgs[i+1] == "ed25519" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected -t ed25519 in args, got %v", cmdRunner.lastArgs)
	}

	// Verify default filename
	foundFilename := false
	for i, arg := range cmdRunner.lastArgs {
		if arg == "-f" && i+1 < len(cmdRunner.lastArgs) {
			if strings.HasSuffix(cmdRunner.lastArgs[i+1], "id_ed25519") {
				foundFilename = true
			}
			break
		}
	}
	if !foundFilename {
		t.Errorf("expected default filename id_ed25519 in args, got %v", cmdRunner.lastArgs)
	}
}

func TestGenerateKey_RSAWithBits(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cmdRunner := &capturedCommandRunner{}
	ks := NewKeyService(cmdRunner, dir)

	_, err := ks.GenerateKey(GenerateKeyRequest{
		Type: KeyTypeRSA,
		Bits: 4096,
	})

	if err == nil {
		t.Skip("ssh-keygen unexpectedly succeeded")
	}

	// Verify -b 4096 is present
	found := false
	for i, arg := range cmdRunner.lastArgs {
		if arg == "-b" && i+1 < len(cmdRunner.lastArgs) && cmdRunner.lastArgs[i+1] == "4096" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected -b 4096 in args, got %v", cmdRunner.lastArgs)
	}
}

func TestGenerateKey_RejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ks := NewKeyService(&fakeCommandRunner{}, dir)

	_, err := ks.GenerateKey(GenerateKeyRequest{
		Type: KeyTypeDSA,
	})
	if err == nil {
		t.Error("expected error for unsupported key type DSA")
	}
	if !strings.Contains(err.Error(), "key type must be") {
		t.Errorf("error should mention key type requirement, got: %v", err)
	}

	_, err = ks.GenerateKey(GenerateKeyRequest{
		Type: KeyTypeECDSA,
	})
	if err == nil {
		t.Error("expected error for unsupported key type ECDSA")
	}
}

func TestGenerateKey_RejectsExistingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	existingKey := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(existingKey, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	ks := NewKeyService(&fakeCommandRunner{}, dir)
	_, err := ks.GenerateKey(GenerateKeyRequest{
		Type:     KeyTypeEd25519,
		Filename: "id_ed25519",
	})
	if err == nil {
		t.Error("expected error for existing key file")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention key already exists, got: %v", err)
	}
}

func TestGenerateKey_InvalidFilename(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ks := NewKeyService(&fakeCommandRunner{}, dir)

	tests := []struct {
		name     string
		filename string
	}{
		{"path separator", "sub/key"},
		{"dot-dot", "..sneaky"},
		{"starts with dot", ".hidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ks.GenerateKey(GenerateKeyRequest{
				Type:     KeyTypeEd25519,
				Filename: tt.filename,
			})
			if err == nil {
				t.Errorf("expected error for filename %q", tt.filename)
			}
		})
	}
}
