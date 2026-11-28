package monetization

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCL3HealthPhaseRetainsLocalResultsButFailsClosedWithoutRemoteReceipts(t *testing.T) {
	report := &CL3HealthReport{
		Status: "failed", LocalContracts: "passed", StagingShowcase: "unverified", ProductionShowcase: "unverified",
		Suites: []CL3HealthSuiteResult{{Name: "tls-alert-transitions", Status: "passed", EvidenceRefs: []string{"api/health/alerts_test.go::TestAlerterCertificateExpiringAndRenewalFailed"}}},
	}
	response := cl3HealthResponse("scenario-to-cloud", report)
	if response.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("phase status = %s, want failed until staging and production receipts exist", response.GetStatus())
	}
	assessment := response.GetAssessment()
	if assessment.GetPhase() != cl3HealthPhase || assessment.GetLocal().GetCurrentLevel() != "L1" || assessment.GetLocal().GetClean() {
		t.Fatalf("assessment = phase %q level %q clean=%t, want local-only L1 and not clean", assessment.GetPhase(), assessment.GetLocal().GetCurrentLevel(), assessment.GetLocal().GetClean())
	}
	if len(assessment.GetFindings()) != 2 || assessment.GetFindings()[0].GetCode() != "cl3.staging_showcase_missing" || assessment.GetFindings()[1].GetCode() != "cl3.production_showcase_missing" {
		t.Fatalf("remote findings = %+v, want staging and production receipt blockers", assessment.GetFindings())
	}
	var native structpb.Struct
	if err := response.GetNativeDetail().UnmarshalTo(&native); err != nil {
		t.Fatalf("unmarshal native phase evidence: %v", err)
	}
	results := native.AsMap()["cl3_production_health"].(map[string]any)["suites"].([]any)
	refs := results[0].(map[string]any)["evidence_refs"].([]any)
	if refs[0] != "api/health/alerts_test.go::TestAlerterCertificateExpiringAndRenewalFailed" {
		t.Fatalf("durable native evidence refs = %v", refs)
	}
}

func TestCL3HealthProviderContractExecutesAndReportsExactPhaseIdentity(t *testing.T) {
	called := false
	handler := Handler{healthRunner: func(_ context.Context, path string) (*CL3HealthReport, error) {
		called = true
		if path != "/repo/scenarios/scenario-to-cloud" {
			t.Fatalf("scenario path = %q", path)
		}
		return &CL3HealthReport{Status: "failed", LocalContracts: "passed", StagingShowcase: "unverified", ProductionShowcase: "unverified"}, nil
	}}
	response, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "scenario-to-cloud", Path: "/repo/scenarios/scenario-to-cloud", Phase: cl3HealthPhase, IncludeExecution: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !called || response.Msg.GetAssessment().GetProvider() != "landing-page-business-suite" || response.Msg.GetAssessment().GetPhase() != cl3HealthPhase {
		t.Fatalf("runner called=%t assessment=%+v", called, response.Msg.GetAssessment())
	}
}

func TestCL3HealthProviderRejectsWrongScenarioAndNoExecutionDoesNotClaimPass(t *testing.T) {
	handler := Handler{healthRunner: func(context.Context, string) (*CL3HealthReport, error) {
		t.Fatal("runner must not execute for the wrong target")
		return nil, nil
	}}
	_, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "landing-page-business-suite", Phase: cl3HealthPhase, IncludeExecution: true}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("wrong target error = %v, want invalid argument", err)
	}
	response, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "scenario-to-cloud", Phase: cl3HealthPhase, IncludeExecution: false}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED {
		t.Fatalf("non-execution status = %s, want not evaluated", response.Msg.GetStatus())
	}
}

func TestCL3RepositoryRootIsPinnedToScenarioAndSafeTestEnvDropsCredentials(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "scenarios", "scenario-to-cloud")
	if err := os.MkdirAll(scenario, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := cl3RepositoryRoot(scenario)
	if err != nil || resolved != root {
		t.Fatalf("repositoryRoot(%q) = %q, %v", scenario, resolved, err)
	}
	if _, err := cl3RepositoryRoot(filepath.Join(root, "scenarios", "landing-page-business-suite")); err == nil {
		t.Fatal("repository root accepted a different target scenario")
	}
	t.Setenv("STRIPE_SECRET_KEY", "must-not-be-forwarded")
	t.Setenv("DATABASE_URL", "postgres://must-not-be-forwarded")
	for _, entry := range cl3SafeTestEnv([]string{"SCENARIO_TO_CLOUD_API_BASE=http://127.0.0.1:18888"}) {
		if entry == "STRIPE_SECRET_KEY=must-not-be-forwarded" || entry == "DATABASE_URL=postgres://must-not-be-forwarded" {
			t.Fatalf("test subprocess inherited credential: %s", entry)
		}
	}
}
