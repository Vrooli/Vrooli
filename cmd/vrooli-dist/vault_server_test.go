package main

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVaultArtifactValidationRejectsUntrustedOrUnpinnedInput(t *testing.T) {
	valid := vaultArtifact{
		URL:          "https://releases.hashicorp.com/vault/1.17.6/vault.zip",
		SHA256:       strings.Repeat("a", 64),
		BinarySHA256: strings.Repeat("b", 64),
		Archive:      "zip",
		BinaryPath:   "vault",
	}
	if err := valid.validate(); err != nil {
		t.Fatalf("valid artifact: %v", err)
	}
	for _, artifact := range []vaultArtifact{
		{URL: "https://example.test/vault.zip", SHA256: valid.SHA256, BinarySHA256: valid.BinarySHA256, Archive: "zip", BinaryPath: "vault"},
		{URL: valid.URL, SHA256: valid.SHA256, BinarySHA256: valid.BinarySHA256, Archive: "zip", BinaryPath: "../vault"},
		{URL: valid.URL, SHA256: "bad", BinarySHA256: valid.BinarySHA256, Archive: "zip", BinaryPath: "vault"},
	} {
		if err := artifact.validate(); err == nil {
			t.Fatalf("artifact validation accepted %#v", artifact)
		}
	}
}

func TestVaultTargetsAreExplicit(t *testing.T) {
	if !isVaultTarget("linux-amd64") || isVaultTarget("linux-386") {
		t.Fatal("Vault target allowlist is incorrect")
	}
}

func TestStageVaultServerUsesReleaseArtifactNaming(t *testing.T) {
	previousFetch := fetchVaultArchive
	defer func() { fetchVaultArchive = previousFetch }()
	previousVerify := verifySHA256File
	verifySHA256File = func(_ string, _ string) error { return nil }
	defer func() { verifySHA256File = previousVerify }()
	previousProvenance := verifyVaultChecksumProvenance
	verifyVaultChecksumProvenance = func(context.Context, vaultArtifactCatalog) error { return nil }
	defer func() { verifyVaultChecksumProvenance = previousProvenance }()

	for _, test := range []struct {
		name       string
		target     string
		binaryPath string
		wantName   string
	}{
		{name: "linux", target: "linux-amd64", binaryPath: "vault", wantName: "vault_linux_amd64"},
		{name: "macos", target: "macos-arm64", binaryPath: "vault", wantName: "vault_darwin_arm64"},
		{name: "windows", target: "windows-amd64", binaryPath: "vault.exe", wantName: "vault_windows_amd64.exe"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "resources", "vault"), 0o755); err != nil {
				t.Fatal(err)
			}
			catalog := fmt.Sprintf(`{"version":"1.17.6","artifacts":{"%s":{"url":"https://releases.hashicorp.com/vault/1.17.6/vault.zip","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","binary_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","archive":"zip","binary_path":"%s"}}}`, test.target, test.binaryPath)
			if err := os.WriteFile(filepath.Join(root, "resources", "vault", "artifacts.json"), []byte(catalog), 0o644); err != nil {
				t.Fatal(err)
			}
			fetchVaultArchive = func(_ context.Context, _ string, destination *os.File) error {
				return writeVaultTestArchive(destination, test.binaryPath)
			}
			output := t.TempDir()
			if err := stageVaultServer(context.Background(), root, output, test.target); err != nil {
				t.Fatalf("stageVaultServer: %v", err)
			}
			if _, err := os.Stat(filepath.Join(output, test.wantName)); err != nil {
				t.Fatalf("expected canonical staged artifact name %q: %v", test.wantName, err)
			}
		})
	}
}

func writeVaultTestArchive(destination *os.File, name string) error {
	archive := zip.NewWriter(destination)
	entry, err := archive.Create(name)
	if err != nil {
		return err
	}
	if _, err := entry.Write([]byte("fixture Vault server")); err != nil {
		return err
	}
	return archive.Close()
}
