package main

import (
	"os"
	"path/filepath"
	"testing"
)

// [REQ:TM-LS-009]
func TestScanSeamsMatchesCallsAndLiteralsWithoutTextFalsePositives(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "src/bypass.go", `package sample
import "os"
func bypass() { _ = os.Rename("a", "b"); _ = "VROOLI_REPO_ROOT" }
`)
	writeSeamTestFile(t, root, "src/clean.go", `package sample
// os.Rename("a", "b") and VROOLI_REPO_ROOT are documentation only.
func clean() {}
`)
	writeSeamTestFile(t, root, "src/aliased.go", `package sample
import filesystem "os"
func aliased() { _ = filesystem.Rename("a", "b") }
`)
	seams := []Seam{
		{ID: "owned-write", Canonical: "config.WriteOwnedFileAtomic", Why: "ownership", Remediation: "use the owned writer", Bypass: SeamBypass{Kind: "call", Pattern: `^os\.Rename$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
		{ID: "repo-root", Canonical: "buildinfo.ResolveSourceRoot", Why: "one root", Remediation: "use buildinfo", Bypass: SeamBypass{Kind: "literal", Pattern: `^VROOLI_REPO_ROOT$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
	}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("expected direct call, aliased call, and literal hits only, got %#v", hits)
	}
}

// [REQ:TM-LS-009]
func TestScanSeamsSkipsCanonicalDeclarationFile(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "config/write.go", `package config
import "os"
func WriteOwnedFileAtomic() { _ = os.Rename("a", "b") }
`)
	seam := Seam{ID: "owned-write", Canonical: "config.WriteOwnedFileAtomic", Why: "ownership", Remediation: "use the owned writer", Bypass: SeamBypass{Kind: "call", Pattern: `^os\.Rename$`}, Scope: SeamScope{Include: []string{"**"}}, Severity: "high"}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("canonical declaration must not report its implementation detail: %#v", hits)
	}
}

// [REQ:TM-LS-009]
func TestScanSeamsMatchesNumericAndQualifiedDurationLiterals(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "src/literals.go", `package sample
import tm "time"
var mode = 0o644
var timeout = 30 * tm.Second
const namedTimeout = 45 * tm.Second
`)
	seams := []Seam{
		{ID: "mode", Canonical: "tuning.PermFile", Why: "named modes", Remediation: "use tuning", Bypass: SeamBypass{Kind: "literal", Pattern: `^0[oO][0-7]+$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
		{ID: "duration", Canonical: "tuning.Duration", Why: "named durations", Remediation: "use tuning", Bypass: SeamBypass{Kind: "literal", Pattern: `^time\.(Second|Minute|Hour|Millisecond|Microsecond|Nanosecond):[0-9]+$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
	}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected numeric and qualified-duration hits, got %#v", hits)
	}
}

// [REQ:TM-LS-009]
func TestScanSeamsHonorsExcludeAndBudget(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "src/one.go", "package sample\nfunc one(){ bypass() }\n")
	writeSeamTestFile(t, root, "generated/two.go", "package sample\nfunc two(){ bypass() }\n")
	seam := Seam{ID: "call", Canonical: "canonical", Why: "reason", Remediation: "fix", Bypass: SeamBypass{Kind: "call", Pattern: `^bypass$`}, Scope: SeamScope{Include: []string{"**"}, Exclude: []string{"generated/**"}}, Severity: "high", Budget: 1}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected one non-excluded hit, got %#v", hits)
	}
	if findings := seamFindings("sample", hits); len(findings) != 0 {
		t.Fatalf("budgeted hit must not gate: %#v", findings)
	}
	seam.Budget = 0
	hits, err = ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if findings := seamFindings("sample", hits); len(findings) != 1 || findings[0].RuleID != "BYPASSED_SEAM" {
		t.Fatalf("expected one gating finding, got %#v", findings)
	}
}

// [REQ:TM-LS-009]
func TestLoadSeamsRejectsUnknownFields(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, canonicalSeamsPath, `{"schemaVersion":"1.0.0","unknown":true,"seams":[]}`)
	if _, err := LoadSeams(root); err == nil {
		t.Fatal("expected unknown field to fail strict decoding")
	}
}

// [REQ:TM-LS-009]
func TestLoadSeamsRejectsTrailingJSON(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, canonicalSeamsPath, `{"schemaVersion":"1.0.0","seams":[]} {}`)
	if _, err := LoadSeams(root); err == nil {
		t.Fatal("expected trailing JSON value to fail strict decoding")
	}
}

// [REQ:TM-LS-009]
func TestRepositoryCanonicalSeamsAreClean(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	if got := seamTreeRoot(filepath.Join(root, "internal")); got != root {
		t.Fatalf("control-plane seam root = %q, want repository root %q", got, root)
	}
	seams, err := LoadSeams(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(seams) != 10 {
		t.Fatalf("expected ten declared seams, got %d", len(seams))
	}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	if findings := seamFindings("control-plane", hits); len(findings) != 0 {
		t.Fatalf("repository seam rules found above-budget bypasses: %#v", findings)
	}
}

func writeSeamTestFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
