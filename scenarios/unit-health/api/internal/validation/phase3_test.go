package validation

import (
	"path/filepath"
	"testing"
)

// --- B2: seam bypass-vs-absent matrix -----------------------------------

func goModForSeam(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "svc_test.go"), "package demo\n\nimport \"testing\"\n\nfunc TestN(t *testing.T) { N() }\n")
}

func TestSeamBypassWithDeclaredSeamIsNotFlagged(t *testing.T) {
	root := t.TempDir()
	goModForSeam(t, root)
	// Production bypasses time.Now() BUT the workspace declares a clock seam.
	writeFile(t, filepath.Join(root, "svc.go"), "package demo\n\nimport \"time\"\n\nfunc N() time.Time { return time.Now() }\n")
	writeFile(t, filepath.Join(root, "clock", "clock.go"), "package clock\n\nimport \"time\"\n\ntype Clock = TimeSource\n\ntype TimeSource interface { Now() time.Time }\n")

	findings := analyzeArchitecture("demo", []Workspace{{ID: "api", Language: "go", RootPath: root}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeMissingInjectableSeam); ok {
		t.Errorf("a declared clock seam should suppress MISSING_INJECTABLE_SEAM, got %v", codes(findings))
	}
}

func TestSeamBypassInCommentIsNotFlagged(t *testing.T) {
	root := t.TempDir()
	goModForSeam(t, root)
	// time.Now() appears only in a comment and a string — AST must ignore both.
	writeFile(t, filepath.Join(root, "svc.go"), "package demo\n\n// N does not call time.Now() directly.\nfunc N() string { return \"time.Now()\" }\n")

	findings := analyzeArchitecture("demo", []Workspace{{ID: "api", Language: "go", RootPath: root}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeMissingInjectableSeam); ok {
		t.Errorf("ambient name in a comment/string must not fire the seam finding, got %v", codes(findings))
	}
}

func TestSeamBypassNoSeamIsFlagged(t *testing.T) {
	root := t.TempDir()
	goModForSeam(t, root)
	writeFile(t, filepath.Join(root, "svc.go"), "package demo\n\nimport \"os\"\n\nfunc N() string { return os.Getenv(\"X\") }\n")

	findings := analyzeArchitecture("demo", []Workspace{{ID: "api", Language: "go", RootPath: root}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeMissingInjectableSeam); !ok {
		t.Errorf("os.Getenv with no env seam should fire MISSING_INJECTABLE_SEAM, got %v", codes(findings))
	}
}

func TestSeamBypassInMainIsNotFlagged(t *testing.T) {
	root := t.TempDir()
	goModForSeam(t, root)
	// The composition root may read config directly.
	writeFile(t, filepath.Join(root, "main.go"), "package demo\n\nimport \"os\"\n\nfunc main() { _ = os.Getenv(\"PORT\") }\n")

	findings := analyzeArchitecture("demo", []Workspace{{ID: "api", Language: "go", RootPath: root}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeMissingInjectableSeam); ok {
		t.Errorf("main.go config read must not fire the seam finding, got %v", codes(findings))
	}
}

// --- B4: local-helper assertion resolution ------------------------------

func TestAssertionThroughLocalHelperIsNotFlagged(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "x_test.go"), `package demo

import "testing"

func assertEqual(t *testing.T, a, b int) {
	if a != b {
		t.Fatalf("want %d got %d", b, a)
	}
}

func TestThroughHelper(t *testing.T) {
	assertEqual(t, 1, 1)
}
`)
	findings := analyzeQuality("demo", root, []Workspace{{ID: "api", Language: "go", RootPath: root}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestNoAssertion); ok {
		t.Errorf("a test asserting through a local helper must not be flagged assertion-free, got %v", codes(findings))
	}
}

func TestThroughSharedTestutilRequireHelper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "x_test.go"), `package demo

import (
    "testing"
    "demo/internal/testutil"
)

func TestResponse(t *testing.T) {
    testutil.RequireHTTPStatus(t, 200, 200)
}
`)
	findings := analyzeQuality("demo", root, []Workspace{{ID: "api", Language: "go", RootPath: root}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestNoAssertion); ok {
		t.Errorf("a test using testutil.Require* must not be flagged assertion-free, got %v", codes(findings))
	}
}

// Semantic edge-case coverage is a behavioral review, not a naming heuristic.

func TestQualityDoesNotInferMissingEdgeCasesFromNames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	// An unrelated string and a test name cannot establish behavioral coverage.
	writeFile(t, filepath.Join(root, "x_test.go"), `package demo

import "testing"

func TestHappy(t *testing.T) {
	got := "no error here, just a label"
	if got == "" {
		t.Fatal("x")
	}
}
`)
	findings := analyzeQuality("demo", root, []Workspace{{ID: "api", Language: "go", RootPath: root}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestMissingEdgeCases); ok {
		t.Errorf("static syntax cannot infer missing semantic cases, got %v", codes(findings))
	}
}

func TestQualityDoesNotTreatErrorMatcherAsEdgeCoverageProof(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "x_test.go"), `package demo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStuff(t *testing.T) {
	require.Error(t, doThing())
}

func doThing() error { return nil }
`)
	findings := analyzeQuality("demo", root, []Workspace{{ID: "api", Language: "go", RootPath: root}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestMissingEdgeCases); ok {
		t.Errorf("an assertion name cannot prove semantic coverage, got %v", codes(findings))
	}
}

// --- B1: per-file coverage findings + advisory default ------------------

func TestPerFileLowCoverageFindings(t *testing.T) {
	root := t.TempDir()
	wsDir := filepath.Join(root, "api")
	// a.go: 0/2 covered (0%, below). b.go: 2/2 (100%, ok). No testing.json =>
	// the 50% advisory default applies.
	writeFile(t, filepath.Join(wsDir, "coverage.out"), "mode: atomic\n"+
		"mod/a.go:1.1,3.2 2 0\n"+
		"mod/b.go:1.1,2.2 2 1\n")
	ws := Workspace{ID: "api", Language: "go", RootPath: wsDir, CoverageCommand: "go test -cover ./..."}
	_, findings := analyzeCoverage("demo", root, []Workspace{ws}, fixedNowStr)

	var perFile, hasA, hasB bool
	for _, f := range findings {
		if f.Code == codeLowCoverage && f.FilePath == "mod/a.go" {
			perFile, hasA = true, true
		}
		if f.Code == codeLowCoverage && f.FilePath == "mod/b.go" {
			hasB = true
		}
	}
	if !perFile || !hasA {
		t.Errorf("expected a per-file LOW_COVERAGE finding for mod/a.go, got %v", codes(findings))
	}
	if hasB {
		t.Errorf("mod/b.go is fully covered and must not be flagged")
	}
}

// --- B5: per-requirement untagged enumeration ---------------------------

func TestLegacyRequirementMentionsDoNotCreateTraceabilityClaims(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "requirements", "core.md"), "- REQ-A-001\n- REQ-B-002\n- REQ-C-003\n")
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	// Only REQ-A-001 is referenced; B and C are untagged.
	writeFile(t, filepath.Join(root, "x_test.go"), "package demo\n\nimport \"testing\"\n\n// covers REQ-A-001\nfunc TestX(t *testing.T){ t.Fatal(\"invalid\") }\n")
	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeQuality("demo", root, []Workspace{ws}, fixedNowStr)

	if _, ok := findingByCode(findings, codeTestUntaggedRequirement); ok {
		t.Errorf("legacy source mentions produced registry evidence: %v", codes(findings))
	}
}
