// Package validationstub provides the shared fixed L2 validation surface used by
// analysis tools that do not yet own a scenario validation implementation.
package validationstub

import (
	"context"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/module"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type handler struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	provider string
}

// Module mounts a consistent placeholder validation contract for provider.
// Owners should replace this module when they implement real validation.
func Module(provider string) module.Module {
	h := &handler{provider: provider}
	return module.Module{Name: "validation", Mount: func(r *mux.Router) {
		path, endpoint := scenariovalidationconnect.NewScenarioValidationServiceHandler(h)
		r.PathPrefix(path).Handler(endpoint)
	}}
}

func (h *handler) assessment(scenario string) *commonv1.MaturityAssessment {
	return &commonv1.MaturityAssessment{
		Scenario: scenario, Provider: h.provider, Phase: h.provider, Version: "1.0.0",
		Local: &commonv1.LocalMaturityAssessment{CurrentLevel: "L2", Clean: true},
		Presentation: &commonv1.PhasePresentation{
			ContractVersion: "v1", Provider: h.provider, Phase: h.provider,
			CurrentLevel: "L2", Clean: true, AtMaximum: true,
			Capabilities: []*commonv1.PhaseCapabilityPresentation{{Id: "local", Label: "Local Maturity", CurrentLevel: "L2", Clean: true, PriorityRank: 1}},
		},
	}
}

func (h *handler) metrics() *commonv1.ExecutionMetrics {
	now := timestamppb.Now()
	return &commonv1.ExecutionMetrics{StartedAt: now, CompletedAt: now}
}

func (h *handler) ValidateScenario(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	scenario := req.Msg.GetScenario()
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{Scenario: scenario, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: h.assessment(scenario), Metrics: h.metrics()}), nil
}

func (h *handler) ValidateTarget(_ context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	target := req.Msg.GetTarget()
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: target, Status: scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, Assessment: h.assessment(target.GetId()), Metrics: h.metrics()}), nil
}
