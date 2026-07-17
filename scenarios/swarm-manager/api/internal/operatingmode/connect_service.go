package operatingmode

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/initiativelock"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// ConnectService implements the typed OperatingModeService Connect contract over
// the *Service chokepoint — the sole transport for the operating-mode subsystem.
// The generated client is what lets the UI and CLI consume one wire contract; the
// bespoke gorilla/mux JSON surface it replaced has been deleted.
type ConnectService struct {
	svc *Service
}

// NewConnectService builds the Connect OperatingModeService over an existing
// *Service.
func NewConnectService(svc *Service) *ConnectService { return &ConnectService{svc: svc} }

// RegisterConnectService mounts the OperatingModeService Connect handler on the
// router — the operating-mode subsystem's only transport.
func RegisterConnectService(router *mux.Router, svc *Service) {
	path, handler := apiconnect.NewOperatingModeServiceHandler(NewConnectService(svc))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

var _ apiconnect.OperatingModeServiceHandler = (*ConnectService)(nil)

// connectError maps the operating-mode service's typed errors to Connect codes:
// lock/round conflicts to Aborted/FailedPrecondition, unknown selectors to
// NotFound, validation failures to InvalidArgument, and unavailable dependencies
// to Unavailable.
func connectError(err error) *connect.Error {
	var conflict *initiativelock.Conflict
	var activeConflict *ActiveItemExecutionsConflict
	var activeRoundConflict *ActiveOperatingModeRoundConflict
	switch {
	case errors.Is(err, ErrApplyModeNotImplemented):
		return connect.NewError(connect.CodeUnimplemented, err)
	case errors.As(err, &activeConflict):
		return activeItemExecutionsConflictError(activeConflict)
	case errors.As(err, &activeRoundConflict):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.As(err, &conflict):
		return connect.NewError(connect.CodeAborted, err)
	case errors.Is(err, ErrRoundNotFound), strings.Contains(err.Error(), "not found"):
		return connect.NewError(connect.CodeNotFound, err)
	case strings.Contains(err.Error(), "requires"), strings.Contains(err.Error(), "unknown operating mode"),
		strings.Contains(err.Error(), "does not define phase"), strings.Contains(err.Error(), "member-item"),
		strings.Contains(err.Error(), "run_id"), strings.Contains(err.Error(), "item_refs"),
		strings.Contains(err.Error(), "must be kind/name"), strings.Contains(err.Error(), "is not a member"),
		strings.Contains(err.Error(), "does not allow"), strings.Contains(err.Error(), "no backlog_sync"),
		strings.Contains(err.Error(), "proposal"), strings.Contains(err.Error(), "mode is required"),
		strings.Contains(err.Error(), "round actions require"), strings.Contains(err.Error(), "out of range"),
		strings.Contains(err.Error(), "blank"):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, agentmanager.ErrNotAvailable):
		return connect.NewError(connect.CodeUnavailable, errors.New("agent-manager is not available"))
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// activeItemExecutionsConflictError builds a FailedPrecondition error carrying
// the structured conflict detail the UI reads to list the affected item
// executions before re-submitting the switch with cancellation — the Connect
// equivalent of the REST handler's `active_item_executions` conflict body.
func activeItemExecutionsConflictError(c *ActiveItemExecutionsConflict) *connect.Error {
	cerr := connect.NewError(connect.CodeFailedPrecondition, c)
	detail, derr := connect.NewErrorDetail(&apipb.OperatingModeActiveItemExecutionsConflict{
		InitiativeName: c.InitiativeName,
		FromMode:       c.FromMode,
		ToMode:         c.ToMode,
		Executions:     activeExecutionsToProto(c.Executions),
	})
	if derr == nil {
		cerr.AddDetail(detail)
	}
	return cerr
}

func invalidArg(msg string) *connect.Error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(msg))
}

func notFoundMode(raw string) *connect.Error {
	return connect.NewError(connect.CodeNotFound, errors.New("unknown operating mode "+raw))
}

// requireMode trims, validates, and normalizes a mode selector the way the REST
// handlers do.
func requireMode(raw string) (Mode, *connect.Error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalidArg("mode is required")
	}
	if !ValidateMode(trimmed) {
		return "", notFoundMode(trimmed)
	}
	return NormalizeMode(trimmed), nil
}

func (s *ConnectService) Catalog(_ context.Context, _ *connect.Request[apipb.OperatingModeCatalogRequest]) (*connect.Response[apipb.OperatingModeCatalogResponse], error) {
	catalog, err := s.svc.Catalog()
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(catalogToProto(catalog)), nil
}

func (s *ConnectService) GetMode(_ context.Context, req *connect.Request[apipb.OperatingModeGetRequest]) (*connect.Response[apipb.OperatingModeDetailResponse], error) {
	mode, cerr := requireMode(req.Msg.GetMode())
	if cerr != nil {
		return nil, cerr
	}
	detail, err := s.svc.GetMode(mode)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(modeDetailToProto(detail)), nil
}

func (s *ConnectService) UpdateMode(_ context.Context, req *connect.Request[apipb.OperatingModeUpdateRequest]) (*connect.Response[apipb.OperatingModeDetailResponse], error) {
	mode, cerr := requireMode(req.Msg.GetMode())
	if cerr != nil {
		return nil, cerr
	}
	var override Override
	if req.Msg.Label != nil {
		v := req.Msg.GetLabel()
		override.Label = &v
	}
	if req.Msg.Description != nil {
		v := req.Msg.GetDescription()
		override.Description = &v
	}
	if !override.HasChanges() {
		return nil, invalidArg("at least one of label or description is required")
	}
	detail, err := s.svc.UpdateMode(mode, override)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(modeDetailToProto(detail)), nil
}

func (s *ConnectService) SimulateMode(ctx context.Context, req *connect.Request[apipb.OperatingModeSimulateRequest]) (*connect.Response[apipb.OperatingModeSimulationResponse], error) {
	preset := strings.TrimSpace(req.Msg.GetPreset())
	if req.Msg.GetDraft() {
		// Draft simulation targets an on-disk mode that may not be registered yet
		// (e.g. one just scaffolded), so it does not go through the registry-backed
		// requireMode validation.
		raw := strings.TrimSpace(req.Msg.GetMode())
		if raw == "" {
			return nil, invalidArg("mode is required")
		}
		result, err := s.svc.SimulateModeDraft(ctx, raw, preset)
		if err != nil {
			return nil, connectError(err)
		}
		return connect.NewResponse(simulationToProto(result)), nil
	}
	mode, cerr := requireMode(req.Msg.GetMode())
	if cerr != nil {
		return nil, cerr
	}
	result, err := s.svc.SimulateMode(ctx, mode, preset)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(simulationToProto(result)), nil
}

func (s *ConnectService) ScaffoldMode(_ context.Context, req *connect.Request[apipb.OperatingModeScaffoldRequest]) (*connect.Response[apipb.OperatingModeScaffoldResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, invalidArg("mode id is required")
	}
	result, err := s.svc.ScaffoldMode(ScaffoldRequest{
		ID:          id,
		Label:       req.Msg.GetLabel(),
		Description: req.Msg.GetDescription(),
		StartFrom:   req.Msg.GetStartFrom(),
		Force:       req.Msg.GetForce(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(scaffoldResultToProto(result)), nil
}

func (s *ConnectService) ValidateMode(_ context.Context, req *connect.Request[apipb.OperatingModeValidateRequest]) (*connect.Response[apipb.OperatingModeValidateResponse], error) {
	id := strings.TrimSpace(req.Msg.GetMode())
	if id == "" {
		return nil, invalidArg("mode is required")
	}
	report, err := s.svc.ValidateModeDraft(id)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(validationReportToProto(report)), nil
}

func (s *ConnectService) RenderSimulationPrompt(ctx context.Context, req *connect.Request[apipb.OperatingModeRenderSimulationRequest]) (*connect.Response[apipb.OperatingModeRenderPromptResponse], error) {
	mode, cerr := requireMode(req.Msg.GetMode())
	if cerr != nil {
		return nil, cerr
	}
	result, err := s.svc.RenderSimulationPrompt(ctx, mode, strings.TrimSpace(req.Msg.GetPreset()), int(req.Msg.GetStepIndex()))
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(renderPromptToProto(result)), nil
}

func (s *ConnectService) GetWorkspace(ctx context.Context, req *connect.Request[apipb.OperatingModeWorkspaceRequest]) (*connect.Response[apipb.OperatingModeWorkspace], error) {
	name := strings.TrimSpace(req.Msg.GetInitiativeName())
	if name == "" {
		return nil, invalidArg("initiative name is required")
	}
	workspace, err := s.svc.Workspace(ctx, name)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(workspaceToProto(workspace)), nil
}

func (s *ConnectService) SwitchMode(ctx context.Context, req *connect.Request[apipb.OperatingModeSwitchRequest]) (*connect.Response[apipb.OperatingModeSwitchResult], error) {
	name := strings.TrimSpace(req.Msg.GetInitiativeName())
	if name == "" || strings.TrimSpace(req.Msg.GetMode()) == "" {
		return nil, invalidArg("initiative name and mode are required")
	}
	result, err := s.svc.SwitchMode(ctx, SwitchModeRequest{
		InitiativeName:             name,
		Mode:                       req.Msg.GetMode(),
		CancelActiveItemExecutions: req.Msg.GetCancelActiveItemExecutions(),
		RequestedBy:                req.Msg.GetRequestedBy(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(switchResultToProto(result)), nil
}

func (s *ConnectService) StartPhase(ctx context.Context, req *connect.Request[apipb.OperatingModeStartPhaseRequest]) (*connect.Response[apipb.OperatingModeRoundEnvelope], error) {
	name := strings.TrimSpace(req.Msg.GetInitiativeName())
	phase := strings.TrimSpace(req.Msg.GetPhase())
	if name == "" || phase == "" {
		return nil, invalidArg("initiative name and phase are required")
	}
	round, err := s.svc.StartPhase(ctx, StartPhaseRequest{
		InitiativeName: name,
		Phase:          phase,
		Note:           req.Msg.GetNote(),
		Inputs:         mapFromStruct(req.Msg.GetInputs()),
		Override:       req.Msg.GetOverride(),
		RequestedBy:    req.Msg.GetRequestedBy(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(roundEnvelopeToProto(round)), nil
}

func (s *ConnectService) StartTargetPhase(ctx context.Context, req *connect.Request[apipb.OperatingModeStartTargetPhaseRequest]) (*connect.Response[apipb.OperatingModeRoundEnvelope], error) {
	mode := strings.TrimSpace(req.Msg.GetMode())
	targetRef := strings.TrimSpace(req.Msg.GetTargetRef())
	if mode == "" || targetRef == "" {
		return nil, invalidArg("mode and target_ref are required")
	}
	round, err := s.svc.StartTargetPhase(ctx, StartTargetPhaseRequest{
		Mode:        mode,
		TargetRef:   targetRef,
		Phase:       req.Msg.GetPhase(),
		Note:        req.Msg.GetNote(),
		Inputs:      mapFromStruct(req.Msg.GetInputs()),
		Override:    req.Msg.GetOverride(),
		RequestedBy: req.Msg.GetRequestedBy(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(roundEnvelopeToProto(round)), nil
}

func (s *ConnectService) RenderLivePrompt(ctx context.Context, req *connect.Request[apipb.OperatingModeRenderLiveRequest]) (*connect.Response[apipb.OperatingModeRenderPromptResponse], error) {
	name := strings.TrimSpace(req.Msg.GetInitiativeName())
	phase := strings.TrimSpace(req.Msg.GetPhase())
	if name == "" || phase == "" {
		return nil, invalidArg("initiative name and phase are required")
	}
	result, err := s.svc.RenderLivePrompt(ctx, name, phase, int(req.Msg.GetRound()), req.Msg.GetNote())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(renderPromptToProto(result)), nil
}

func (s *ConnectService) RefreshRound(ctx context.Context, req *connect.Request[apipb.OperatingModeRoundActionRequest]) (*connect.Response[apipb.OperatingModeRoundEnvelope], error) {
	name, mode, cerr := s.roundAction(req.Msg.GetInitiativeName(), req.Msg.GetMode(), req.Msg.GetRound())
	if cerr != nil {
		return nil, cerr
	}
	round, err := s.svc.RefreshRound(ctx, name, mode, int(req.Msg.GetRound()))
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(roundEnvelopeToProto(round)), nil
}

func (s *ConnectService) CancelRound(ctx context.Context, req *connect.Request[apipb.OperatingModeRoundActionRequest]) (*connect.Response[apipb.OperatingModeRoundEnvelope], error) {
	name, mode, cerr := s.roundAction(req.Msg.GetInitiativeName(), req.Msg.GetMode(), req.Msg.GetRound())
	if cerr != nil {
		return nil, cerr
	}
	round, err := s.svc.CancelRound(ctx, name, mode, int(req.Msg.GetRound()))
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(roundEnvelopeToProto(round)), nil
}

func (s *ConnectService) CompleteItems(ctx context.Context, req *connect.Request[apipb.OperatingModeCompleteItemsRequest]) (*connect.Response[apipb.OperatingModeBacklogSyncResult], error) {
	name, roundNumber, cerr := requireRound(req.Msg.GetInitiativeName(), req.Msg.GetRound())
	if cerr != nil {
		return nil, cerr
	}
	if strings.TrimSpace(req.Msg.GetRunId()) == "" {
		return nil, invalidArg("run_id is required")
	}
	mode, err := s.svc.ResolveRoundActionMode(name, req.Msg.GetMode())
	if err != nil {
		return nil, connectError(err)
	}
	result, err := s.svc.CompleteItems(ctx, CompleteItemsRequest{
		InitiativeName: name,
		Mode:           string(mode),
		Round:          roundNumber,
		RunID:          req.Msg.GetRunId(),
		ItemRefs:       req.Msg.GetItemRefs(),
		RequestedBy:    req.Msg.GetRequestedBy(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(backlogSyncResultToProto(result)), nil
}

func (s *ConnectService) ApplyBacklogSync(ctx context.Context, req *connect.Request[apipb.OperatingModeApplyBacklogSyncRequest]) (*connect.Response[apipb.OperatingModeBacklogSyncResult], error) {
	name, roundNumber, cerr := requireRound(req.Msg.GetInitiativeName(), req.Msg.GetRound())
	if cerr != nil {
		return nil, cerr
	}
	if strings.TrimSpace(req.Msg.GetRunId()) == "" {
		return nil, invalidArg("run_id is required")
	}
	mode, err := s.svc.ResolveRoundActionMode(name, req.Msg.GetMode())
	if err != nil {
		return nil, connectError(err)
	}
	result, err := s.svc.ApplyBacklogSync(ctx, ApplyBacklogSyncRequest{
		InitiativeName:      name,
		Mode:                string(mode),
		Round:               roundNumber,
		RunID:               req.Msg.GetRunId(),
		AcceptedMutationIDs: req.Msg.GetAcceptedMutationIds(),
		RequestedBy:         req.Msg.GetRequestedBy(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(backlogSyncResultToProto(result)), nil
}

// requireRound validates the initiative + round selector shared by the round
// action RPCs.
func requireRound(initiativeName string, round int32) (string, int, *connect.Error) {
	name := strings.TrimSpace(initiativeName)
	if name == "" {
		return "", 0, invalidArg("initiative name is required")
	}
	if round <= 0 {
		return "", 0, invalidArg("round must be a positive integer")
	}
	return name, int(round), nil
}

// roundAction resolves the initiative + mode for refresh/cancel, letting the
// service derive the mode from the initiative when the request omits it.
func (s *ConnectService) roundAction(initiativeName, rawMode string, round int32) (string, Mode, *connect.Error) {
	name, _, cerr := requireRound(initiativeName, round)
	if cerr != nil {
		return "", "", cerr
	}
	mode, err := s.svc.ResolveRoundActionMode(name, rawMode)
	if err != nil {
		return "", "", connectError(err)
	}
	return name, mode, nil
}
