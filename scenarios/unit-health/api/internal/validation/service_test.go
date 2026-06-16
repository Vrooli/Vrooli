package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"unit-health/internal/discovery"

	"github.com/vrooli/maturity-go/assessment"
)

// fakeDiscoverer returns a canned inventory so tests never touch Code Facts.
type fakeDiscoverer struct {
	inv discovery.Inventory
	err error
}

func (f fakeDiscoverer) Discover(context.Context, string, string, bool) (discovery.Inventory, error) {
	return f.inv, f.err
}

func loadSpec(t *testing.T) *assessment.Spec {
	t.Helper()
	// Resolve repo root by walking up to the scenario maturity.json.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// .../scenarios/unit-health/api/internal/validation -> .../scenarios/unit-health
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatalf("read maturity.json: %v", err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	return spec
}

func newService(disc discovery.Discoverer, spec *assessment.Spec) *Service {
	fixedNow := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	return &Service{Discoverer: disc, Spec: spec, Now: func() time.Time { return fixedNow }}
}

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
	if ws.Status != "ready" || ws.TestCommand != "go test ./..." || ws.CoverageCommand == "" {
		t.Errorf("go workspace not ready: %+v", ws)
	}
	if len(resp.Plan.Commands) != 1 {
		t.Errorf("expected 1 planned command, got %+v", resp.Plan.Commands)
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
