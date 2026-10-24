package main

import (
	"testing"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/vrooli/resources/testkit"
)

func TestDeliveryModelPolicyHasOnlyOperatorAuthorizedTargets(t *testing.T) {
	catalog, err := agentcatalog.ReadCodingRoleCatalog("codex", "../model-policy.json")
	if err != nil {
		t.Fatal(err)
	}
	for roleName, models := range map[string][]string{"code.delivery": {"gpt-6-luna"}, "judgment.supervision": {"gpt-6-sol"}} {
		role, ok := catalog.Roles[roleName]
		if !ok || role.Model != models[0] || role.CanonicalModel != "codex/"+models[0] || len(role.Fallbacks) != len(models)-1 || len(role.Models) != len(models) || role.Challenger != nil {
			t.Fatalf("%s must have only the authorized ordered models: %+v", roleName, role)
		}
		for index, model := range models {
			selection := role.Models[index]
			if selection.Model != model || selection.Source != "codex-subscription" || selection.Effort == nil || *selection.Effort != "medium" {
				t.Fatalf("%s candidate %d must use %s subscription/medium: %+v", roleName, index, model, selection)
			}
			if index > 0 && role.Fallbacks[index-1] != model {
				t.Fatalf("%s fallback order differs: %v", roleName, role.Fallbacks)
			}
		}
	}
	for _, model := range []string{"gpt-6-sol"} {
		roles := catalog.RestrictedModels[model]
		if len(roles) != 1 || roles[0] != "judgment.supervision" {
			t.Fatalf("%s must remain reserved for supervision: %v", model, roles)
		}
	}
}

func TestNewAppConfiguresResourceApp(t *testing.T) {
	h := testkit.Handlers(t)
	if h.Stdout == nil {
		t.Fatal("test harness must provide stdout")
	}
	app, err := newApp()
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	if app == nil {
		t.Fatal("newApp() returned nil app")
	}
	if app.CLI == nil {
		t.Fatal("newApp() returned nil CLI")
	}
	if app.StaleChecker == nil {
		t.Fatal("newApp() returned nil stale checker")
	}
	if app.StaleChecker.SourceContextPath != ".." {
		t.Fatalf("SourceContextPath = %q, want %q", app.StaleChecker.SourceContextPath, "..")
	}
	if app.StaleChecker.ManifestSourcePath != "resource.json" {
		t.Fatalf("ManifestSourcePath = %q, want %q", app.StaleChecker.ManifestSourcePath, "resource.json")
	}
	if len(app.StaleChecker.FreshnessInputs) != 4 {
		t.Fatalf("FreshnessInputs len = %d, want 4", len(app.StaleChecker.FreshnessInputs))
	}
	if got, want := app.StaleChecker.FreshnessInputs[0], "cli/**"; got != want {
		t.Fatalf("FreshnessInputs[0] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[1], "resource.json"; got != want {
		t.Fatalf("FreshnessInputs[1] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[2], "../../packages/cli-core"; got != want {
		t.Fatalf("FreshnessInputs[2] = %q, want %q", got, want)
	}
	if got, want := app.StaleChecker.FreshnessInputs[3], "../../packages/proto/gen/.vrooli-proto-artifact.json"; got != want {
		t.Fatalf("FreshnessInputs[3] = %q, want %q", got, want)
	}
}
