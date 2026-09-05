package ssh

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverKeysParsesCandidatesAndSkipsUnsafeFiles(t *testing.T) {
	sshDir := t.TempDir()
	keygen := writeExecutable(t, "ssh-keygen", "#!/bin/sh\nprintf '%s\\n' '2048 SHA256:testfp sample@example.com (RSA)'\n")
	platform := &FakePlatform{
		SSHDirPath:   sshDir,
		HomeDirPath:  filepath.Dir(sshDir),
		SSHKeygenBin: keygen,
	}

	writeFile(t, filepath.Join(sshDir, "id_rsa"), "-----BEGIN RSA PRIVATE KEY-----\nbody\n")
	writeFile(t, filepath.Join(sshDir, "id_rsa.pub"), "ssh-rsa AAA sample@example.com\n")
	writeFile(t, filepath.Join(sshDir, "config"), "Host github.com\n")
	writeFile(t, filepath.Join(sshDir, "notes"), "not a private key\n")
	if err := os.Mkdir(filepath.Join(sshDir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	keys, err := DiscoverKeys(platform, sshDir)
	if err != nil {
		t.Fatalf("DiscoverKeys() error = %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("DiscoverKeys() returned %d keys, want 1: %#v", len(keys), keys)
	}

	got := keys[0]
	if got.Filename != "id_rsa" {
		t.Fatalf("Filename = %q, want id_rsa", got.Filename)
	}
	if got.Type != KeyTypeRSA {
		t.Fatalf("Type = %q, want rsa", got.Type)
	}
	if got.Bits != 2048 {
		t.Fatalf("Bits = %d, want 2048", got.Bits)
	}
	if got.Fingerprint != "SHA256:testfp" {
		t.Fatalf("Fingerprint = %q, want SHA256:testfp", got.Fingerprint)
	}
	if got.Comment != "sample@example.com" {
		t.Fatalf("Comment = %q, want sample@example.com", got.Comment)
	}
	if !got.HasPublic {
		t.Fatal("HasPublic = false, want true")
	}
}

func TestDiscoverKeysHandlesMissingAndInvalidDirectories(t *testing.T) {
	platform := &FakePlatform{SSHDirPath: t.TempDir()}

	keys, err := DiscoverKeys(platform, filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("DiscoverKeys(missing) error = %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("DiscoverKeys(missing) returned %d keys, want 0", len(keys))
	}

	notDir := filepath.Join(t.TempDir(), "file")
	writeFile(t, notDir, "content")
	if _, err := DiscoverKeys(platform, notDir); err == nil {
		t.Fatal("DiscoverKeys(file) error = nil, want error")
	}
}

func TestReadPublicKeyExpandsPathAndReturnsFingerprint(t *testing.T) {
	homeDir := t.TempDir()
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.Mkdir(sshDir, 0o700); err != nil {
		t.Fatalf("mkdir ssh dir: %v", err)
	}
	keygen := writeExecutable(t, "ssh-keygen", "#!/bin/sh\nprintf '%s\\n' '256 SHA256:pubfp github-key (ED25519)'\n")
	platform := &FakePlatform{
		SSHDirPath:   sshDir,
		HomeDirPath:  homeDir,
		SSHKeygenBin: keygen,
	}
	writeFile(t, filepath.Join(sshDir, "id_ed25519.pub"), "ssh-ed25519 AAA github-key\n")

	publicKey, fingerprint, err := ReadPublicKey(platform, "~/.ssh/id_ed25519")
	if err != nil {
		t.Fatalf("ReadPublicKey() error = %v", err)
	}
	if publicKey != "ssh-ed25519 AAA github-key" {
		t.Fatalf("publicKey = %q, want trimmed public key", publicKey)
	}
	if fingerprint != "SHA256:pubfp" {
		t.Fatalf("fingerprint = %q, want SHA256:pubfp", fingerprint)
	}
}

func TestBuildKeygenArgs(t *testing.T) {
	tests := []struct {
		name string
		req  GenerateKeyRequest
		want []string
	}{
		{
			name: "ed25519 with default comment",
			req:  GenerateKeyRequest{Type: KeyTypeEd25519},
			want: []string{"-t", "ed25519", "-f", "/tmp/key", "-C", "github-key", "-N", ""},
		},
		{
			name: "rsa includes requested bits and comment",
			req:  GenerateKeyRequest{Type: KeyTypeRSA, Bits: 3072, Comment: "work"},
			want: []string{"-t", "rsa", "-b", "3072", "-f", "/tmp/key", "-C", "work", "-N", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildKeygenArgs(tt.req, "/tmp/key"); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildKeygenArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestGenerateKeyRejectsExistingFileBeforeRunningKeygen(t *testing.T) {
	sshDir := t.TempDir()
	platform := &FakePlatform{SSHDirPath: sshDir, HomeDirPath: filepath.Dir(sshDir)}
	writeFile(t, filepath.Join(sshDir, "github_ed25519"), "existing")

	_, _, err := GenerateKey(platform, GenerateKeyRequest{Type: KeyTypeEd25519})
	if err == nil {
		t.Fatal("GenerateKey() error = nil, want existing key error")
	}
	if !strings.Contains(err.Error(), "key already exists") {
		t.Fatalf("GenerateKey() error = %q, want existing key error", err.Error())
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeExecutable(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
	return path
}
