package capabilities

import (
	"context"
	"errors"
	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"log"

	"connectrpc.com/connect"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities/capabilities_v1connect"

	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities"
)

// Action failures carry their class so the browser can tell a missing
// machine or an unreachable control plane from a malformed request. Every
// action error used to be invalid_argument, which named the caller as the
// cause of failures it could not have avoided.
var (
	ErrTargetNotFound     = errors.New("target not found")
	ErrUnavailable        = errors.New("unavailable")
	ErrFailedPrecondition = errors.New("failed precondition")
	ErrPermissionDenied   = errors.New("permission denied")
)

// runActionCode keeps a control-plane status code when the failure came from
// one, and otherwise classifies by the sentinel the adapter wrapped.
func runActionCode(err error) connect.Code {
	var remote *connect.Error
	switch {
	case errors.As(err, &remote):
		return remote.Code()
	case errors.Is(err, ErrTargetNotFound):
		return connect.CodeNotFound
	case errors.Is(err, ErrUnavailable):
		return connect.CodeUnavailable
	case errors.Is(err, ErrFailedPrecondition):
		return connect.CodeFailedPrecondition
	case errors.Is(err, ErrPermissionDenied):
		return connect.CodePermissionDenied
	case errors.Is(err, context.DeadlineExceeded):
		return connect.CodeDeadlineExceeded
	case errors.Is(err, context.Canceled):
		return connect.CodeCanceled
	default:
		return connect.CodeInvalidArgument
	}
}

// Deps wires the seams the Connect capabilities handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	capabilitiesconnect.UnimplementedCapabilitiesServiceHandler
	deps Deps
}

type agentUpdateReader interface {
	AgentUpdate(context.Context, string, string) (AgentUpdateResult, error)
}

type agentUpdateActionRunner interface {
	RunAgentUpdateAction(ctx context.Context, targetID, agentID, action, cleanupPlanID string) (AgentUpdateActionResult, error)
}

type agentUsageReader interface {
	AgentUsage(context.Context, string, string) (AgentUsageResult, error)
}

type harnessLister interface {
	ListHarnesses(context.Context, string) ([]HarnessDescriptor, error)
}

func (h *connectHandler) ListHarnesses(ctx context.Context, req *connect.Request[capabilitiesv1.ListHarnessesRequest]) (*connect.Response[capabilitiesv1.ListHarnessesResponse], error) {
	lister, ok := h.deps.Service.(harnessLister)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, ErrUnavailable)
	}
	descriptors, err := lister.ListHarnesses(ctx, req.Msg.GetTargetId())
	if err != nil {
		return nil, connect.NewError(runActionCode(err), err)
	}
	response := &capabilitiesv1.ListHarnessesResponse{TargetId: req.Msg.GetTargetId()}
	for _, d := range descriptors {
		item := &capabilitiesv1.HarnessDescriptor{
			AgentId: d.AgentID, Resource: d.Resource, DisplayName: d.DisplayName, Icon: d.Icon,
			SupportsContextLimit: d.SupportsContextLimit, SupportsUsageMode: d.SupportsUsageMode,
			SupportsModelSelection: d.SupportsModelSelection, SupportsProviderSelection: d.SupportsProviderSelection,
		}
		for _, w := range d.UsageWindows {
			item.UsageWindows = append(item.UsageWindows, &capabilitiesv1.HarnessUsageWindow{Id: w.ID, WindowMinutes: w.WindowMinutes})
		}
		response.Harnesses = append(response.Harnesses, item)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) AgentUsage(ctx context.Context, req *connect.Request[capabilitiesv1.AgentUsageRequest]) (*connect.Response[capabilitiesv1.AgentUsageResponse], error) {
	reader, ok := h.deps.Service.(agentUsageReader)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, ErrUnavailable)
	}
	result, err := reader.AgentUsage(ctx, req.Msg.GetTargetId(), req.Msg.GetAgentId())
	if err != nil {
		return nil, connect.NewError(runActionCode(err), err)
	}
	response := &capabilitiesv1.AgentUsageResponse{TargetId: result.TargetID, AgentId: result.AgentID, Match: result.Match, Reason: result.Reason, Source: result.Source, ObservedAt: result.ObservedAt}
	for _, w := range result.Windows {
		response.Windows = append(response.Windows, &capabilitiesv1.AgentUsageWindow{Pool: w.Pool, WindowMinutes: w.WindowMinutes, UsedPercent: w.UsedPercent, ResetsAt: w.ResetsAt, ObservedAt: w.ObservedAt})
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) AgentUpdate(ctx context.Context, req *connect.Request[capabilitiesv1.AgentUpdateRequest]) (*connect.Response[capabilitiesv1.AgentUpdateResponse], error) {
	reader, ok := h.deps.Service.(agentUpdateReader)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, ErrUnavailable)
	}
	result, err := reader.AgentUpdate(ctx, req.Msg.GetTargetId(), req.Msg.GetAgentId())
	if err != nil {
		return nil, connect.NewError(runActionCode(err), err)
	}
	return connect.NewResponse(agentUpdateToProto(result)), nil
}

func agentUpdateToProto(result AgentUpdateResult) *capabilitiesv1.AgentUpdateResponse {
	checks := make([]*capabilitiesv1.AgentUpdateCheck, 0, len(result.Checks))
	for _, check := range result.Checks {
		checks = append(checks, &capabilitiesv1.AgentUpdateCheck{Id: check.ID, Passed: check.Passed, Message: check.Message})
	}
	return &capabilitiesv1.AgentUpdateResponse{
		TargetId: result.TargetID, AgentId: result.AgentID, Resource: result.Resource,
		Installed: result.Installed, Pinned: result.Pinned, Latest: result.Latest, Status: result.Status,
		CheckedAt: result.CheckedAt, SourceKind: result.SourceKind, SourceId: result.SourceID,
		ProbeVerified: result.ProbeVerified, Note: result.Note,
		State: result.State, PrimaryAction: result.PrimaryAction, OperatorCommand: result.OperatorCommand, Reason: result.Reason,
		Checks: checks, InstallKind: result.InstallKind, RollbackVersion: result.RollbackVersion, NewerThanReviewed: result.NewerThanReviewed,
		InstallState: result.InstallState, SignInState: result.SignInState, UpdateState: result.UpdateState,
	}
}

func (h *connectHandler) RunAgentUpdateAction(ctx context.Context, req *connect.Request[capabilitiesv1.RunAgentUpdateActionRequest]) (*connect.Response[capabilitiesv1.RunAgentUpdateActionResponse], error) {
	runner, ok := h.deps.Service.(agentUpdateActionRunner)
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, ErrUnavailable)
	}
	result, err := runner.RunAgentUpdateAction(ctx, req.Msg.GetTargetId(), req.Msg.GetAgentId(), req.Msg.GetAction(), req.Msg.GetCleanupPlanId())
	if err != nil {
		return nil, connect.NewError(runActionCode(err), err)
	}
	return connect.NewResponse(&capabilitiesv1.RunAgentUpdateActionResponse{
		Success: result.Success, Message: result.Message, OperationId: result.OperationID, Update: agentUpdateToProto(result.Update), CleanupPlan: cleanupPlanToProto(result.CleanupPlan),
	}), nil
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// CapabilitiesServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Get(ctx context.Context, _ *connect.Request[capabilitiesv1.GetRequest]) (*connect.Response[capabilitiesv1.GetResponse], error) {
	snap := h.deps.Service.Resolve(ctx)
	resp := &capabilitiesv1.GetResponse{
		Capabilities:    statesToProto(snap.Capabilities),
		Timestamp:       snap.Timestamp,
		SessionBackends: backendsToProto(snap.BackendOptions),
		DefaultBackend:  snap.DefaultBackend,
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) Liveness(ctx context.Context, _ *connect.Request[capabilitiesv1.LivenessRequest]) (*connect.Response[capabilitiesv1.LivenessResponse], error) {
	snap := h.deps.Service.Liveness(ctx)
	resp := &capabilitiesv1.LivenessResponse{
		Capabilities: statesToProto(snap.Capabilities),
		Timestamp:    snap.Timestamp,
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) RunAction(ctx context.Context, req *connect.Request[capabilitiesv1.RunActionRequest]) (*connect.Response[capabilitiesv1.RunActionResponse], error) {
	result, err := h.deps.Service.RunAction(ctx, ActionRequest{
		CapabilityID: req.Msg.GetCapabilityId(),
		ActionKind:   req.Msg.GetActionKind(),
		TargetID:     req.Msg.GetTargetId(),
	})
	if err != nil {
		return nil, connect.NewError(runActionCode(err), err)
	}
	resp := &capabilitiesv1.RunActionResponse{
		Success:      result.Success,
		Status:       result.Status,
		Message:      result.Message,
		CapabilityId: result.CapabilityID,
		ActionKind:   result.ActionKind,
		OperationId:  result.OperationID,
		Capabilities: statesToProto(result.Snapshot.Capabilities),
		Timestamp:    result.Snapshot.Timestamp,
	}
	return connect.NewResponse(resp), nil
}

func statesToProto(in []CapabilityState) []*capabilitiesv1.CapabilityState {
	out := make([]*capabilitiesv1.CapabilityState, len(in))
	for i, s := range in {
		out[i] = &capabilitiesv1.CapabilityState{
			Id:                     s.ID,
			Name:                   s.Name,
			Description:            s.Description,
			DependencyKind:         s.DependencyKind,
			DependencySlug:         s.DependencySlug,
			Features:               s.Features,
			Status:                 s.Status,
			Message:                s.Message,
			CheckedAt:              s.CheckedAt,
			ReasonCode:             s.ReasonCode,
			ActionKind:             s.ActionKind,
			ActionLabel:            s.ActionLabel,
			OperatorCommand:        s.OperatorCommand,
			FeatureStatus:          s.FeatureStatus,
			FeatureReason:          s.FeatureReason,
			FeatureOperatorCommand: s.FeatureOperatorCommand,
			ProviderStatus:         s.ProviderStatus,
			ProviderFeatures:       s.ProviderFeatures,
			Uses:                   integrationUses(s.Uses),
			Installation:           localInstallation(s.Installation),
		}
	}
	return out
}

func backendsToProto(in []BackendOption) []*capabilitiesv1.BackendOption {
	out := make([]*capabilitiesv1.BackendOption, len(in))
	for i, b := range in {
		out[i] = &capabilitiesv1.BackendOption{
			Id:              b.ID,
			DisplayName:     b.DisplayName,
			Description:     b.Description,
			SurvivesRestart: b.SurvivesRestart,
			Available:       b.Available,
			Reason:          b.Reason,
		}
	}
	return out
}

func cleanupPlanToProto(plan *HarnessCleanupPlan) *capabilitiesv1.HarnessCleanupPlan {
	if plan == nil {
		return nil
	}
	return &capabilitiesv1.HarnessCleanupPlan{Id: plan.ID, Remove: plan.Remove, Preserved: plan.Preserved}
}

func integrationUses(uses []capreg.Use) []*commonv1.IntegrationUse {
	out := make([]*commonv1.IntegrationUse, len(uses))
	for i, use := range uses {
		out[i] = &commonv1.IntegrationUse{Id: use.ID, Name: use.Name, Description: use.Description, Group: use.Group}
	}
	return out
}
func localInstallation(installation *capreg.Installation) *commonv1.LocalInstallation {
	if installation == nil {
		return nil
	}
	return &commonv1.LocalInstallation{Location: installation.Location, Version: installation.Version, Selected: installation.Selected}
}
