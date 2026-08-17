package binaryfetch

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tarBytes(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFetchDirSupportsRegisteredTarZstdDecompressor(t *testing.T) {
	archive := tarBytes(t, map[string][]byte{"bin/tool": bigBinary(), "lib/runtime": []byte("runtime")})
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, _ = gz.Write(archive)
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	srv := serve(t, compressed.Bytes())
	remove := RegisterArchiveDecompressor("tar.zst", func(r io.Reader) (io.ReadCloser, error) {
		return gzip.NewReader(r)
	})
	defer remove()
	optDir := filepath.Join(t.TempDir(), "opt")
	entry, err := FetchDir(context.Background(), Target{Name: "tool", URL: srv.URL, SHA256: sha256hex(compressed.Bytes()), Archive: "tar.zst", Layout: "dir", BinPath: "bin/tool"}, optDir, nil)
	if err != nil {
		t.Fatalf("FetchDir(tar.zst): %v", err)
	}
	if entry != filepath.Join(optDir, "bin", "tool") {
		t.Fatalf("entry = %q", entry)
	}
}

func TestTreeDigestChangesWhenOneFileChanges(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin"), []byte("one"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "runtime"), []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := TreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "runtime"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := TreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("TreeDigest did not change: %s", first)
	}
}

func TestFetchOCIExtractsDigestPinnedLayer(t *testing.T) {
	layer := tarBytes(t, map[string][]byte{"bin/server": bigBinary()})
	sum := sha256.Sum256(layer)
	digest := hex.EncodeToString(sum[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/example/blobs/sha256:"+digest {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(layer)
	}))
	defer server.Close()
	entry, err := FetchOCI(context.Background(), AcquisitionTarget{Image: server.URL + "/example@sha256:" + digest, BinPath: "bin/server"}, filepath.Join(t.TempDir(), "opt"), nil)
	if err != nil {
		t.Fatalf("FetchOCI: %v", err)
	}
	if _, err := os.Stat(entry); err != nil {
		t.Fatalf("OCI entry %q: %v", entry, err)
	}
}

func TestOCIReferenceDefaultsLibraryNamespace(t *testing.T) {
	base, repository, digest, err := ociReference("library/redis@sha256:" + strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if base != "https://registry-1.docker.io/v2/library/redis" || repository != "library/redis" || digest != strings.Repeat("a", 64) {
		t.Fatalf("ociReference = %q, %q, %q", base, repository, digest)
	}
}

func TestVerifyProvenanceRejectsWrongFingerprint(t *testing.T) {
	oldLookPath, oldRun := provenanceLookPath, provenanceRun
	defer func() { provenanceLookPath, provenanceRun = oldLookPath, oldRun }()
	provenanceLookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	provenanceRun = func(_ context.Context, executable string, _ ...string) ([]byte, error) {
		if filepath.Base(executable) == "gpg" {
			return []byte("fpr:::::::::AABBCCDDEEFF00112233445566778899AABBCCDD:"), nil
		}
		return nil, nil
	}
	_, err := VerifyProvenance(context.Background(), &Provenance{Kind: "gpg-checksums", Fingerprint: "00112233445566778899AABBCCDDFFEEDDCCBBAA"}, "key", "manifest", "signature")
	if err == nil {
		t.Fatal("VerifyProvenance accepted a wrong fingerprint")
	}
}
