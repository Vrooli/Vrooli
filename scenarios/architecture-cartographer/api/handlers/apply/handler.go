// Package apply is the Connect-RPC surface for the apply domain.
// v0.1 supports PlanApply (deterministic), ListApplyHistory (empty),
// GetBuildBaseline (empty). RunApply surfaces CodeUnimplemented per
// the plan; the seam interfaces are in place so v0.2's executor can
// drop in without touching the wire shape.
package apply

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/apply"

	"connectrpc.com/connect"
	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply/apply_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements apply_v1connect.ApplyServiceHandler.
type Handler struct {
	apply_v1connect.UnimplementedApplyServiceHandler
	svc apply.Service
}

// NewHandler constructs the Connect handler.
func NewHandler(svc apply.Service) *Handler { return &Handler{svc: svc} }

var _ apply_v1connect.ApplyServiceHandler = (*Handler)(nil)

func (h *Handler) PlanApply(ctx context.Context, req *connect.Request[applyv1.PlanApplyRequest]) (*connect.Response[applyv1.PlanApplyResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	domain := strings.TrimSpace(req.Msg.GetDomain())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	if domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("domain is required"))
	}
	plan, dry, err := h.svc.PlanApply(ctx, apply.PlanInput{
		Scenario:    scenario,
		Domain:      domain,
		ConflictIDs: append([]string(nil), req.Msg.GetConflictIds()...),
		DryRun:      req.Msg.GetDryRun() || req.Header().Get("X-Dry-Run") == "true",
	})
	if err != nil {
		return nil, connect.NewError(apply.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&applyv1.PlanApplyResponse{
		Plan:   planToProto(plan),
		DryRun: dry,
	}), nil
}

func (h *Handler) RunApply(ctx context.Context, req *connect.Request[applyv1.RunApplyRequest]) (*connect.Response[applyv1.RunApplyResponse], error) {
	_, err := h.svc.RunApply(ctx, req.Msg.GetPlanId(), req.Msg.GetAcknowledgeV01Unimplemented())
	if err != nil {
		return nil, connect.NewError(apply.ErrorToConnectCode(err), err)
	}
	// Unreachable in v0.1; included for symmetry once execution lands.
	return connect.NewResponse(&applyv1.RunApplyResponse{}), nil
}

func (h *Handler) ListApplyHistory(ctx context.Context, req *connect.Request[applyv1.ListApplyHistoryRequest]) (*connect.Response[applyv1.ListApplyHistoryResponse], error) {
	page, err := h.svc.ListApplyHistory(ctx, apply.ListRunsFilter{
		Scenario: strings.TrimSpace(req.Msg.GetScenario()),
		Domain:   strings.TrimSpace(req.Msg.GetDomain()),
		PageSize: int(req.Msg.GetPageSize()),
	})
	if err != nil {
		return nil, connect.NewError(apply.ErrorToConnectCode(err), err)
	}
	out := &applyv1.ListApplyHistoryResponse{NextPageToken: page.NextPageToken}
	for _, r := range page.Runs {
		out.Runs = append(out.Runs, runToProto(r))
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) GetBuildBaseline(ctx context.Context, req *connect.Request[applyv1.GetBuildBaselineRequest]) (*connect.Response[applyv1.GetBuildBaselineResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	bl, err := h.svc.GetBuildBaseline(ctx, scenario)
	if err != nil {
		return nil, connect.NewError(apply.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&applyv1.GetBuildBaselineResponse{Baseline: baselineToProto(bl)}), nil
}

// -------------------------- proto<->domain --------------------------

func planToProto(p apply.Plan) *applyv1.Plan {
	out := &applyv1.Plan{
		Id:       p.ID,
		Scenario: p.Scenario,
		Domain:   p.Domain,
	}
	for _, op := range p.Operations {
		out.Operations = append(out.Operations, &applyv1.Operation{
			Id:       op.ID,
			Kind:     opKindToProto(op.Kind),
			FromPath: op.FromPath,
			ToPath:   op.ToPath,
		})
	}
	return out
}

func runToProto(r apply.ApplyRun) *applyv1.ApplyRun {
	out := &applyv1.ApplyRun{
		Id:       r.ID,
		PlanId:   r.PlanID,
		Scenario: r.Scenario,
		Domain:   r.Domain,
		Status:   statusToProto(r.Status),
	}
	if !r.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(r.StartedAt)
	}
	return out
}

func baselineToProto(b apply.BuildBaseline) *applyv1.BuildBaseline {
	out := &applyv1.BuildBaseline{
		Scenario: b.Scenario,
	}
	if !b.CapturedAt.IsZero() {
		out.CapturedAt = timestamppb.New(b.CapturedAt)
	}
	return out
}

func opKindToProto(k apply.OperationKind) applyv1.OperationKind {
	switch k {
	case apply.OperationKindMoveFile:
		return applyv1.OperationKind_OPERATION_KIND_MOVE_FILE
	case apply.OperationKindRewriteImport:
		return applyv1.OperationKind_OPERATION_KIND_REWRITE_IMPORT
	case apply.OperationKindDeleteFile:
		return applyv1.OperationKind_OPERATION_KIND_DELETE_FILE
	case apply.OperationKindCreateFile:
		return applyv1.OperationKind_OPERATION_KIND_CREATE_FILE
	default:
		return applyv1.OperationKind_OPERATION_KIND_UNSPECIFIED
	}
}

func statusToProto(s apply.ApplyStatus) applyv1.ApplyStatus {
	switch s {
	case apply.ApplyStatusPlanned:
		return applyv1.ApplyStatus_APPLY_STATUS_PLANNED
	case apply.ApplyStatusRunning:
		return applyv1.ApplyStatus_APPLY_STATUS_RUNNING
	case apply.ApplyStatusBuildGreen:
		return applyv1.ApplyStatus_APPLY_STATUS_BUILD_GREEN
	case apply.ApplyStatusBuildRed:
		return applyv1.ApplyStatus_APPLY_STATUS_BUILD_RED
	case apply.ApplyStatusReverted:
		return applyv1.ApplyStatus_APPLY_STATUS_REVERTED
	case apply.ApplyStatusCommitted:
		return applyv1.ApplyStatus_APPLY_STATUS_COMMITTED
	default:
		return applyv1.ApplyStatus_APPLY_STATUS_UNSPECIFIED
	}
}
