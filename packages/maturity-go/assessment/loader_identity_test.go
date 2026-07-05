package assessment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestLoadSpecFromScenario(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "measures-health")
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	raw, err := json.Marshal(testDescriptor(validSpec()))
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "test-genie.json"), raw, 0o644); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}

	spec, err := LoadSpecFromScenario(dir)
	if err != nil {
		t.Fatalf("LoadSpecFromScenario: %v", err)
	}
	if spec.Provider != "measures-health" || spec.Phase != "measures" {
		t.Fatalf("loaded spec identity wrong: %+v", spec)
	}
}

func TestLoadSpecFromScenarioMissing(t *testing.T) {
	if _, err := LoadSpecFromScenario(t.TempDir()); err == nil {
		t.Fatal("expected error for missing test-genie.json")
	}
}

func TestLoadSpecFromScenarioInvalid(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "measures-health")
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "test-genie.json"), []byte(`{"scenario":"","phase":"","maturity":{}}`), 0o644); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}
	if _, err := LoadSpecFromScenario(dir); err == nil {
		t.Fatal("expected validation error for invalid spec")
	}
}

func TestLoadSpecFromScenarioRejectsRetiredMaturityFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "measures-health")
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	raw, err := json.Marshal(testDescriptor(validSpec()))
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "test-genie.json"), raw, 0o644); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "maturity.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write retired spec: %v", err)
	}
	if _, err := LoadSpecFromScenario(dir); err == nil {
		t.Fatal("expected leftover maturity.json to fail")
	}
}

func testDescriptor(spec Spec) map[string]any {
	return map[string]any{
		"scenario": spec.Provider,
		"phase":    spec.Phase,
		"maturity": spec,
	}
}

func TestParseEmbeddedSpecStampsDescriptorIdentity(t *testing.T) {
	raw := []byte(`{
		"version":"1.0.0",
		"levels":[{"id":"L0","name":"Missing","description":"Missing","entry_criteria":[],"exit_criteria":[]}],
		"findings":{
			"CODE":{"local_level_impact":"L0","global_impact":"advisory","dimension":"standards","severity_default":"SEVERITY_WARNING","recommended_skill_ids":[],"fix_class":"manual","reason":"Needs review."}
		},
		"fallback":{"local_level_impact":"L0","global_impact":"unknown","dimension":"standards","severity_default":"SEVERITY_WARNING"}
	}`)
	spec, err := ParseEmbeddedSpec(raw, "provider", "phase")
	if err != nil {
		t.Fatalf("ParseEmbeddedSpec returned error: %v", err)
	}
	if spec.Provider != "provider" || spec.Phase != "phase" {
		t.Fatalf("identity = %s/%s, want provider/phase", spec.Provider, spec.Phase)
	}
}

func TestParseEmbeddedSpecRejectsMismatchedDuplicatedIdentity(t *testing.T) {
	raw := []byte(`{
		"provider":"other",
		"phase":"phase",
		"version":"1.0.0",
		"levels":[{"id":"L0","name":"Missing","description":"Missing","entry_criteria":[],"exit_criteria":[]}],
		"findings":{
			"CODE":{"local_level_impact":"L0","global_impact":"advisory","dimension":"standards","severity_default":"SEVERITY_WARNING","recommended_skill_ids":[],"fix_class":"manual","reason":"Needs review."}
		},
		"fallback":{"local_level_impact":"L0","global_impact":"unknown","dimension":"standards","severity_default":"SEVERITY_WARNING"}
	}`)
	if _, err := ParseEmbeddedSpec(raw, "provider", "phase"); err == nil {
		t.Fatal("expected mismatched provider to fail")
	}
}

func buildValidAssessment(t *testing.T) *commonv1.MaturityAssessment {
	t.Helper()
	a, err := BuildProtoAssessment(BuildInput{
		Scenario: "measures-health",
		Spec:     validSpec(),
		Findings: []Finding{{
			Code:     "measures.uncovered-domain",
			Severity: "ERROR",
			Title:    "Uncovered domain",
		}},
	})
	if err != nil {
		t.Fatalf("build assessment: %v", err)
	}
	return a
}

func TestRequireIdentityMatches(t *testing.T) {
	a := buildValidAssessment(t)
	if err := RequireIdentity("measures-health", "measures", a); err != nil {
		t.Fatalf("RequireIdentity should pass: %v", err)
	}
}

func TestRequireIdentityProviderMismatch(t *testing.T) {
	a := buildValidAssessment(t)
	if err := RequireIdentity("proto-health", "measures", a); err == nil {
		t.Fatal("expected provider mismatch error")
	}
}

func TestRequireIdentityPhaseMismatch(t *testing.T) {
	a := buildValidAssessment(t)
	if err := RequireIdentity("measures-health", "structure", a); err == nil {
		t.Fatal("expected phase mismatch error")
	}
}

func TestRequireIdentityInvalidAssessment(t *testing.T) {
	if err := RequireIdentity("measures-health", "measures", nil); err == nil {
		t.Fatal("expected error for nil assessment")
	}
}

func TestRequireIdentityEmptyExpectationsSkipChecks(t *testing.T) {
	a := buildValidAssessment(t)
	if err := RequireIdentity("", "", a); err != nil {
		t.Fatalf("empty expectations should only validate: %v", err)
	}
}

func TestBuildValidationResponseAttachesMetrics(t *testing.T) {
	a := buildValidAssessment(t)
	metrics := &commonv1.ExecutionMetrics{WallClockMs: 1840}
	resp, err := BuildValidationResponse("measures-health", a, nil, metrics)
	if err != nil {
		t.Fatalf("BuildValidationResponse: %v", err)
	}
	if resp.GetMetrics().GetWallClockMs() != 1840 {
		t.Fatalf("metrics not attached: %+v", resp.GetMetrics())
	}
}

func TestBuildValidationResponseNilMetricsUnset(t *testing.T) {
	a := buildValidAssessment(t)
	resp, err := BuildValidationResponse("measures-health", a, nil, nil)
	if err != nil {
		t.Fatalf("BuildValidationResponse: %v", err)
	}
	if resp.GetMetrics() != nil {
		t.Fatal("nil metrics should leave the field unset")
	}
}
