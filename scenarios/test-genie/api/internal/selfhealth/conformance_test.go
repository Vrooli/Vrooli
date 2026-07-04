package selfhealth

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
	raw, err := json.Marshal(testGenieDescriptor(provider, phase, spec))
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test-genie.json"), raw, 0o644); err != nil {
		t.Fatalf("write descriptor: %v", err)
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

func probeFn(resp *scenariovalidationv1.ValidateScenarioResponse, err error) ConformanceProbe {
	return func(context.Context, string, string, time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return resp, err
	}
}

// writeSpecWithFindings writes a descriptor-embedded maturity spec carrying
// explicit fixability declarations so the autofix-coverage lens has real data
// to roll up.
func writeSpecWithFindings(t *testing.T, repoRoot, provider, phase string, findings map[string]assessment.FindingMapping) {
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
		Findings: findings,
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L0",
			GlobalImpact:     assessment.ImpactUnknown,
			Dimension:        "measures",
			SeverityDefault:  "WARNING",
		},
	}
	raw, err := json.Marshal(testGenieDescriptor(provider, phase, spec))
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test-genie.json"), raw, 0o644); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}
}

func testGenieDescriptor(provider, phase string, spec assessment.Spec) map[string]any {
	return map[string]any{
		"scenario": provider,
		"phase":    phase,
		"maturity": spec,
	}
}

func TestScanProviderAutofixCoverage(t *testing.T) {
	repo := t.TempDir()
	writeSpecWithFindings(t, repo, "proto-health", "proto", map[string]assessment.FindingMapping{
		"DONE":    {GlobalImpact: assessment.ImpactAdvisory, FixClass: assessment.FixClassAuto, FixerStatus: assessment.FixerStatusImplemented},
		"GAP":     {GlobalImpact: assessment.ImpactAdvisory, FixClass: assessment.FixClassAuto, FixerStatus: assessment.FixerStatusPending},
		"BY_HAND": {GlobalImpact: assessment.ImpactAdvisory, FixClass: assessment.FixClassManual, FixReason: "needs judgment"},
	})
	pr := scanProvider(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.Autofix.Implemented != 1 || pr.Autofix.Pending != 1 || pr.Autofix.Manual != 1 {
		t.Fatalf("coverage = %+v, want implemented=1 pending=1 manual=1", pr.Autofix)
	}
	if !pr.Autofix.DeclarationComplete {
		t.Fatalf("DeclarationComplete=false, want true (all findings declared)")
	}
	// The autofix lens is advisory: it must not change the hard-dim adoption score.
	if pr.AdoptionScore != 1.0 {
		t.Fatalf("adoption_score = %v, want 1.0 (autofix is advisory)", pr.AdoptionScore)
	}
}

func TestScanProviderAutofixCoverageWhenUnreachable(t *testing.T) {
	// Coverage is a declaration property — available even when the provider is
	// not running, so the lens never goes dark on an environmental signal.
	repo := t.TempDir()
	writeSpecWithFindings(t, repo, "proto-health", "proto", map[string]assessment.FindingMapping{
		"GAP": {GlobalImpact: assessment.ImpactAdvisory, FixClass: assessment.FixClassAuto, FixerStatus: assessment.FixerStatusPending},
	})
	pr := scanProvider(context.Background(), probeFn(nil, errors.New("connection refused")), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.Reachable {
		t.Fatal("expected unreachable")
	}
	if pr.Autofix.Pending != 1 || pr.Autofix.Total != 1 {
		t.Fatalf("coverage = %+v, want pending=1 total=1 even when unreachable", pr.Autofix)
	}
}

func TestScanProviderFullAdoption(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	pr := scanProvider(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
	if !pr.Reachable || !pr.ContractValid || !pr.IdentityOK || !pr.SpecValid || !pr.MetricsAdopted {
		t.Fatalf("expected full adoption, got %+v", pr)
	}
	if pr.AdoptionScore != 1.0 {
		t.Fatalf("adoption_score = %v, want 1.0", pr.AdoptionScore)
	}
	if len(pr.Violations) != 0 {
		t.Fatalf("unexpected violations: %v", pr.Violations)
	}
	if pr.HasHardViolation() {
		t.Fatal("full adoption should have no hard violation")
	}
}

func TestScanProviderMetricsRequired(t *testing.T) {
	// Plan 3 Part B: metrics adoption is no longer advisory. A reachable
	// provider whose response carries no ExecutionMetrics is a hard violation.
	repo := t.TempDir()
	writeSpec(t, repo, "cli-health", "contracts")
	pr := scanProvider(context.Background(), probeFn(validProbeResponse("cli-health", "contracts", false), nil), repo, "test-genie", "contracts", "cli-health", time.Second)
	if pr.MetricsAdopted {
		t.Fatal("expected metrics_adopted=false when the response carries no metrics")
	}
	if !pr.HasHardViolation() {
		t.Fatalf("a reachable provider that dropped metrics must be a hard violation: %+v", pr)
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
	if err := os.WriteFile(filepath.Join(dir, "test-genie.json"), []byte(`{"scenario":"proto-health","phase":"proto","maturity":{"provider":""}}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	pr := scanProvider(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.SpecValid {
		t.Fatal("expected spec_valid=false for broken descriptor maturity")
	}
	if !pr.HasHardViolation() {
		t.Fatal("broken spec must be a hard violation")
	}
}

func TestScanProviderUnreachableNotHard(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	pr := scanProvider(context.Background(), probeFn(nil, errors.New("connection refused")), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.Reachable {
		t.Fatal("expected unreachable")
	}
	if pr.HasHardViolation() {
		t.Fatal("unreachable (env) should not be a hard violation when the spec is valid")
	}
	if len(pr.Violations) == 0 {
		t.Fatal("unreachable should still be reported as a violation")
	}
}

func TestScanProviderIdentityMismatch(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	pr := scanProvider(context.Background(), probeFn(validProbeResponse("imposter", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.IdentityOK {
		t.Fatal("expected identity mismatch")
	}
	if !pr.HasHardViolation() {
		t.Fatal("identity mismatch among reachable providers is a hard violation")
	}
}

func TestConformanceScannerScansCatalog(t *testing.T) {
	repo := t.TempDir()
	// Seed a couple of provider specs; unseeded providers degrade to spec-invalid
	// (reported), but the scan must enumerate the full delegated catalog.
	writeSpec(t, repo, "proto-health", "proto")
	report := ConformanceScanner{
		RepoRoot: repo,
		Probe:    probeFn(validProbeResponse("proto-health", "proto", true), nil),
	}.Scan(context.Background())
	if report.Target != DefaultScanTarget {
		t.Fatalf("target = %q, want %q", report.Target, DefaultScanTarget)
	}
	if len(report.Providers) == 0 {
		t.Fatal("expected the scan to enumerate delegated providers")
	}
	// Results are sorted by phase.
	for i := 1; i < len(report.Providers); i++ {
		if report.Providers[i-1].Phase > report.Providers[i].Phase {
			t.Fatalf("providers not sorted by phase: %+v", report.Providers)
		}
	}
}

func TestConformanceScannerFiltersBySubject(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "brand-manager", "branding")
	report := ConformanceScanner{
		RepoRoot: repo,
		Subject:  "branding",
		Probe:    probeFn(validProbeResponse("brand-manager", "branding", true), nil),
	}.Scan(context.Background())
	if len(report.Providers) != 1 {
		t.Fatalf("providers = %d, want 1: %+v", len(report.Providers), report.Providers)
	}
	if got := report.Providers[0]; got.Phase != "branding" || got.Provider != "brand-manager" {
		t.Fatalf("unexpected provider: %+v", got)
	}
}
