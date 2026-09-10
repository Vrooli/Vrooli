package execution

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/workflowcontract"
)

func reviewedLimits() *identity.ExecutionLimits {
	return &identity.ExecutionLimits{MaxSlices: 64, MaxTokens: 2000000, MaxWallSeconds: 604800, MaxTurns: 1000, MaxChargeMicroUSD: 30000000, MaxChildren: 256, MaxNodeAttempts: 512, MaxRetries: 64}
}

func TestQueueBindsReviewedExecutionLimitsToOwnerGrant(t *testing.T) {
	root := t.TempDir()
	limits := reviewedLimits()
	mustWriteBacklogItem(t, root, "execute", "reviewed-limits", map[string]any{"name": "reviewed-limits", "title": "Reviewed campaign", "status": "ready", "execution_strategy": adaptiveImprovementStrategy, "execution_limits": limits})
	workflow := &stubPhasedPlanWorkflow{}
	service := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, "executions.json"), PlanRenderer: testPlanRenderer(), PhasedPlanWorkflow: workflow, TransitionRegistry: testTransitionRegistry(t)})
	if _, err := service.QueueBacklog(t.Context(), CreateRequest{BacklogKind: "execute", BacklogName: "reviewed-limits", Mode: ModeManual, MaxSlices: 65, Force: true}); err == nil {
		t.Fatal("force bypassed the reviewed slice allowance")
	}
	queued, err := service.QueueBacklog(t.Context(), CreateRequest{BacklogKind: "execute", BacklogName: "reviewed-limits", Mode: ModeManual})
	if err != nil {
		t.Fatal(err)
	}
	if queued.MaxSlices != 64 || queued.ExecutionLimits == nil || queued.ApprovalDigest == "" {
		t.Fatalf("reviewed allowance omitted: %+v", queued)
	}
	started, err := service.Start(t.Context(), queued.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	grant := workflow.invocation.EngagementGrant
	if grant == nil || grant.MaxTokens != int32(limits.MaxTokens) || grant.MaxWallTimeSeconds != int32(limits.MaxWallSeconds) || grant.MaxTurns != int32(limits.MaxTurns) || grant.MaxChildren != 256 || grant.MaxConcurrency != 1 {
		t.Fatalf("owner grant=%+v", grant)
	}
	if workflow.invocation.ApprovalDigest != started.ApprovalDigest || workflow.invocation.GrantDigest != workflowcontract.GrantDigest(started.WorkflowGrant) {
		t.Fatal("owner grant was not bound to persisted acceptance")
	}
	stored, err := service.Get(t.Context(), started.ExecutionID)
	if err != nil || stored.WorkflowGrant == nil || stored.WorkflowGrant.MaxTokens != limits.MaxTokens {
		t.Fatalf("dispatch reservation was not durable: %+v %v", stored, err)
	}
}

func TestAggregateLimitsRetainUnknownReservationAndSpendOnlyRemainder(t *testing.T) {
	limits := reviewedLimits()
	item := backlogItem{Kind: "execute", Name: "aggregate", ExecutionLimits: limits, PlanAcceptance: &planAcceptance{SubjectVersion: "subject", PlanContentHash: "plan"}}
	approval := digestStrings("subject", "plan")
	newRecord := func() Record {
		return Record{ExecutionID: "next", BacklogKind: item.Kind, BacklogName: item.Name, MaxSlices: 64, ExecutionLimits: limits.Clone(), ApprovalDigest: approval}
	}
	prior := newRecord()
	prior.ExecutionID = "prior"
	service := &Service{}
	if err := service.prepareExecutionGrantLocked(t.Context(), nil, &prior, item); err != nil {
		t.Fatal(err)
	}
	prior.SettledUsage = &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, Tokens: 500000, Turns: 100, WallSeconds: 600, ChargeMicroUSD: 1000000, Children: 10, NodeAttempts: 20, Retries: 2, Slices: 7}
	next := newRecord()
	if err := service.prepareExecutionGrantLocked(t.Context(), []Record{prior}, &next, item); err != nil {
		t.Fatal(err)
	}
	grant := next.WorkflowGrant
	if grant.MaxTokens != 1500000 || grant.MaxTurns != 900 || grant.MaxWallTimeSeconds != 604200 || grant.MaxChargeMicroUSD != 29000000 || grant.MaxChildren != 246 || grant.MaxNodeAttempts != 492 || grant.MaxRetries != 61 || next.MaxSlices != 57 {
		t.Fatalf("retry restored spent budget: %+v slices=%d", grant, next.MaxSlices)
	}
	// A fresh service after restart consumes the same persisted owner receipt.
	replayed := newRecord()
	if err := (&Service{}).prepareExecutionGrantLocked(t.Context(), []Record{prior}, &replayed, item); err != nil || workflowcontract.GrantDigest(replayed.WorkflowGrant) != workflowcontract.GrantDigest(next.WorkflowGrant) {
		t.Fatalf("restart changed remainder: %v", err)
	}
	prior.SettledUsage = nil
	next = newRecord()
	if err := service.prepareExecutionGrantLocked(t.Context(), []Record{prior}, &next, item); err == nil || !strings.Contains(err.Error(), "unresolved dispatch reservation") || next.WorkflowGrant != nil {
		t.Fatalf("unknown receipt released reservation: %+v %v", next, err)
	}
}

func TestQueueWithoutReviewedLimitsKeepsSixSliceDefault(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "execute", "ordinary", map[string]any{"name": "ordinary", "title": "Ordinary", "status": "ready"})
	service := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, "executions.json"), PlanRenderer: testPlanRenderer(), PhasedPlanWorkflow: &stubPhasedPlanWorkflow{}, TransitionRegistry: testTransitionRegistry(t)})
	if _, err := service.QueueBacklog(t.Context(), CreateRequest{BacklogKind: "execute", BacklogName: "ordinary", Mode: ModeManual, MaxSlices: 7}); err == nil {
		t.Fatal("unreviewed larger allowance admitted")
	}
	queued, err := service.QueueBacklog(t.Context(), CreateRequest{BacklogKind: "execute", BacklogName: "ordinary", Mode: ModeManual})
	if err != nil || queued.MaxSlices != 6 || queued.ExecutionLimits != nil || queued.WorkflowGrant != nil {
		t.Fatalf("default allowance changed: %+v %v", queued, err)
	}
}

func TestExternalRetriesConsumeReviewedAllowanceWithoutRestoringInternalRetries(t *testing.T) {
	limits := reviewedLimits()
	limits.MaxRetries = 2
	item := backlogItem{Kind: "execute", Name: "retry-count", ExecutionLimits: limits, PlanAcceptance: &planAcceptance{SubjectVersion: "subject", PlanContentHash: "plan"}}
	newRetry := func(id string) Record {
		return Record{ExecutionID: id, BacklogKind: item.Kind, BacklogName: item.Name, Operation: "retry", ExecutionLimits: limits.Clone(), ApprovalDigest: digestStrings("subject", "plan")}
	}
	service := &Service{}
	original := newRetry("original")
	original.Operation = "generator"
	if err := service.prepareExecutionGrantLocked(t.Context(), nil, &original, item); err != nil {
		t.Fatal(err)
	}
	original.SettledUsage = &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, Tokens: 1, Turns: 1, WallSeconds: 1, Children: 1, NodeAttempts: 1, Slices: 1}
	first := newRetry("retry-1")
	if err := service.prepareExecutionGrantLocked(t.Context(), []Record{original}, &first, item); err != nil || first.WorkflowGrant.MaxRetries != 1 {
		t.Fatalf("first outer retry did not consume a retry: %+v %v", first.WorkflowGrant, err)
	}
	first.SettledUsage = &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, Tokens: 1, Turns: 1, WallSeconds: 1, Children: 1, NodeAttempts: 1, Slices: 1}
	second := newRetry("retry-2")
	if err := service.prepareExecutionGrantLocked(t.Context(), []Record{original, first}, &second, item); err != nil || second.WorkflowGrant.MaxRetries != 0 || !second.WorkflowGrant.RetryLimitSet {
		t.Fatalf("last outer retry must run with zero internal retries: %+v %v", second.WorkflowGrant, err)
	}
	// Replaying the durable reservation does not charge its outer retry twice.
	digest := workflowcontract.GrantDigest(second.WorkflowGrant)
	if err := service.prepareExecutionGrantLocked(t.Context(), []Record{original, first, second}, &second, item); err != nil || workflowcontract.GrantDigest(second.WorkflowGrant) != digest {
		t.Fatalf("reservation replay changed allowance: %+v %v", second.WorkflowGrant, err)
	}
	second.SettledUsage = first.SettledUsage
	third := newRetry("retry-3")
	if err := service.prepareExecutionGrantLocked(t.Context(), []Record{original, first, second}, &third, item); err == nil || third.WorkflowGrant != nil {
		t.Fatalf("third outer retry escaped aggregate retry limit: %+v %v", third, err)
	}
	first.SettledUsage = &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, Tokens: 1, Turns: 1, WallSeconds: 1, Retries: 1}
	next := newRetry("retry-after-internal")
	if err := service.prepareExecutionGrantLocked(t.Context(), []Record{original, first}, &next, item); err == nil {
		t.Fatal("owner-internal and Swarm-outer retries did not share the reviewed allowance")
	}
	first.Operation = "generator"
	next.Operation = "generator"
	if err := service.prepareExecutionGrantLocked(t.Context(), []Record{original, first}, &next, item); err == nil {
		t.Fatal("fresh Queue/Start operation labels bypassed the reviewed retry allowance")
	}
}

type uniqueGrantWorkflow struct{ stubPhasedPlanWorkflow }

func (w *uniqueGrantWorkflow) StartWorkflow(ctx context.Context, invocation agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	started, err := w.stubPhasedPlanWorkflow.StartWorkflow(ctx, invocation)
	started.ExecutionID = fmt.Sprintf("wfx-%d", w.startCalls)
	return started, err
}

func TestFreshQueueStartCannotBypassReviewedRetryLimit(t *testing.T) {
	root := t.TempDir()
	limits := reviewedLimits()
	limits.MaxRetries = 1
	item := map[string]any{"name": "requeued", "title": "Same accepted work", "status": "ready", "execution_strategy": adaptiveImprovementStrategy, "execution_limits": limits}
	workflow := &uniqueGrantWorkflow{}
	service := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, "executions.json"), PlanRenderer: testPlanRenderer(), PhasedPlanWorkflow: workflow, TransitionRegistry: testTransitionRegistry(t)})
	for attempt := 0; attempt < 3; attempt++ {
		mustWriteBacklogItem(t, root, "execute", "requeued", item)
		queued, err := service.QueueBacklog(t.Context(), CreateRequest{BacklogKind: "execute", BacklogName: "requeued", Mode: ModeManual})
		if err != nil {
			t.Fatal(err)
		}
		started, err := service.Start(t.Context(), queued.ExecutionID)
		if attempt == 2 {
			if err == nil || !strings.Contains(err.Error(), "aggregate execution allowance is exhausted") {
				t.Fatalf("fresh queue bypassed retries: %+v %v", started, err)
			}
			if workflow.startCalls != 2 {
				t.Fatalf("exhausted queue dispatched %d workflows", workflow.startCalls)
			}
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if started.Operation == "retry" || started.WorkflowGrant.MaxRetries != 1-attempt {
			t.Fatalf("ordinary queue did not consume the same allowance: %+v", started)
		}
		records, err := service.store.Load()
		if err != nil {
			t.Fatal(err)
		}
		for index := range records {
			if records[index].ExecutionID == started.ExecutionID {
				records[index].Status = StatusFailed
				records[index].SettledUsage = &workflowcontract.Usage{TokensKnown: true, ChargeMeasured: true, Tokens: 1, Turns: 1, WallSeconds: 1, Children: 1, NodeAttempts: 1, Slices: 1}
			}
		}
		if err := service.store.Save(records); err != nil {
			t.Fatal(err)
		}
	}
}
