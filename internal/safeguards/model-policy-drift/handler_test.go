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
	for _, runner := range runners {
		path := filepath.Join(root, "resources", runner, "model-policy.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		data := `{"roles":{"code.default":{"model":"missing","fallbacks":[],"description":"x","capabilities":["code"]},"code.fast":{"model":"missing","description":"x","capabilities":["code"]},"code.smart":{"model":"missing","description":"x","capabilities":["code"]},"code.cheap":{"model":"missing","description":"x","capabilities":["code"]}},"provenance":{"source":"fixture","observed_at":"2026-08-04"}}`
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("VROOLI_"+strings.ToUpper(strings.ReplaceAll(runner, "-", "_"))+"_MODELS", "present")
	}
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "model_policy_drift"}).Inspect(hostreqkit.Host{}, hostreqspec.ResolvedRequirement{Name: "model_policy_drift"})
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
	for _, runner := range runners {
		t.Setenv("VROOLI_"+strings.ToUpper(strings.ReplaceAll(runner, "-", "_"))+"_MODELS", "")
	}
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "model_policy_drift"}).Inspect(hostreqkit.Host{}, hostreqspec.ResolvedRequirement{Name: "model_policy_drift"})
	if status.ExecutionState != hostreqkit.ExecutionPending || !strings.Contains(strings.Join(status.Notes, "\n"), "not_measured") {
		t.Fatalf("not measured status = %+v", status)
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
	t.Setenv("VROOLI_CODEX_MODELS", "present,new-model")
	findings, err := validateAgainstLive(context.Background(), "codex", path)
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
