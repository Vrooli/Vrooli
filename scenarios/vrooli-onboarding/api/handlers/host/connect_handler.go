package host

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/targetmodel"
	hostv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host"
	hostv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host/hostv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/host"
	"google.golang.org/protobuf/types/known/structpb"
)

type connectHandler struct{ service host.Service }

func NewConnectHandler(service host.Service) hostv1connect.HostServiceHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListHostRequirements(ctx context.Context, req *connect.Request[hostv1.ListHostRequirementsRequest]) (*connect.Response[hostv1.ListHostRequirementsResponse], error) {
	tools, safeguards, err := h.service.ListRequirements(ctx, req.Msg.GetTarget())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list host requirements: %w", err))
	}
	return connect.NewResponse(&hostv1.ListHostRequirementsResponse{Tools: requirementsToProto(tools), Safeguards: requirementsToProto(safeguards)}), nil
}

func (h *connectHandler) GetHostFacts(ctx context.Context, req *connect.Request[hostv1.GetHostFactsRequest]) (*connect.Response[hostv1.GetHostFactsResponse], error) {
	facts, err := h.service.Facts(ctx, req.Msg.GetTarget())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get host facts: %w", err))
	}
	response := &hostv1.GetHostFactsResponse{Available: facts.Available, Reason: facts.Reason, Gpus: facts.GPUs, Platform: facts.Platform}
	if facts.CPUCount != nil {
		value := int32(*facts.CPUCount)
		response.CpuCount = &value
	}
	if facts.MemoryTotalBytes != nil {
		response.MemoryTotalBytes = facts.MemoryTotalBytes
	}
	if facts.MemoryAvailableBytes != nil {
		response.MemoryAvailableBytes = facts.MemoryAvailableBytes
	}
	if facts.DiskFreeBytes != nil {
		response.DiskFreeBytes = facts.DiskFreeBytes
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ListTargets(ctx context.Context, _ *connect.Request[hostv1.ListTargetsRequest]) (*connect.Response[hostv1.ListTargetsResponse], error) {
	targets, discoveryError, err := h.service.Targets(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("list targets: %w", err))
	}
	return connect.NewResponse(&hostv1.ListTargetsResponse{Targets: targetsToProto(targets), Error: discoveryError}), nil
}

func (h *connectHandler) PatchHostSafeguardConfig(ctx context.Context, req *connect.Request[hostv1.PatchHostSafeguardConfigRequest]) (*connect.Response[hostv1.PatchHostSafeguardConfigResponse], error) {
	value := any(nil)
	if req.Msg.GetValue() != nil {
		value = req.Msg.GetValue().AsInterface()
	}
	if err := h.service.PatchSafeguardConfig(ctx, req.Msg.GetTarget(), req.Msg.GetSafeguardName(), req.Msg.GetConfigKey(), value); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&hostv1.PatchHostSafeguardConfigResponse{Status: "committed"}), nil
}

func (h *connectHandler) SetNotificationRecipient(ctx context.Context, req *connect.Request[hostv1.SetNotificationRecipientRequest]) (*connect.Response[hostv1.SetNotificationRecipientResponse], error) {
	if err := h.service.SetRecipient(ctx, req.Msg.GetTarget(), req.Msg.GetSubject()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&hostv1.SetNotificationRecipientResponse{Status: "committed"}), nil
}

func requirementsToProto(items []host.Requirement) []*hostv1.HostRequirement {
	result := make([]*hostv1.HostRequirement, 0, len(items))
	for _, item := range items {
		result = append(result, &hostv1.HostRequirement{Name: item.Name, Required: item.Required, Reason: item.Reason, Notes: item.Notes, Description: item.Description, Risk: item.Risk, Privilege: item.Privilege, Bundling: item.Bundling, Platforms: item.Platforms, Commands: item.Commands, ConfigSchema: structValue(item.ConfigSchema), Config: structValue(item.Config), Status: item.Status, Disposition: disposition(item.Disposition), Detail: item.Detail, Remediation: item.Remediation})
	}
	return result
}

func structValue(values map[string]any) *structpb.Struct {
	if len(values) == 0 {
		return nil
	}
	value, err := structpb.NewStruct(values)
	if err != nil {
		return nil
	}
	return value
}

func disposition(value string) hostv1.SafeguardDisposition {
	switch value {
	case "ready":
		return hostv1.SafeguardDisposition_SAFEGUARD_DISPOSITION_READY
	case "missing":
		return hostv1.SafeguardDisposition_SAFEGUARD_DISPOSITION_MISSING
	case "permission_denied":
		return hostv1.SafeguardDisposition_SAFEGUARD_DISPOSITION_PERMISSION_DENIED
	case "content_mismatch":
		return hostv1.SafeguardDisposition_SAFEGUARD_DISPOSITION_CONTENT_MISMATCH
	case "deferred":
		return hostv1.SafeguardDisposition_SAFEGUARD_DISPOSITION_DEFERRED
	case "unsupported":
		return hostv1.SafeguardDisposition_SAFEGUARD_DISPOSITION_UNSUPPORTED
	case "not_applicable":
		return hostv1.SafeguardDisposition_SAFEGUARD_DISPOSITION_NOT_APPLICABLE
	default:
		return hostv1.SafeguardDisposition_SAFEGUARD_DISPOSITION_UNSPECIFIED
	}
}

func targetsToProto(items []targetmodel.Target) []*hostv1.Target {
	result := make([]*hostv1.Target, 0, len(items))
	for _, item := range items {
		checks := make([]*hostv1.ReadinessCheck, 0, len(item.Readiness))
		for _, check := range item.Readiness {
			checks = append(checks, &hostv1.ReadinessCheck{Identity: check.Identity, Label: check.Label, Passed: check.Passed, State: string(check.State), Version: check.Version, Detail: check.Detail, RecoveryAction: check.RecoveryAction})
		}
		result = append(result, &hostv1.Target{Id: item.ID, Name: item.Label, Status: item.Health.Status, Os: item.OS, Architecture: item.Architecture, Kind: item.DeviceKind, Online: item.Transport.Available, Available: item.Available, Reason: item.Reason, NextAction: item.NextAction, Capabilities: item.Capabilities, Scopes: item.Scopes, Readiness: checks})
	}
	return result
}
