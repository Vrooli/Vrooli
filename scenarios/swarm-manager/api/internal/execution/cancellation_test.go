package execution

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitionrunner"
	"swarm-manager/internal/workflowcontract"
)

type lostAckCancellationWorkflow struct {
	stubPhasedPlanWorkflow
	lookupErr    error
	lookupCalls  int
	beforeLookup func()
}

func (w *lostAckCancellationWorkflow) InspectWorkflowStart(context.Context, string, string) (*domainpb.WorkflowExecution, error) {
	w.lookupCalls++
	if w.beforeLookup != nil {
		w.beforeLookup()
	}
	if w.lookupErr != nil {
		return nil, w.lookupErr
	}
	in := w.invocation
	return &domainpb.WorkflowExecution{Id: "original-lost-ack-owner", Owner: in.Owner, WorkflowKey: in.WorkflowKey, IdempotencyKey: in.IdempotencyKey, DefinitionDigest: "sha256:original", ApprovalDigest: in.ApprovalDigest, GrantDigest: in.GrantDigest, EngagementGrant: in.EngagementGrant, Input: in.Input}, nil
}

func TestCancelPendingLostAcknowledgementUsesOnlyOriginalOwner(t *testing.T) {
	for _, reviewed := range []bool{false, true} {
		t.Run(fmt.Sprintf("reviewed=%v", reviewed), func(t *testing.T) {
			root, name := t.TempDir(), "lost-start-cancel"
			item := map[string]any{"name": name, "title": name, "status": "ready", "execution_strategy": adaptiveImprovementStrategy}
			if reviewed {
				item["execution_limits"] = reviewedLimits()
			}
			mustWriteBacklogItem(t, root, "execute", name, item)
			workflow := &lostAckCancellationWorkflow{stubPhasedPlanWorkflow: stubPhasedPlanWorkflow{startErr: errors.New("lost start response"), collectErr: agentmanager.ErrWorkflowNotReady}, lookupErr: agentmanager.ErrWorkflowNotFound}
			newService := func() *Service {
				return NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"), PlanRenderer: testPlanRenderer(), PhasedPlanWorkflow: workflow, AgentService: &stubAgentService{}, TransitionRegistry: testTransitionRegistry(t)})
			}
			service := newService()
			queued, err := service.QueueBacklog(t.Context(), CreateRequest{BacklogKind: "execute", BacklogName: name, Mode: ModeManual})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Start(t.Context(), queued.ExecutionID); err == nil {
				t.Fatal("lost start response was hidden")
			}
			workflow.beforeLookup = func() {
				records, index, err := service.loadRecordLocked(queued.ExecutionID)
				if err != nil || records[index].Status != StatusCancelling || records[index].FinishedAt != "" {
					t.Fatalf("owner read preceded durable local withdrawal: %+v %v", records, err)
				}
			}
			pending, err := service.Cancel(t.Context(), queued.ExecutionID)
			if err != nil || pending.Status != StatusCancelling || pending.SettledUsage != nil || pending.Cancellation.AcknowledgedAt != "" {
				t.Fatalf("absent original owner became zero usage: %+v %v", pending, err)
			}
			if workflow.startCalls != 1 || workflow.cancelCalls != 0 {
				t.Fatalf("redispatched or cancelled unbound owner: %+v", workflow)
			}
			if _, err := service.Retry(t.Context(), RetryRequest{ExecutionID: queued.ExecutionID}); err == nil {
				t.Fatal("unresolved reservation admitted retry")
			}
			service = newService()
			workflow.lookupErr = nil
			pending, err = service.Cancel(t.Context(), queued.ExecutionID)
			if err != nil || pending.Status != StatusCancelling || pending.Cancellation.AcknowledgedAt == "" {
				t.Fatalf("original owner not recovered: %+v %v", pending, err)
			}
			zero := int64(0)
			workflow.collectErr = nil
			workflow.completion = agentmanager.InvocationCompletion{ExecutionID: "original-lost-ack-owner", DefinitionDigest: "sha256:original", ApprovalDigest: workflow.invocation.ApprovalDigest, GrantDigest: workflow.invocation.GrantDigest, Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED, BudgetUsage: &domainpb.WorkflowBudgetUsage{AccountingComplete: true, Tokens: 23, Turns: 1, Children: 1}, ChargeReceipt: &domainpb.ChargeReceipt{Measured: true, AmountMicroUsd: &zero}, WallSeconds: 1}
			if err := service.reconcilePendingCancellations(t.Context()); err != nil {
				t.Fatal(err)
			}
			settled, err := service.Cancel(t.Context(), queued.ExecutionID)
			if err != nil || settled.Status != StatusCanceled || settled.SettledUsage == nil || settled.SettledUsage.Tokens != 23 || workflow.startCalls != 1 || workflow.cancelCalls != 1 {
				t.Fatalf("original receipt did not settle exactly once: %+v %v start=%d cancel=%d", settled, err, workflow.startCalls, workflow.cancelCalls)
			}
		})
	}
}

func TestCancelPersistsRevocationBeforeUnavailableOwner(t *testing.T) {
	service, workflow, started, _ := setupPhasedPlanExecution(t, "cancel-offline")
	service.agentService = &stubAgentService{}
	workflow.collectErr = errors.New("owner unavailable")
	keys := []string{}
	workflow.cancelHook = func(key string) error {
		keys = append(keys, key)
		records, err := service.store.Load()
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			if record.ExecutionID == started.ExecutionID && (record.Status != StatusCancelling || record.FinishedAt != "") {
				t.Fatalf("owner was called before durable local revocation: %+v", record)
			}
		}
		return errors.New("owner unavailable")
	}
	for i := 0; i < 2; i++ {
		result, err := service.Cancel(t.Context(), started.ExecutionID)
		if err != nil || result.Status != StatusCancelling || result.FinishedAt != "" || !strings.Contains(result.FailureReason, "cancellation") {
			t.Fatalf("pending cancellation was hidden: %+v %v", result, err)
		}
	}
	if len(keys) != 2 || keys[0] != keys[1] {
		t.Fatalf("retry changed cancellation identity: %v", keys)
	}
	if _, err := service.Start(t.Context(), started.ExecutionID); err == nil {
		t.Fatal("revoked execution restarted")
	}
	if _, err := service.Retry(t.Context(), RetryRequest{ExecutionID: started.ExecutionID}); err == nil || !strings.Contains(err.Error(), "cancelling") {
		t.Fatalf("unsettled cancellation did not block retry: %v", err)
	}
	if state := transitionApplyStateFor(t, service, workflowCorrelationFor(t, service, started).ExecutionID); state == transitionrun.ApplyStateComplete {
		t.Fatal("transport failure closed unresolved correlation")
	}
	if workflow.startCalls != 1 {
		t.Fatalf("created another owner execution: %d", workflow.startCalls)
	}
}

func TestCancelAcknowledgementRetainsReservationAndRestartSettlesOnce(t *testing.T) {
	root := t.TempDir()
	name := "cancel-accounting"
	mustWriteBacklogItem(t, root, "execute", name, map[string]any{
		"name": name, "title": "Cancellation accounting", "status": "ready",
		"execution_strategy": adaptiveImprovementStrategy, "execution_limits": reviewedLimits(),
	})
	workflow := &stubPhasedPlanWorkflow{collectErr: agentmanager.ErrWorkflowNotReady}
	newService := func() *Service {
		return NewService(ServiceConfig{
			DataRoot: root, StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
			PlanRenderer: testPlanRenderer(), PhasedPlanWorkflow: workflow,
			AgentService: &stubAgentService{}, TransitionRegistry: testTransitionRegistry(t),
		})
	}
	service := newService()
	queued, err := service.QueueBacklog(t.Context(), CreateRequest{BacklogKind: "execute", BacklogName: name, Mode: ModeManual})
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.Start(t.Context(), queued.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	correlation := workflowCorrelationFor(t, service, started)
	grantDigest := workflowcontract.GrantDigest(started.WorkflowGrant)
	pending, err := service.Cancel(t.Context(), started.ExecutionID)
	if err != nil || pending.Status != StatusCancelling || pending.Cancellation.AcknowledgedAt == "" || pending.Cancellation.SettledAt != "" || pending.FinishedAt != "" {
		t.Fatalf("acknowledgement was treated as terminal settlement: %+v %v", pending, err)
	}
	if countActiveExecutions([]Record{pending}) != 1 || !isInFlightRecord(pending) || !service.HasActiveForBacklog(t.Context(), "execute", name) {
		t.Fatal("unsettled cancellation released active capacity or plan acceptance guard")
	}

	// A fresh service reads the persisted authority withdrawal and reuses the
	// original owner identity. Positive but incomplete totals do not settle.
	service = newService()
	zero := int64(0)
	workflow.collectErr = nil
	workflow.completion = agentmanager.InvocationCompletion{
		ExecutionID: correlation.ExecutionID, DefinitionDigest: correlation.DefinitionDigest,
		ApprovalDigest: started.ApprovalDigest, GrantDigest: grantDigest,
		Status:        domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED,
		BudgetUsage:   &domainpb.WorkflowBudgetUsage{Tokens: 20, Turns: 2, Children: 1, NodeAttempts: 1},
		ChargeReceipt: &domainpb.ChargeReceipt{Measured: true, AmountMicroUsd: &zero}, WallSeconds: 1,
		Attempts: []*domainpb.WorkflowNodeAttempt{{NodeId: "slice", Ordinal: 1, RunId: "run-1"}},
	}
	if err := service.reconcilePendingCancellations(t.Context()); err != nil {
		t.Fatal(err)
	}
	pending, err = service.Get(t.Context(), started.ExecutionID)
	if err != nil || pending.Status != StatusCancelling || pending.SettledUsage != nil || pending.Cancellation.RequestID != "cancel-"+started.ExecutionID {
		t.Fatalf("restart released unknown accounting or changed identity: %+v %v", pending, err)
	}
	if _, err := service.Retry(t.Context(), RetryRequest{ExecutionID: started.ExecutionID}); err == nil || !strings.Contains(err.Error(), "cancelling") {
		t.Fatalf("restart admitted retry before final accounting: %v", err)
	}
	if err := service.applyPlanExecuteTransition(t.Context(), started.ExecutionID, transitionrunner.Outcome{TransitionKey: "plan.execute"}); err == nil || !strings.Contains(err.Error(), "cancellation") {
		t.Fatalf("late result could apply after authority withdrawal: %v", err)
	}
	staleProjection := started
	staleProjection.Status = StatusCompleted
	if err := service.saveReconciledProjection(staleProjection); err != nil {
		t.Fatal(err)
	}
	projectionRecords, projectionIndex, err := service.loadRecordLocked(started.ExecutionID)
	if err != nil || !reflect.DeepEqual(projectionRecords[projectionIndex], pending) {
		t.Fatalf("stale callback erased durable cancellation: %+v %v", projectionRecords, err)
	}
	if transitionApplyStateFor(t, service, correlation.ExecutionID) == transitionrun.ApplyStateComplete {
		t.Fatal("unknown accounting closed owner correlation")
	}

	workflow.completion.BudgetUsage.AccountingComplete = true
	if err := service.reconcilePendingCancellations(t.Context()); err != nil {
		t.Fatal(err)
	}
	settled, err := service.Get(t.Context(), started.ExecutionID)
	if err != nil || settled.Status != StatusCanceled || settled.Cancellation.SettledAt == "" || settled.Cancellation.ReconciledAt == "" || settled.FinishedAt == "" || settled.SettledUsage == nil {
		t.Fatalf("terminal original-owner accounting did not settle: %+v %v", settled, err)
	}
	if settled.SettledUsage.Tokens != 20 || settled.SettledUsage.Turns != 2 || !settled.SettledUsage.ChargeMeasured || !settled.SettledUsage.TokensKnown || settled.SettledUsage.ChargeMicroUSD != 0 || workflowcontract.GrantDigest(settled.WorkflowGrant) != grantDigest || settled.ApprovalDigest != started.ApprovalDigest {
		t.Fatalf("terminal settlement changed grant or usage: %+v", settled)
	}
	if transitionApplyStateFor(t, service, correlation.ExecutionID) != transitionrun.ApplyStateComplete {
		t.Fatal("settled cancellation left the correlation open")
	}
	item, err := service.loadBacklogItem("execute", name)
	if err != nil || item.Status != backlogStatusReady {
		t.Fatalf("settled cancellation did not restore backlog: %+v %v", item, err)
	}
	// Retry accounting receives the remainder of the same approval, never the
	// original full allowance. No second owner dispatch is needed for this proof.
	next := Record{ExecutionID: "next", BacklogKind: "execute", BacklogName: name, MaxSlices: settled.MaxSlices, ExecutionLimits: settled.ExecutionLimits.Clone(), ApprovalDigest: settled.ApprovalDigest}
	if err := service.prepareExecutionGrantLocked(t.Context(), []Record{settled}, &next, item); err != nil || next.WorkflowGrant.MaxTokens != started.WorkflowGrant.MaxTokens-20 || next.WorkflowGrant.MaxTurns != started.WorkflowGrant.MaxTurns-2 || next.MaxSlices != settled.MaxSlices-1 {
		t.Fatalf("canceled attempt replenished the retry allowance: %+v %v", next, err)
	}
	collectCalls := workflow.collectCalls
	if err := service.reconcilePendingCancellations(t.Context()); err != nil {
		t.Fatal(err)
	}
	repeated, err := service.Cancel(t.Context(), started.ExecutionID)
	if err != nil || !reflect.DeepEqual(repeated, settled) || workflow.collectCalls != collectCalls || workflow.cancelCalls != 1 || workflow.startCalls != 1 {
		t.Fatalf("replay changed terminal accounting or repeated owner work: %+v %v; cancel=%d collect=%d start=%d", repeated, err, workflow.cancelCalls, workflow.collectCalls, workflow.startCalls)
	}
}

func TestCancelDefaultWorkflowUsesOriginalUngrantBinding(t *testing.T) {
	service, workflow, started, _ := setupPhasedPlanExecution(t, "cancel-default-binding")
	if started.ApprovalDigest == "" || started.WorkflowGrant != nil || workflow.invocation.ApprovalDigest != "" || workflow.invocation.GrantDigest != "" {
		t.Fatalf("fixture must distinguish local acceptance from ungrant owner invocation: %+v %+v", started, workflow.invocation)
	}
	correlation := workflowCorrelationFor(t, service, started)
	zero := int64(0)
	workflow.completion = agentmanager.InvocationCompletion{
		ExecutionID: correlation.ExecutionID, DefinitionDigest: correlation.DefinitionDigest,
		Status:        domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED,
		BudgetUsage:   &domainpb.WorkflowBudgetUsage{AccountingComplete: true, Tokens: 21, Turns: 1},
		ChargeReceipt: &domainpb.ChargeReceipt{Measured: true, AmountMicroUsd: &zero}, WallSeconds: 1,
	}
	settled, err := service.Cancel(t.Context(), started.ExecutionID)
	if err != nil || settled.Status != StatusCanceled || settled.SettledUsage == nil || settled.SettledUsage.Tokens != 21 || settled.ApprovalDigest != started.ApprovalDigest {
		t.Fatalf("default cancellation stranded by a binding never sent to the owner: %+v %v", settled, err)
	}
}

func TestCancellationCleanupPreservesImmediateRetryBacklogStatus(t *testing.T) {
	service, workflow, started, _ := setupPhasedPlanExecution(t, "cancel-retry-race")
	service.agentService = &stubAgentService{}
	correlation := workflowCorrelationFor(t, service, started)
	zero := int64(0)
	workflow.completion = agentmanager.InvocationCompletion{ExecutionID: correlation.ExecutionID, DefinitionDigest: correlation.DefinitionDigest, Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED, BudgetUsage: &domainpb.WorkflowBudgetUsage{AccountingComplete: true}, ChargeReceipt: &domainpb.ChargeReceipt{Measured: true, AmountMicroUsd: &zero}, WallSeconds: 1}
	if canceled, err := service.Cancel(t.Context(), started.ExecutionID); err != nil || canceled.Status != StatusCanceled {
		t.Fatalf("settle: %+v %v", canceled, err)
	}
	workflow.start = agentmanager.WorkflowStart{ExecutionID: "workflow-retry", RunID: "run-retry", DefinitionDigest: "sha256:def"}
	if retry, err := service.Retry(t.Context(), RetryRequest{ExecutionID: started.ExecutionID}); err != nil || retry.Status != StatusStarting {
		t.Fatalf("retry: %+v %v", retry, err)
	}
	before, err := service.loadBacklogItem("execute", started.BacklogName)
	if err != nil || before.Status == backlogStatusReady {
		t.Fatalf("retry did not own active backlog standing: %+v %v", before, err)
	}
	if err := service.reconcilePendingCancellations(t.Context()); err != nil {
		t.Fatal(err)
	}
	item, err := service.loadBacklogItem("execute", started.BacklogName)
	if err != nil || item.Status != before.Status {
		t.Fatalf("old cancellation overwrote new execution activity: %+v %v", item, err)
	}
}
