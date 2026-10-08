package nutrition

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/nutrition"
	"nutrition-planner/internal/decimalx"
	internal "nutrition-planner/internal/nutrition"
	workspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	repo    internal.TargetRepository
	intakes internal.IntakeRepository
	ws      workspace.Service
	logger  *log.Logger
}

func NewConnectHandler(repo internal.TargetRepository, intakes internal.IntakeRepository, ws workspace.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{repo: repo, intakes: intakes, ws: ws, logger: logger}
}

func (h *connectHandler) scope(ctx context.Context, id string) error {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if id == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.ws.Get(ctx, id, p.Subject); err != nil {
		var nf workspace.ErrNotFound
		if errors.As(err, &nf) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		var f workspace.ErrForbidden
		if errors.As(err, &f) {
			return connect.NewError(connect.CodePermissionDenied, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

func (h *connectHandler) ListTargets(ctx context.Context, req *connect.Request[v1.ListTargetsRequest]) (*connect.Response[v1.ListTargetsResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	items, err := h.repo.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListTargetsResponse{Targets: make([]*v1.Target, 0, len(items))}
	for _, item := range items {
		out.Targets = append(out.Targets, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateTarget(ctx context.Context, req *connect.Request[v1.CreateTargetRequest]) (*connect.Response[v1.CreateTargetResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	t, err := fromProto(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	created, err := h.repo.Create(ctx, t)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.CreateTargetResponse{Target: toProto(created)}), nil
}

func (h *connectHandler) GetTarget(ctx context.Context, req *connect.Request[v1.GetTargetRequest]) (*connect.Response[v1.GetTargetResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	t, err := h.repo.Get(ctx, req.Msg.Id, req.Msg.WorkspaceId, req.Msg.Revision)
	if err != nil {
		var nf internal.TargetNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetTargetResponse{Target: toProto(t)}), nil
}

func (h *connectHandler) EvaluateScope(ctx context.Context, req *connect.Request[v1.EvaluateScopeRequest]) (*connect.Response[v1.EvaluateScopeResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	if req.Msg.TargetId == "" || req.Msg.NutrientId == "" || req.Msg.Scope == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target_id, nutrient_id, and scope are required"))
	}
	target, err := h.repo.Get(ctx, req.Msg.TargetId, req.Msg.WorkspaceId, req.Msg.TargetRevision)
	if err != nil {
		var nf internal.TargetNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	intakes := make([]internal.Intake, 0, len(req.Msg.Intakes))
	for _, item := range req.Msg.Intakes {
		if item == nil {
			continue
		}
		planned, parseErr := parseOptionalDecimal(item.Planned)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("planned: invalid decimal"))
		}
		actual, parseErr := parseOptionalDecimal(item.Actual)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("actual: invalid decimal"))
		}
		intakes = append(intakes, internal.Intake{Date: item.Date, NutrientID: item.NutrientId, Planned: planned, Actual: actual, Recorded: item.Recorded, Past: item.Past})
	}
	result, err := internal.AggregateScope(req.Msg.NutrientId, req.Msg.Scope, intakes)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	evaluation := internal.Evaluate(result, target)
	return connect.NewResponse(&v1.EvaluateScopeResponse{NutrientId: result.NutrientID, Known: result.Known.String(), Complete: result.Complete, Unresolved: result.Unresolved, Status: string(evaluation.Status), Reason: evaluation.Reason}), nil
}

func (h *connectHandler) ListIntakes(ctx context.Context, req *connect.Request[v1.ListIntakesRequest]) (*connect.Response[v1.ListIntakesResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	if h.intakes == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("intake storage is unavailable"))
	}
	items, err := h.intakes.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListIntakesResponse{Events: make([]*v1.IntakeEvent, 0, len(items))}
	for _, item := range items {
		out.Events = append(out.Events, intakeToProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RecordIntake(ctx context.Context, req *connect.Request[v1.RecordIntakeRequest]) (*connect.Response[v1.RecordIntakeResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	if h.intakes == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("intake storage is unavailable"))
	}
	amount, err := decimalx.Parse(req.Msg.Amount)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("amount: invalid decimal"))
	}
	event := internal.IntakeEvent{ID: req.Msg.Id, WorkspaceID: req.Msg.WorkspaceId, Date: req.Msg.Date, RecipeID: req.Msg.RecipeId, RecipeRevision: req.Msg.RecipeRevision, NutrientID: req.Msg.NutrientId, Amount: amount, Unit: req.Msg.Unit, Reason: req.Msg.Reason, CorrectionOf: req.Msg.CorrectionOf}
	if err := h.intakes.Append(ctx, req.Msg.WorkspaceId, event); err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	return connect.NewResponse(&v1.RecordIntakeResponse{Event: intakeToProto(event)}), nil
}

func intakeToProto(event internal.IntakeEvent) *v1.IntakeEvent {
	return &v1.IntakeEvent{Id: event.ID, Date: event.Date, RecipeId: event.RecipeID, RecipeRevision: event.RecipeRevision, NutrientId: event.NutrientID, Amount: event.Amount.String(), Unit: event.Unit, Reason: event.Reason, CorrectionOf: event.CorrectionOf, RecordedAt: event.RecordedAt().Format(time.RFC3339Nano)}
}

func parseOptionalDecimal(value string) (decimalx.Decimal, error) {
	if value == "" {
		return decimalx.Unknown, nil
	}
	return decimalx.Parse(value)
}

func fromProto(req *v1.CreateTargetRequest) (internal.Target, error) {
	lower, err := decimalx.Parse(req.Lower)
	if err != nil {
		return internal.Target{}, errors.New("lower: invalid decimal")
	}
	upper, err := decimalx.Parse(req.Upper)
	if err != nil {
		return internal.Target{}, errors.New("upper: invalid decimal")
	}
	from, err := time.Parse(time.RFC3339Nano, req.EffectiveFrom)
	if err != nil {
		return internal.Target{}, errors.New("effective_from: invalid timestamp")
	}
	var to *time.Time
	if req.EffectiveTo != "" {
		v, e := time.Parse(time.RFC3339Nano, req.EffectiveTo)
		if e != nil {
			return internal.Target{}, errors.New("effective_to: invalid timestamp")
		}
		to = &v
	}
	return internal.Target{WorkspaceID: req.WorkspaceId, NutrientID: req.NutrientId, Lower: lower, Upper: upper, Period: req.Period, Scope: req.Scope, Enforcement: req.Enforcement, Provenance: req.Provenance, EffectiveFrom: from, EffectiveTo: to, Active: req.Active}, nil
}

func toProto(t internal.Target) *v1.Target {
	out := &v1.Target{Id: t.ID, Revision: t.Revision, NutrientId: t.NutrientID, Lower: t.Lower.String(), Upper: t.Upper.String(), Period: t.Period, Scope: t.Scope, Enforcement: t.Enforcement, Provenance: t.Provenance, EffectiveFrom: t.EffectiveFrom.Format(time.RFC3339Nano), Active: t.Active}
	if t.EffectiveTo != nil {
		out.EffectiveTo = t.EffectiveTo.Format(time.RFC3339Nano)
	}
	return out
}
