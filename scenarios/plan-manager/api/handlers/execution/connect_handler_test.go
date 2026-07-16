package execution

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"

	internalexecution "plan-manager/internal/execution"
	internalplans "plan-manager/internal/plans"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// fakeExecutionService is a minimal stand-in for internalexecution.Service.
type fakeExecutionService struct {
	execution internalexecution.Execution
	pctx      internalexecution.PhaseContext
	plan      internalplans.Plan
	handoff   internalexecution.Handoff
	nudges    []internalexecution.CompletionNudge
	points    []internalexecution.VelocityPoint
	step      internalexecution.GuidedStep
	complete  bool
	err       error

	gotPlanID      string
	gotRunID       string
	gotPlanOrExec  string
	gotExecutionID string
	gotPhaseID     string
	gotToStatus    internalplans.PhaseStatus
	gotOverride    string
	gotFeedback    string
	gotReason      string
	gotAuthor      string
	gotInputs      internalexecution.CompletionInputs
}

func (f *fakeExecutionService) Start(_ context.Context, planID, runID string) (internalexecution.Execution, internalexecution.PhaseContext, internalexecution.GuidedStep, error) {
	f.gotPlanID, f.gotRunID = planID, runID
	return f.execution, f.pctx, f.step, f.err
}

func (f *fakeExecutionService) GetStatus(_ context.Context, executionID string) (internalexecution.Execution, internalexecution.PhaseContext, internalexecution.GuidedStep, error) {
	f.gotExecutionID = executionID
	return f.execution, f.pctx, f.step, f.err
}

func (f *fakeExecutionService) GetContext(_ context.Context, executionID, phaseID string) (internalexecution.Execution, internalexecution.PhaseContext, internalexecution.GuidedStep, error) {
	f.gotExecutionID, f.gotPhaseID = executionID, phaseID
	return f.execution, f.pctx, f.step, f.err
}

func (f *fakeExecutionService) Resume(_ context.Context, planOrExecution, phaseID, runID string) (internalexecution.Execution, internalexecution.PhaseContext, internalexecution.GuidedStep, error) {
	f.gotPlanOrExec, f.gotPhaseID, f.gotRunID = planOrExecution, phaseID, runID
	return f.execution, f.pctx, f.step, f.err
}

func (f *fakeExecutionService) ContinueExecution(_ context.Context, planOrExecution, phaseID, runID string) (internalexecution.Execution, internalexecution.PhaseContext, internalexecution.GuidedStep, error) {
	f.gotPlanOrExec, f.gotPhaseID, f.gotRunID = planOrExecution, phaseID, runID
	return f.execution, f.pctx, f.step, f.err
}

func (f *fakeExecutionService) AbandonExecution(_ context.Context, executionID, reason, actor string) (internalexecution.Execution, bool, internalexecution.GuidedStep, error) {
	f.gotExecutionID, f.gotReason, f.gotAuthor = executionID, reason, actor
	return f.execution, false, f.step, f.err
}

func (f *fakeExecutionService) SyncBaseline(_ context.Context, executionID string) (internalexecution.Execution, internalexecution.PhaseContext, internalexecution.GuidedStep, error) {
	f.gotExecutionID = executionID
	return f.execution, f.pctx, f.step, f.err
}

func (f *fakeExecutionService) AmendScope(_ context.Context, executionID string, _ internalexecution.ScopeAmendmentRequest) (internalexecution.Execution, internalexecution.PhaseContext, internalexecution.GuidedStep, error) {
	f.gotExecutionID = executionID
	return f.execution, f.pctx, f.step, f.err
}

func (f *fakeExecutionService) AdoptBaseline(_ context.Context, executionID string, _ internalexecution.BaselineAdoptionRequest) (internalexecution.Execution, internalexecution.PhaseContext, internalexecution.GuidedStep, error) {
	f.gotExecutionID = executionID
	return f.execution, f.pctx, f.step, f.err
}

func (f *fakeExecutionService) RepairSourceScope(_ context.Context, executionID string, _ internalexecution.SourceScopeRepairRequest) (internalexecution.Execution, internalexecution.PhaseContext, internalexecution.GuidedStep, error) {
	f.gotExecutionID = executionID
	return f.execution, f.pctx, f.step, f.err
}

func (f *fakeExecutionService) GetNext(_ context.Context, executionID string) (internalexecution.PhaseContext, bool, internalexecution.GuidedStep, error) {
	f.gotExecutionID = executionID
	return f.pctx, f.complete, f.step, f.err
}

func (f *fakeExecutionService) TransitionPhase(_ context.Context, executionID, phaseID string, inputs internalexecution.PhaseTransitionInputs) (internalexecution.Execution, internalplans.Plan, internalexecution.GuidedStep, error) {
	f.gotExecutionID, f.gotPhaseID, f.gotToStatus, f.gotOverride, f.gotFeedback = executionID, phaseID, inputs.ToStatus, inputs.ValidationOverrideReason, inputs.FeedbackOverrideReason
	return f.execution, f.plan, f.step, f.err
}

func (f *fakeExecutionService) Complete(_ context.Context, executionID string, inputs internalexecution.CompletionInputs) (internalexecution.Handoff, []internalexecution.CompletionNudge, internalexecution.GuidedStep, error) {
	f.gotExecutionID, f.gotInputs = executionID, inputs
	return f.handoff, f.nudges, f.step, f.err
}

func (f *fakeExecutionService) PartialHandoff(_ context.Context, executionID string, inputs internalexecution.CompletionInputs) (internalexecution.Handoff, []internalexecution.CompletionNudge, internalexecution.GuidedStep, error) {
	f.gotExecutionID, f.gotInputs = executionID, inputs
	return f.handoff, f.nudges, f.step, f.err
}

func (f *fakeExecutionService) GetHandoff(_ context.Context, executionID string) (internalexecution.Handoff, internalexecution.GuidedStep, error) {
	f.gotExecutionID = executionID
	return f.handoff, f.step, f.err
}

func (f *fakeExecutionService) GetVelocity(_ context.Context, planID string) ([]internalexecution.VelocityPoint, internalexecution.GuidedStep, error) {
	f.gotPlanID = planID
	return f.points, f.step, f.err
}

var _ internalexecution.Service = (*fakeExecutionService)(nil)

func newExecutionHandler(svc internalexecution.Service) *connectHandler {
	return NewConnectHandler(Deps{Service: svc, Logger: log.New(io.Discard, "", 0)})
}

func TestStartSuccess(t *testing.T) {
	svc := &fakeExecutionService{
		execution: internalexecution.Execution{ID: "e1", PlanID: "p1", RunID: "run-1"},
		step:      internalexecution.GuidedStep{StepKind: "execution_started", NextActions: []internalexecution.NextAction{{ID: "status", Kind: internalexecution.NextActionRecommended, Argv: []string{"exec", "status", "e1"}}}},
	}
	h := newExecutionHandler(svc)

	resp, err := h.Start(context.Background(), connect.NewRequest(&executionv1.StartRequest{PlanId: "p1", RunId: "run-1"}))
	require.NoError(t, err)
	require.Equal(t, "e1", resp.Msg.GetExecution().GetId())
	require.Equal(t, "execution_started", resp.Msg.GetStep().GetStepKind())
	require.Equal(t, []string{"exec", "status", "e1"}, resp.Msg.GetStep().GetNextActions()[0].GetArgv())
	require.Equal(t, "p1", svc.gotPlanID)
	require.Equal(t, "run-1", svc.gotRunID)
}

func TestGetStatusSuccess(t *testing.T) {
	svc := &fakeExecutionService{
		execution: internalexecution.Execution{ID: "e1", CurrentPhaseID: "ph-1"},
		pctx: internalexecution.PhaseContext{
			CurrentPhase: internalplans.Phase{ID: "ph-1", Title: "Cur"},
			HasCurrent:   true,
			Completeness: internalexecution.CompletenessPartial,
		},
	}
	h := newExecutionHandler(svc)

	resp, err := h.GetStatus(context.Background(), connect.NewRequest(&executionv1.GetStatusRequest{ExecutionId: "e1"}))
	require.NoError(t, err)
	require.Equal(t, "e1", resp.Msg.GetExecution().GetId())
	require.NotNil(t, resp.Msg.GetContext())
	require.Equal(t, "ph-1", resp.Msg.GetContext().GetCurrentPhase().GetId())
	require.Equal(t, sharedv1.Completeness_COMPLETENESS_PARTIAL, resp.Msg.GetContext().GetCompleteness())
	require.Equal(t, "e1", svc.gotExecutionID)
}

func TestGetContextSuccess(t *testing.T) {
	svc := &fakeExecutionService{
		execution: internalexecution.Execution{ID: "e1", CurrentPhaseID: "ph-1"},
		pctx: internalexecution.PhaseContext{
			CurrentPhase: internalplans.Phase{ID: "ph-2", Title: "Requested"},
			HasCurrent:   true,
		},
	}
	h := newExecutionHandler(svc)

	resp, err := h.GetContext(context.Background(), connect.NewRequest(&executionv1.GetContextRequest{ExecutionId: "e1", PhaseId: "ph-2"}))
	require.NoError(t, err)
	require.Equal(t, "e1", resp.Msg.GetExecution().GetId())
	require.Equal(t, "ph-2", resp.Msg.GetContext().GetCurrentPhase().GetId())
	require.Equal(t, "e1", svc.gotExecutionID)
	require.Equal(t, "ph-2", svc.gotPhaseID)
}

func TestResumeSuccess(t *testing.T) {
	svc := &fakeExecutionService{
		execution: internalexecution.Execution{ID: "e1", CurrentPhaseID: "ph-2"},
		pctx: internalexecution.PhaseContext{
			CurrentPhase: internalplans.Phase{ID: "ph-2", Title: "Requested"},
			HasCurrent:   true,
		},
	}
	h := newExecutionHandler(svc)

	resp, err := h.Resume(context.Background(), connect.NewRequest(&executionv1.ResumeRequest{PlanOrExecution: "plan-1", PhaseId: "ph-2", RunId: "run-1"}))
	require.NoError(t, err)
	require.Equal(t, "e1", resp.Msg.GetExecution().GetId())
	require.Equal(t, "ph-2", resp.Msg.GetContext().GetCurrentPhase().GetId())
	require.Equal(t, "plan-1", svc.gotPlanOrExec)
	require.Equal(t, "ph-2", svc.gotPhaseID)
	require.Equal(t, "run-1", svc.gotRunID)
}

func TestContinueExecutionSuccess(t *testing.T) {
	svc := &fakeExecutionService{
		execution: internalexecution.Execution{ID: "e1", PlanID: "plan-1", CurrentPhaseID: "ph-1"},
		pctx:      internalexecution.PhaseContext{CurrentPhase: internalplans.Phase{ID: "ph-1"}, HasCurrent: true},
		step:      internalexecution.GuidedStep{StepKind: "phase_context", NextActions: []internalexecution.NextAction{{ID: "transition-active", Kind: internalexecution.NextActionRecommended}}},
	}
	h := newExecutionHandler(svc)

	resp, err := h.ContinueExecution(context.Background(), connect.NewRequest(&executionv1.ContinueExecutionRequest{PlanOrExecution: "plan-1", PhaseId: "ph-1", RunId: "run-1"}))
	require.NoError(t, err)
	require.Equal(t, "plan-1", svc.gotPlanOrExec)
	require.Equal(t, "ph-1", svc.gotPhaseID)
	require.Equal(t, "run-1", svc.gotRunID)
	require.Equal(t, "e1", resp.Msg.GetExecution().GetId())
	require.Equal(t, "phase_context", resp.Msg.GetStep().GetStepKind())
}

func TestAbandonExecutionSuccess(t *testing.T) {
	svc := &fakeExecutionService{execution: internalexecution.Execution{
		ID: "e1", PlanID: "plan-1", LifecycleState: internalexecution.ExecutionLifecycleAbandoned,
		AbandonedReason: "accidental", AbandonedAt: "2026-07-16T12:00:00Z", AbandonedBy: "agent-7",
	}}
	h := newExecutionHandler(svc)

	resp, err := h.AbandonExecution(context.Background(), connect.NewRequest(&executionv1.AbandonExecutionRequest{ExecutionId: "e1", Reason: "accidental", Actor: "agent-7"}))
	require.NoError(t, err)
	require.Equal(t, "e1", svc.gotExecutionID)
	require.Equal(t, "accidental", svc.gotReason)
	require.Equal(t, "agent-7", svc.gotAuthor)
	require.Equal(t, "abandoned", resp.Msg.GetExecution().GetLifecycleState())
}

func TestGetNextSuccess(t *testing.T) {
	svc := &fakeExecutionService{complete: true, pctx: internalexecution.PhaseContext{Completeness: internalexecution.CompletenessFull}}
	h := newExecutionHandler(svc)

	resp, err := h.GetNext(context.Background(), connect.NewRequest(&executionv1.GetNextRequest{ExecutionId: "e1"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetComplete())
	require.NotNil(t, resp.Msg.GetContext())
}

func TestTransitionPhaseForwardsStatus(t *testing.T) {
	svc := &fakeExecutionService{
		execution: internalexecution.Execution{ID: "e1"},
		plan:      internalplans.Plan{ID: "p1", Status: internalplans.PlanStatusActive},
	}
	h := newExecutionHandler(svc)

	resp, err := h.TransitionPhase(context.Background(), connect.NewRequest(&executionv1.TransitionPhaseRequest{
		ExecutionId: "e1",
		PhaseId:     "ph-1",
		ToStatus:    sharedv1.PhaseStatus_PHASE_STATUS_DONE,
		ValidationOverride: &executionv1.ValidationOverride{
			Reason: "validated externally",
		},
		FeedbackOverride: &executionv1.FeedbackOverride{
			Reason: "feedback reviewed externally",
		},
	}))
	require.NoError(t, err)
	require.Equal(t, "e1", resp.Msg.GetExecution().GetId())
	require.Equal(t, "p1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "ph-1", svc.gotPhaseID)
	require.Equal(t, internalplans.PhaseStatusDone, svc.gotToStatus, "proto status must be translated to the domain enum")
	require.Equal(t, "validated externally", svc.gotOverride)
	require.Equal(t, "feedback reviewed externally", svc.gotFeedback)
}

func TestCompleteForwardsInputs(t *testing.T) {
	svc := &fakeExecutionService{
		handoff: internalexecution.Handoff{ID: "h1", Completeness: internalexecution.CompletenessFull},
		nudges:  []internalexecution.CompletionNudge{{Kind: "file_bug", Satisfied: true}},
	}
	h := newExecutionHandler(svc)

	resp, err := h.Complete(context.Background(), connect.NewRequest(&executionv1.CompleteRequest{
		ExecutionId: "e1", Tokens: 1000, Iterations: 4,
	}))
	require.NoError(t, err)
	require.Equal(t, "h1", resp.Msg.GetHandoff().GetId())
	require.Len(t, resp.Msg.GetNudges(), 1)
	require.Equal(t, int64(1000), svc.gotInputs.Tokens)
	require.Equal(t, int32(4), svc.gotInputs.Iterations)
}

func TestGetHandoffSuccess(t *testing.T) {
	svc := &fakeExecutionService{handoff: internalexecution.Handoff{ID: "h1", PlanID: "p1"}}
	h := newExecutionHandler(svc)

	resp, err := h.GetHandoff(context.Background(), connect.NewRequest(&executionv1.GetHandoffRequest{ExecutionId: "e1"}))
	require.NoError(t, err)
	require.Equal(t, "h1", resp.Msg.GetHandoff().GetId())
	require.Equal(t, "p1", resp.Msg.GetHandoff().GetPlanId())
}

func TestGetVelocitySuccess(t *testing.T) {
	svc := &fakeExecutionService{points: []internalexecution.VelocityPoint{
		{ID: "v1", PlanID: "p1", Tokens: 10},
	}}
	h := newExecutionHandler(svc)

	resp, err := h.GetVelocity(context.Background(), connect.NewRequest(&executionv1.GetVelocityRequest{PlanId: "p1"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetPoints(), 1)
	require.Equal(t, "v1", resp.Msg.GetPoints()[0].GetId())
	require.Equal(t, "p1", svc.gotPlanID)
}

// TestExecutionErrorMapping asserts each execution/plans sentinel maps to the
// documented Connect code (see internal/execution/service_error_mapping.go).
func TestExecutionErrorMapping(t *testing.T) {
	t.Run("invalid_execution_is_invalid_argument", func(t *testing.T) {
		h := newExecutionHandler(&fakeExecutionService{err: internalexecution.ErrInvalidExecution{Reason: "plan id is required"}})
		_, err := h.Start(context.Background(), connect.NewRequest(&executionv1.StartRequest{}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("active_execution_conflict_has_typed_recovery_detail", func(t *testing.T) {
		h := newExecutionHandler(&fakeExecutionService{err: internalexecution.ErrActiveExecutionConflict{PlanID: "p", ExecutionIDs: []string{"e1", "e2"}}})
		_, err := h.Start(context.Background(), connect.NewRequest(&executionv1.StartRequest{PlanId: "p"}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
		connectErr := new(connect.Error)
		require.ErrorAs(t, err, &connectErr)
		require.Len(t, connectErr.Details(), 1)
		value, detailErr := connectErr.Details()[0].Value()
		require.NoError(t, detailErr)
		detail := value.(*executionv1.ActiveExecutionConflict)
		require.Equal(t, []string{"e1", "e2"}, detail.GetExecutionIds())
		require.Equal(t, []string{"plan-manager exec resume e1", "plan-manager exec resume e2"}, detail.GetResumeCommands())
	})
	t.Run("execution_not_found_is_not_found", func(t *testing.T) {
		h := newExecutionHandler(&fakeExecutionService{err: internalexecution.ErrExecutionNotFound{ID: "x"}})
		_, err := h.GetStatus(context.Background(), connect.NewRequest(&executionv1.GetStatusRequest{ExecutionId: "x"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("plan_not_found_is_not_found", func(t *testing.T) {
		h := newExecutionHandler(&fakeExecutionService{err: internalplans.ErrPlanNotFound{ID: "p"}})
		_, err := h.Start(context.Background(), connect.NewRequest(&executionv1.StartRequest{PlanId: "p"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("phase_not_found_is_not_found", func(t *testing.T) {
		h := newExecutionHandler(&fakeExecutionService{err: internalplans.ErrPhaseNotFound{PlanID: "p", PhaseID: "ph"}})
		_, err := h.TransitionPhase(context.Background(), connect.NewRequest(&executionv1.TransitionPhaseRequest{ExecutionId: "e", PhaseId: "ph"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("invalid_plan_is_invalid_argument", func(t *testing.T) {
		h := newExecutionHandler(&fakeExecutionService{err: internalplans.ErrInvalidPlan{Reason: "bad"}})
		_, err := h.TransitionPhase(context.Background(), connect.NewRequest(&executionv1.TransitionPhaseRequest{ExecutionId: "e", PhaseId: "ph"}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("unknown_error_is_internal", func(t *testing.T) {
		h := newExecutionHandler(&fakeExecutionService{err: errors.New("boom")})
		_, err := h.GetStatus(context.Background(), connect.NewRequest(&executionv1.GetStatusRequest{ExecutionId: "x"}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}
