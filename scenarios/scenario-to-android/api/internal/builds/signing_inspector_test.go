package builds

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectAPKSigningCertificateReadsApksignerDigest(t *testing.T) {
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	useFakeAndroidSDK(t, digest)

	got, err := inspectAPKSigningCertificate(context.Background(), filepath.Join(t.TempDir(), "app.apk"))
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ToLower(digest) {
		t.Fatalf("signer digest = %q, want %q", got, digest)
	}
}

func useFakeAndroidSDK(t *testing.T, digest string) string {
	t.Helper()
	sdkRoot := t.TempDir()
	tool := filepath.Join(sdkRoot, "build-tools", "35.0.0", "apksigner")
	if err := os.MkdirAll(filepath.Join(sdkRoot, "platforms"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(tool), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nprintf '%s\\n' 'Signer #1 certificate SHA-256 digest: " + digest + "'\n"
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANDROID_SDK_ROOT", sdkRoot)
	t.Setenv("ANDROID_HOME", "")
	return tool
}

func TestValidSHA256Digest(t *testing.T) {
	if !validSHA256Digest(strings.Repeat("A", 64)) {
		t.Fatal("expected uppercase SHA-256 digest to be accepted")
	}
	for _, invalid := range []string{"", strings.Repeat("g", 64), strings.Repeat("a", 63), strings.Repeat("a", 65)} {
		if validSHA256Digest(invalid) {
			t.Fatalf("accepted invalid digest %q", invalid)
		}
	}
}

func TestInspectVerifiedAPKSigningCertificateRejectsChangedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.apk")
	requireBytes := []byte("APK bytes changed after resolution")
	if err := os.WriteFile(path, requireBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	expected := sha256.Sum256([]byte("previously verified APK bytes"))
	if _, err := inspectVerifiedAPKSigningCertificate(file, expected[:]); err == nil || err.Error() != "verified APK changed before signer inspection" {
		t.Fatalf("expected byte-change refusal, got %v", err)
	}
}
