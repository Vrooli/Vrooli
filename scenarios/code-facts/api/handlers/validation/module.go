package validation

import (
	"context"

	"code-facts/internal/module"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type handler struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
}

func metricsFor() *commonv1.ExecutionMetrics {
	now := timestamppb.Now()
	return &commonv1.ExecutionMetrics{StartedAt: now, CompletedAt: now}
}

func Module() module.Module {
	h := &handler{}
	return module.Module{Name: "validation", Mount: func(r *mux.Router) {
		path, endpoint := scenariovalidationconnect.NewScenarioValidationServiceHandler(h)
		r.PathPrefix(path).Handler(endpoint)
	}}
}

func assessmentFor(scenario string) *commonv1.MaturityAssessment {
	return &commonv1.MaturityAssessment{Scenario: scenario, Provider: "code-facts", Phase: "code-facts", Version: "1.0.0", Local: &commonv1.LocalMaturityAssessment{CurrentLevel: "L2", Clean: true}, Presentation: &commonv1.PhasePresentation{ContractVersion: "v1", Provider: "code-facts", Phase: "code-facts", CurrentLevel: "L2", Clean: true, AtMaximum: true, Capabilities: []*commonv1.PhaseCapabilityPresentation{{Id: "local", Label: "Local Maturity", CurrentLevel: "L2", Clean: true, PriorityRank: 1}}}}
}

func (h *handler) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	scenario := req.Msg.GetScenario()
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{Scenario: scenario, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: assessmentFor(scenario), Metrics: metricsFor()}), nil
}

func (h *handler) ValidateTarget(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	target := req.Msg.GetTarget()
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: target, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: assessmentFor(target.GetId()), Metrics: metricsFor()}), nil
}
