package packs

import (
	"os"
	"path/filepath"
	"testing"

	"structure-health/internal/packs/scan"
)

// writeFixture lays down a file under root, creating parent dirs.
func writeFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// envViolatingAPIFile is a Go API file with an unvalidated os.Getenv usage,
// which the migrated env-validation rule flags (the auditor titles it
// "Environment variable missing validation").
const envViolatingAPIFile = `package main

import (
	"fmt"
	"os"
)

func main() {
	value := os.Getenv("SOME_REQUIRED_VAR")
	fmt.Println(value)
}
`

func buildScan(t *testing.T, scenario, root string) *scan.Context {
	t.Helper()
	sc, err := scan.Build(scenario, root)
	if err != nil {
		t.Fatalf("scan.Build: %v", err)
	}
	return sc
}

func codeCounts(fs []finding) map[string]int {
	out := map[string]int{}
	for _, f := range fs {
		out[f.Code]++
	}
	return out
}

// finding is a local alias so the test does not import internal/rules directly.
type finding = struct {
	Code        string
	Severity    string
	Title       string
	Message     string
	Location    string
	Remediation string
	Surface     string
}

func toLocal(t *testing.T, root, scenario string, recognized bool, profileID string) []finding {
	t.Helper()
	sc := buildScan(t, scenario, root)
	raw := Evaluate(profileID, recognized, sc)
	out := make([]finding, 0, len(raw))
	for _, f := range raw {
		out = append(out, finding{
			Code: f.Code, Severity: f.Severity, Title: f.Title,
			Message: f.Message, Location: f.Location, Remediation: f.Remediation, Surface: f.Surface,
		})
	}
	return out
}

// TestPerFileRuleFeedsAPIFiles proves the per-file config pack runs against
// api/ source and produces the env-validation finding (parity with the
// auditor's per-file standards scan).
func TestPerFileRuleFeedsAPIFiles(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "api/main.go", envViolatingAPIFile)

	fs := toLocal(t, root, "demo", true, DefaultProfileID)
	counts := codeCounts(fs)
	if counts["PROFILE_ENV_VALIDATION"] == 0 {
		t.Fatalf("expected PROFILE_ENV_VALIDATION finding from api/main.go, got %v", counts)
	}
	// Location must be scenario-relative, not absolute.
	for _, f := range fs {
		if f.Code == "PROFILE_ENV_VALIDATION" {
			if filepath.IsAbs(f.Location) {
				t.Fatalf("location should be scenario-relative, got %q", f.Location)
			}
		}
	}
}

func TestSharedPackageSourceBypassIsBlockingForRecognizedScenarios(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "ui/vite.config.ts", `export default { resolve: { alias: { "@vrooli/audio-capture-browser": "../../../packages/audio-capture-browser/src/index.ts" } } }`)

	fs := toLocal(t, root, "demo", true, DefaultProfileID)
	counts := codeCounts(fs)
	if counts["SCENARIO_SHARED_PACKAGE_BYPASS"] != 1 {
		t.Fatalf("expected one shared package bypass finding, got %v", counts)
	}
}

// TestUnrecognizedProfileDowngradesToAdvisory proves the core de-rigidification
// promise: a scenario whose profile is not the recognized default is never
// failed by react-vite/Go conventions — every pack finding becomes an advisory
// PROFILE_CONFORMANCE_VIOLATION warning instead of a blocking error.
// [REQ:SH-PROF-002]
func TestUnrecognizedProfileDowngradesToAdvisory(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "api/main.go", envViolatingAPIFile)

	enforced := toLocal(t, root, "demo", true, DefaultProfileID)
	if len(enforced) == 0 {
		t.Fatal("fixture produced no enforced findings; cannot test downgrade")
	}

	advisory := toLocal(t, root, "demo", false, "rust-axum")
	if len(advisory) != len(enforced) {
		t.Fatalf("advisory should preserve finding count: enforced=%d advisory=%d", len(enforced), len(advisory))
	}
	for _, f := range advisory {
		if f.Severity == "error" {
			t.Fatalf("unrecognized profile must not emit error severity, got %q for %q", f.Severity, f.Title)
		}
		if f.Code != "PROFILE_CONFORMANCE_VIOLATION" {
			t.Fatalf("unrecognized profile findings must be advisory PROFILE_CONFORMANCE_VIOLATION, got %q", f.Code)
		}
	}
}

// TestRecognizedProfileEnforcesSeverity proves critical/high-severity migrated
// rules block under the default profile (error severity), so verdicts match
// today's. A scenario missing its required top-level files trips the
// critical-severity Required Structure rule.
// [REQ:SH-PROF-002]
func TestRecognizedProfileEnforcesSeverity(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".vrooli/service.json", `{"name":"demo"}`)

	fs := toLocal(t, root, "demo", true, DefaultProfileID)
	counts := codeCounts(fs)
	if counts["PROFILE_REQUIRED_STRUCTURE"] == 0 {
		t.Fatalf("expected PROFILE_REQUIRED_STRUCTURE finding for a scenario missing required files, got %v", counts)
	}
	var sawError bool
	for _, f := range fs {
		if f.Code == "PROFILE_REQUIRED_STRUCTURE" && f.Severity == "error" {
			sawError = true
		}
	}
	if !sawError {
		t.Fatal("PROFILE_REQUIRED_STRUCTURE must be error severity under the default profile")
	}
}

// TestDeterministicOrder proves repeated evaluation yields identical findings.
func TestDeterministicOrder(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "api/main.go", envViolatingAPIFile)
	writeFixture(t, root, ".vrooli/service.json", `{"name":"demo"}`)

	a := toLocal(t, root, "demo", true, DefaultProfileID)
	b := toLocal(t, root, "demo", true, DefaultProfileID)
	if len(a) != len(b) {
		t.Fatalf("nondeterministic count: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("finding %d differs between runs:\n  %+v\n  %+v", i, a[i], b[i])
		}
	}
}
