package validation

import (
	"connectrpc.com/connect"
	"context"
	"github.com/gorilla/mux"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"go-code-graph/internal/module"
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
func (h *handler) assessment(s string) *commonv1.MaturityAssessment {
	return &commonv1.MaturityAssessment{Scenario: s, Provider: "go-code-graph", Phase: "go-code-graph", Version: "1.0.0", Local: &commonv1.LocalMaturityAssessment{CurrentLevel: "L2", Clean: true}, Presentation: &commonv1.PhasePresentation{ContractVersion: "v1", Provider: "go-code-graph", Phase: "go-code-graph", CurrentLevel: "L2", Clean: true, AtMaximum: true, Capabilities: []*commonv1.PhaseCapabilityPresentation{{Id: "local", Label: "Local Maturity", CurrentLevel: "L2", Clean: true, PriorityRank: 1}}}}
}
func (h *handler) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	s := req.Msg.GetScenario()
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{Scenario: s, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: h.assessment(s), Metrics: metricsFor()}), nil
}
func (h *handler) ValidateTarget(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	t := req.Msg.GetTarget()
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: t, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: h.assessment(t.GetId()), Metrics: metricsFor()}), nil
}
