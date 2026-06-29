package validation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	repoRoot := t.TempDir()
	root := filepath.Join(repoRoot, "scenarios", "demo")
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"demo","displayName":"Demo","description":"Demo scenario."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head><title>Demo</title></head><body><main>Demo</main></body></html>`)
	writeMaturitySpec(t, repoRoot)

	handler := &Handler{repoRoot: repoRoot}
	resp, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario returned error: %v", err)
	}
	if resp.Msg.GetMetrics() == nil {
		t.Fatal("ValidateScenario must attach execution metrics")
	}
	if resp.Msg.GetMetrics().GetWallClockMs() < 0 {
		t.Fatalf("wall_clock_ms = %d, want non-negative", resp.Msg.GetMetrics().GetWallClockMs())
	}
}

func TestFixRPCPreviewAndApply(t *testing.T) {
	repoRoot := t.TempDir()
	root := filepath.Join(repoRoot, "scenarios", "demo")
	writeFile(t, root, ".vrooli/service.json", `{"service":{"name":"demo","displayName":"Demo","description":"Demo scenario."}}`)
	writeFile(t, root, "ui/index.html", `<!doctype html><html><head><title>Demo</title></head><body></body></html>`)

	handler := &Handler{repoRoot: repoRoot}
	req := connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: "demo",
		RuleIds:  []string{"has-color-system"},
	})

	preview, err := handler.PreviewFix(context.Background(), req)
	if err != nil {
		t.Fatalf("PreviewFix returned error: %v", err)
	}
	if got := len(preview.Msg.GetCandidates()); got != 1 {
		t.Fatalf("PreviewFix candidates = %d, want 1", got)
	}
	if preview.Msg.GetCandidates()[0].GetApplied() {
		t.Fatal("PreviewFix candidate must not be marked applied")
	}
	if _, err := os.Stat(filepath.Join(root, "ui", "src", "design-tokens.css")); !os.IsNotExist(err) {
		t.Fatalf("PreviewFix wrote design tokens or unexpected stat error: %v", err)
	}

	applied, err := handler.ApplyFix(context.Background(), req)
	if err != nil {
		t.Fatalf("ApplyFix returned error: %v", err)
	}
	if got := len(applied.Msg.GetCandidates()); got != 1 {
		t.Fatalf("ApplyFix candidates = %d, want 1", got)
	}
	if !applied.Msg.GetCandidates()[0].GetApplied() {
		t.Fatal("ApplyFix candidate must be marked applied")
	}
	if _, err := os.Stat(filepath.Join(root, "ui", "src", "design-tokens.css")); err != nil {
		t.Fatalf("ApplyFix did not write design tokens: %v", err)
	}
}

func writeMaturitySpec(t *testing.T, repoRoot string) {
	t.Helper()
	spec := assessment.Spec{
		Provider: "brand-manager",
		Phase:    "branding",
		Version:  "test",
		Levels: []assessment.Level{
			{ID: "L0", Name: "base"},
			{ID: "L1", Name: "ready"},
		},
		Findings: map[string]assessment.FindingMapping{},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L1",
			GlobalImpact:     assessment.ImpactHardeningGap,
			Dimension:        "branding",
			SeverityDefault:  "SEVERITY_WARNING",
		},
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal maturity spec: %v", err)
	}
	writeFile(t, filepath.Join(repoRoot, "scenarios", "brand-manager"), ".vrooli/maturity.json", string(raw))
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}
