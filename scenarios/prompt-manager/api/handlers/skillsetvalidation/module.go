package skillsetvalidation

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	maturityassessment "github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"prompt-manager/internal/skillset"
)

func NewConnectMount(repoRoot string) (string, http.Handler) {
	h := &handler{repoRoot: repoRoot}
	path, endpoint := scenariovalidationconnect.NewScenarioValidationServiceHandler(h)
	return path, endpoint
}

type handler struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	repoRoot string
}

func (h *handler) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	scenario := req.Msg.GetScenario()
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario is required"))
	}
	result := skillset.Validate(h.repoRoot, scenario)
	clean := result.Status == "ok"
	level := "L1"
	if clean {
		level = "L2"
	}
	if result.Status == "unavailable" {
		level = "L0"
	}
	now := time.Now()
	levels := []*commonv1.LocalMaturityLevel{{Id: "L0", Name: "Skill set unavailable", StatusLabel: "Unavailable", NextUnlock: "Readable service metadata."}, {Id: "L1", Name: "Skill set inspectable", StatusLabel: "Foundation", NextUnlock: "Clear skill-set findings."}, {Id: "L2", Name: "Skill set clean", StatusLabel: "Complete", CapabilitySummary: "Every owed skill role is declared and reachable."}}
	local := &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: func() string {
		if clean {
			return ""
		}
		if level == "L0" {
			return "L1"
		}
		return "L2"
	}(), Clean: clean, BlockingFindingCodes: result.Findings, Levels: levels}
	assessment := &commonv1.MaturityAssessment{Scenario: scenario, Provider: "prompt-manager", Phase: "skill-set", Version: "2.0.0", Local: local}
	assessment.Presentation = maturityassessment.BuildPhasePresentation(assessment)
	detail, _ := structpb.NewStruct(map[string]any{"roles": result.Roles, "findings": result.Findings})
	native, _ := anypb.New(detail)
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if !clean {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{Scenario: scenario, Status: status, Assessment: assessment, NativeDetail: native, Metrics: &commonv1.ExecutionMetrics{StartedAt: timestamppb.New(now), CompletedAt: timestamppb.Now()}}), nil
}
