package modelpolicydrift

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestInspectReportsMeasuredDriftAndNeverAppliesIt(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PROJECT_ROOT", root)
	models := map[string]any{}
	for _, runner := range runners {
		path := filepath.Join(root, "resources", runner, "model-policy.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		data := `{"roles":{"code.default":{"model":"missing","fallbacks":[],"description":"x","capabilities":["code"]},"code.fast":{"model":"missing","description":"x","capabilities":["code"]},"code.smart":{"model":"missing","description":"x","capabilities":["code"]},"code.cheap":{"model":"missing","description":"x","capabilities":["code"]}},"provenance":{"source":"fixture","observed_at":"2026-08-04"}}`
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		models[runner] = []any{"present"}
	}
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "model_policy_drift"}).Inspect(hostreqkit.Host{}, hostreqspec.ResolvedRequirement{Name: "model_policy_drift", Config: map[string]any{"models": models}})
	if status.ExecutionState != hostreqkit.ExecutionPending || status.Applied {
		t.Fatalf("status = %+v", status)
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), "missing_primary_model") {
		t.Fatalf("notes do not identify measured drift: %v", status.Notes)
	}
	status, err := NewHandler(hostreqkit.SafeguardManifest{Name: "model_policy_drift"}).Apply(hostreqkit.Host{}, status, hostreqkit.EnsureOptions{})
	if err != nil || status.Applied {
		t.Fatalf("apply mutated drift safeguard: status=%+v err=%v", status, err)
	}
}

func TestInspectDistinguishesNotMeasured(t *testing.T) {
	t.Setenv("PROJECT_ROOT", t.TempDir())
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "model_policy_drift"}).Inspect(hostreqkit.Host{}, hostreqspec.ResolvedRequirement{Name: "model_policy_drift"})
	if status.ExecutionState != hostreqkit.ExecutionPending || !strings.Contains(strings.Join(status.Notes, "\n"), "not_measured") {
		t.Fatalf("not measured status = %+v", status)
	}
}

func TestInspectDoesNotEscalatePartialCatalogWarnings(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PROJECT_ROOT", root)
	models := map[string]any{}
	for _, runner := range runners {
		path := filepath.Join(root, "resources", runner, "model-policy.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		data := `{"provenance":{"source":"fixture","observed_at":"2026-09-28"},"roles":{"code.default":{"model":"gpt-6-luna"}}}`
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		models[runner] = []any{"present"}
	}
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "model_policy_drift"}).Inspect(hostreqkit.Host{}, hostreqspec.ResolvedRequirement{Name: "model_policy_drift", Config: map[string]any{"models": models, "models_exhaustive": false}})
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent || !status.Applied {
		t.Fatalf("partial catalog warnings escalated: %+v", status)
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), "unconfirmed_primary_model") {
		t.Fatalf("status did not preserve warning evidence: %v", status.Notes)
	}
}

func TestValidateAgainstLiveReportsStaleAndUnadoptedModels(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "resources", "codex", "model-policy.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"staleness_budget_days":14,"provenance":{"observed_at":"2020-01-01"},"roles":{"code.default":{"model":"present"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := validateAgainstLive(context.Background(), "codex", path, map[string]any{"models": map[string]any{"codex": []any{"present", "new-model"}}})
	if err != nil {
		t.Fatal(err)
	}
	text := ""
	for _, finding := range findings {
		text += finding.Type + " "
	}
	if !strings.Contains(text, "catalog_stale") || !strings.Contains(text, "unnamed_live_model") {
		t.Fatalf("findings=%+v", findings)
	}
}

func TestDiscoverUsesInstalledCodexBeforeCompatibilityCache(t *testing.T) {
	runnerDir := t.TempDir()
	codex := filepath.Join(runnerDir, "codex")
	if err := os.WriteFile(codex, []byte("#!/bin/sh\nprintf '%s' '{\"models\":[{\"slug\":\"gpt-6-luna\"},{\"slug\":\"gpt-6-sol\"}]}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), []byte(`{"models":["stale-model"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", runnerDir)
	t.Setenv("HOME", home)

	models, err := discover(context.Background(), "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !models["gpt-6-luna"] || !models["gpt-6-sol"] || models["stale-model"] {
		t.Fatalf("models=%v; expected installed runner catalog to win over cache", models)
	}
}

func TestValidateAgainstLiveRejectsCompatibilityCacheAsAvailabilityProof(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "resources", "codex", "model-policy.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"roles":{"code.delivery":{"model":"gpt-6-luna"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), []byte(`{"models":["gpt-5.6-luna"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runnerDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(runnerDir, "codex"), []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", runnerDir)

	_, err := validateAgainstLive(context.Background(), "codex", path)
	if err == nil || !strings.Contains(err.Error(), "non-authoritative") {
		t.Fatalf("err=%v; stale compatibility cache must not produce a model-absence finding", err)
	}
}
