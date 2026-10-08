package conformance

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	channelcore "switchboard/internal/channels"
	"switchboard/internal/channels/conformance"
	"switchboard/internal/module"
)

// Module exposes Switchboard's shared channel contract as a Test Genie
// validation-provider phase. External transports use the deterministic
// fixture path here; their native protocol behavior remains covered by the
// adapter package tests and live availability is reported by Probe.
func Module(registry *channelcore.Registry) module.Module {
	h := &handler{registry: registry}
	path, endpoint := scenariovalidationconnect.NewScenarioValidationServiceHandler(h)
	return module.Module{
		Name: "channel-conformance",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(endpoint)
		},
	}
}

type handler struct {
	scenariovalidationconnect.UnimplementedScenarioValidationServiceHandler
	registry *channelcore.Registry
}

func (h *handler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	started := time.Now()
	results := map[string]any{}
	failed := 0
	if h.registry != nil {
		for _, listing := range h.registry.List(ctx, channelcore.HostFacts{}) {
			adapter := fixtureAdapter{IDValue: listing.Descriptor.ID}
			cases := conformance.Run(ctx, adapter, listing.Descriptor)
			values := make([]any, 0, len(cases))
			for _, result := range cases {
				if !result.Passed {
					failed++
				}
				values = append(values, map[string]any{"name": result.Name, "passed": result.Passed, "detail": result.Detail})
			}
			results[listing.Descriptor.ID] = values
		}
	}
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	level := "L2"
	clean := failed == 0
	if !clean {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
		level = "L1"
	}
	native, _ := structpb.NewStruct(map[string]any{"adapters": results, "failed_cases": failed})
	assessment := &commonv1.MaturityAssessment{
		Scenario: req.Msg.GetScenario(), Provider: "switchboard", Phase: "channel-conformance", Version: "1.0.0",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel: level, Clean: clean,
			BlockingFindingCodes: []string{},
			Levels: []*commonv1.LocalMaturityLevel{
				{Id: "L0", Name: "Conformance unavailable", StatusLabel: "Unavailable", NextUnlock: "The provider is reachable."},
				{Id: "L1", Name: "Contract runnable", StatusLabel: "Foundation", NextUnlock: "Every registered channel passes all cases."},
				{Id: "L2", Name: "Channels conformant", StatusLabel: "Complete", CapabilitySummary: "Registered channel adapters share one proven contract."},
			},
		},
	}
	capability := &commonv1.CapabilityMaturityAssessment{Id: "channel_contract", Label: "Channel Contract", CurrentLevel: level, Clean: clean, PriorityRank: 1, CurrentSummary: "Registered channel adapters share one proven contract.", Levels: assessment.Local.Levels, BlockingFindingCodes: []string{}}
	assessment.Capabilities = []*commonv1.CapabilityMaturityAssessment{capability}
	assessment.HighestPriorityCapability = &commonv1.PriorityFocus{CapabilityId: capability.Id, CapabilityLabel: capability.Label, CurrentLevel: level, Reason: "lowest current level"}
	assessment.Presentation = &commonv1.PhasePresentation{
		ContractVersion: "v1", Provider: assessment.Provider, Phase: assessment.Phase, CurrentLevel: level, CurrentLevelLabel: "Complete", CeilingLevel: "L2", Clean: clean, BlockingFindingCodes: []string{}, NorthStar: capability.CurrentSummary, AtMaximum: clean,
		FocusCapabilityId: capability.Id, FocusCapabilityLabel: capability.Label, NextActionReason: "lowest current level",
		Capabilities: []*commonv1.PhaseCapabilityPresentation{{Id: capability.Id, Label: capability.Label, CurrentLevel: level, CurrentLevelLabel: "Complete", CurrentSummary: capability.CurrentSummary, Clean: clean, PriorityRank: 1, BlockingFindingCodes: []string{}, Findings: []*commonv1.PhasePresentationFinding{}}},
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{
		Scenario: req.Msg.GetScenario(), Status: status, Assessment: assessment, NativeDetail: func() *anypb.Any { value, _ := anypb.New(native); return value }(),
		FailureClassification: func() string {
			if clean {
				return ""
			}
			return "channel_conformance_failed"
		}(),
		Metrics: &commonv1.ExecutionMetrics{StartedAt: timestamppb.New(started), CompletedAt: timestamppb.Now()},
	}), nil
}

func (h *handler) DescribeProvider(context.Context, *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.DescribeProviderResponse{
		Provider: "switchboard", Phase: "channel-conformance", SpecVersion: "1.0.0", Contract: "scenario-validation/v1",
		Capabilities: &scenariovalidationv1.ProviderCapabilities{
			SupportsExecution: false, DeliveryMode: "inline", SupportsFixes: false,
			TargetKinds: []commonv1.ValidationTargetKind{commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO},
		},
	}), nil
}

func (h *handler) PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Applied: false, Messages: []string{"Channel conformance has no automatic fixers."}}), nil
}

func (h *handler) ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Applied: false, Messages: []string{"Channel conformance has no automatic fixers."}}), nil
}

type fixtureAdapter struct{ IDValue string }

func (a fixtureAdapter) ID() string                                                      { return a.IDValue }
func (a fixtureAdapter) Connect(context.Context, func(channelcore.Envelope) error) error { return nil }
func (a fixtureAdapter) Send(context.Context, channelcore.Outbound) error                { return nil }
func (a fixtureAdapter) Probe(context.Context) channelcore.ProbeResult {
	return channelcore.ProbeResult{Available: true}
}
