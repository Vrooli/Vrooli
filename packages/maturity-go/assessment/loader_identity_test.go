package assessment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestLoadSpecFromScenario(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	raw, err := json.Marshal(validSpec())
	if err != nil {
		t.Fatalf("marshal spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "maturity.json"), raw, 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
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
		t.Fatal("expected error for missing maturity.json")
	}
}

func TestLoadSpecFromScenarioInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "maturity.json"), []byte(`{"provider":""}`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if _, err := LoadSpecFromScenario(dir); err == nil {
		t.Fatal("expected validation error for invalid spec")
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
