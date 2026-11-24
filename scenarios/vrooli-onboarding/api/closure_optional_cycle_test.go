package main

import (
	"path/filepath"
	"testing"
)

func TestClosureRetainsOptionalTryStartCycleAsUnion(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha"},"dependencies":{"scenarios":{"beta":{"required":false,"startup_policy":"try_start"}}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "beta", ".vrooli", "service.json"), `{"service":{"name":"beta"},"dependencies":{"scenarios":{"alpha":{"required":false,"startup_policy":"try_start"}}}}`)

	closure, err := resolveClosureForState(root, []ScenarioReadModel{{Name: "alpha", Enabled: true}}, OperatorState{})
	if err != nil {
		t.Fatalf("optional try_start cycle should remain a closure union: %v", err)
	}
	if len(closure.Scenarios) != 2 || closure.Scenarios[0].Name != "alpha" || closure.Scenarios[1].Name != "beta" {
		t.Fatalf("closure scenarios = %#v, want alpha and beta", closure.Scenarios)
	}
}
