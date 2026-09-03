package main

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type integrationsConnectService struct{ server *Server }

func (s integrationsConnectService) List(ctx context.Context, _ *connect.Request[commonv1.ListIntegrationsRequest]) (*connect.Response[commonv1.ListIntegrationsResponse], error) {
	snap := s.server.integrationSnapshot(ctx, false)
	return connect.NewResponse(&commonv1.ListIntegrationsResponse{Integrations: messagesFromStates(snap.States), GeneratedAt: snap.GeneratedAt.Format("2006-01-02T15:04:05Z07:00")}), nil
}
func (s integrationsConnectService) Refresh(ctx context.Context, _ *connect.Request[commonv1.RefreshIntegrationsRequest]) (*connect.Response[commonv1.ListIntegrationsResponse], error) {
	snap := s.server.integrationSnapshot(ctx, true)
	return connect.NewResponse(&commonv1.ListIntegrationsResponse{Integrations: messagesFromStates(snap.States), GeneratedAt: snap.GeneratedAt.Format("2006-01-02T15:04:05Z07:00")}), nil
}
func (s integrationsConnectService) Get(ctx context.Context, req *connect.Request[commonv1.GetIntegrationRequest]) (*connect.Response[commonv1.GetIntegrationResponse], error) {
	for _, state := range s.server.integrationSnapshot(ctx, false).States {
		if state.ID == req.Msg.GetIntegrationId() {
			return connect.NewResponse(&commonv1.GetIntegrationResponse{Integration: messageFromState(state), Features: featureMessages(state)}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, errors.New("integration not found"))
}

func featureMessages(state capabilityregistry.State) []*commonv1.FeatureState {
	out := make([]*commonv1.FeatureState, 0, len(state.Features))
	for _, feature := range state.Features {
		status := state.FeatureStatus[feature]
		out = append(out, &commonv1.FeatureState{FeatureId: feature, Status: featureStatus(status), ReasonCode: state.FeatureReason[feature], Message: state.Message, CheckedAt: state.CheckedAt})
	}
	return out
}

func featureStatus(status string) commonv1.FeatureStatus {
	switch status {
	case "available":
		return commonv1.FeatureStatus_FEATURE_STATUS_COMPATIBLE
	case "compatible":
		return commonv1.FeatureStatus_FEATURE_STATUS_COMPATIBLE
	case "incompatible":
		return commonv1.FeatureStatus_FEATURE_STATUS_INCOMPATIBLE
	case "missing":
		return commonv1.FeatureStatus_FEATURE_STATUS_MISSING
	case "unknown":
		return commonv1.FeatureStatus_FEATURE_STATUS_UNKNOWN
	default:
		return commonv1.FeatureStatus_FEATURE_STATUS_UNSPECIFIED
	}
}
func (s integrationsConnectService) RunAction(ctx context.Context, req *connect.Request[commonv1.RunIntegrationActionRequest]) (*connect.Response[commonv1.RunIntegrationActionResponse], error) {
	if !req.Msg.GetConfirmed() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("confirmation required"))
	}
	for _, state := range s.server.integrationSnapshot(ctx, false).States {
		if state.ID == req.Msg.GetIntegrationId() {
			if actionKindName(req.Msg.GetAction()) != state.ActionKind {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("action is not eligible"))
			}
			if state.ActionKind == capabilityregistry.ActionKindScenarioStart || state.ActionKind == capabilityregistry.ActionKindScenarioRestart {
				result, err := s.server.actionService.Run(ctx, capabilityregistry.LifecycleActionRequest{IntegrationID: state.ID, ActionKind: state.ActionKind})
				if err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, err)
				}
				return connect.NewResponse(&commonv1.RunIntegrationActionResponse{IntegrationId: state.ID, Action: req.Msg.GetAction(), Status: result.Status, Message: result.Message, RequestId: req.Msg.GetRequestId()}), nil
			}
			return connect.NewResponse(&commonv1.RunIntegrationActionResponse{IntegrationId: state.ID, Action: req.Msg.GetAction(), Status: "owner_guidance", Message: state.OperatorCommand, RequestId: req.Msg.GetRequestId()}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, errors.New("integration not found"))
}

func actionKindName(v commonv1.ActionKind) capabilityregistry.ActionKind {
	switch v {
	case commonv1.ActionKind_ACTION_KIND_OWNER_GUIDANCE:
		return capabilityregistry.ActionKindOwnerGuidance
	case commonv1.ActionKind_ACTION_KIND_SCENARIO_START:
		return capabilityregistry.ActionKindScenarioStart
	case commonv1.ActionKind_ACTION_KIND_SCENARIO_RESTART:
		return capabilityregistry.ActionKindScenarioRestart
	case commonv1.ActionKind_ACTION_KIND_OPERATOR_COMMAND:
		return capabilityregistry.ActionKindOperatorCommand
	default:
		return capabilityregistry.ActionKindNone
	}
}

func (s integrationsConnectService) integrationMessages(ctx context.Context, force bool) []*commonv1.Integration {
	return messagesFromStates(s.server.integrationSnapshot(ctx, force).States)
}
func messagesFromStates(states []capabilityregistry.State) []*commonv1.Integration {
	out := make([]*commonv1.Integration, 0, len(states))
	for _, state := range states {
		out = append(out, messageFromState(state))
	}
	return out
}
func messageFromState(state capabilityregistry.State) *commonv1.Integration {
	features := make([]*commonv1.Feature, 0, len(state.Features))
	for _, id := range state.Features {
		requirement := state.FeatureRequirements[id]
		features = append(features, &commonv1.Feature{Id: id, ContractVersion: requirement.ContractVersion, ExpectedUnit: requirement.ExpectedUnit})
	}
	return &commonv1.Integration{Id: state.ID, Origin: state.Origin, Name: state.Name, Description: state.Description, DependencyKind: dependencyKind(state.DependencyKind), DependencySlug: state.DependencySlug, Criticality: criticality(state.ResolvedCriticality()), Platform: platformVerdict(state.Platform), Enabled: state.Enabled, Required: state.Required, StartupPolicy: state.StartupPolicy, Features: features, Lifecycle: &commonv1.LifecycleState{Status: lifecycleStatus(state.Status), ReasonCode: state.ReasonCode, Message: state.Message, CheckedAt: state.CheckedAt}, Action: &commonv1.ActionPolicy{Kind: actionKind(state.ActionKind), Label: state.ActionLabel, Eligible: state.ActionKind != "", RequiresConfirmation: state.ActionKind != "", OwnerGuidance: state.OperatorCommand}}
}

func criticality(v capabilityregistry.Criticality) commonv1.Criticality {
	if v == capabilityregistry.CriticalityRequired {
		return commonv1.Criticality_CRITICALITY_REQUIRED
	}
	return commonv1.Criticality_CRITICALITY_OPTIONAL
}

func platformVerdict(v capabilityregistry.PlatformVerdict) *commonv1.PlatformVerdict {
	if v.Support == "" && v.Reason == "" {
		return nil
	}
	result := &commonv1.PlatformVerdict{Reason: v.Reason}
	switch v.Support {
	case capabilityregistry.PlatformSupported:
		result.Support = commonv1.PlatformSupport_PLATFORM_SUPPORT_SUPPORTED
	case capabilityregistry.PlatformDegraded:
		result.Support = commonv1.PlatformSupport_PLATFORM_SUPPORT_DEGRADED
	case capabilityregistry.PlatformUnsupported:
		result.Support = commonv1.PlatformSupport_PLATFORM_SUPPORT_UNSUPPORTED
	}
	return result
}
func dependencyKind(v capabilityregistry.DependencyKind) commonv1.DependencyKind {
	switch v {
	case capabilityregistry.DependencyScenario:
		return commonv1.DependencyKind_DEPENDENCY_KIND_SCENARIO
	case capabilityregistry.DependencyResource:
		return commonv1.DependencyKind_DEPENDENCY_KIND_RESOURCE
	case capabilityregistry.DependencyControlPlane:
		return commonv1.DependencyKind_DEPENDENCY_KIND_CONTROL_PLANE
	}
	return commonv1.DependencyKind_DEPENDENCY_KIND_UNSPECIFIED
}
func lifecycleStatus(v capabilityregistry.Status) commonv1.LifecycleStatus {
	switch v {
	case capabilityregistry.StatusAvailable:
		return commonv1.LifecycleStatus_LIFECYCLE_STATUS_AVAILABLE
	case capabilityregistry.StatusUnavailable:
		return commonv1.LifecycleStatus_LIFECYCLE_STATUS_UNAVAILABLE
	case capabilityregistry.StatusUnknown:
		return commonv1.LifecycleStatus_LIFECYCLE_STATUS_UNKNOWN
	}
	return commonv1.LifecycleStatus_LIFECYCLE_STATUS_UNSPECIFIED
}
func actionKind(v capabilityregistry.ActionKind) commonv1.ActionKind {
	switch v {
	case capabilityregistry.ActionKindOwnerGuidance:
		return commonv1.ActionKind_ACTION_KIND_OWNER_GUIDANCE
	case capabilityregistry.ActionKindScenarioStart:
		return commonv1.ActionKind_ACTION_KIND_SCENARIO_START
	case capabilityregistry.ActionKindScenarioRestart:
		return commonv1.ActionKind_ACTION_KIND_SCENARIO_RESTART
	case capabilityregistry.ActionKindOperatorCommand:
		return commonv1.ActionKind_ACTION_KIND_OPERATOR_COMMAND
	}
	return commonv1.ActionKind_ACTION_KIND_UNSPECIFIED
}
