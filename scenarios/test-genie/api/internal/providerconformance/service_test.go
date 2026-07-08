package providerconformance

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

// fixtureRepo lays out <root>/scenarios/<name>/.vrooli/test-genie.json plus a
// docs file so a mutation-free descriptor validates clean.
// conformantDoc is a remediation doc that satisfies the required skeleton (all
// five H2 headings), so the shared fixture is Phase Capability Contract-clean and
// individual tests break only what they assert.
const conformantDoc = `# Fixture

## North Star
Everything is verified.

## The rungs and their gates
L0 → L4.

## What each finding means
Each code caps a rung.

## The canonical fix
Do the thing.

## How to verify
Run the check.
`

func fixtureRepo(t *testing.T, scenario string, mutate func(map[string]any)) (repoRoot, scenarioDir string) {
	t.Helper()
	repoRoot = t.TempDir()
	scenarioDir = filepath.Join(repoRoot, "scenarios", scenario)
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	docsRel := filepath.ToSlash(filepath.Join("scenarios", scenario, "README.md"))
	if err := os.WriteFile(filepath.Join(scenarioDir, "README.md"), []byte(conformantDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	descriptor := map[string]any{
		"schemaVersion": "1.0.0",
		"scenario":      scenario,
		"phase":         "demo",
		"displayName":   "Demo",
		"description":   "Fixture provider phase.",
		"source":        "validation-provider",
		"timeout":       "30s",
		"validation":    map[string]any{"contract": "scenario-validation/v1"},
		"applicability": map[string]any{
			"default": "not_applicable",
			"any":     []any{map[string]any{"fileExists": ".vrooli/demo.json"}},
		},
		"policy": map[string]any{
			"selection":         "default_when_applicable",
			"providerReadiness": "required_when_applicable",
			"providerLifecycle": "start_if_needed",
			"freshness":         "require_live_contract",
			"resultGating":      "gating",
			"unavailable":       "fail",
		},
		"runnability": map[string]any{},
		"docs":        map[string]any{"path": docsRel},
		"maturity": map[string]any{
			"version": "1.0.0",
			"capabilities": []any{
				map[string]any{
					"id":          "cap",
					"label":       "Capability",
					"description": "Fixture capability.",
					"levels": []any{
						map[string]any{"id": "L0", "name": "zero", "entry_criteria": []any{"start"}, "exit_criteria": []any{"leave L0"}, "next_unlock": "reach L1"},
						map[string]any{"id": "L1", "name": "one", "entry_criteria": []any{"enter L1"}, "exit_criteria": []any{"stay clean"}, "capability_summary": "The fixture capability is fully realized."},
					},
				},
			},
			"findings": map[string]any{
				"DEMO_FINDING": map[string]any{
					"capability_id":      "cap",
					"local_level_impact": "L1",
					"global_impact":      "capability_gap",
					"dimension":          "contracts",
					"severity_default":   "SEVERITY_ERROR",
					"clean_requirement":  "required",
					"fix_class":          "manual",
					"reason":             "fixture judgment",
				},
			},
			"fallback": map[string]any{
				"capability_id":      "cap",
				"local_level_impact": "L1",
				"global_impact":      "unknown",
				"severity_default":   "SEVERITY_WARNING",
				"clean_requirement":  "advisory",
			},
		},
	}
	if mutate != nil {
		mutate(descriptor)
	}
	raw, err := json.MarshalIndent(descriptor, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "test-genie.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return repoRoot, scenarioDir
}

func findingCodes(report Report) []string {
	codes := make([]string, 0, len(report.Findings))
	for _, f := range report.Findings {
		codes = append(codes, f.Code)
	}
	return codes
}

func requireCode(t *testing.T, report Report, code string) Finding {
	t.Helper()
	for _, f := range report.Findings {
		if f.Code == code {
			return f
		}
	}
	t.Fatalf("finding %s absent; got %v", code, findingCodes(report))
	return Finding{}
}

func TestValidateScenarioCleanDescriptor(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", nil)
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("want clean report, got findings %v", findingCodes(report))
	}
	if report.Summary.Status() != "passed" {
		t.Fatalf("status = %q, want passed", report.Summary.Status())
	}
	if report.Phase != "demo" {
		t.Fatalf("phase = %q, want demo", report.Phase)
	}
	if report.Probed {
		t.Fatal("no probe seam configured; Probed must be false")
	}
}

func TestValidateScenarioDescriptorMissing(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioDir := filepath.Join(repoRoot, "scenarios", "bare")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := New(repoRoot).ValidateScenario(context.Background(), "bare", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodeDescriptorMissing)
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %q, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioInvalidJSON(t *testing.T) {
	repoRoot, scenarioDir := fixtureRepo(t, "demo-provider", nil)
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "test-genie.json"), []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodeDescriptorInvalid)
	if report.ProbeSkipReason == "" {
		t.Fatal("broken descriptor must record a probe skip reason")
	}
}

func TestValidateScenarioIdentityMismatch(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", func(d map[string]any) {
		d["scenario"] = "someone-else"
	})
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodeIdentityMismatch)
}

func TestValidateScenarioStaleMaturityFile(t *testing.T) {
	repoRoot, scenarioDir := fixtureRepo(t, "demo-provider", nil)
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "maturity.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodeStaleMaturityFile)
}

func TestValidateScenarioUnsafePolicy(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", func(d map[string]any) {
		policy := d["policy"].(map[string]any)
		// providerReadiness none + a lifecycle that manages the provider is the
		// canonical unsafe combination the policy validator rejects.
		policy["providerReadiness"] = "none"
		policy["providerLifecycle"] = "start_if_needed"
	})
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodePolicyUnsafe)
}

func TestValidateScenarioInvalidMaturity(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", func(d map[string]any) {
		maturity := d["maturity"].(map[string]any)
		delete(maturity, "version")
	})
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodeMaturityInvalid)
}

func TestValidateScenarioDocsMissing(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", func(d map[string]any) {
		d["docs"] = map[string]any{"path": "scenarios/demo-provider/docs/absent.md"}
	})
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	finding := requireCode(t, report, CodeDocsMissing)
	if finding.Severity != SeverityWarning {
		t.Fatalf("docs finding severity = %q, want warning", finding.Severity)
	}
	if report.Summary.Status() != "passed" {
		t.Fatalf("advisory docs finding must not fail the report; status = %q", report.Summary.Status())
	}
}

func TestValidateScenarioAutofixDeclarationIncomplete(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", func(d map[string]any) {
		maturity := d["maturity"].(map[string]any)
		findings := maturity["findings"].(map[string]any)
		findings["UNDECLARED_FINDING"] = map[string]any{
			"capability_id":      "cap",
			"local_level_impact": "L1",
			"global_impact":      "capability_gap",
			"dimension":          "contracts",
			"severity_default":   "SEVERITY_ERROR",
			"clean_requirement":  "required",
		}
	})
	report, err := New(repoRoot).ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodeAutofixDeclarationIncomplete)
}

// probeResponse builds a contract-valid ValidateScenarioResponse whose
// assessment identity is provider/phase.
func probeResponse(t *testing.T, repoRoot, provider, phase, target string, metrics *commonv1.ExecutionMetrics) *scenariovalidationv1.ValidateScenarioResponse {
	t.Helper()
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", provider))
	if err != nil {
		t.Fatalf("load fixture spec: %v", err)
	}
	spec.Provider = provider
	spec.Phase = phase
	maturity, err := assessment.BuildProtoAssessment(assessment.BuildInput{Scenario: target, Spec: *spec})
	if err != nil {
		t.Fatalf("build fixture assessment: %v", err)
	}
	resp, err := assessment.BuildValidationResponse(target, maturity, nil, metrics)
	if err != nil {
		t.Fatalf("build fixture response: %v", err)
	}
	return resp
}

func TestValidateScenarioLiveProbeClean(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", nil)
	service := New(repoRoot)
	service.Probe = func(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		if provider != "demo-provider" {
			t.Fatalf("probe provider = %q, want demo-provider", provider)
		}
		return probeResponse(t, repoRoot, "demo-provider", "demo", target, &commonv1.ExecutionMetrics{}), nil
	}
	report, err := service.ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if !report.Probed {
		t.Fatal("probe seam configured; Probed must be true")
	}
	if len(report.Findings) != 0 {
		t.Fatalf("want clean live report, got findings %v", findingCodes(report))
	}
}

func TestValidateScenarioLiveProbeUnreachable(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", nil)
	service := New(repoRoot)
	service.Probe = func(context.Context, string, string, time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return nil, errors.New("connection refused")
	}
	report, err := service.ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	finding := requireCode(t, report, CodeProviderUnreachable)
	if finding.Severity != SeverityWarning {
		t.Fatalf("unreachable severity = %q, want warning (environmental)", finding.Severity)
	}
	if report.Summary.Status() != "passed" {
		t.Fatalf("unreachable provider must not hard-fail; status = %q", report.Summary.Status())
	}
}

func TestValidateScenarioLiveProbeContractInvalid(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", nil)
	service := New(repoRoot)
	service.Probe = func(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return &scenariovalidationv1.ValidateScenarioResponse{
			Scenario: target,
			Status:   scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED,
		}, nil
	}
	report, err := service.ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodeContractInvalid)
	if report.Summary.Status() != "failed" {
		t.Fatalf("contract violation must fail; status = %q", report.Summary.Status())
	}
}

func TestValidateScenarioLiveProbeIdentityMismatch(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", nil)
	service := New(repoRoot)
	service.Probe = func(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return probeResponse(t, repoRoot, "demo-provider", "other-phase", target, &commonv1.ExecutionMetrics{}), nil
	}
	report, err := service.ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodeContractIdentityMismatch)
}

func TestValidateScenarioLiveProbeMetricsMissing(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "demo-provider", nil)
	service := New(repoRoot)
	service.Probe = func(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		return probeResponse(t, repoRoot, "demo-provider", "demo", target, nil), nil
	}
	report, err := service.ValidateScenario(context.Background(), "demo-provider", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireCode(t, report, CodeMetricsMissing)
}

func TestValidateScenarioSelfTargetSkipsProbe(t *testing.T) {
	repoRoot, _ := fixtureRepo(t, "test-genie", func(d map[string]any) {
		d["scenario"] = "test-genie"
	})
	service := New(repoRoot)
	service.Probe = func(context.Context, string, string, time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
		t.Fatal("self target must never trigger a live probe (recursion guard)")
		return nil, nil
	}
	report, err := service.ValidateScenario(context.Background(), "test-genie", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if report.Probed {
		t.Fatal("Probed must be false for the self target")
	}
	if report.ProbeSkipReason == "" {
		t.Fatal("self target must record a probe skip reason")
	}
	if len(report.Findings) != 0 {
		t.Fatalf("want clean self report, got findings %v", findingCodes(report))
	}
}

func TestValidateScenarioRequiresTarget(t *testing.T) {
	if _, err := New("").ValidateScenario(context.Background(), "", ""); err == nil {
		t.Fatal("want error when scenario and path are both empty")
	}
}
