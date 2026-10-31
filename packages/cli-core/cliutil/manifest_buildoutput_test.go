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

// The installed-CLI stale checker uses the same canonical manifest enumeration
// as the stat-cache manifest above. Pin the build-output exclusion at that
// shared seam because a mismatch can restart a scenario API.
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
	for _, name := range []string{"coverage.out", "cov.out", "cover.out", "api.log", "pkg.test", "jscpd-report.json", ".vrooli-closure-go_module.json", ".vrooli-closure-go_module.json.tmp"} {
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

func TestCoverageOutputRootsDoNotHideNestedGoSource(t *testing.T) {
	root := t.TempDir()
	writeOutputTestFile(t, filepath.Join(root, ".vrooli", "repo-contract.json"), "{}\n")
	writeOutputTestFile(t, filepath.Join(root, ".gitignore"), "/coverage/\npackages/*/coverage/\nscenarios/*/ui/coverage/\ntemplates/scenarios/*/ui/coverage/\n")
	writeOutputTestFile(t, filepath.Join(root, "api", "internal", "checks", "coverage", "check.go"), "package coverage\nconst Value = 1\n")
	writeOutputTestFile(t, filepath.Join(root, "api", "internal", "ui", "coverage", "check.go"), "package coverage\nconst UIValue = 1\n")
	writeOutputTestFile(t, filepath.Join(root, "coverage", "index.html"), "old root report")
	writeOutputTestFile(t, filepath.Join(root, "scenarios", "demo", "ui", "coverage", "index.html"), "old scenario ui report")
	spec := FreshnessSpec{SourceRoot: root, ContextRoot: root, Inputs: []string{"."}}
	manifest, err := ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	foundSource := false
	foundNestedUISource := false
	for _, file := range manifest.Files {
		if file.Rel == "api/internal/checks/coverage/check.go" {
			foundSource = true
		}
		if file.Rel == "api/internal/ui/coverage/check.go" {
			foundNestedUISource = true
		}
		if file.Rel == "coverage/index.html" || file.Rel == "scenarios/demo/ui/coverage/index.html" {
			t.Errorf("generated coverage output was recorded as input: %s", file.Rel)
		}
	}
	if !foundSource {
		t.Fatal("nested Go source under a coverage package was omitted from the freshness manifest")
	}
	if !foundNestedUISource {
		t.Fatal("nested Go source under an internal ui/coverage package was omitted from the freshness manifest")
	}

	writeOutputTestFile(t, filepath.Join(root, "coverage", "index.html"), "new root report")
	writeOutputTestFile(t, filepath.Join(root, "scenarios", "demo", "ui", "coverage", "index.html"), "new scenario ui report")
	verdict, err := EvaluateFreshness(spec, manifest, nil)
	if err != nil {
		t.Fatalf("evaluate after output churn: %v", err)
	}
	if verdict.Stale {
		t.Fatalf("coverage output churn made inputs stale (%s: %s)", verdict.Reason, verdict.File)
	}

	writeOutputTestFile(t, filepath.Join(root, "api", "internal", "checks", "coverage", "check.go"), "package coverage\nconst Value = 2\n")
	verdict, err = EvaluateFreshness(spec, manifest, nil)
	if err != nil {
		t.Fatalf("evaluate after source edit: %v", err)
	}
	if !verdict.Stale || verdict.File != "api/internal/checks/coverage/check.go" {
		t.Fatalf("nested Go source edit was not reported stale: %+v", verdict)
	}
}

func TestCoverageOutputDirClassificationIsPathSpecific(t *testing.T) {
	root := t.TempDir()
	writeOutputTestFile(t, filepath.Join(root, ".vrooli", "repo-contract.json"), "{}\n")
	writeOutputTestFile(t, filepath.Join(root, ".gitignore"), "/coverage/\npackages/*/coverage/\nscenarios/*/ui/coverage/\ntemplates/scenarios/*/ui/coverage/\n")
	for _, path := range []string{"coverage", "packages/demo/coverage", "scenarios/demo/ui/coverage", "templates/scenarios/demo/ui/coverage"} {
		if !isCoverageOutputDir(root, path) {
			t.Errorf("expected generated coverage output path %q to match", path)
		}
	}
	for _, path := range []string{"ui/coverage", "api/internal/coverage", "api/internal/checks/coverage", "api/internal/ui/coverage", "scenarios/demo/coverage", "scenarios/demo/api/internal/coverage", "scenarios/demo/api/internal/ui/coverage"} {
		if isCoverageOutputDir(root, path) {
			t.Errorf("authored source path %q was classified as coverage output", path)
		}
	}
}

func TestNoGitFallbackKeepsNestedCoverageSourceAndSkipsDeclaredReportRoots(t *testing.T) {
	root := t.TempDir()
	writeOutputTestFile(t, filepath.Join(root, ".vrooli", "repo-contract.json"), "{}\n")
	writeOutputTestFile(t, filepath.Join(root, ".gitignore"), "/coverage/\npackages/*/coverage/\nscenarios/*/ui/coverage/\ntemplates/scenarios/*/ui/coverage/\n")
	writeOutputTestFile(t, filepath.Join(root, "api", "internal", "checks", "coverage", "check.go"), "package coverage\nconst Value = 1\n")
	writeOutputTestFile(t, filepath.Join(root, "api", "internal", "ui", "coverage", "check.go"), "package coverage\nconst UIValue = 1\n")
	writeOutputTestFile(t, filepath.Join(root, "coverage", "index.html"), "root report")
	writeOutputTestFile(t, filepath.Join(root, "scenarios", "demo", "ui", "coverage", "index.html"), "scenario UI report")
	writeOutputTestFile(t, filepath.Join(root, "scenarios", "demo", "coverage", "authored.go"), "package coverage\n")

	files, source, err := EnumerateInputs(root, []string{"."}, EnumerateDeps{
		Command: func(string, ...string) ([]byte, error) { return nil, os.ErrNotExist },
	})
	if err != nil {
		t.Fatalf("EnumerateInputs fallback: %v", err)
	}
	if source != EnumerationWalk {
		t.Fatalf("enumeration source = %q, want walk fallback", source)
	}
	want := map[string]bool{
		"api/internal/checks/coverage/check.go": true,
		"api/internal/ui/coverage/check.go":     true,
		"scenarios/demo/coverage/authored.go":   true,
	}
	for _, file := range files {
		if file == "coverage/index.html" || file == "scenarios/demo/ui/coverage/index.html" {
			t.Errorf("fallback included declared report root %q", file)
		}
		delete(want, file)
	}
	if len(want) != 0 {
		t.Fatalf("fallback omitted authored coverage paths: %v; got %v", want, files)
	}

	spec := FreshnessSpec{SourceRoot: root}
	manifest, err := ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("compute manifest through empty-input fallback: %v", err)
	}
	manifestFiles := make(map[string]bool, len(manifest.Files))
	for _, file := range manifest.Files {
		manifestFiles[file.Rel] = true
	}
	if !manifestFiles["api/internal/checks/coverage/check.go"] || !manifestFiles["api/internal/ui/coverage/check.go"] || !manifestFiles["scenarios/demo/coverage/authored.go"] {
		t.Fatalf("empty-input fallback manifest omitted authored coverage paths: %v", manifestFiles)
	}
	if manifestFiles["coverage/index.html"] || manifestFiles["scenarios/demo/ui/coverage/index.html"] {
		t.Fatalf("empty-input fallback manifest included a declared report root: %v", manifestFiles)
	}
	writeOutputTestFile(t, filepath.Join(root, "api", "internal", "checks", "coverage", "check.go"), "package coverage\nconst Value = 2\n")
	verdict, err := EvaluateFreshness(spec, manifest, nil)
	if err != nil {
		t.Fatalf("evaluate empty-input fallback after source edit: %v", err)
	}
	if !verdict.Stale || verdict.File != "api/internal/checks/coverage/check.go" {
		t.Fatalf("empty-input fallback did not detect nested source edit: %+v", verdict)
	}
}

// Source trees can contain links used by fixtures or development tooling. A
// broken link is not a source file and must not make a successful build
// impossible to stamp.
func TestBrokenSymlinkIsIgnoredByFreshnessManifest(t *testing.T) {
	root := t.TempDir()
	writeOutputTestFile(t, filepath.Join(root, "main.go"), "package main\n")
	if err := os.Symlink(filepath.Join(root, "does-not-exist"), filepath.Join(root, "broken-fixture")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	spec := FreshnessSpec{SourceRoot: root, ContextRoot: root, Inputs: []string{"."}}
	manifest, err := ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("compute with broken symlink: %v", err)
	}
	for _, file := range manifest.Files {
		if file.Rel == "broken-fixture" {
			t.Fatal("broken symlink was recorded as a freshness input")
		}
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
