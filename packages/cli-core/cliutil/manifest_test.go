package cliutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if !mtime.IsZero() {
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatalf("chtimes %s: %v", path, err)
		}
	}
}

func goModuleSpec(root string) FreshnessSpec {
	return FreshnessSpec{
		SourceRoot:   root,
		ContextRoot:  root,
		Inputs:       []string{"."},
		SkipSuffixes: []string{"_test.go"},
	}
}

// manifestWrittenAfter returns a write-time newer than every input file so the
// stat-cache trusts unchanged (size,mtime) without re-hashing (no racy window).
func manifestWrittenAfter(base time.Time) int64 {
	return base.Add(time.Hour).UnixNano()
}

func TestComputeAndEvaluateFreshnessFresh(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-2 * time.Hour)
	writeFile(t, filepath.Join(root, "main.go"), "package main", base)
	writeFile(t, filepath.Join(root, "util.go"), "package main // util", base)

	spec := goModuleSpec(root)
	m, err := ComputeFreshnessManifest(spec, "binaries", nil, manifestWrittenAfter(base))
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if len(m.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(m.Files), m.Files)
	}
	v, err := EvaluateFreshness(spec, m, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if v.Stale {
		t.Fatalf("expected fresh, got stale: %+v", v)
	}
}

func TestEvaluateFreshnessTouchedSameContentIsFresh(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-2 * time.Hour)
	main := filepath.Join(root, "main.go")
	writeFile(t, main, "package main", base)

	spec := goModuleSpec(root)
	written := manifestWrittenAfter(base)
	m, err := ComputeFreshnessManifest(spec, "binaries", nil, written)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Touch the file (new mtime) but keep content identical. mtime stays below
	// the manifest write time, so it re-hashes and finds the same content.
	touch := base.Add(10 * time.Minute)
	if err := os.Chtimes(main, touch, touch); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	v, err := EvaluateFreshness(spec, m, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if v.Stale {
		t.Fatalf("touch with identical content must be fresh, got %+v", v)
	}
}

func TestEvaluateFreshnessContentChangedIsStale(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-2 * time.Hour)
	main := filepath.Join(root, "main.go")
	writeFile(t, main, "package main", base)

	spec := goModuleSpec(root)
	m, err := ComputeFreshnessManifest(spec, "binaries", nil, manifestWrittenAfter(base))
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	writeFile(t, main, "package main // changed", base.Add(5*time.Minute))
	v, err := EvaluateFreshness(spec, m, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	// A longer edit changes size; the engine catches that before hashing.
	if !v.Stale || v.File != "main.go" {
		t.Fatalf("expected stale naming main.go, got %+v", v)
	}
}

func TestEvaluateFreshnessTestFileExcluded(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-2 * time.Hour)
	writeFile(t, filepath.Join(root, "main.go"), "package main", base)

	spec := goModuleSpec(root)
	m, err := ComputeFreshnessManifest(spec, "binaries", nil, manifestWrittenAfter(base))
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Add a brand-new _test.go file after stamping; it must not register as a
	// new input (test files are excluded by SkipSuffixes).
	writeFile(t, filepath.Join(root, "main_test.go"), "package main", time.Now())
	v, err := EvaluateFreshness(spec, m, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if v.Stale {
		t.Fatalf("a new _test.go must not mark stale, got %+v", v)
	}
}

func TestEvaluateFreshnessRacyIndexRehash(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-2 * time.Hour)
	main := filepath.Join(root, "main.go")
	writeFile(t, main, "package main", base)

	spec := goModuleSpec(root)
	// Manifest write time EQUALS the file mtime: the racy-index rule forces a
	// re-hash even though (size,mtime) appear unchanged. We then change the
	// content while preserving size and mtime; the re-hash must catch it.
	written := base.UnixNano()
	m, err := ComputeFreshnessManifest(spec, "binaries", nil, written)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Same length ("package main" == "package MAIN"), same mtime, new content.
	if err := os.WriteFile(main, []byte("package MAIN"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chtimes(main, base, base); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	v, err := EvaluateFreshness(spec, m, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !v.Stale {
		t.Fatalf("racy-index rule must re-hash and detect same-size same-mtime content change, got %+v", v)
	}
}

func TestEvaluateFreshnessKeyInputChange(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-2 * time.Hour)
	writeFile(t, filepath.Join(root, "main.go"), "package main", base)

	spec := goModuleSpec(root)
	m, err := ComputeFreshnessManifest(spec, "binaries", map[string]string{"toolchain": "go1.25.0"}, manifestWrittenAfter(base))
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Identical source, different toolchain -> stale (false-negative defense).
	v, err := EvaluateFreshness(spec, m, map[string]string{"toolchain": "go1.26.0"})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !v.Stale || v.File != "toolchain" {
		t.Fatalf("toolchain change must mark stale naming the key, got %+v", v)
	}
	// Assert the key actually participates in the aggregate digest.
	other, _ := ComputeFreshnessManifest(spec, "binaries", map[string]string{"toolchain": "go1.26.0"}, manifestWrittenAfter(base))
	if other.Digest == m.Digest {
		t.Fatal("toolchain key input must change the aggregate digest")
	}
}

func TestFreshnessManifestRoundTrip(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-2 * time.Hour)
	writeFile(t, filepath.Join(root, "main.go"), "package main", base)

	spec := goModuleSpec(root)
	m, err := ComputeFreshnessManifest(spec, "binaries", map[string]string{"toolchain": "go1.25.0"}, manifestWrittenAfter(base))
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	path := FreshnessManifestPath(filepath.Join(root, "bin", "app"))
	if err := WriteFreshnessManifest(path, m); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, ok, err := ReadFreshnessManifest(path)
	if err != nil || !ok {
		t.Fatalf("read: ok=%v err=%v", ok, err)
	}
	if got.Digest != m.Digest || len(got.Files) != len(m.Files) {
		t.Fatalf("round-trip mismatch: %+v vs %+v", got, m)
	}

	missing, ok, err := ReadFreshnessManifest(FreshnessManifestPath(filepath.Join(root, "bin", "absent")))
	if err != nil {
		t.Fatalf("read absent: %v", err)
	}
	if ok {
		t.Fatalf("absent manifest must report ok=false, got %+v", missing)
	}
}

func TestEvaluateFreshnessVersionMismatch(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-2 * time.Hour)
	writeFile(t, filepath.Join(root, "main.go"), "package main", base)
	spec := goModuleSpec(root)
	m, err := ComputeFreshnessManifest(spec, "binaries", nil, manifestWrittenAfter(base))
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	m.Version = 999
	v, err := EvaluateFreshness(spec, m, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !v.Stale {
		t.Fatal("version mismatch must be stale (bootstrap)")
	}
}
