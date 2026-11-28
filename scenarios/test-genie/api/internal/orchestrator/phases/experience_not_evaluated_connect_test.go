package phases

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"test-genie/internal/orchestrator/phases/validationprovider"
	"test-genie/internal/orchestrator/workspace"
)

type notEvaluatedValidationOwner struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
}

// The Connect fixture stands in for a provider whose runtime evaluator is
// unavailable. UI Health's production runner against an unreachable BAS port
// has its own regression; this test proves Test Genie's wire and phase path.
func (notEvaluatedValidationOwner) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if req.Msg.GetScenario() != "fixture" || req.Msg.GetPhase() != "experience" || !req.Msg.GetIncludeExecution() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errUnexpectedExperienceRequest)
	}
	phaseAssessment := &commonv1.MaturityAssessment{
		Scenario: "fixture", Provider: "ui-health", Phase: "experience", Version: "1",
		Local:              &commonv1.LocalMaturityAssessment{CurrentLevel: "L1", NextLevel: "L2", Clean: true},
		FindingsBySeverity: map[string]int32{"SEVERITY_INFO": 1},
		Findings: []*commonv1.AssessmentFinding{{
			Severity: "SEVERITY_INFO",
			Code:     "runtime_skipped_bas_unavailable",
			Message:  "runtime render could not collect BAS evidence at a typed failure stage",
			Maturity: &commonv1.FindingMaturity{GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY},
			Evidence: []*commonv1.AssessmentEvidence{{
				Kind:    "ui_health_runtime_unavailable.v1",
				Summary: "BAS runtime collection stopped at get_execution_timeline",
				Payload: []byte(`{"schemaVersion":1,"stage":"get_execution_timeline","correlationId":"2d6ae1c9-36ab-4e02-b6a7-8382f8f7c09d","executionId":"78fffc0f-a6c4-4f46-9f0e-1a1c3a630ac1","transportKind":"connect_error","connectCode":"unavailable"}`),
			}},
		}},
	}
	phaseAssessment.Presentation = assessment.BuildPhasePresentation(phaseAssessment)
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{
		Scenario:   "fixture",
		Status:     scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED,
		Assessment: phaseAssessment,
	}), nil
}

var errUnexpectedExperienceRequest = errors.New("unexpected canonical experience request")

func TestCanonicalExperienceConnectRunDoesNotPassProviderNotEvaluated(t *testing.T) {
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(notEvaluatedValidationOwner{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	previousResolve := validationprovider.ResolveBaseURL
	previousClient := validationprovider.NewClient
	validationprovider.ResolveBaseURL = func(context.Context, string) (string, error) { return server.URL, nil }
	validationprovider.NewClient = func(timeout time.Duration, baseURL string) validationprovider.Client {
		return scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
	}
	t.Cleanup(func() {
		validationprovider.ResolveBaseURL = previousResolve
		validationprovider.NewClient = previousClient
	})

	root := t.TempDir()
	env := workspace.Environment{RunID: "experience-unreachable-bas", ScenarioName: "fixture", ScenarioDir: root}
	report := providerRunner(validationprovider.Provider{Phase: "experience", ProviderScenario: "ui-health", IncludeExecution: true}, nil)(context.Background(), env, io.Discard)
	if !report.NotEvaluated || report.Err != nil || report.OutcomeStatus != "not_evaluated" {
		t.Fatalf("canonical experience phase passed or lost the not-evaluated verdict: %+v", report)
	}
	if len(report.Findings) != 1 || len(report.Findings[0].GetEvidence()) != 1 {
		t.Fatalf("canonical phase lost runtime diagnostic finding: %+v", report.Findings)
	}
	phaseEvidence := report.Findings[0].GetEvidence()[0]
	if phaseEvidence.GetKind() != "ui_health_runtime_unavailable.v1" || !strings.Contains(string(phaseEvidence.GetPayload()), `"stage":"get_execution_timeline"`) {
		t.Fatalf("canonical phase changed typed runtime diagnostic: %+v", phaseEvidence)
	}
	raw, err := os.ReadFile(filepath.Join(root, "coverage", "runs", env.RunID, "phase-results", "experience.json"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted struct {
		Status           string `json:"status"`
		FindingCount     int    `json:"finding_count"`
		FindingsArtifact string `json:"findings_artifact"`
	}
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Status != "not_evaluated" {
		t.Fatalf("persisted canonical experience status = %q, want not_evaluated", persisted.Status)
	}
	if persisted.FindingCount != 1 || persisted.FindingsArtifact == "" {
		t.Fatalf("persisted phase pointer lost finding reference: %+v", persisted)
	}
}
