package providercontract

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func writeSpec(t *testing.T, repoRoot, provider, phase string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "scenarios", provider, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	spec := assessment.Spec{
		Provider: provider,
		Phase:    phase,
		Version:  "1",
		Levels:   []assessment.Level{{ID: "L0", Name: "base"}, {ID: "L1", Name: "next"}},
		Findings: map[string]assessment.FindingMapping{},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L0",
			GlobalImpact:     assessment.ImpactUnknown,
			Dimension:        "measures",
			SeverityDefault:  "WARNING",
		},
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "maturity.json"), raw, 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
}

func validProbeResponse(provider, phase string, withMetrics bool) *scenariovalidationv1.ValidateScenarioResponse {
	resp := &scenariovalidationv1.ValidateScenarioResponse{
		Scenario: "test-genie",
		Status:   scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: &commonv1.MaturityAssessment{
			Scenario: "test-genie",
			Provider: provider,
			Phase:    phase,
			Version:  "1",
			Local:    &commonv1.LocalMaturityAssessment{CurrentLevel: "L1"},
		},
	}
	if withMetrics {
		resp.Metrics = &commonv1.ExecutionMetrics{WallClockMs: 1840}
	}
	return resp
}

func TestScanProviderFullAdoption(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	probe := func(context.Context, string, string, time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return validProbeResponse("proto-health", "proto", true), nil
	}
	restore := swapProbe(probe)
	defer restore()

	pr := scanProvider(context.Background(), repo, "test-genie", "proto", "proto-health", time.Second)
	if !pr.Reachable || !pr.ContractValid || !pr.IdentityOK || !pr.SpecValid || !pr.MetricsAdopted {
		t.Fatalf("expected full adoption, got %+v", pr)
	}
	if pr.AdoptionScore != 1.0 {
		t.Fatalf("adoption_score = %v, want 1.0", pr.AdoptionScore)
	}
	if len(pr.Violations) != 0 {
		t.Fatalf("unexpected violations: %v", pr.Violations)
	}
	if pr.hasHardViolation() {
		t.Fatal("full adoption should have no hard violation")
	}
}

func TestScanProviderMetricsAdvisory(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "cli-health", "contracts")
	probe := func(context.Context, string, string, time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return validProbeResponse("cli-health", "contracts", false), nil
	}
	restore := swapProbe(probe)
	defer restore()

	pr := scanProvider(context.Background(), repo, "test-genie", "contracts", "cli-health", time.Second)
	if pr.MetricsAdopted {
		t.Fatal("expected metrics_adopted=false for un-migrated provider")
	}
	if pr.hasHardViolation() {
		t.Fatalf("missing metrics must not be a hard violation: %+v", pr)
	}
	if pr.AdoptionScore != 0.8 {
		t.Fatalf("adoption_score = %v, want 0.8", pr.AdoptionScore)
	}
}

func TestScanProviderBrokenSpec(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, "scenarios", "proto-health", ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "maturity.json"), []byte(`{"provider":""}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	probe := func(context.Context, string, string, time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return validProbeResponse("proto-health", "proto", true), nil
	}
	restore := swapProbe(probe)
	defer restore()

	pr := scanProvider(context.Background(), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.SpecValid {
		t.Fatal("expected spec_valid=false for broken maturity.json")
	}
	if !pr.hasHardViolation() {
		t.Fatal("broken spec must be a hard violation")
	}
}

func TestScanProviderUnreachableNotHard(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	probe := func(context.Context, string, string, time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return nil, errors.New("connection refused")
	}
	restore := swapProbe(probe)
	defer restore()

	pr := scanProvider(context.Background(), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.Reachable {
		t.Fatal("expected unreachable")
	}
	if pr.hasHardViolation() {
		t.Fatal("unreachable (env) should not be a hard violation when the spec is valid")
	}
	if len(pr.Violations) == 0 {
		t.Fatal("unreachable should still be reported as a violation")
	}
}

func TestScanProviderIdentityMismatch(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	probe := func(context.Context, string, string, time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return validProbeResponse("imposter", "proto", true), nil
	}
	restore := swapProbe(probe)
	defer restore()

	pr := scanProvider(context.Background(), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.IdentityOK {
		t.Fatal("expected identity mismatch")
	}
	if !pr.hasHardViolation() {
		t.Fatal("identity mismatch among reachable providers is a hard violation")
	}
}

func swapProbe(fn func(context.Context, string, string, time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error)) func() {
	prev := scanProbe
	scanProbe = fn
	return func() { scanProbe = prev }
}
