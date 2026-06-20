package reconcile

import (
	"testing"

	"structure-health/internal/intent"
	"structure-health/internal/profile"
)

func find(m Model, surface string) (SurfaceState, bool) {
	for _, s := range m.Surfaces {
		if s.Surface == surface {
			return s, true
		}
	}
	return SurfaceState{}, false
}

func TestBuildReconcile(t *testing.T) {
	in := intent.Intent{
		Name:       "demo",
		CLIEnabled: true,
		Ports: map[string]intent.Port{
			"api": {EnvVar: "API_PORT"},
			"ui":  {EnvVar: "UI_PORT"},
		},
	}
	p := profile.Profile{
		Surfaces: []profile.Surface{
			{ID: "api", Kind: "api", Language: "go"},
			// ui declared but NOT detected (missing implementation)
			// worker detected but NOT declared
			{ID: "worker", Kind: "worker", Language: "go"},
		},
	}
	m := Build("demo", "/tmp/demo", in, p)

	api, ok := find(m, "api")
	if !ok || !api.Declared || !api.Actual {
		t.Fatalf("api should be declared+actual: %+v", api)
	}
	ui, ok := find(m, "ui")
	if !ok || !ui.Declared || ui.Actual {
		t.Fatalf("ui should be declared but not actual: %+v", ui)
	}
	cli, ok := find(m, "cli")
	if !ok || !cli.Declared {
		t.Fatalf("cli should be declared via cli.enabled: %+v", cli)
	}
	worker, ok := find(m, "worker")
	if !ok || worker.Declared || !worker.Actual {
		t.Fatalf("worker should be actual but not declared: %+v", worker)
	}
}
