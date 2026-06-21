package rules

import (
	"os"
	"path/filepath"
	"testing"

	"structure-health/internal/intent"
	"structure-health/internal/profile"
	"structure-health/internal/reconcile"
)

// conformantRoot writes a minimal react-vite/Go scenario skeleton that passes
// the universal structure rules. The root is named "demo" so the directory base
// (which evalAt adopts as the scenario name) stays consistent with the fixture's
// demo-api binary references and demo cli command.
func conformantRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "Makefile"), "start:\n\tvrooli scenario start\n")
	mustWrite(t, filepath.Join(root, "README.md"), "# demo\n")
	for _, dir := range []string{"api", "ui", "cli"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(root, ".vrooli", "service.json"), "{}")
	return root
}

func conformantIntent() intent.Intent {
	return intent.Intent{
		Name:       filepath.Base("demo"),
		CLIEnabled: true,
		Ports: map[string]intent.Port{
			"api": {EnvVar: "API_PORT", Range: "15000-19999"},
			"ui":  {EnvVar: "UI_PORT", Range: "20000-24999"},
		},
		Lifecycle: intent.Lifecycle{
			Health: intent.Health{Checks: []intent.HealthCheck{{Type: "http", Critical: true}}},
			Setup: intent.Phase{
				Steps: []intent.Step{{Name: "build-api", Run: "cd api && go build ."}, {Name: "build-ui", Run: "cd ui && pnpm build"}},
				Condition: &intent.Condition{Checks: []intent.FreshCheck{
					{Type: "binaries", Targets: []string{"api/demo-api"}},
					{Type: "ui-bundle", BundlePath: "ui/dist/index.html", SourceDir: "ui/src"},
					{Type: "cli", Command: "demo"},
				}},
			},
			Develop: intent.Phase{Steps: []intent.Step{
				{Name: "start-api", Run: "cd api && ./demo-api", Background: true},
				{Name: "start-ui", Run: "cd ui && node server.js", Background: true},
			}},
		},
	}
}

func fullProfile() profile.Profile {
	return profile.Profile{
		ID:         profile.DefaultProfileID,
		Recognized: true,
		Surfaces: []profile.Surface{
			{ID: "api", Kind: "api", Language: "go"},
			{ID: "ui", Kind: "ui", Framework: "react-vite"},
			{ID: "cli", Kind: "cli", Language: "go"},
		},
	}
}

func evalAt(root string, in intent.Intent, p profile.Profile) []Finding {
	// service.json name defaults to the directory base so SERVICE_NAME_MISMATCH
	// does not fire in the conformant case.
	in.Name = filepath.Base(root)
	model := reconcile.Build(in.Name, root, in, p)
	return Evaluate(Input{Model: model, ServiceJSONReadable: true})
}

func codes(findings []Finding) map[string]int {
	out := map[string]int{}
	for _, f := range findings {
		out[f.Code]++
	}
	return out
}

// [REQ:SH-RULE-001] [REQ:SH-PROF-001]
func TestConformantScenarioHasNoFindings(t *testing.T) {
	root := conformantRoot(t)
	got := evalAt(root, conformantIntent(), fullProfile())
	if len(got) != 0 {
		t.Fatalf("conformant scenario should have no findings, got: %+v", got)
	}
}

// [REQ:SH-RULE-001]
func TestServiceJSONMissing(t *testing.T) {
	root := t.TempDir()
	model := reconcile.Build("demo", root, intent.Intent{}, profile.Profile{})
	got := Evaluate(Input{Model: model, ServiceJSONReadable: false})
	if codes(got)["SERVICE_JSON_MISSING"] == 0 {
		t.Fatalf("expected SERVICE_JSON_MISSING, got %+v", got)
	}
}

// [REQ:SH-RULE-001]
func TestServiceNameMismatch(t *testing.T) {
	root := conformantRoot(t)
	in := conformantIntent()
	in.Name = "not-the-dir"
	model := reconcile.Build("not-the-dir", root, in, fullProfile())
	got := Evaluate(Input{Model: model, ServiceJSONReadable: true})
	if codes(got)["SERVICE_NAME_MISMATCH"] == 0 {
		t.Fatalf("expected SERVICE_NAME_MISMATCH, got %+v", got)
	}
}

// [REQ:SH-RULE-002]
func TestMissingFreshnessCheck(t *testing.T) {
	root := conformantRoot(t)
	in := conformantIntent()
	in.Lifecycle.Setup.Condition = nil // drop all freshness checks
	got := evalAt(root, in, fullProfile())
	if codes(got)["FRESHNESS_CHECK_MISSING"] == 0 {
		t.Fatalf("expected FRESHNESS_CHECK_MISSING, got %+v", got)
	}
}

// [REQ:SH-RULE-003]
func TestMissingHealthCheck(t *testing.T) {
	root := conformantRoot(t)
	in := conformantIntent()
	in.Lifecycle.Health.Checks = nil
	got := evalAt(root, in, fullProfile())
	if codes(got)["HEALTH_CHECK_MISSING"] == 0 {
		t.Fatalf("expected HEALTH_CHECK_MISSING, got %+v", got)
	}
}

// [REQ:SH-RULE-004]
func TestProductionServeNonconformant(t *testing.T) {
	root := conformantRoot(t)
	in := conformantIntent()
	in.Lifecycle.Develop.Steps = []intent.Step{
		{Name: "start-api", Run: "cd api && ./demo-api", Background: true},
		{Name: "start-ui", Run: "cd ui && pnpm dev", Background: true},
	}
	got := evalAt(root, in, fullProfile())
	if codes(got)["PRODUCTION_SERVE_NONCONFORMANT"] == 0 {
		t.Fatalf("expected PRODUCTION_SERVE_NONCONFORMANT, got %+v", got)
	}
}

// [REQ:SH-RULE-004]
func TestPortBandNonconformant(t *testing.T) {
	root := conformantRoot(t)
	in := conformantIntent()
	// api declared in a non-canonical band (and a wrong env var) → fixable.
	in.Ports["api"] = intent.Port{EnvVar: "PORT", Range: "10000-12000"}
	got := evalAt(root, in, fullProfile())
	if codes(got)["PORT_BAND_NONCONFORMANT"] == 0 {
		t.Fatalf("expected PORT_BAND_NONCONFORMANT, got %+v", got)
	}
}

// [REQ:SH-RULE-004]
func TestAPIBinaryNameNonconformant(t *testing.T) {
	root := conformantRoot(t)
	in := conformantIntent()
	in.Lifecycle.Develop.Steps = []intent.Step{
		{Name: "start-api", Run: "cd api && ./wrong-binary", Background: true},
		{Name: "start-ui", Run: "cd ui && node server.js", Background: true},
	}
	got := evalAt(root, in, fullProfile())
	if codes(got)["API_BINARY_NAME_NONCONFORMANT"] == 0 {
		t.Fatalf("expected API_BINARY_NAME_NONCONFORMANT, got %+v", got)
	}
}

// [REQ:SH-RULE-005]
func TestReconcileMismatchDeclaredNotActual(t *testing.T) {
	root := conformantRoot(t)
	in := conformantIntent()
	p := fullProfile()
	// Drop the UI surface from actual → declared-but-not-detected.
	p.Surfaces = p.Surfaces[:1] // api only
	p.Surfaces = append(p.Surfaces, profile.Surface{ID: "cli", Kind: "cli", Language: "go"})
	got := evalAt(root, in, p)
	if codes(got)["SURFACE_RECONCILE_MISMATCH"] == 0 {
		t.Fatalf("expected SURFACE_RECONCILE_MISMATCH, got %+v", got)
	}
}

// [REQ:SH-RULE-006]
func TestInvalidDependencyPolicy(t *testing.T) {
	root := conformantRoot(t)
	in := conformantIntent()
	in.Deps.Resources = map[string]intent.Dependency{
		"postgres": {StartupPolicy: "bogus"},
	}
	got := evalAt(root, in, fullProfile())
	if codes(got)["DEPENDENCY_DECLARATION_INVALID"] == 0 {
		t.Fatalf("expected DEPENDENCY_DECLARATION_INVALID, got %+v", got)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
