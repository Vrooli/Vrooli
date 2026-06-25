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
	decision  internalexecution.Decision
	finding   internalexecution.Finding
	findings  []internalexecution.Finding
	handoff   internalexecution.Handoff
	nudges    []internalexecution.CompletionNudge
	points    []internalexecution.VelocityPoint
	complete  bool
	err       error

	gotPlanID      string
	gotRunID       string
	gotExecutionID string
	gotPhaseID     string
	gotToStatus    internalplans.PhaseStatus
	gotSummary     string
	gotDetail      string
	gotTitle       string
	gotInputs      internalexecution.CompletionInputs
	gotFindingID   string
	gotTriage      internalexecution.FindingTriage
}

func (f *fakeExecutionService) Start(_ context.Context, planID, runID string) (internalexecution.Execution, error) {
	f.gotPlanID, f.gotRunID = planID, runID
	return f.execution, f.err
}

func (f *fakeExecutionService) GetStatus(_ context.Context, executionID string) (internalexecution.Execution, internalexecution.PhaseContext, error) {
	f.gotExecutionID = executionID
	return f.execution, f.pctx, f.err
}

func (f *fakeExecutionService) GetNext(_ context.Context, executionID string) (internalexecution.PhaseContext, bool, error) {
	f.gotExecutionID = executionID
	return f.pctx, f.complete, f.err
}

func (f *fakeExecutionService) TransitionPhase(_ context.Context, executionID, phaseID string, to internalplans.PhaseStatus) (internalexecution.Execution, internalplans.Plan, error) {
	f.gotExecutionID, f.gotPhaseID, f.gotToStatus = executionID, phaseID, to
	return f.execution, f.plan, f.err
}

func (f *fakeExecutionService) RecordDecision(_ context.Context, executionID, phaseID, summary, detail string) (internalexecution.Decision, error) {
	f.gotExecutionID, f.gotPhaseID, f.gotSummary, f.gotDetail = executionID, phaseID, summary, detail
	return f.decision, f.err
}

func (f *fakeExecutionService) RecordFinding(_ context.Context, executionID, phaseID, title, detail string) (internalexecution.Finding, error) {
	f.gotExecutionID, f.gotPhaseID, f.gotTitle, f.gotDetail = executionID, phaseID, title, detail
	return f.finding, f.err
}

func (f *fakeExecutionService) Complete(_ context.Context, executionID string, inputs internalexecution.CompletionInputs) (internalexecution.Handoff, []internalexecution.CompletionNudge, error) {
	f.gotExecutionID, f.gotInputs = executionID, inputs
	return f.handoff, f.nudges, f.err
}

func (f *fakeExecutionService) GetHandoff(_ context.Context, executionID string) (internalexecution.Handoff, error) {
	f.gotExecutionID = executionID
	return f.handoff, f.err
}

func (f *fakeExecutionService) ListCandidateFindings(_ context.Context, executionID string) ([]internalexecution.Finding, error) {
	f.gotExecutionID = executionID
	return f.findings, f.err
}

func (f *fakeExecutionService) TriageFinding(_ context.Context, findingID string, triage internalexecution.FindingTriage) (internalexecution.Finding, error) {
	f.gotFindingID, f.gotTriage = findingID, triage
	return f.finding, f.err
}

func (f *fakeExecutionService) GetVelocity(_ context.Context, planID string) ([]internalexecution.VelocityPoint, error) {
	f.gotPlanID = planID
	return f.points, f.err
}

var _ internalexecution.Service = (*fakeExecutionService)(nil)

func newExecutionHandler(svc internalexecution.Service) *connectHandler {
	return NewConnectHandler(Deps{Service: svc, Logger: log.New(io.Discard, "", 0)})
}

func TestStartSuccess(t *testing.T) {
	svc := &fakeExecutionService{execution: internalexecution.Execution{ID: "e1", PlanID: "p1", RunID: "run-1"}}
	h := newExecutionHandler(svc)

	resp, err := h.Start(context.Background(), connect.NewRequest(&executionv1.StartRequest{PlanId: "p1", RunId: "run-1"}))
	require.NoError(t, err)
	require.Equal(t, "e1", resp.Msg.GetExecution().GetId())
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
	}))
	require.NoError(t, err)
	require.Equal(t, "e1", resp.Msg.GetExecution().GetId())
	require.Equal(t, "p1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "ph-1", svc.gotPhaseID)
	require.Equal(t, internalplans.PhaseStatusDone, svc.gotToStatus, "proto status must be translated to the domain enum")
}

func TestRecordDecisionForwardsArgs(t *testing.T) {
	svc := &fakeExecutionService{decision: internalexecution.Decision{ID: "d1", Summary: "s"}}
	h := newExecutionHandler(svc)

	resp, err := h.RecordDecision(context.Background(), connect.NewRequest(&executionv1.RecordDecisionRequest{
		ExecutionId: "e1", PhaseId: "ph-1", Summary: "s", Detail: "det",
	}))
	require.NoError(t, err)
	require.Equal(t, "d1", resp.Msg.GetDecision().GetId())
	require.Equal(t, "s", svc.gotSummary)
	require.Equal(t, "det", svc.gotDetail)
}

func TestRecordFindingForwardsArgs(t *testing.T) {
	svc := &fakeExecutionService{finding: internalexecution.Finding{ID: "f1", Triage: internalexecution.TriageCandidate}}
	h := newExecutionHandler(svc)

	resp, err := h.RecordFinding(context.Background(), connect.NewRequest(&executionv1.RecordFindingRequest{
		ExecutionId: "e1", PhaseId: "ph-1", Title: "bug", Detail: "det",
	}))
	require.NoError(t, err)
	require.Equal(t, "f1", resp.Msg.GetFinding().GetId())
	require.Equal(t, sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE, resp.Msg.GetFinding().GetTriage())
	require.Equal(t, "bug", svc.gotTitle)
}

func TestCompleteForwardsInputs(t *testing.T) {
	svc := &fakeExecutionService{
		handoff: internalexecution.Handoff{ID: "h1", Completeness: internalexecution.CompletenessFull},
		nudges:  []internalexecution.CompletionNudge{{Kind: "file_bugs", Satisfied: true}},
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

func TestListCandidateFindingsSuccess(t *testing.T) {
	svc := &fakeExecutionService{findings: []internalexecution.Finding{
		{ID: "f1", Triage: internalexecution.TriageCandidate},
		{ID: "f2", Triage: internalexecution.TriageCandidate},
	}}
	h := newExecutionHandler(svc)

	resp, err := h.ListCandidateFindings(context.Background(), connect.NewRequest(&executionv1.ListCandidateFindingsRequest{ExecutionId: "e1"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetFindings(), 2)
	require.Equal(t, "f1", resp.Msg.GetFindings()[0].GetId())
}

func TestTriageFindingForwardsTriage(t *testing.T) {
	svc := &fakeExecutionService{finding: internalexecution.Finding{ID: "f1", Triage: internalexecution.TriagePromoted}}
	h := newExecutionHandler(svc)

	resp, err := h.TriageFinding(context.Background(), connect.NewRequest(&executionv1.TriageFindingRequest{
		FindingId: "f1",
		Triage:    sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED,
	}))
	require.NoError(t, err)
	require.Equal(t, sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED, resp.Msg.GetFinding().GetTriage())
	require.Equal(t, "f1", svc.gotFindingID)
	require.Equal(t, internalexecution.TriagePromoted, svc.gotTriage, "proto triage must be translated to the domain enum")
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
	t.Run("execution_not_found_is_not_found", func(t *testing.T) {
		h := newExecutionHandler(&fakeExecutionService{err: internalexecution.ErrExecutionNotFound{ID: "x"}})
		_, err := h.GetStatus(context.Background(), connect.NewRequest(&executionv1.GetStatusRequest{ExecutionId: "x"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("finding_not_found_is_not_found", func(t *testing.T) {
		h := newExecutionHandler(&fakeExecutionService{err: internalexecution.ErrFindingNotFound{ID: "f"}})
		_, err := h.TriageFinding(context.Background(), connect.NewRequest(&executionv1.TriageFindingRequest{FindingId: "f", Triage: sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED}))
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
