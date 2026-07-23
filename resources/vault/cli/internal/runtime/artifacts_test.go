package runtime

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactCatalogPinsSupportedTargets(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "artifacts.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := ParseArtifactCatalog(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"linux-amd64", "linux-arm64", "macos-amd64", "macos-arm64", "windows-amd64"} {
		if _, ok := catalog.Artifacts[target]; !ok {
			t.Fatalf("catalog missing %s", target)
		}
	}
}

func TestArtifactCatalogRejectsUntrustedArtifactURL(t *testing.T) {
	_, err := ParseArtifactCatalog([]byte(`{"schema_version":1,"version":"1.17.6","checksum_manifest_url":"https://releases.hashicorp.com/vault/sums","checksum_signature_url":"https://releases.hashicorp.com/vault/sums.sig","artifacts":{"linux-amd64":{"url":"https://example.invalid/vault.zip","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","archive":"zip","binary_path":"vault"}}}`))
	if err == nil || !strings.Contains(err.Error(), "releases.hashicorp.com") {
		t.Fatalf("ParseArtifactCatalog() error = %v, want trusted source denial", err)
	}
}

func TestFetchArtifactVerifiesAndExtractsDeclaredBinary(t *testing.T) {
	archive := testVaultArchive(t, "vault", []byte("fixture-vault"))
	sum := sha256.Sum256(archive)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "bin", "vault")
	binarySum := sha256.Sum256([]byte("fixture-vault"))
	path, err := FetchArtifact(context.Background(), server.Client(), ArtifactTarget{URL: server.URL, SHA256: fmt.Sprintf("%x", sum), BinarySHA256: fmt.Sprintf("%x", binarySum), Archive: "zip", BinaryPath: "vault"}, destination)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fixture-vault" {
		t.Fatalf("extracted binary = %q", data)
	}
}

func TestFetchArtifactRejectsChecksumMismatch(t *testing.T) {
	archive := testVaultArchive(t, "vault", []byte("fixture-vault"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	_, err := FetchArtifact(context.Background(), server.Client(), ArtifactTarget{URL: server.URL, SHA256: strings.Repeat("0", 64), BinarySHA256: strings.Repeat("0", 64), Archive: "zip", BinaryPath: "vault"}, filepath.Join(t.TempDir(), "vault"))
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("FetchArtifact() error = %v", err)
	}
}

func testVaultArchive(t *testing.T, path string, body []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
