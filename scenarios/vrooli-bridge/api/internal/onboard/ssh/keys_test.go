package ssh

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePortAppliesDefaultAndEnforcesTCPRange(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   int
		want    int
		wantErr bool
	}{
		{name: "default", input: 0, want: DefaultPort},
		{name: "minimum", input: 1, want: 1},
		{name: "maximum", input: 65535, want: 65535},
		{name: "negative", input: -1, wantErr: true},
		{name: "too large", input: 65536, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizePort(%d) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("normalizePort(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestInvalidPortStopsConnectionAndKeyCopyBeforeSideEffects(t *testing.T) {
	runner := &scriptedRunner{}
	svc := NewService(t.TempDir(), WithRunner(runner))
	for _, port := range []int{-1, 65536} {
		got := svc.TestConnection(context.Background(), TestConnectionRequest{Host: "example.test", Port: port})
		if got.Status != StatusInvalidInput {
			t.Errorf("TestConnection port %d status = %q, want %q", port, got.Status, StatusInvalidInput)
		}
	}
	if runner.calls != 0 {
		t.Fatalf("invalid ports reached connection runner %d times", runner.calls)
	}
	for _, port := range []int{-1, 65536} {
		got := (ExecKeyCopier{}).CopyKey(context.Background(), CopyKeyRequest{Host: "example.test", Port: port})
		if got.Status != StatusInvalidInput {
			t.Errorf("CopyKey port %d status = %q, want %q", port, got.Status, StatusInvalidInput)
		}
	}
}

func TestGenerateKeyWritesSecurePermsAndFingerprint(t *testing.T) {
	requireSSHTools(t)

	dir := t.TempDir()
	svc := NewService(dir)

	info, err := svc.GenerateKey(GenerateKeyRequest{Type: KeyTypeEd25519, Filename: "bridge-onboard"})
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	keyPath := filepath.Join(dir, "bridge-onboard")
	assertMode(t, keyPath, 0o600) // private key owner-only
	assertMode(t, dir, 0o700)     // state dir owner-only
	assertMode(t, keyPath+".pub", 0o644)

	if info.Type != KeyTypeEd25519 {
		t.Errorf("key type = %q, want ed25519", info.Type)
	}
	if !strings.HasPrefix(info.Fingerprint, "SHA256:") {
		t.Errorf("fingerprint = %q, want SHA256: prefix", info.Fingerprint)
	}

	pub, fp, err := svc.ReadPublicKey(keyPath)
	if err != nil {
		t.Fatalf("ReadPublicKey: %v", err)
	}
	if !strings.HasPrefix(pub, "ssh-ed25519 ") {
		t.Errorf("public key = %q, want ssh-ed25519 prefix", pub)
	}
	if fp != info.Fingerprint {
		t.Errorf("read fingerprint %q != generated %q", fp, info.Fingerprint)
	}
}

func TestGenerateKeyRefusesToOverwriteExisting(t *testing.T) {
	requireSSHTools(t)

	dir := t.TempDir()
	svc := NewService(dir)
	if _, err := svc.GenerateKey(GenerateKeyRequest{Type: KeyTypeEd25519, Filename: "bridge-onboard"}); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	if _, err := svc.GenerateKey(GenerateKeyRequest{Type: KeyTypeEd25519, Filename: "bridge-onboard"}); err == nil {
		t.Fatal("expected an error regenerating over an existing key")
	}
}

func TestValidateKeyFilename(t *testing.T) {
	bad := []string{"", "../evil", "a/b", "..", strings.Repeat("x", 65), "-leading-dash"}
	for _, f := range bad {
		if err := ValidateKeyFilename(f); err == nil {
			t.Errorf("ValidateKeyFilename(%q) should fail", f)
		}
	}
	for _, f := range []string{"bridge-onboard", "id_ed25519", "_key1"} {
		if err := ValidateKeyFilename(f); err != nil {
			t.Errorf("ValidateKeyFilename(%q) should pass, got %v", f, err)
		}
	}
}

func TestBuildArgsPinsBridgeKnownHosts(t *testing.T) {
	cfg := NewConfig("h", 2222, "u", "/state/key", "/state/known_hosts")
	args := strings.Join(buildSSHArgs(cfg, TestConnectionOptions()), " ")
	if !strings.Contains(args, "UserKnownHostsFile=/state/known_hosts") {
		t.Errorf("expected UserKnownHostsFile pin, got: %s", args)
	}
	if !strings.Contains(args, "GlobalKnownHostsFile=/dev/null") {
		t.Errorf("expected global known_hosts ignored, got: %s", args)
	}
	if !strings.Contains(args, "StrictHostKeyChecking=accept-new") {
		t.Errorf("expected accept-new, got: %s", args)
	}
	if !strings.Contains(args, "-p 2222") {
		t.Errorf("expected custom port, got: %s", args)
	}
}
