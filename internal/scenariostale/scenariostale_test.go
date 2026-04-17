package scenariostale

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckNoSourcesReturnsNoSourcesStatus(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "ui"))

	result, err := Check(dir, "empty", Options{})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Status != StatusNoSources {
		t.Fatalf("expected StatusNoSources, got %q", result.Status)
	}
	if _, err := os.Stat(result.SidecarPath); !os.IsNotExist(err) {
		t.Fatalf("sidecar should not be written for no-sources scenario")
	}
}

func TestCheckInitialBaselineWritesSidecarAndDoesNotWarn(t *testing.T) {
	dir := scaffoldScenario(t)

	result, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Status != StatusInitialBaseline {
		t.Fatalf("expected StatusInitialBaseline, got %q", result.Status)
	}
	if FormatWarning(result) != "" {
		t.Fatalf("InitialBaseline should not produce a warning; got %q", FormatWarning(result))
	}
	if _, err := os.Stat(result.SidecarPath); err != nil {
		t.Fatalf("sidecar not written: %v", err)
	}

	var payload sidecarPayload
	readPayload(t, result.SidecarPath, &payload)
	if payload.SourceHash != result.Fingerprint {
		t.Fatalf("sidecar fingerprint mismatch: want %q, got %q", result.Fingerprint, payload.SourceHash)
	}
	if payload.Version != SidecarVersion {
		t.Fatalf("sidecar version mismatch: want %d, got %d", SidecarVersion, payload.Version)
	}
	if len(payload.Files) != 2 {
		t.Fatalf("expected 2 tracked files (api + cli), got %d", len(payload.Files))
	}
}

func TestCheckFreshAfterUnchangedRerun(t *testing.T) {
	dir := scaffoldScenario(t)
	_, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("initial Check: %v", err)
	}

	second, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("second Check: %v", err)
	}
	if second.Status != StatusFresh {
		t.Fatalf("expected StatusFresh, got %q", second.Status)
	}
	if FormatWarning(second) != "" {
		t.Fatalf("Fresh should not produce a warning")
	}
}

func TestCheckStaleWhenSourceChangesWithoutRebuild(t *testing.T) {
	dir := scaffoldScenario(t)
	_, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("initial Check: %v", err)
	}

	// Edit a source file without touching the binary mtime.
	writeFile(t, filepath.Join(dir, "api", "main.go"), "package main\nfunc main(){ println(\"edited\") }\n")

	result, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("Check after edit: %v", err)
	}
	if result.Status != StatusStale {
		t.Fatalf("expected StatusStale, got %q", result.Status)
	}
	warning := FormatWarning(result)
	if !strings.Contains(warning, "WARNING: scenario 'foo' binary is stale") {
		t.Fatalf("warning missing expected header, got:\n%s", warning)
	}
	if !strings.Contains(warning, "1 Go file has changed") {
		t.Fatalf("warning missing change count, got:\n%s", warning)
	}
	if !strings.Contains(warning, "--no-stale-check") {
		t.Fatalf("warning missing --no-stale-check hint, got:\n%s", warning)
	}
	if len(result.ChangedFiles) != 1 || result.ChangedFiles[0] != "api/main.go" {
		t.Fatalf("expected api/main.go in ChangedFiles, got %v", result.ChangedFiles)
	}
}

func TestCheckRebuildDetectedWhenBinaryMtimeChanges(t *testing.T) {
	dir := scaffoldScenario(t)
	_, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("initial Check: %v", err)
	}

	// Edit a source file, then bump the api binary mtime to simulate a rebuild.
	writeFile(t, filepath.Join(dir, "api", "main.go"), "package main\nfunc main(){ println(\"rebuilt\") }\n")
	bumpMtime(t, filepath.Join(dir, "api", "foo-api"))

	result, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("Check after rebuild: %v", err)
	}
	if result.Status != StatusRebuildDetected {
		t.Fatalf("expected StatusRebuildDetected, got %q", result.Status)
	}
	if FormatWarning(result) != "" {
		t.Fatalf("RebuildDetected should not produce a warning")
	}

	// A subsequent Check with no further edits should be Fresh.
	third, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("third Check: %v", err)
	}
	if third.Status != StatusFresh {
		t.Fatalf("expected StatusFresh after rebuild, got %q", third.Status)
	}
}

func TestCheckIgnoresTestFiles(t *testing.T) {
	dir := scaffoldScenario(t)
	writeFile(t, filepath.Join(dir, "api", "extra_test.go"), "package main\nimport \"testing\"\nfunc TestX(t *testing.T){}\n")

	first, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("initial Check: %v", err)
	}
	if first.GoFileCount != 2 {
		t.Fatalf("expected 2 non-test Go files, got %d", first.GoFileCount)
	}

	// Mutating a _test.go file must not change the fingerprint.
	writeFile(t, filepath.Join(dir, "api", "extra_test.go"), "package main\nimport \"testing\"\nfunc TestY(t *testing.T){}\n")

	second, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("second Check: %v", err)
	}
	if second.Status != StatusFresh {
		t.Fatalf("editing _test.go should not produce staleness; got %q", second.Status)
	}
}

func TestCheckRecoversFromCorruptSidecar(t *testing.T) {
	dir := scaffoldScenario(t)
	sidecar := filepath.Join(dir, SidecarFile)
	writeFile(t, sidecar, "{not json")

	result, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("Check with corrupt sidecar returned error: %v", err)
	}
	if result.Status != StatusInitialBaseline {
		t.Fatalf("expected StatusInitialBaseline on corrupt sidecar, got %q", result.Status)
	}
	var payload sidecarPayload
	readPayload(t, sidecar, &payload)
	if payload.SourceHash == "" {
		t.Fatalf("sidecar not rewritten after corruption")
	}
}

func TestCheckSkipsVendorAndNodeModules(t *testing.T) {
	dir := scaffoldScenario(t)
	writeFile(t, filepath.Join(dir, "api", "vendor", "dep", "dep.go"), "package dep\n")
	writeFile(t, filepath.Join(dir, "api", "node_modules", "pkg", "pkg.go"), "package pkg\n")
	writeFile(t, filepath.Join(dir, "api", "testdata", "fixture.go"), "package fixture\n")

	result, err := Check(dir, "foo", Options{Clock: fixedClock()})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.GoFileCount != 2 {
		t.Fatalf("vendor/node_modules/testdata must be skipped; got %d files", result.GoFileCount)
	}
}

func TestCheckRequiresScenarioDir(t *testing.T) {
	if _, err := Check("", "x", Options{}); err == nil {
		t.Fatal("expected error for empty scenario dir")
	}
	if _, err := Check(filepath.Join(t.TempDir(), "does-not-exist"), "x", Options{}); err == nil {
		t.Fatal("expected error for missing scenario dir")
	}
}

func TestFormatWarningForNonStaleStatusesIsEmpty(t *testing.T) {
	for _, status := range []Status{StatusFresh, StatusRebuildDetected, StatusInitialBaseline, StatusNoSources} {
		r := Result{Scenario: "foo", Status: status}
		if FormatWarning(r) != "" {
			t.Fatalf("status %q should produce empty warning", status)
		}
	}
}

// ---- helpers ----

func scaffoldScenario(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "api", "main.go"), "package main\nfunc main(){}\n")
	writeFile(t, filepath.Join(dir, "cli", "main.go"), "package main\nfunc main(){}\n")
	writeExecutable(t, filepath.Join(dir, "api", "foo-api"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(dir, "cli", "foo"), "#!/bin/sh\nexit 0\n")
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func bumpMtime(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	newTime := info.ModTime().Add(5 * time.Second)
	if err := os.Chtimes(path, newTime, newTime); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func fixedClock() func() time.Time {
	now := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	return func() time.Time { return now }
}

func readPayload(t *testing.T, path string, out *sidecarPayload) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal sidecar: %v", err)
	}
}
