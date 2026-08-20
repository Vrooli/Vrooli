package validation

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func writeCompanionFixture(t *testing.T, root string) {
	t.Helper()
	registry := companionRegistry{
		SchemaVersion: "1.0.0",
		Companions: []companionExport{{
			Owner:           "api-core/schedule",
			OwnerImportPath: "github.com/vrooli/api-core/schedule",
			ImportPath:      "github.com/vrooli/api-core/scheduletest",
			Symbols: []companionSymbol{{
				Name:    "FakeClock",
				Kind:    "type",
				Methods: []string{"Now", "Advance", "NewTimer", "NewTicker", "Sleep"},
			}, {
				// A companion export whose name is common enough to collide
				// with unrelated local constructors.
				Name:      "New",
				Kind:      "function",
				Signature: "func(start time.Time) *FakeClock",
			}},
		}, {
			Owner:      "api-core/apihttp",
			ImportPath: "github.com/vrooli/api-core/apihttptest",
			Symbols: []companionSymbol{{
				Name:      "MustDecodeJSON",
				Kind:      "function",
				Signature: "func[T any](t *testing.T, body []byte) T",
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

// TestCompanionReachabilityHoldsWithoutDependencyAnalyzer pins the behaviour
// that made the error severity unreachable in practice: a workspace whose
// go.mod requires the companion's module is provably able to import it, so the
// verdict must not soften to an advisory suggestion just because Scenario
// Dependency Analyzer is stopped.
func TestCompanionReachabilityHoldsWithoutDependencyAnalyzer(t *testing.T) {
	root := t.TempDir()
	writeCompanionFixture(t, root)
	writeFakeClockFixture(t, root)
	writeFile(t, filepath.Join(root, "go.mod"), `module fixture

go 1.25

require github.com/vrooli/api-core v0.0.0
`)
	ws := Workspace{ID: "api", Language: "go", RootPath: root}

	// The zero closure is exactly what the service passes when the resolver
	// returned an error.
	findings := analyzeGoCompanionDeclarations("demo", ws, fixedNowStr, DependencyClosure{})
	f, ok := findingByCode(findings, codeCompanionReimplemented)
	if !ok || f.Severity != "error" {
		t.Fatalf("go.mod-proven companion = %+v, want COMPANION_REIMPLEMENTED/error", findings)
	}
	if !strings.Contains(f.Evidence, "closure_source=go.mod") {
		t.Errorf("evidence must name the go.mod source, got %q", f.Evidence)
	}
}

// TestCompanionReachabilityStaysUnprovenWithoutEvidence keeps the fix from
// escalating everything: a workspace that does not require the module has not
// proven anything, and the finding stays advisory.
func TestCompanionReachabilityStaysUnprovenWithoutEvidence(t *testing.T) {
	root := t.TempDir()
	writeCompanionFixture(t, root)
	writeFakeClockFixture(t, root)
	writeFile(t, filepath.Join(root, "go.mod"), "module fixture\n\ngo 1.25\n")
	ws := Workspace{ID: "api", Language: "go", RootPath: root}

	findings := analyzeGoCompanionDeclarations("demo", ws, fixedNowStr, DependencyClosure{})
	if _, ok := findingByCode(findings, codeCompanionReimplemented); ok {
		t.Fatalf("unrequired companion must not report an error: %+v", findings)
	}
	if _, ok := findingByCode(findings, codeCompanionAvailable); !ok {
		t.Fatalf("unrequired companion should still be suggested: %+v", findings)
	}
}

// TestModuleClosureDoesNotLeakBetweenWorkspaces guards the per-workspace merge.
// api and cli are separate modules; one requiring api-core must not make the
// other look as though it does.
func TestModuleClosureDoesNotLeakBetweenWorkspaces(t *testing.T) {
	shared := DependencyClosure{Imports: map[string]bool{}, Source: "scenario-dependency-analyzer"}

	requiring := t.TempDir()
	writeFile(t, filepath.Join(requiring, "go.mod"), "module api\n\ngo 1.25\n\nrequire github.com/vrooli/api-core v0.0.0\n")
	widened := mergeModuleClosure(requiring, shared)
	if !widened.Available || !importReachable("github.com/vrooli/api-core/scheduletest", widened) {
		t.Fatalf("requiring workspace closure = %+v, want api-core reachable", widened)
	}

	bare := t.TempDir()
	writeFile(t, filepath.Join(bare, "go.mod"), "module cli\n\ngo 1.25\n")
	if second := mergeModuleClosure(bare, shared); importReachable("github.com/vrooli/api-core/scheduletest", second) {
		t.Fatalf("closure leaked into an unrelated workspace: %+v", second)
	}
	if len(shared.Imports) != 0 {
		t.Fatalf("mergeModuleClosure mutated its argument: %+v", shared)
	}
}

// TestStructuralMatchCatchesUnexportedReimplementation covers the blind spot
// that let the fleet's clock fakes pass: the ordinary Go idiom keeps a fake
// unexported, so `fakeClock` never matches `FakeClock` by name however exactly
// it rebuilds it.
func TestStructuralMatchCatchesUnexportedReimplementation(t *testing.T) {
	root := t.TempDir()
	writeCompanionFixture(t, root)
	writeFile(t, filepath.Join(root, "go.mod"), "module fixture\n\ngo 1.25\n\nrequire github.com/vrooli/api-core v0.0.0\n")
	writeFile(t, filepath.Join(root, "scheduler.go"), `package fixture

import _ "github.com/vrooli/api-core/schedule"
`)
	writeFile(t, filepath.Join(root, "scheduler_test.go"), `package fixture

type fakeClock struct{}

func (c *fakeClock) Now()     {}
func (c *fakeClock) Advance() {}
`)
	ws := Workspace{ID: "api", Language: "go", RootPath: root}

	findings := analyzeGoCompanionDeclarations("demo", ws, fixedNowStr, DependencyClosure{})
	f, ok := findingByCode(findings, codeCompanionReimplemented)
	if !ok {
		t.Fatalf("unexported clock fake = %+v, want COMPANION_REIMPLEMENTED", findings)
	}
	if f.Symbol != "fakeClock" {
		t.Errorf("symbol = %q, want fakeClock", f.Symbol)
	}
	if !strings.Contains(f.Evidence, "match=shape") {
		t.Errorf("evidence must record how the match was made, got %q", f.Evidence)
	}
	if !strings.Contains(f.Remediation, "FakeClock") {
		t.Errorf("remediation must name the companion symbol, got %q", f.Remediation)
	}
}

// TestAdaptedHelperMatchSurvivesSignatureDrift covers the tier between exact
// and structural matching: the same named helper rebuilt around a different
// input. vrooli-autoheal's MustDecodeJSON reads the recorder where the
// companion reads the body, and exact matching cannot see that.
func TestAdaptedHelperMatchSurvivesSignatureDrift(t *testing.T) {
	root := t.TempDir()
	writeCompanionFixture(t, root)
	writeFile(t, filepath.Join(root, "go.mod"), "module fixture\n\ngo 1.25\n\nrequire github.com/vrooli/api-core v0.0.0\n")
	writeFile(t, filepath.Join(root, "testutil", "http.go"), `package testutil

import (
	"net/http/httptest"
	"testing"
)

func MustDecodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	var out T
	return out
}

func New(handler int) *httptest.Server { return nil }
`)
	ws := Workspace{ID: "api", Language: "go", RootPath: root}

	findings := analyzeGoCompanionDeclarations("demo", ws, fixedNowStr, DependencyClosure{})
	f, ok := findingByCode(findings, codeCompanionReimplemented)
	if !ok || f.Symbol != "MustDecodeJSON" {
		t.Fatalf("adapted helper = %+v, want COMPANION_REIMPLEMENTED for MustDecodeJSON", findings)
	}
	if !strings.Contains(f.Evidence, "match=adapted") {
		t.Errorf("evidence must record how the match was made, got %q", f.Evidence)
	}
	// New collides with scheduletest.New by name but takes no testing handle,
	// so it is an ordinary constructor and must not be accused.
	for _, other := range findings {
		if other.Symbol == "New" {
			t.Errorf("plain constructor must not match a companion by name alone: %+v", other)
		}
	}
}

// TestStructuralMatchIgnoresProductionAndThinFakes holds the two limits that
// keep the structural rule from becoming noise: production types are not
// duplicates of a test companion, and a single shared method is not evidence.
func TestStructuralMatchIgnoresProductionAndThinFakes(t *testing.T) {
	root := t.TempDir()
	writeCompanionFixture(t, root)
	writeFile(t, filepath.Join(root, "go.mod"), "module fixture\n\ngo 1.25\n\nrequire github.com/vrooli/api-core v0.0.0\n")
	writeFile(t, filepath.Join(root, "runtime.go"), `package fixture

import _ "github.com/vrooli/api-core/schedule"

type systemClock struct{}

func (c *systemClock) Now()     {}
func (c *systemClock) Advance() {}
`)
	writeFile(t, filepath.Join(root, "thin_test.go"), `package fixture

type stubTimer struct{}

func (s *stubTimer) Now() {}
`)
	ws := Workspace{ID: "api", Language: "go", RootPath: root}

	for _, f := range analyzeGoCompanionDeclarations("demo", ws, fixedNowStr, DependencyClosure{}) {
		if f.Symbol == "systemClock" {
			t.Errorf("production type must not be reported as a companion duplicate: %+v", f)
		}
		if f.Symbol == "stubTimer" {
			t.Errorf("single-method fake is not evidence of reimplementation: %+v", f)
		}
	}
}

// TestShapeMatchRequiresTheOwnedSeamInScope is the guard against the class of
// false positive shape matching invites. swarm-manager's fakeGoalReader exposes
// {Get, List} — a subset of databasetest.SliceRepo — while being a domain
// reader with no connection to a repository. Its package never imports
// api-core/database, and that is what tells the two apart.
func TestShapeMatchRequiresTheOwnedSeamInScope(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".vrooli", "test-companions.json"), `{
  "schema_version": "1.0.0",
  "companions": [
    {
      "owner": "api-core/database",
      "owner_import_path": "github.com/vrooli/api-core/database",
      "import_path": "github.com/vrooli/api-core/databasetest",
      "symbols": [
        {"name": "SliceRepo", "kind": "type", "methods": ["Create", "Get", "List"]}
      ]
    }
  ],
  "seams": []
}
`)
	writeFile(t, filepath.Join(root, "go.mod"), "module fixture\n\ngo 1.25\n\nrequire github.com/vrooli/api-core v0.0.0\n")
	writeFile(t, filepath.Join(root, "aisearch", "service_test.go"), `package aisearch

type fakeGoalReader struct{}

func (f *fakeGoalReader) Get()  {}
func (f *fakeGoalReader) List() {}
`)
	writeFile(t, filepath.Join(root, "store", "store_test.go"), `package store

import _ "github.com/vrooli/api-core/database"

type fakeRepo struct{}

func (f *fakeRepo) Get()  {}
func (f *fakeRepo) List() {}
`)
	ws := Workspace{ID: "api", Language: "go", RootPath: root}

	findings := analyzeGoCompanionDeclarations("demo", ws, fixedNowStr, DependencyClosure{})
	reported := map[string]bool{}
	for _, f := range findings {
		reported[f.Symbol] = true
	}
	if reported["fakeGoalReader"] {
		t.Errorf("a domain fake in a package that never uses the seam must not match: %+v", findings)
	}
	if !reported["fakeRepo"] {
		t.Errorf("a fake in a package that uses the seam must still match: %+v", findings)
	}
}

// TestCanonicalCompanionExclusionIsImportPathScoped proves the self-exclusion
// follows the registry rather than a directory name. A scenario package that
// merely shares a base name with a registered export is the local duplicate
// these rules exist to catch, and excluding it by name would hide it.
func TestCanonicalCompanionExclusionIsImportPathScoped(t *testing.T) {
	registry := companionRegistry{
		SchemaVersion: companionRegistrySchemaVersion,
		Seams: []companionExport{{
			Owner:      "api-core/schedule",
			ImportPath: "github.com/vrooli/api-core/schedule",
			Symbols:    []companionSymbol{{Name: "Clock", Kind: "interface", Methods: []string{"Now"}}},
		}},
	}

	owner := t.TempDir()
	writeFile(t, filepath.Join(owner, "go.mod"), "module github.com/vrooli/api-core\n\ngo 1.25\n")
	if !isCanonicalCompanionDeclaration(owner, filepath.Join(owner, "schedule", "schedule.go"), registry) {
		t.Error("the owning package must be excluded from its own rules")
	}

	consumer := t.TempDir()
	writeFile(t, filepath.Join(consumer, "go.mod"), "module search-hub\n\ngo 1.25\n")
	if isCanonicalCompanionDeclaration(consumer, filepath.Join(consumer, "internal", "schedule", "clock.go"), registry) {
		t.Error("a scenario package sharing a base name must still be analyzed")
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
