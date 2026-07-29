package selfhealth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
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
	phaseAssessment := &commonv1.MaturityAssessment{
		Scenario: "test-genie",
		Provider: provider,
		Phase:    phase,
		Version:  "1",
		Local:    &commonv1.LocalMaturityAssessment{CurrentLevel: "L1"},
	}
	phaseAssessment.Presentation = assessment.BuildPhasePresentation(phaseAssessment)
	resp := &scenariovalidationv1.ValidateScenarioResponse{
		Scenario:   "test-genie",
		Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: phaseAssessment,
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

func fixProbeFn(preview, applied *scenariovalidationv1.FixResponse, err error) FixConformanceProbe {
	return func(context.Context, string, string, string, []string, time.Duration) (*scenariovalidationv1.FixResponse, *scenariovalidationv1.FixResponse, error) {
		return preview, applied, err
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
		"schemaVersion": "1.0.0",
		"scenario":      provider,
		"phase":         phase,
		"displayName":   phase,
		"description":   "Fixture descriptor for provider conformance tests.",
		"source":        "validation-provider",
		"orderHint":     100,
		"timeout":       "30s",
		"findingSource": phase,
		"validation": map[string]any{
			"contract": "scenario-validation/v1",
		},
		"applicability": map[string]any{
			"default": "applies",
		},
		"policy": map[string]any{
			"selection":         "default_when_applicable",
			"providerReadiness": "required_when_applicable",
			"providerLifecycle": "start_if_needed",
			"freshness":         "require_live_contract",
			"resultGating":      "gating",
			"unavailable":       "fail",
		},
		"runnability": map[string]any{
			"needsUI":           false,
			"needsAPI":          false,
			"requiredResources": []string{},
		},
		"docs": map[string]any{
			"path": "scenarios/test-genie/docs/phases/" + phase + "/README.md",
		},
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
	pr := ScanProvider(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
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
	pr := ScanProvider(context.Background(), probeFn(nil, errors.New("connection refused")), repo, "test-genie", "proto", "proto-health", time.Second)
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
	pr := ScanProvider(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
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
	if pr.Classification != ConformanceCompliant || len(pr.ReasonCodes) != 0 {
		t.Fatalf("classification = %q reasons=%v, want compliant with no reasons", pr.Classification, pr.ReasonCodes)
	}
}

func TestScanProviderValidatesImplementedFixContractInIsolation(t *testing.T) {
	repo := t.TempDir()
	writeSpecWithFindings(t, repo, "proto-health", "proto", map[string]assessment.FindingMapping{
		"FIX": {GlobalImpact: assessment.ImpactAdvisory, FixClass: assessment.FixClassAuto, FixerStatus: assessment.FixerStatusImplemented},
	})
	preview := &scenariovalidationv1.FixResponse{
		Scenario: "test-genie",
		Candidates: []*scenariovalidationv1.FixCandidate{{
			RuleId: "FIX", FilePath: "ui/fixture.txt", Before: "before", After: "after",
		}},
	}
	applied := &scenariovalidationv1.FixResponse{
		Scenario: "test-genie", Applied: true,
		Candidates: []*scenariovalidationv1.FixCandidate{{
			RuleId: "FIX", FilePath: "ui/fixture.txt", Before: "before", After: "after", Applied: true,
		}},
	}
	pr := ScanProviderWithFixProbe(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), fixProbeFn(preview, applied, nil), repo, "test-genie", "proto", "proto-health", time.Second)
	if !pr.FixContractRequired || !pr.FixContractValid || pr.HasHardViolation() {
		t.Fatalf("implemented fixture contract should be valid: %+v", pr)
	}
}

func TestScanProviderRejectsUnsafeOrUntruthfulFixContract(t *testing.T) {
	repo := t.TempDir()
	writeSpecWithFindings(t, repo, "proto-health", "proto", map[string]assessment.FindingMapping{
		"FIX": {GlobalImpact: assessment.ImpactAdvisory, FixClass: assessment.FixClassAuto, FixerStatus: assessment.FixerStatusImplemented},
	})
	preview := &scenariovalidationv1.FixResponse{Scenario: "test-genie", Candidates: []*scenariovalidationv1.FixCandidate{{RuleId: "FIX", FilePath: "../outside", After: "after"}}}
	applied := &scenariovalidationv1.FixResponse{Scenario: "test-genie", Applied: true, Candidates: []*scenariovalidationv1.FixCandidate{{RuleId: "FIX", FilePath: "../outside", After: "after", Applied: true}}}
	pr := ScanProviderWithFixProbe(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), fixProbeFn(preview, applied, nil), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.FixContractValid || !pr.HasHardViolation() || !containsViolation(pr.ReasonCodes, ReasonFixContractInvalid) {
		t.Fatalf("unsafe fix contract must be hard violation: %+v", pr)
	}
}

func TestScanProviderMetricsRequired(t *testing.T) {
	// Plan 3 Part B: metrics adoption is no longer advisory. A reachable
	// provider whose response carries no ExecutionMetrics is a hard violation.
	repo := t.TempDir()
	writeSpec(t, repo, "cli-health", "contracts")
	// Isolate the metrics dimension: this provider does answer DescribeProvider,
	// so only metrics is missing out of the six scored dimensions.
	restore := stubDescribeProbe(nil)
	defer restore()

	pr := ScanProvider(context.Background(), probeFn(validProbeResponse("cli-health", "contracts", false), nil), repo, "test-genie", "contracts", "cli-health", time.Second)
	if pr.MetricsAdopted {
		t.Fatal("expected metrics_adopted=false when the response carries no metrics")
	}
	if !pr.HasHardViolation() {
		t.Fatalf("a reachable provider that dropped metrics must be a hard violation: %+v", pr)
	}
	if want := 5.0 / 6.0; pr.AdoptionScore != want {
		t.Fatalf("adoption_score = %v, want %v", pr.AdoptionScore, want)
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
	pr := ScanProvider(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.SpecValid {
		t.Fatal("expected spec_valid=false for broken descriptor maturity")
	}
	if !pr.HasHardViolation() {
		t.Fatal("broken spec must be a hard violation")
	}
}

func TestScanProviderDescriptorPolicyViolationIsHard(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	path := filepath.Join(repo, "scenarios", "proto-health", ".vrooli", "test-genie.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read descriptor: %v", err)
	}
	raw = []byte(strings.Replace(string(raw), `"providerLifecycle":"start_if_needed"`, `"providerLifecycle":"maybe"`, 1))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}

	pr := ScanProvider(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.SpecValid {
		t.Fatal("expected spec_valid=false for invalid descriptor policy")
	}
	if !pr.HasHardViolation() {
		t.Fatal("descriptor policy violations must be hard violations")
	}
	if !containsViolation(pr.Violations, "code=invalid_provider_lifecycle_policy") {
		t.Fatalf("violations = %v, want invalid_provider_lifecycle_policy", pr.Violations)
	}
}

func TestScanProviderRetiredMaturityFileIsHard(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	if err := os.WriteFile(filepath.Join(repo, "scenarios", "proto-health", ".vrooli", "maturity.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write retired maturity file: %v", err)
	}

	pr := ScanProvider(context.Background(), probeFn(validProbeResponse("proto-health", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.SpecValid {
		t.Fatal("expected spec_valid=false while retired maturity.json remains")
	}
	if !pr.HasHardViolation() {
		t.Fatal("retired maturity.json must be a hard descriptor violation")
	}
	if !containsViolation(pr.Violations, "code=leftover_maturity_json") {
		t.Fatalf("violations = %v, want leftover_maturity_json", pr.Violations)
	}
}

func TestScanProviderUnreachableNotHard(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	pr := ScanProvider(context.Background(), probeFn(nil, errors.New("connection refused")), repo, "test-genie", "proto", "proto-health", time.Second)
	if pr.Reachable {
		t.Fatal("expected unreachable")
	}
	if pr.HasHardViolation() {
		t.Fatal("unreachable (env) should not be a hard violation when the spec is valid")
	}
	if len(pr.Violations) == 0 {
		t.Fatal("unreachable should still be reported as a violation")
	}
	if pr.Classification != ConformanceUnavailable || !containsViolation(pr.ReasonCodes, ReasonProviderUnreachable) {
		t.Fatalf("classification = %q reasons=%v, want unavailable/provider_unreachable", pr.Classification, pr.ReasonCodes)
	}
}

func TestNativePhaseExemptionIsExplicit(t *testing.T) {
	entry := nativePhaseExemption("future-native")
	if entry.Classification != ConformanceExempt || !containsViolation(entry.ReasonCodes, ReasonNativePhase) {
		t.Fatalf("native catalog phase must be explicit exemption: %+v", entry)
	}
}

func TestScanProviderIdentityMismatch(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "proto-health", "proto")
	pr := ScanProvider(context.Background(), probeFn(validProbeResponse("imposter", "proto", true), nil), repo, "test-genie", "proto", "proto-health", time.Second)
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

func TestProviderDefaultTargetUsesApplicabilityValidExperienceFixture(t *testing.T) {
	if got := ProviderDefaultTarget("experience-manager"); got != "experience-manager" {
		t.Fatalf("experience-manager target = %q", got)
	}
	if got := ProviderDefaultTarget("brand-manager"); got != DefaultScanTarget {
		t.Fatalf("brand-manager target = %q, want default %q", got, DefaultScanTarget)
	}
}

func containsViolation(violations []string, needle string) bool {
	for _, violation := range violations {
		if strings.Contains(violation, needle) {
			return true
		}
	}
	return false
}

// stubDescribeProbe replaces the live DescribeProvider probe for one test and
// returns a restore func. Without it, unit tests would attempt real discovery.
func stubDescribeProbe(err error) func() {
	prev := DescribeProbe
	DescribeProbe = func(context.Context, string, time.Duration) error { return err }
	return func() { DescribeProbe = prev }
}

func TestScanProviderReportsDescribeProviderAdoption(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "cli-health", "contracts")

	restore := stubDescribeProbe(nil)
	pr := ScanProvider(context.Background(), probeFn(validProbeResponse("cli-health", "contracts", true), nil), repo, "test-genie", "contracts", "cli-health", time.Second)
	restore()
	if !pr.DescribeAdopted {
		t.Fatalf("adopting provider reported describeAdopted=false: %+v", pr)
	}
	if pr.AdoptionScore != 1.0 {
		t.Fatalf("adoption_score = %v, want 1.0 for a fully conformant provider", pr.AdoptionScore)
	}
}

func TestScanProviderFlagsUnadoptedDescribeProviderWithoutFailingIt(t *testing.T) {
	repo := t.TempDir()
	writeSpec(t, repo, "cli-health", "contracts")

	restore := stubDescribeProbe(connect.NewError(connect.CodeUnimplemented, errors.New("not adopted")))
	pr := ScanProvider(context.Background(), probeFn(validProbeResponse("cli-health", "contracts", true), nil), repo, "test-genie", "contracts", "cli-health", time.Second)
	restore()

	if pr.DescribeAdopted {
		t.Fatal("provider returning Unimplemented was scored as having adopted DescribeProvider")
	}
	// The cost must be named...
	var named bool
	for _, v := range pr.Violations {
		if strings.Contains(v, "DescribeProvider not adopted") {
			named = true
		}
	}
	if !named {
		t.Errorf("the fallback cost was not surfaced in violations: %+v", pr.Violations)
	}
	// ...but non-adoption is advisory during migration, so it must not turn an
	// otherwise-healthy provider into a hard violation.
	if pr.HasHardViolation() {
		t.Errorf("non-adoption was treated as a hard violation while the fleet is still migrating: %+v", pr)
	}
}
