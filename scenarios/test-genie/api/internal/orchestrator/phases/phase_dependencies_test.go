package phases

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/workspace"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func swapDependencyDriftSeam(t *testing.T, fn func(ctx context.Context, scenario string) ([]byte, int, error)) {
	t.Helper()
	prev := runDependencyDrift
	runDependencyDrift = fn
	t.Cleanup(func() { runDependencyDrift = prev })
}

func dependencyEnv(t *testing.T) workspace.Environment {
	t.Helper()
	dir := t.TempDir()
	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  dir,
		TestDir:      filepath.Join(dir, "test"),
		AppRoot:      filepath.Dir(dir),
	}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "testing.json"), []byte(`{
		"dependencies": {
			"runtime_versions": {},
			"go_modules": {"enabled": false},
			"node_packages": {"enabled": false},
			"scenarios": {"enabled": false},
			"resources": {"health_policy": "skip"}
		}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "service.json"), []byte(`{
		"name": "demo",
		"dependencies": {
			"resources": {},
			"scenarios": {}
		}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return env
}

func TestDependencyDriftArchFindings_MapsSourceAndStableID(t *testing.T) {
	report, err := parseDependencyDriftOutput([]byte(`{
		"findings": [
			{
				"scenario": "demo",
				"dependency": "proto-health",
				"kind": "undeclared-but-used",
				"severity": "WARNING",
				"message": "demo imports proto-health but does not declare it",
				"evidence": [
					{"source": "proto_import", "to_file": "proto-health/v1/shared/surface.proto"}
				]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("parseDependencyDriftOutput: %v", err)
	}
	got := dependencyDriftArchFindings("demo", report)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got))
	}
	if got[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY {
		t.Errorf("source = %v, want FINDING_SOURCE_DEPENDENCY", got[0].GetSource())
	}
	if got[0].GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
		t.Errorf("severity = %v, want WARNING", got[0].GetSeverity())
	}
	if got[0].GetStableId() == "" {
		t.Error("stable id must be stamped")
	}
	if len(got[0].GetEvidence()) != 1 || got[0].GetEvidence()[0].GetKind() != "proto_import" {
		t.Fatalf("evidence not translated: %+v", got[0].GetEvidence())
	}
}

func TestRunDependenciesPhase_DriftUnavailableSkipsGracefully(t *testing.T) {
	swapDependencyDriftSeam(t, func(_ context.Context, _ string) ([]byte, int, error) {
		return nil, 0, errors.New("not installed")
	})

	var buf bytes.Buffer
	report := runDependenciesPhase(context.Background(), dependencyEnv(t), io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("unavailable drift producer must not fail dependency phase: %v", report.Err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("skipped drift producer should emit no findings, got %d", len(report.Findings))
	}
}

func TestRunDependenciesPhase_DriftFindingIsAttached(t *testing.T) {
	swapDependencyDriftSeam(t, func(_ context.Context, scenario string) ([]byte, int, error) {
		return []byte(`{
			"findings": [
				{"scenario":"` + scenario + `","dependency":"code-facts","kind":"undeclared-but-used","severity":"WARNING","message":"missing declaration"}
			]
		}`), 0, nil
	})

	var buf bytes.Buffer
	report := runDependenciesPhase(context.Background(), dependencyEnv(t), io.MultiWriter(&buf, io.Discard))
	if report.Err != nil {
		t.Fatalf("dependency phase should still pass: %v", report.Err)
	}
	if len(report.Findings) != 1 || report.Findings[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY {
		t.Fatalf("expected 1 dependency finding, got %+v", report.Findings)
	}
}
