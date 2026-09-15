package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyAcceptsDeclaredSidecar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc-parse.wasm")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := Verify(path, "f16d05ec6b29248d2c61adb1e9263f78e4f7bace1b955014a2d17872cfe4064d")
	if err != nil {
		t.Fatal(err)
	}
	if digest != "f16d05ec6b29248d2c61adb1e9263f78e4f7bace1b955014a2d17872cfe4064d" {
		t.Fatalf("digest = %q", digest)
	}
}

func TestVerifyRejectsChecksumMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc-parse.wasm")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(path, "00000000000000000000000000000000"); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestResolverUsesEnvironmentArtifactAndSidecar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "module.wasm")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".sha256", []byte("f16d05ec6b29248d2c61adb1e9263f78e4f7bace1b955014a2d17872cfe4064d  module.wasm\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := Resolver{Environment: func(key string) string {
		if key == WASIEnvironment {
			return path
		}
		return ""
	}}
	got, err := r.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != path || got.SHA256 != "f16d05ec6b29248d2c61adb1e9263f78e4f7bace1b955014a2d17872cfe4064d" {
		t.Fatalf("artifact = %+v", got)
	}
}

func TestResolverInstalledDataDirDoesNotFallBackToSource(t *testing.T) {
	dir := t.TempDir()
	sourceRoot := filepath.Join(dir, "source")
	if err := os.MkdirAll(filepath.Join(sourceRoot, "artifacts"), 0o700); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(sourceRoot, "artifacts", "doc-parse.wasm")
	if err := os.WriteFile(sourcePath, []byte("source-fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath+".sha256", []byte("93b9a5429155bfcee56fd1f993b106cc588f138b27598b0e066d8e1af21bf086\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolver := Resolver{SourceRoot: sourceRoot, InstalledDataDir: filepath.Join(dir, "installed"), Environment: func(string) string { return "" }}
	if _, err := resolver.Resolve(); err == nil {
		t.Fatal("expected missing installed artifact to remain unhealthy")
	}
}
