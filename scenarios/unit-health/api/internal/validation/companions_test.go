package validation

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func writeCompanionFixture(t *testing.T, root string) {
	t.Helper()
	registry := companionRegistry{
		SchemaVersion: "1.0.0",
		Companions: []companionExport{{
			Owner:      "api-core/schedule",
			ImportPath: "github.com/vrooli/api-core/scheduletest",
			Symbols: []companionSymbol{{
				Name:    "FakeClock",
				Kind:    "type",
				Methods: []string{"Now", "Advance", "NewTimer", "NewTicker", "Sleep"},
			}},
		}},
		Seams: []companionExport{{
			Owner:      "api-core/schedule",
			ImportPath: "github.com/vrooli/api-core/schedule",
			Symbols: []companionSymbol{{
				Name:    "Clock",
				Kind:    "interface",
				Methods: []string{"Now", "Sleep", "NewTimer", "NewTicker"},
			}},
		}},
	}
	raw, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".vrooli", "test-companions.json"), string(raw))
}

func writeFakeClockFixture(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "fake_clock.go"), `package fixture

type FakeClock struct{}
func (FakeClock) Now() {}
func (FakeClock) Advance() {}
func (FakeClock) NewTimer() {}
func (FakeClock) NewTicker() {}
func (FakeClock) Sleep() {}
`)
}

func TestCompanionRulesUseClosureForSeverity(t *testing.T) {
	root := t.TempDir()
	writeCompanionFixture(t, root)
	writeFakeClockFixture(t, root)
	ws := Workspace{ID: "api", Language: "go", RootPath: root}

	reachable := DependencyClosure{
		Imports:   map[string]bool{"github.com/vrooli/api-core/scheduletest": true},
		Source:    "scenario-dependency-analyzer",
		Available: true,
	}
	findings := analyzeGoCompanionDeclarations("demo", ws, fixedNowStr, reachable)
	f, ok := findingByCode(findings, codeCompanionReimplemented)
	if !ok || f.Severity != "error" {
		t.Fatalf("reachable companion = %+v, want COMPANION_REIMPLEMENTED/error", findings)
	}
	if f.Symbol != "FakeClock" || f.Remediation == "" {
		t.Fatalf("reachable finding lacks symbol/remediation: %+v", f)
	}

	outside := reachable
	outside.Imports = map[string]bool{}
	findings = analyzeGoCompanionDeclarations("demo", ws, fixedNowStr, outside)
	f, ok = findingByCode(findings, codeCompanionAvailable)
	if !ok || f.Severity != "info" {
		t.Fatalf("outside companion = %+v, want COMPANION_AVAILABLE/info", findings)
	}
}

func TestSeamRuleUsesReachableCanonicalOwner(t *testing.T) {
	root := t.TempDir()
	writeCompanionFixture(t, root)
	writeFile(t, filepath.Join(root, "clock.go"), `package fixture

type Clock interface {
	Now()
	Sleep()
	NewTimer()
	NewTicker()
}
`)
	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	closure := DependencyClosure{
		Imports:   map[string]bool{"github.com/vrooli/api-core/schedule": true},
		Source:    "scenario-dependency-analyzer",
		Available: true,
	}
	findings := analyzeGoCompanionDeclarations("demo", ws, fixedNowStr, closure)
	f, ok := findingByCode(findings, codeSeamReimplemented)
	if !ok || f.Severity != "error" || f.Symbol != "Clock" {
		t.Fatalf("reachable seam = %+v, want SEAM_REIMPLEMENTED/error for Clock", findings)
	}
}

func TestAdoptedCompanionSatisfiesTestUtilRequirement(t *testing.T) {
	root := t.TempDir()
	writeCompanionFixture(t, root)
	writeFile(t, filepath.Join(root, "go.mod"), "module fixture\n\ngo 1.25\n")
	for _, name := range []string{"a", "b", "c"} {
		writeFile(t, filepath.Join(root, name+"_test.go"), "package fixture\n\nimport _ \"github.com/vrooli/api-core/scheduletest\"\n")
	}
	findings := analyzeGoArchitectureWithClosure("demo", Workspace{ID: "api", Language: "go", RootPath: root}, fixedNowStr, DependencyClosure{})
	if _, ok := findingByCode(findings, codeTestUtilMissing); ok {
		t.Fatalf("adopted companion must satisfy TEST_UTIL_MISSING: %+v", findings)
	}
	if _, ok := findingByCode(findings, codeUnitProjectionDrift); ok {
		t.Fatalf("projection drift must not require a local meta-test without local testutil: %+v", findings)
	}
}

func TestCompanionRuleWaiversSuppressAllNewCodes(t *testing.T) {
	waivers := []unitPolicyWaiver{
		{Finding: codeSeamReimplemented, Reason: "migration in progress", Owner: "testing-team", Evidence: "ticket-1", ExpiresAt: "2099-01-01T00:00:00Z"},
		{Finding: codeCompanionReimplemented, Reason: "migration in progress", Owner: "testing-team", Evidence: "ticket-2", ExpiresAt: "2099-01-01T00:00:00Z"},
		{Finding: codeCompanionAvailable, Reason: "migration scheduled", Owner: "testing-team", Evidence: "ticket-3", ExpiresAt: "2099-01-01T00:00:00Z"},
	}
	findings := []Finding{
		{Code: codeSeamReimplemented},
		{Code: codeCompanionReimplemented},
		{Code: codeCompanionAvailable},
	}
	marked := applyUnitPolicyWaivers(findings, waivers, "demo", "testing.json", fixedNowStr)
	if len(marked) != len(waivers) {
		t.Fatalf("waiver result length = %d, want %d", len(marked), len(waivers))
	}
	for _, finding := range marked {
		if !finding.Suppressed {
			t.Errorf("finding %s was not suppressed", finding.Code)
		}
	}
}
