package monetization

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"landing-page-business-suite-api/internal/credentialleak"
)

func TestCredentialLeakPhaseWithoutExecutionIsNotEvaluated(t *testing.T) {
	handler := Handler{}
	response, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "landing-page-business-suite", Phase: credentialLeakPhase, IncludeExecution: false,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED {
		t.Fatalf("status = %s, want NOT_EVALUATED", response.Msg.GetStatus())
	}
	if response.Msg.GetAssessment().GetPhase() != credentialLeakPhase || response.Msg.GetAssessment().GetLocal().GetClean() {
		t.Fatalf("assessment = %+v, want blocked not-evaluated phase", response.Msg.GetAssessment())
	}
}

func TestCredentialLeakPhaseRunsOwnerProofAndFailsClosedUntilArtifactsAreScanned(t *testing.T) {
	called := false
	handler := Handler{credentialLeakRunner: func(_ context.Context, path string) (*CredentialLeakReport, error) {
		called = true
		if path != "/repo/scenarios/landing-page-business-suite" {
			t.Fatalf("scenario path = %q", path)
		}
		return &CredentialLeakReport{Status: "partial", LocalContracts: "passed", NegativeControl: "detected_all_families", TestGenArtifacts: "unverified"}, nil
	}}
	response, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "landing-page-business-suite", Path: "/repo/scenarios/landing-page-business-suite", Phase: credentialLeakPhase, IncludeExecution: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !called || response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("runner called=%t status=%s, want executed and failed closed on artifact gap", called, response.Msg.GetStatus())
	}
	if response.Msg.GetAssessment().GetLocal().GetCurrentLevel() != "L1" || response.Msg.GetAssessment().GetLocal().GetClean() {
		t.Fatalf("assessment = %+v, want local-only L1", response.Msg.GetAssessment().GetLocal())
	}
}

func TestCredentialLeakProviderContractAndTargetGuard(t *testing.T) {
	description, err := (Handler{}).DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{Phase: credentialLeakPhase}))
	if err != nil {
		t.Fatal(err)
	}
	if description.Msg.GetProvider() != "landing-page-business-suite" || description.Msg.GetPhase() != credentialLeakPhase || !description.Msg.GetCapabilities().GetSupportsExecution() {
		t.Fatalf("provider descriptor = %+v", description.Msg)
	}
	handler := Handler{credentialLeakRunner: func(context.Context, string) (*CredentialLeakReport, error) {
		t.Fatal("runner must not execute for another scenario")
		return nil, nil
	}}
	_, err = handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "scenario-to-cloud", Phase: credentialLeakPhase, IncludeExecution: true}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("wrong-target error = %v", err)
	}
}

func TestCredentialLeakSafeTestEnvironmentDropsPaidCredentials(t *testing.T) {
	for _, key := range []string{"STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "AWS_SECRET_ACCESS_KEY", "SENDGRID_API_KEY", "LPBS_SERVICE_SECRET", "DATABASE_URL"} {
		t.Setenv(key, "synthetic-must-not-forward")
	}
	for _, entry := range cl3SafeTestEnv(nil) {
		if strings.Contains(entry, "synthetic-must-not-forward") {
			t.Fatal("credential-bearing parent environment value reached proof subprocess")
		}
	}
}

func TestCredentialLeakOutputBufferMarksExactBoundaryTruncation(t *testing.T) {
	var output credentialLeakBoundedBuffer
	if _, err := output.Write(make([]byte, 32<<10)); err != nil {
		t.Fatal(err)
	}
	if output.truncated {
		t.Fatal("exact-boundary write was marked truncated before any byte was lost")
	}
	if _, err := output.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	if !output.truncated || output.Len() != 32<<10 {
		t.Fatal("discarded bytes after exact boundary were not reported as truncation")
	}
}

func TestLPBSRepositoryRootRequiresOwningScenarioPath(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "scenarios", "landing-page-business-suite")
	if err := os.MkdirAll(scenario, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module synthetic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if resolved, err := lpbsRepositoryRoot(scenario); err != nil || resolved != root {
		t.Fatalf("LPBS root = %q, %v", resolved, err)
	}
	if _, err := lpbsRepositoryRoot(filepath.Join(root, "scenarios", "scenario-to-cloud")); err == nil {
		t.Fatal("root guard accepted a non-LPBS target")
	}
}

func TestRunCredentialLeakProofScansRealOwnerSubprocessesAndReturnsMetadataOnly(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal("resolve repository root")
	}
	scenario := filepath.Join(root, "scenarios", "landing-page-business-suite")
	report, err := RunCredentialLeakProof(context.Background(), scenario)
	if err != nil {
		t.Fatalf("credential leak owner phase failed before producing metadata: %v", err)
	}
	if report.NegativeControl != "detected_all_families" || len(report.Suites) != 12 {
		t.Fatalf("phase report incomplete: negative control %q, suites %d", report.NegativeControl, len(report.Suites))
	}
	for _, suite := range report.Suites {
		if suite.Status != "passed" || suite.ExitCode != 0 || len(suite.Findings) != 0 || suite.OutputScannedBytes == 0 {
			t.Fatalf("owner suite %q did not produce a clean scanned receipt: %+v", suite.Name, suite)
		}
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal("phase metadata serialization failed")
	}
	data, err := os.ReadFile(filepath.Join(scenario, ".vrooli", "service.json"))
	if err != nil {
		t.Fatal("credential family declarations unavailable")
	}
	families, err := credentialleak.ManifestFamilies(data)
	if err != nil {
		t.Fatal("credential family metadata unavailable")
	}
	scanner, err := credentialleak.NewScanner(credentialleak.SyntheticCanaries(families))
	if err != nil {
		t.Fatal("synthetic leak scanner unavailable")
	}
	findings, err := scanner.Scan(credentialleak.Observation{Channel: "provider-report-json", Data: encoded})
	if err != nil || len(findings) != 0 {
		t.Fatal("phase report contains synthetic credential material")
	}
	if report.TestGenArtifacts != "unverified" || report.LocalContracts != "passed" || report.InventoryStatus != "complete" {
		t.Fatalf("phase improperly advanced despite inventory/artifact gaps: local=%q inventory=%q artifacts=%q", report.LocalContracts, report.InventoryStatus, report.TestGenArtifacts)
	}
	t.Logf("metadata-only local proof report: %s", encoded)
}
