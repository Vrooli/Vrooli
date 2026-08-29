package cliutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeOutputTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A test run rewrites coverage.out. If that counted as an input change the
// binary would look stale after every test run, and any auto-rebuild policy
// would rebuild it, and the next test run would invalidate it again — a
// self-perpetuating restart treadmill. This pins that it cannot happen.
func TestCoverageProfileChurnDoesNotMakeBinaryStale(t *testing.T) {
	root := t.TempDir()
	writeOutputTestFile(t, filepath.Join(root, "main.go"), "package main\nfunc main(){}\n")
	writeOutputTestFile(t, filepath.Join(root, "coverage.out"), "mode: set\n")

	spec := FreshnessSpec{SourceRoot: root, ContextRoot: root}
	manifest, err := ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	for _, f := range manifest.Files {
		if filepath.Base(f.Rel) == "coverage.out" {
			t.Fatal("coverage.out was recorded as a build input")
		}
	}

	writeOutputTestFile(t, filepath.Join(root, "coverage.out"), "mode: set\nmain.go:1.1,2.2 1 1\n")
	v, err := EvaluateFreshness(spec, manifest, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if v.Stale {
		t.Errorf("a rewritten coverage profile made the binary stale (%s: %s) — this is the restart treadmill", v.Reason, v.File)
	}
}

// The installed-CLI stale checker still uses the legacy content fingerprint,
// not the stat-cache manifest above. Pin the same build-output exclusion there
// because a mismatch causes an auto-rebuild and can restart a scenario API.
func TestCoverageProfileChurnDoesNotChangeDeclaredInputFingerprint(t *testing.T) {
	root := t.TempDir()
	writeOutputTestFile(t, filepath.Join(root, "api", "main.go"), "package main\nfunc main(){}\n")
	writeOutputTestFile(t, filepath.Join(root, "api", "coverage.out"), "mode: set\n")
	spec := FreshnessSpec{SourceRoot: root, ContextRoot: root, Inputs: []string{"api/**"}}

	before, err := ComputeFreshnessFingerprint(spec)
	if err != nil {
		t.Fatalf("compute before: %v", err)
	}
	writeOutputTestFile(t, filepath.Join(root, "api", "coverage.out"), "mode: set\nmain.go:1.1,2.2 1 1\n")
	after, err := ComputeFreshnessFingerprint(spec)
	if err != nil {
		t.Fatalf("compute after: %v", err)
	}
	if before != after {
		t.Fatalf("coverage churn changed declared-input fingerprint: before=%s after=%s", before, after)
	}
}

func TestBuildOutputsAreExcludedFromInputs(t *testing.T) {
	root := t.TempDir()
	writeOutputTestFile(t, filepath.Join(root, "main.go"), "package main\n")
	for _, name := range []string{"coverage.out", "cov.out", "cover.out", "api.log", "pkg.test", "jscpd-report.json"} {
		writeOutputTestFile(t, filepath.Join(root, name), "output\n")
	}

	manifest, err := ComputeFreshnessManifest(FreshnessSpec{SourceRoot: root, ContextRoot: root}, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if len(manifest.Files) != 1 || filepath.Base(manifest.Files[0].Rel) != "main.go" {
		t.Fatalf("expected only main.go as input, got %+v", manifest.Files)
	}
}

// Manifests written before build outputs were excluded still list them. Those
// must not all evaluate stale at once, which would force a fleet-wide rebuild.
func TestLegacyManifestListingBuildOutputIsNotStale(t *testing.T) {
	root := t.TempDir()
	writeOutputTestFile(t, filepath.Join(root, "main.go"), "package main\n")
	spec := FreshnessSpec{SourceRoot: root, ContextRoot: root}
	manifest, err := ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	// Simulate a manifest recorded before the exclusion existed.
	manifest.Files = append(manifest.Files, FileManifestEntry{
		Rel: "coverage.out", Size: 9, MTimeNS: 1, Hash: "deadbeef",
	})

	v, err := EvaluateFreshness(spec, manifest, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if v.Stale {
		t.Errorf("a legacy manifest listing coverage.out evaluated stale (%s: %s); every pre-existing manifest would rebuild at once", v.Reason, v.File)
	}
}

// The exclusion must not blind the check to real source changes.
func TestRealSourceChangeStillDetected(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "main.go")
	writeOutputTestFile(t, src, "package main\nfunc main(){}\n")
	spec := FreshnessSpec{SourceRoot: root, ContextRoot: root}
	manifest, err := ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	writeOutputTestFile(t, src, "package main\nfunc main(){println(1)}\n")

	v, err := EvaluateFreshness(spec, manifest, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !v.Stale {
		t.Error("a real source change was not detected")
	}
}
