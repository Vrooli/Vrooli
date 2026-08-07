package structureprovider

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type fakeValidationService struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	gotTarget *commonv1.ValidationTarget
}

func (f *fakeValidationService) ValidateTarget(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	f.gotTarget = req.Msg.GetTarget()
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{
		Target: f.gotTarget,
		Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED,
		Assessment: &commonv1.MaturityAssessment{Findings: []*commonv1.AssessmentFinding{
			{Code: "PROJECT_CONFIG_SURFACE", Message: "unapproved entry", Location: ".vrooli/baselines"},
		}},
	}), nil
}

func TestValidateDelegatesProjectTargetAndPreservesFinding(t *testing.T) {
	service := &fakeValidationService{}
	_, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()

	output, err := (Provider{
		ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil },
	}).Validate(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if service.gotTarget.GetKind() != commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT || service.gotTarget.GetId() != TargetID || service.gotTarget.GetRoot() != "/repo" {
		t.Fatalf("target = %#v", service.gotTarget)
	}
	if output.Success {
		t.Fatal("output.Success = true, want delegated failure")
	}
	if output.Report.Checks[4].Name != "project_config_surface" || output.Report.Checks[4].Passed {
		t.Fatalf("project config check = %#v", output.Report.Checks[4])
	}
}

func TestValidateReturnsExplicitUnavailableError(t *testing.T) {
	_, err := (Provider{
		ResolveURL: func(context.Context, string) (string, error) { return "", errors.New("offline") },
	}).Validate(context.Background(), "/repo")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v, want ErrUnavailable", err)
	}
}
