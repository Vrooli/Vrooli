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

func TestFetchAndFetchDirSupportRegisteredTarXZDecompressor(t *testing.T) {
	archive := tarBytes(t, map[string][]byte{"bin/tool": bigBinary(), "lib/runtime": []byte("runtime")})
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	if _, err := gz.Write(archive); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	remove := RegisterArchiveDecompressor("txz", func(r io.Reader) (io.ReadCloser, error) {
		return gzip.NewReader(r)
	})
	defer remove()
	server := serve(t, compressed.Bytes())
	digest := sha256hex(compressed.Bytes())

	filePath, err := Fetch(context.Background(), Target{
		Name: "tool", URL: server.URL, SHA256: digest, Archive: "tar.xz", BinPath: "bin/tool",
	}, filepath.Join(t.TempDir(), "bin"), nil)
	if err != nil {
		t.Fatalf("Fetch(tar.xz): %v", err)
	}
	if data, err := os.ReadFile(filePath); err != nil || !bytes.Equal(data, bigBinary()) {
		t.Fatalf("single-file extraction = %d bytes, err %v", len(data), err)
	}

	optDir := filepath.Join(t.TempDir(), "opt")
	entry, err := FetchDir(context.Background(), Target{
		Name: "tool", URL: server.URL, SHA256: digest, Archive: "tar.xz", Layout: "dir", BinPath: "bin/tool",
	}, optDir, nil)
	if err != nil {
		t.Fatalf("FetchDir(tar.xz): %v", err)
	}
	if entry != filepath.Join(optDir, "bin", "tool") {
		t.Fatalf("entry = %q", entry)
	}
	if data, err := os.ReadFile(filepath.Join(optDir, "lib", "runtime")); err != nil || string(data) != "runtime" {
		t.Fatalf("directory extraction runtime = %q, err %v", data, err)
	}
}

func TestFetchDirRejectsCorruptRegisteredTarFooter(t *testing.T) {
	archive := tarBytes(t, map[string][]byte{"bin/tool": bigBinary(), "lib/runtime": []byte("runtime")})
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	if _, err := gz.Write(archive); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	data := append([]byte(nil), compressed.Bytes()...)
	data[len(data)-1] ^= 0x01
	remove := RegisterArchiveDecompressor("tar.xz", func(r io.Reader) (io.ReadCloser, error) {
		return gzip.NewReader(r)
	})
	defer remove()
	server := serve(t, data)
	_, err := FetchDir(context.Background(), Target{
		Name: "tool", URL: server.URL, SHA256: sha256hex(data), Archive: "tar.xz", Layout: "dir", BinPath: "bin/tool",
	}, filepath.Join(t.TempDir(), "opt"), nil)
	if err == nil {
		t.Fatal("FetchDir accepted a corrupt registered-tar footer")
	}
}

func TestRegisteredTarXZUsesBoundedWalkerAndRequiresCodec(t *testing.T) {
	plain := writeTarGz(t, []tarEntry{
		{hdr: tar.Header{Name: "bin/tool"}, body: "safe"},
		{hdr: tar.Header{Name: "../escape"}, body: "blocked"},
	})
	compressed, err := os.ReadFile(plain)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "fixture.tar.xz")
	if err := os.WriteFile(path, compressed, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectArchiveBounded(path, "tar.xz", ExtractOptions{}); err == nil || !strings.Contains(err.Error(), "no decompressor registered for tar.xz") {
		t.Fatalf("unregistered codec error = %v", err)
	}
	remove := RegisterArchiveDecompressor("tar.xz", func(r io.Reader) (io.ReadCloser, error) {
		return gzip.NewReader(r)
	})
	defer remove()
	if _, err := InspectArchiveBounded(path, "tar.xz", ExtractOptions{}); violationOf(t, err) != ViolationTraversal {
		t.Fatalf("InspectArchiveBounded violation = %v; want traversal", err)
	}
	if _, err := ExtractArchiveBounded(path, "tar.xz", filepath.Join(t.TempDir(), "stage"), ExtractOptions{}); violationOf(t, err) != ViolationTraversal {
		t.Fatalf("ExtractArchiveBounded violation = %v; want traversal", err)
	}
}

func TestRegisteredTarXZHonorsExpandedSizeBounds(t *testing.T) {
	plain := writeTarGz(t, []tarEntry{{hdr: tar.Header{Name: "data"}, body: "123456789"}})
	compressed, err := os.ReadFile(plain)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "fixture.tar.xz")
	if err := os.WriteFile(path, compressed, 0o600); err != nil {
		t.Fatal(err)
	}
	remove := RegisterArchiveDecompressor("tar.xz", func(r io.Reader) (io.ReadCloser, error) {
		return gzip.NewReader(r)
	})
	defer remove()
	if _, err := InspectArchiveBounded(path, "tar.xz", ExtractOptions{MaxExpandedBytes: 8}); violationOf(t, err) != ViolationTooLarge {
		t.Fatalf("InspectArchiveBounded violation = %v; want too_large", err)
	}
	if _, err := ExtractArchiveBounded(path, "tar.xz", filepath.Join(t.TempDir(), "stage"), ExtractOptions{MaxExpandedBytes: 8}); violationOf(t, err) != ViolationTooLarge {
		t.Fatalf("ExtractArchiveBounded violation = %v; want too_large", err)
	}
}

func TestRegisteredSingleBinaryExtractionValidatesCompressedFooter(t *testing.T) {
	archive := writeTarGz(t, []tarEntry{{hdr: tar.Header{Name: "bin/tool", Mode: 0o755}, body: "safe"}})
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] ^= 0x01 // Corrupt the gzip trailer checksum.
	archive = filepath.Join(t.TempDir(), "fixture.tar.xz")
	if err := os.WriteFile(archive, data, 0o600); err != nil {
		t.Fatal(err)
	}
	remove := RegisterArchiveDecompressor("tar.xz", func(r io.Reader) (io.ReadCloser, error) {
		return gzip.NewReader(r)
	})
	defer remove()
	dest := filepath.Join(t.TempDir(), "tool")
	if err := extractBinary(archive, "tar.xz", "bin/tool", dest); err == nil {
		t.Fatal("single-binary extraction accepted a corrupt compressed footer")
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

func TestRemoveDanglingOCISymlinksPrunesDanglingAndEmptyDirectoryLinks(t *testing.T) {
	root := t.TempDir()
	emptyDir := filepath.Join(root, "usr", "local", "share", "man")
	populatedDir := filepath.Join(root, "usr", "share", "doc")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(populatedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(populatedDir, "README"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	links := map[string]string{
		filepath.Join(root, "usr", "local", "man"):           "share/man",
		filepath.Join(root, "usr", "share", "documentation"): "doc",
		filepath.Join(root, "dangling"):                      "missing",
	}
	for path, target := range links {
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
	}

	if err := removeDanglingOCISymlinks(root); err != nil {
		t.Fatalf("removeDanglingOCISymlinks: %v", err)
	}
	for _, path := range []string{filepath.Join(root, "usr", "local", "man"), filepath.Join(root, "dangling")} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Errorf("pruned path %q still exists (err %v)", path, err)
		}
	}
	if target, err := os.Readlink(filepath.Join(root, "usr", "share", "documentation")); err != nil || target != "doc" {
		t.Errorf("populated directory symlink = %q, %v; want doc", target, err)
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

func TestFetchOCIGrantsExtractorAccessToRestrictedImageFiles(t *testing.T) {
	var layer bytes.Buffer
	tw := tar.NewWriter(&layer)
	body := bytes.Repeat([]byte("runtime configuration\n"), 64)
	if err := tw.WriteHeader(&tar.Header{Name: "etc/runtime.conf", Mode: 0, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	digest := sha256hex(layer.Bytes())
	server := serve(t, layer.Bytes())
	defer server.Close()
	optDir := filepath.Join(t.TempDir(), "opt")
	if _, err := FetchOCI(context.Background(), AcquisitionTarget{Image: server.URL + "/example@sha256:" + digest, BinPath: "etc/runtime.conf"}, optDir, nil); err != nil {
		t.Fatalf("FetchOCI restricted file: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(optDir, "etc", "runtime.conf"))
	if err != nil {
		t.Fatalf("read extracted restricted file: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("extracted body = %q, want %q", got, body)
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
