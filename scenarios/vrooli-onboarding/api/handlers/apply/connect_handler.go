package apply

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/shared"
	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type connectHandler struct {
	service applydomain.Service
}

func NewConnectHandler(service applydomain.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) StartApply(ctx context.Context, req *connect.Request[applyv1.StartApplyRequest]) (*connect.Response[applyv1.StartApplyResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("apply admission request is required"))
	}
	run, err := h.service.StartApply(ctx, applydomain.StartRequest{
		Target: req.Msg.GetTarget(), PlanID: req.Msg.GetPlanId(), PlanDigest: req.Msg.GetPlanDigest(),
		ExpectedRevision: req.Msg.GetExpectedRevision(), ConsentReceiptID: req.Msg.GetConsentReceiptId(), IdempotencyKey: req.Msg.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, applyError("start apply", err)
	}
	return connect.NewResponse(&applyv1.StartApplyResponse{Run: runProto(run)}), nil
}

func (h *connectHandler) ReviewApply(ctx context.Context, req *connect.Request[applyv1.ReviewApplyRequest]) (*connect.Response[applyv1.ReviewApplyResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("apply review request is required"))
	}
	review, err := h.service.ReviewApply(ctx, applydomain.ReviewRequest{
		Target: req.Msg.GetTarget(), PlanID: req.Msg.GetPlanId(), PlanDigest: req.Msg.GetPlanDigest(), ExpectedRevision: req.Msg.GetExpectedRevision(),
	})
	if err != nil {
		return nil, applyError("review apply", err)
	}
	return connect.NewResponse(&applyv1.ReviewApplyResponse{
		Target: review.Target, PlanId: review.PlanID, PlanDigest: review.PlanDigest, Revision: review.Revision,
		ConsentReceiptId: review.ConsentReceiptID, ExpiresAt: timestamp(review.ExpiresAt),
	}), nil
}

func (h *connectHandler) CancelApply(ctx context.Context, req *connect.Request[applyv1.CancelApplyRequest]) (*connect.Response[applyv1.CancelApplyResponse], error) {
	if req == nil || req.Msg == nil || strings.TrimSpace(req.Msg.GetRunId()) == "" || strings.ContainsAny(req.Msg.GetRunId(), "/\\") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("a valid apply run id is required"))
	}
	run, err := h.service.CancelApply(ctx, strings.TrimSpace(req.Msg.GetRunId()))
	if err != nil {
		return nil, applyError("cancel apply", err)
	}
	return connect.NewResponse(&applyv1.CancelApplyResponse{Run: runProto(run)}), nil
}

func (h *connectHandler) GetApplyRun(ctx context.Context, req *connect.Request[applyv1.GetApplyRunRequest]) (*connect.Response[applyv1.GetApplyRunResponse], error) {
	runID := ""
	if req != nil && req.Msg != nil {
		runID = strings.TrimSpace(req.Msg.GetRunId())
	}
	if runID == "" || strings.ContainsAny(runID, "/\\") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("a valid apply run id is required"))
	}
	run, err := h.service.GetApplyRun(ctx, runID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read apply run: %w", err))
	}
	return connect.NewResponse(runProto(run)), nil
}

func (h *connectHandler) GetApplyPlan(ctx context.Context, req *connect.Request[applyv1.GetApplyPlanRequest]) (*connect.Response[applyv1.GetApplyPlanResponse], error) {
	target := "local"
	if req != nil && req.Msg != nil && strings.TrimSpace(req.Msg.GetTarget()) != "" {
		target = strings.TrimSpace(req.Msg.GetTarget())
	}
	plan, err := h.service.GetApplyPlan(ctx, target)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read apply plan: %w", err))
	}
	items := make([]*applyv1.ApplyItem, 0, len(plan.Items))
	for _, item := range plan.Items {
		items = append(items, itemProto(item))
	}
	return connect.NewResponse(&applyv1.GetApplyPlanResponse{
		Items: items, Target: plan.Target, PlanId: plan.PlanID, PlanDigest: plan.Digest, Revision: plan.Revision,
		ExpiresAt: timestamp(plan.ExpiresAt),
	}), nil
}

func applyError(operation string, err error) error {
	message := strings.ToLower(err.Error())
	code := connect.CodeInternal
	switch {
	case strings.Contains(message, "required"), strings.Contains(message, "invalid"):
		code = connect.CodeInvalidArgument
	case strings.Contains(message, "stale"), strings.Contains(message, "changed"), strings.Contains(message, "expired"), strings.Contains(message, "does not match"):
		code = connect.CodeAborted
	case strings.Contains(message, "idempotency") && strings.Contains(message, "already"):
		code = connect.CodeAlreadyExists
	case strings.Contains(message, "not found"):
		code = connect.CodeNotFound
	}
	return connect.NewError(code, fmt.Errorf("%s: %w", operation, err))
}

func runProto(run applydomain.Run) *applyv1.GetApplyRunResponse {
	steps := make([]*applyv1.ApplyStep, 0, len(run.Steps))
	for _, step := range run.Steps {
		steps = append(steps, stepProto(step))
	}
	blockers := make([]*sharedv1.CompletionBlocker, 0, len(run.Blockers))
	for _, blocker := range run.Blockers {
		blockers = append(blockers, blockerProto(blocker))
	}
	degraded := make([]*sharedv1.CompletionBlocker, 0, len(run.Degraded))
	for _, blocker := range run.Degraded {
		degraded = append(degraded, blockerProto(blocker))
	}
	return &applyv1.GetApplyRunResponse{
		RunId: run.ID, Status: runState(run.Status), LegacyStatus: run.Status, SelectionDigest: run.SelectionDigest,
		StartedAt: timestamp(run.StartedAt), CompletedAt: timestamp(run.CompletedAt), Error: run.Error, Steps: steps,
		Blockers: blockers, Degraded: degraded, DegradedDigest: run.DegradedDigest, RunnerPid: int64(run.RunnerPID), Heartbeat: timestamp(run.Heartbeat),
	}
}

func itemProto(item applydomain.Item) *applyv1.ApplyItem {
	return &applyv1.ApplyItem{Id: item.ID, Kind: item.Kind, Name: item.Name, Dependencies: item.Dependencies, Required: item.Required, Privileged: item.Privileged, ObservedState: item.ObservedState}
}

func stepProto(step applydomain.Step) *applyv1.ApplyStep {
	return &applyv1.ApplyStep{Id: step.ID, Kind: step.Kind, Name: step.Name, Dependencies: step.Dependencies, Required: step.Required, Privileged: step.Privileged, ObservedState: step.ObservedState, State: stepState(step.State), LegacyOutcome: step.LegacyOutcome, Disposition: step.Disposition, Error: step.Error, Remediation: step.Remediation, BlockedBy: step.BlockedBy, ErrorCode: step.ErrorCode, StartedAt: timestamp(step.StartedAt), CompletedAt: timestamp(step.CompletedAt)}
}

func blockerProto(blocker applydomain.Blocker) *sharedv1.CompletionBlocker {
	return &sharedv1.CompletionBlocker{Kind: blocker.Kind, Name: blocker.Name, Reason: blocker.Reason, Remediation: blocker.Remediation}
}

func runState(value string) applyv1.ApplyRunState {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending":
		return applyv1.ApplyRunState_APPLY_RUN_STATE_PENDING
	case "applying":
		return applyv1.ApplyRunState_APPLY_RUN_STATE_APPLYING
	case "applied":
		return applyv1.ApplyRunState_APPLY_RUN_STATE_APPLIED
	case "already_satisfied":
		return applyv1.ApplyRunState_APPLY_RUN_STATE_ALREADY_SATISFIED
	case "partially_applied":
		return applyv1.ApplyRunState_APPLY_RUN_STATE_PARTIALLY_APPLIED
	case "configuration_incomplete":
		return applyv1.ApplyRunState_APPLY_RUN_STATE_CONFIGURATION_INCOMPLETE
	case "failed":
		return applyv1.ApplyRunState_APPLY_RUN_STATE_FAILED
	case "cancelled":
		return applyv1.ApplyRunState_APPLY_RUN_STATE_CANCELLED
	case "indeterminate":
		return applyv1.ApplyRunState_APPLY_RUN_STATE_INDETERMINATE
	default:
		return applyv1.ApplyRunState_APPLY_RUN_STATE_UNSPECIFIED
	}
}

func stepState(value string) applyv1.ApplyStepState {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_PENDING
	case "applying":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_APPLYING
	case "applied":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_APPLIED
	case "already_satisfied":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_ALREADY_SATISFIED
	case "blocked":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_BLOCKED
	case "failed":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_FAILED
	case "timed_out":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_TIMED_OUT
	case "needs_elevation":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_NEEDS_ELEVATION
	case "not_applicable":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_NOT_APPLICABLE
	case "skipped_self":
		return applyv1.ApplyStepState_APPLY_STEP_STATE_SKIPPED_SELF
	default:
		return applyv1.ApplyStepState_APPLY_STEP_STATE_UNSPECIFIED
	}
}

func timestamp(value string) *timestamppb.Timestamp {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return timestamppb.New(parsed)
	}
	return nil
}
