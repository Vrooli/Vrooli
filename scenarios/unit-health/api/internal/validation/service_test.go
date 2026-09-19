package validation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"unit-health/internal/discovery"
	"unit-health/internal/evidence"
	"unit-health/internal/testquality"

	"github.com/vrooli/maturity-go/assessment"
)

func TestMaturityWithoutFindingsDoesNotClaimExecutionOrReview(t *testing.T) {
	svc := &Service{Spec: loadSpec(t)}
	rationale := svc.assessMaturity(nil).Rationale
	for _, claim := range []string{"does not establish test execution", "complete assessment coverage", "behavioral review"} {
		if !strings.Contains(rationale, claim) {
			t.Fatalf("missing limit %q: %s", claim, rationale)
		}
	}
}

// fakeDiscoverer returns a canned inventory so tests never touch Code Facts.
type fakeDiscoverer struct {
	inv discovery.Inventory
	err error
}

func (f fakeDiscoverer) Discover(context.Context, string, string, string, bool) (discovery.Inventory, error) {
	return f.inv, f.err
}

func loadSpec(t *testing.T) *assessment.Spec {
	t.Helper()
	// Resolve repo root by walking up to the scenario descriptor.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// .../scenarios/unit-health/api/internal/validation -> .../scenarios/unit-health
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	spec, err := assessment.LoadSpecFromScenario(root)
	if err != nil {
		t.Fatalf("load descriptor maturity: %v", err)
	}
	return spec
}

func newService(disc discovery.Discoverer, spec *assessment.Spec) *Service {
	fixedNow := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	return &Service{Discoverer: disc, Spec: spec, RequirementRegistryReader: fixtureRegistry{}, Now: func() time.Time { return fixedNow }}
}

type fixtureRegistry struct {
	registry RequirementRegistry
	err      error
}

func TestCachedTraceabilityCannotBecomeCurrentPassingEvidence(t *testing.T) {
	prior := Response{RunID: "prior", Traceability: &testquality.TraceabilityReport{SchemaVersion: "requirement-traceability/v1", Links: []testquality.RequirementLink{{ID: "UH-CORE-001", Registration: "registered", Execution: testquality.ExecutionPassed, RunID: "old-command", Reason: testquality.ReasonNone}}}}
	payload, err := json.Marshal(prior)
	if err != nil {
		t.Fatal(err)
	}
	response, ok := cachedResponse(evidence.Record{Payload: payload}, "current")
	if !ok || response.Traceability == nil {
		t.Fatal("cache decode failed")
	}
	link := response.Traceability.Links[0]
	if link.Execution != testquality.ExecutionUnknown || link.Registration != "unknown" || link.Reason != testquality.StaleEvidence || link.RunID != "old-command" {
		t.Fatalf("cache manufactured current evidence: %+v", link)
	}
}

func (r fixtureRegistry) Read(context.Context, string) (RequirementRegistry, error) {
	if r.registry.SchemaVersion == "" && r.err == nil {
		return RequirementRegistry{SchemaVersion: "requirement-registry/v1", Requirements: []RequirementDeclaration{}}, nil
	}
	return r.registry, r.err
}

// [REQ:UH-BOUND-001]
func TestValidateNoSurfacesIsDegradedL0(t *testing.T) {
	spec := loadSpec(t)
	svc := newService(fakeDiscoverer{inv: discovery.Inventory{Scenario: "empty", DegradedReason: "Code Facts returned no surfaces"}}, spec)

	resp, err := svc.Validate(context.Background(), Request{Scenario: "empty"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if resp.Status != "degraded" {
		t.Errorf("status = %q, want degraded", resp.Status)
	}
	if resp.Maturity.Label != "L0" {
		t.Errorf("maturity = %q, want L0", resp.Maturity.Label)
	}
	if !hasFinding(resp.Findings, codeTestSurfaceAbsent) {
		t.Errorf("expected %s finding, got %+v", codeTestSurfaceAbsent, resp.Findings)
	}
	if len(resp.Plan.Commands) != 0 {
		t.Errorf("expected no planned commands, got %d", len(resp.Plan.Commands))
	}
}

// [REQ:UH-CORE-001]
func TestValidateGoSurfaceIsReady(t *testing.T) {
	spec := loadSpec(t)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "api", "go.mod"), "module x\n\ngo 1.25\n")
	inv := discovery.Inventory{
		Scenario: "demo", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api"), Status: "known"}},
	}
	svc := newService(fakeDiscoverer{inv: inv}, spec)

	resp, err := svc.Validate(context.Background(), Request{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if resp.Status != "passed" {
		t.Errorf("status = %q, want passed; findings=%+v", resp.Status, resp.Findings)
	}
	if len(resp.Workspaces) != 1 || resp.Workspaces[0].CanonicalFramework != "go test" {
		t.Fatalf("workspaces = %+v", resp.Workspaces)
	}
	ws := resp.Workspaces[0]
	if ws.Status != "ready" || ws.TestCommand != "go test -trimpath ./..." || ws.CoverageCommand == "" {
		t.Errorf("go workspace not ready: %+v", ws)
	}
	if len(resp.Plan.Commands) != 1 {
		t.Errorf("expected 1 planned command, got %+v", resp.Plan.Commands)
	}
}

type qualityDependencyResolver struct{}

func (qualityDependencyResolver) Resolve(context.Context, string, string, string) (DependencyClosure, error) {
	return DependencyClosure{Available: true, Source: "test fixture"}, nil
}

func TestValidateGoQualityWithoutExecutingTests(t *testing.T) {
	root := t.TempDir()
	apiRoot := filepath.Join(root, "api")
	writeFile(t, filepath.Join(apiRoot, "go.mod"), "module example.test/api\n\ngo 1.25\n")
	writeFile(t, filepath.Join(apiRoot, "quality_test.go"), `package example
import "testing"
func TestObserved(t *testing.T) { if 2+3 != 5 { t.Fatal("sum") } } // [REQ:UH-CORE-001]
func TestLogging(t *testing.T) { t.Log("no assertion") }
`)
	svc := newService(fakeDiscoverer{inv: discovery.Inventory{
		Scenario: "demo", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "api", Kind: "api", Language: "go", RootPath: apiRoot, Status: "known"}},
	}}, loadSpec(t))
	svc.DependencyResolver = qualityDependencyResolver{}
	svc.RequirementRegistryReader = fixtureRegistry{registry: RequirementRegistry{SchemaVersion: "requirement-registry/v1", Requirements: []RequirementDeclaration{{ID: "UH-CORE-001", Validations: []RequirementResponsibility{{Phase: "integration"}}}}}}
	resp, err := svc.Validate(context.Background(), Request{Scenario: "demo", IncludeExecution: false})
	if err != nil {
		t.Fatal(err)
	}
	report := resp.TestQuality.Normalized()
	if (report.UnavailableReason != "" && report.UnavailableReason != testquality.ReasonNone) || len(report.Results) != 2 {
		t.Fatalf("source quality unavailable: %+v", report)
	}
	statuses := map[string]testquality.Status{}
	for _, row := range report.Results {
		if row.EvidenceKind != testquality.Static || row.Target.Workspace != "api" {
			t.Fatalf("incorrect source provenance: %+v", row)
		}
		statuses[row.Target.TestID] = row.Status
	}
	if statuses["TestObserved"] != testquality.CheckedClean || statuses["TestLogging"] != testquality.Violation {
		t.Fatalf("wrong scoped observations: %+v", statuses)
	}
	if resp.Traceability == nil || len(resp.Traceability.Requirements) != 1 || resp.Traceability.Requirements[0].Applicability != "not_applicable" || len(resp.Traceability.Links) != 1 || resp.Traceability.Links[0].Execution != testquality.ExecutionNotRun || resp.Traceability.Links[0].Registration != "registered" || resp.Traceability.EvidenceUnavailableReason != testquality.NotExecuted {
		t.Fatalf("dry-run traceability invented execution/ownership: %+v", resp.Traceability)
	}
}

func TestValidateHonorsWorkspaceFilterBeforePlanningAndAnalysis(t *testing.T) {
	spec := loadSpec(t)
	root := t.TempDir()
	apiRoot := filepath.Join(root, "api")
	cliRoot := filepath.Join(root, "cli")
	writeFile(t, filepath.Join(apiRoot, "go.mod"), "module example.test/api\n\ngo 1.25\n")
	writeFile(t, filepath.Join(cliRoot, "go.mod"), "module example.test/cli\n\ngo 1.25\n")
	inv := discovery.Inventory{Scenario: "demo", TargetKind: "scenario", RootPath: root, Surfaces: []discovery.Surface{
		{ID: "api", Kind: "api", Language: "go", RootPath: apiRoot, Status: "known"},
		{ID: "cli", Kind: "cli", Language: "go", RootPath: cliRoot, Status: "known"},
	}}
	resp, err := newService(fakeDiscoverer{inv: inv}, spec).Validate(context.Background(), Request{Scenario: "demo", Workspaces: []string{"api"}})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(resp.Surfaces) != 1 || resp.Surfaces[0].ID != "api" || len(resp.Workspaces) != 1 || resp.Workspaces[0].ID != "api" || len(resp.Plan.Commands) != 1 || resp.Plan.Commands[0].WorkspaceID != "api" {
		t.Fatalf("filtered response surfaces=%+v workspaces=%+v plan=%+v", resp.Surfaces, resp.Workspaces, resp.Plan)
	}
}

func TestValidateUnknownWorkspaceFilterDoesNotFallBackToAllSurfaces(t *testing.T) {
	spec := loadSpec(t)
	root := t.TempDir()
	apiRoot := filepath.Join(root, "api")
	writeFile(t, filepath.Join(apiRoot, "go.mod"), "module example.test/api\n\ngo 1.25\n")
	inv := discovery.Inventory{Scenario: "demo", TargetKind: "scenario", RootPath: root, Surfaces: []discovery.Surface{{ID: "api", Kind: "api", Language: "go", RootPath: apiRoot, Status: "known"}}}
	resp, err := newService(fakeDiscoverer{inv: inv}, spec).Validate(context.Background(), Request{Scenario: "demo", Workspaces: []string{"missing"}})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(resp.Surfaces) != 0 || len(resp.Workspaces) != 0 || len(resp.Plan.Commands) != 0 || !hasFinding(resp.Findings, codeTestSurfaceAbsent) {
		t.Fatalf("unknown filter fell back to discovered surfaces: surfaces=%+v workspaces=%+v plan=%+v findings=%v", resp.Surfaces, resp.Workspaces, resp.Plan, codes(resp.Findings))
	}
}

func TestValidateJestUISurfaceIsNoncanonical(t *testing.T) {
	spec := loadSpec(t)
	root := t.TempDir()
	uiDir := filepath.Join(root, "ui")
	writeFile(t, filepath.Join(uiDir, "package.json"), `{
	  "name": "ui",
	  "scripts": {"test": "jest"},
	  "devDependencies": {"jest": "^29.0.0", "react": "^18.0.0"}
	}`)
	inv := discovery.Inventory{
		Scenario: "uidemo", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "ui", Kind: "ui", Language: "typescript", Framework: "react", RootPath: uiDir, PackageManager: "pnpm", Status: "known"}},
	}
	svc := newService(fakeDiscoverer{inv: inv}, spec)

	resp, err := svc.Validate(context.Background(), Request{Scenario: "uidemo"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !hasFinding(resp.Findings, codeTestFrameworkNoncanon) {
		t.Errorf("expected %s finding, got %+v", codeTestFrameworkNoncanon, codes(resp.Findings))
	}
	if resp.Status != "failed" {
		t.Errorf("status = %q, want failed (noncanonical is an error)", resp.Status)
	}
	// Jest is an L2 blocker; current level should be below L2.
	if resp.Maturity.Label != "L1" {
		t.Errorf("maturity = %q, want L1 (blocked at L2 by noncanonical framework)", resp.Maturity.Label)
	}
}

func TestValidateVitestUIIsReady(t *testing.T) {
	spec := loadSpec(t)
	root := t.TempDir()
	uiDir := filepath.Join(root, "ui")
	writeFile(t, filepath.Join(uiDir, "package.json"), `{
	  "name": "ui",
	  "scripts": {"test": "vitest run", "test:coverage": "vitest run --coverage"},
	  "devDependencies": {"vitest": "^2.0.0", "react": "^18.0.0", "vite": "^5.0.0"}
	}`)
	inv := discovery.Inventory{
		Scenario: "uidemo", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "ui", Kind: "ui", Language: "typescript", Framework: "react-vite", RootPath: uiDir, PackageManager: "pnpm", Status: "known"}},
	}
	svc := newService(fakeDiscoverer{inv: inv}, spec)

	resp, err := svc.Validate(context.Background(), Request{Scenario: "uidemo"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if resp.Status != "passed" {
		t.Errorf("status = %q, want passed; findings=%v", resp.Status, codes(resp.Findings))
	}
	ws := resp.Workspaces[0]
	if ws.CanonicalFramework != "vitest" || ws.TestCommand != "pnpm test" || ws.CoverageCommand != "pnpm test:coverage" {
		t.Errorf("vitest workspace wrong: %+v", ws)
	}
}

func TestValidatePythonIsDegradedFallback(t *testing.T) {
	spec := loadSpec(t)
	root := t.TempDir()
	pyDir := filepath.Join(root, "worker")
	writeFile(t, filepath.Join(pyDir, "pyproject.toml"), "[project]\nname='x'\n")
	inv := discovery.Inventory{
		Scenario: "pydemo", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "worker", Kind: "worker", Language: "python", RootPath: pyDir, Status: "known"}},
	}
	svc := newService(fakeDiscoverer{inv: inv}, spec)

	resp, err := svc.Validate(context.Background(), Request{Scenario: "pydemo"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !hasFinding(resp.Findings, codeUnsupportedParseUnit) {
		t.Errorf("expected %s finding, got %v", codeUnsupportedParseUnit, codes(resp.Findings))
	}
	ws := resp.Workspaces[0]
	if ws.Status != "degraded" || ws.CanonicalFramework != "pytest" {
		t.Errorf("python workspace = %+v", ws)
	}
	// UNSUPPORTED_PARSE_UNIT is info/advisory and must not block maturity.
	if resp.Status == "failing" {
		t.Errorf("python fallback should not be failing; status=%q", resp.Status)
	}
}

// [REQ:UH-BOUND-002]
func TestValidatePackageTargetUsesPackageKindWithoutScenarioMaturity(t *testing.T) {
	spec := loadSpec(t)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.test/package\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "seams.go"), `package packageapi

import "time"

type LocalClock interface { Now() time.Time }
type WiderClock interface { Now() time.Time; Sleep(time.Duration) }
`)
	inv := discovery.Inventory{
		Scenario: "api-core", TargetKind: "package", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "ignored-code-facts-surface", Language: "go", RootPath: root, Status: "known"}},
	}
	resp, err := newService(fakeDiscoverer{inv: inv}, spec).Validate(context.Background(), Request{Scenario: "api-core", TargetKind: "package"})
	if err != nil {
		t.Fatalf("Validate package: %v", err)
	}
	if resp.TargetKind != "package" {
		t.Fatalf("target kind = %q, want package", resp.TargetKind)
	}
	if len(resp.Workspaces) != 1 || resp.Workspaces[0].RootPath != root {
		t.Fatalf("package workspaces = %+v, want one workspace rooted at package", resp.Workspaces)
	}
	if resp.Maturity.Label != "L0" || resp.Maturity.Rationale == "" {
		t.Fatalf("package maturity = %+v, want explicit non-scenario L0", resp.Maturity)
	}
	if hasFinding(resp.Findings, codeTestUtilMissing) || hasFinding(resp.Findings, codeUnitPolicyInvalid) {
		t.Fatalf("package received scenario-shaped findings: %+v", resp.Findings)
	}
	if !hasFinding(resp.Findings, codeSeamDuplicatedInPackage) {
		t.Fatalf("package seam analyzer did not report duplicate interfaces: %+v", resp.Findings)
	}
}

func TestValidateUnsupportedTargetKindFailsWithoutClaimingMaturity(t *testing.T) {
	spec := loadSpec(t)
	root := t.TempDir()
	resp, err := newService(fakeDiscoverer{inv: discovery.Inventory{Scenario: "ollama", TargetKind: "resource", RootPath: root}}, spec).Validate(context.Background(), Request{Scenario: "ollama", TargetKind: "resource"})
	if err != nil {
		t.Fatalf("Validate resource: %v", err)
	}
	if resp.Status != "failed" || resp.TargetKind != "resource" {
		t.Fatalf("unsupported response = %+v", resp)
	}
	if !hasFinding(resp.Findings, codeUnsupportedTargetKind) {
		t.Fatalf("missing %s finding: %+v", codeUnsupportedTargetKind, resp.Findings)
	}
	if resp.Maturity.Label != "L0" {
		t.Fatalf("unsupported maturity = %+v, want L0", resp.Maturity)
	}
}

func TestValidateResponseExposesSuppressedFindingsSeparately(t *testing.T) {
	spec := loadSpec(t)
	root := t.TempDir()
	profile := reactViteUnitPolicyProfile()
	ui := profile.PolicyClasses["react_vite_ui"]
	ui.Coverage.MinimumPercent = 70
	profile.PolicyClasses["react_vite_ui"] = ui
	profile.Customization.Waivers = []unitPolicyWaiver{{
		Finding:   codeUnitPolicyWeakened,
		Reason:    "temporary policy exception while generated tests migrate",
		Owner:     "unit-health",
		ExpiresAt: "2026-07-01T00:00:00Z",
		Evidence:  "rec-123",
	}}
	writeUnitPolicyProfile(t, root, profile)
	resp, err := newService(fakeDiscoverer{inv: discovery.Inventory{
		Scenario: "demo", RootPath: root,
		Surfaces: []discovery.Surface{
			{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api"), Status: "known"},
			{ID: "cli", Kind: "cli", Language: "go", RootPath: filepath.Join(root, "cli"), Status: "known"},
			{ID: "ui", Kind: "ui", Language: "typescript", Framework: "vite", RootPath: filepath.Join(root, "ui"), Status: "known"},
		},
	}}, spec).Validate(context.Background(), Request{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if hasFinding(resp.Findings, codeUnitPolicyWeakened) {
		t.Fatalf("suppressed finding remained active: %+v", resp.Findings)
	}
	if len(resp.SuppressedFindings) != 1 || resp.SuppressedFindings[0].Code != codeUnitPolicyWeakened {
		t.Fatalf("suppressed findings = %+v, want one weakened-policy finding", resp.SuppressedFindings)
	}
	reasons := resp.SuppressedFindings[0].SuppressionReasons
	if len(reasons) != 1 || reasons[0].Reason != profile.Customization.Waivers[0].Reason || reasons[0].Owner != "unit-health" || reasons[0].Evidence != "rec-123" || reasons[0].ExpiresAt != "2026-07-01T00:00:00Z" {
		t.Fatalf("validated waiver details lost: %+v", reasons)
	}
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func codes(findings []Finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.Code)
	}
	return out
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

func TestBuildArtifacts(t *testing.T) {
	resp := Response{
		RunID:      "uh-123",
		TargetPath: "/repo/scenarios/demo",
		CommandResults: []CommandResult{
			{Name: "api test", Command: "go test ./...", WorkingDirectory: "/repo/scenarios/demo/api"},
			{Name: "", Command: "bats cli", WorkingDirectory: "/repo/scenarios/demo/cli"},
			{Name: "no-dir", Command: "noop"}, // no working dir -> skipped
		},
		Workspaces: []Workspace{
			{ID: "api", RootPath: "/repo/scenarios/demo/api", CoverageCommand: "go test -cover ./..."},
			{ID: "ui", RootPath: "/repo/scenarios/demo/ui", CoverageCommand: "pnpm test:coverage"}, // no coverage rows -> skipped
			{ID: "cli", RootPath: "/repo/scenarios/demo/cli"},                                      // no coverage command -> skipped
		},
		Coverage: []CoverageTarget{{SurfaceID: "api", FilePath: "a.go"}, {SurfaceID: "api", FilePath: "b.go"}},
	}

	arts := buildArtifacts(resp)

	want := []Artifact{
		{Label: "Validation run", Kind: "run", Reference: "uh-123"},
		{Label: "Target", Kind: "target", Reference: "/repo/scenarios/demo"},
		{Label: "api test", Kind: "command", Reference: "/repo/scenarios/demo/api"},
		{Label: "bats cli", Kind: "command", Reference: "/repo/scenarios/demo/cli"}, // empty name falls back to command
		{Label: "Coverage (api)", Kind: "coverage", Reference: "/repo/scenarios/demo/api"},
	}
	if len(arts) != len(want) {
		t.Fatalf("artifact count = %d, want %d: %+v", len(arts), len(want), arts)
	}
	for i, w := range want {
		if arts[i] != w {
			t.Errorf("artifact[%d] = %+v, want %+v", i, arts[i], w)
		}
	}
}
