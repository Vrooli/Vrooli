package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"
	"agent-manager/internal/workflowruntime"
	"github.com/google/uuid"
)

type sameRunContinuationLauncher struct{ *fakeRunLauncher }

func (l sameRunContinuationLauncher) Continue(_ context.Context, req workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := *req.SourceRunID
	l.byKey[req.IdempotencyKey] = id
	state := l.states[id]
	state.Terminal, state.TokensKnown, state.ChargeMeasured = false, false, false
	l.states[id] = state
	return state, nil
}

func TestSameRunContinuationUsageConservedThroughNormalAndRecovery(t *testing.T) {
	for _, mode := range []string{"normal", "successful-late", "cancelled-late"} {
		t.Run(mode, func(t *testing.T) {
			launcher := newFakeRunLauncher()
			o, repos := newRelayOrchestrator(t, launcher)
			o.workflowEngine.Children = sameRunContinuationLauncher{launcher}
			revision := relayDefinition()
			revision.Definition.Nodes[1] = domain.WorkflowNode{ID: "b", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "a", PromptTemplate: "correct"}}
			if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{revision}); err != nil {
				t.Fatal(err)
			}
			x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: mode})
			if err != nil {
				t.Fatal(err)
			}
			id := runIDForNode(t, repos.WorkflowExecutions, x.ID, "a")
			launcher.complete(id, map[string]any{"step": "a"})
			state := launcher.states[id]
			state.Tokens, state.Turns, state.TokensKnown, state.ChargeMeasured = 10, 1, true, true
			launcher.states[id] = state
			if _, err := o.driveWorkflowExecution(t.Context(), x.ID); err != nil {
				t.Fatal(err)
			}
			if got := runIDForNode(t, repos.WorkflowExecutions, x.ID, "b"); got != id {
				t.Fatalf("continuation replaced original run: %s != %s", got, id)
			}
			before, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
			if before.BudgetUsage.Tokens != 10 {
				t.Fatalf("first receipt=%+v", before.BudgetUsage)
			}
			launcher.complete(id, map[string]any{"step": "b"})
			state = launcher.states[id]
			state.Tokens, state.Turns = 15, 2 // owner cumulative 10 + correction 5
			state.TokensKnown, state.ChargeMeasured = mode == "normal", mode == "normal"
			launcher.states[id] = state
			var finished *domain.WorkflowExecution
			if mode == "cancelled-late" {
				if _, err := o.CancelWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: x.ID, IdempotencyKey: "cancel", Reason: "stop correction"}); err == nil {
					t.Fatal("old receipt settled unfinished correction")
				}
			} else {
				finished, err = o.driveWorkflowExecution(t.Context(), x.ID)
				if err != nil || finished.Status != domain.WorkflowExecutionSucceeded || finished.BudgetUsage.Tokens != 15 {
					t.Fatalf("normal advancement duplicated cumulative usage: %+v %v", finished, err)
				}
			}
			restarted, _ := reopenRelayOrchestrator(t, repos, launcher)
			restarted.workflowEngine.Children = sameRunContinuationLauncher{launcher}
			if mode != "normal" {
				if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
					t.Fatal(err)
				}
				pending, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
				if pending.BudgetUsage.AccountingComplete {
					t.Fatal("unknown correction receipt marked complete")
				}
				state = launcher.states[id]
				state.TokensKnown, state.ChargeMeasured = true, true
				launcher.states[id] = state
			}
			if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
				t.Fatal(err)
			}
			settled, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
			if !settled.BudgetUsage.AccountingComplete || settled.BudgetUsage.Tokens != 15 || settled.BudgetUsage.Turns != 2 || settled.BudgetUsage.Children != 2 || settled.BudgetUsage.NodeAttempts != 2 {
				t.Fatalf("recovery duplicated cumulative usage: %+v", settled)
			}
			if finished != nil && (!settled.EndedAt.Equal(*finished.EndedAt) || !bytes.Equal(settled.Output, finished.Output)) {
				t.Fatal("accounting recovery changed original success")
			}
			if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
				t.Fatal(err)
			}
			again, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
			if again.Version != settled.Version || again.BudgetUsage != settled.BudgetUsage || len(launcher.states) != 1 || len(launcher.byKey) != 2 {
				t.Fatal("replay redispatched or duplicated usage")
			}
		})
	}
}

func TestWorkflowRecoverySettlesSuccessfulLateAccountingWithoutChangingOutcome(t *testing.T) {
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{relayDefinition()}); err != nil {
		t.Fatal(err)
	}
	x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "success-late-accounting"})
	if err != nil {
		t.Fatal(err)
	}
	a := runIDForNode(t, repos.WorkflowExecutions, x.ID, "a")
	launcher.complete(a, map[string]any{"step": "a"})
	// Accounting reads cannot advance an unfinished child into its next run.
	if state, err := (workflowSubworkflowLauncher{o: o}).InspectTerminalAccounting(t.Context(), x.ID); err != nil || state.Terminal || len(launcher.byKey) != 1 {
		t.Fatalf("accounting read drove unfinished work: %+v %v", state, err)
	}
	if _, err := o.driveWorkflowExecution(t.Context(), x.ID); err != nil {
		t.Fatal(err)
	}
	b := runIDForNode(t, repos.WorkflowExecutions, x.ID, "b")
	launcher.complete(b, map[string]any{"step": "b"})
	finished, err := o.driveWorkflowExecution(t.Context(), x.ID)
	if err != nil || finished.Status != domain.WorkflowExecutionSucceeded || finished.BudgetUsage.AccountingComplete || finished.EndedAt == nil {
		t.Fatalf("success fixture: %+v %v", finished, err)
	}
	restarted, _ := reopenRelayOrchestrator(t, repos, launcher)
	if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
		t.Fatal(err)
	}
	unknown, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
	if unknown.BudgetUsage.AccountingComplete || unknown.Status != finished.Status || !unknown.EndedAt.Equal(*finished.EndedAt) {
		t.Fatal("unknown successful accounting changed original outcome")
	}
	launcher.mu.Lock()
	for id, state := range launcher.states {
		state.TokensKnown, state.ChargeMeasured, state.Tokens, state.Turns = true, true, 15, 2
		launcher.states[id] = state
	}
	launcher.mu.Unlock()
	if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
		t.Fatal(err)
	}
	settled, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
	if settled.Status != domain.WorkflowExecutionSucceeded || !settled.EndedAt.Equal(*finished.EndedAt) || !bytes.Equal(settled.Output, finished.Output) || !settled.BudgetUsage.AccountingComplete || settled.BudgetUsage.Tokens != 30 || settled.BudgetUsage.Turns != 4 || settled.BudgetUsage.Children != 2 || settled.BudgetUsage.NodeAttempts != 2 {
		t.Fatalf("successful receipt not reconciled in place: %+v", settled)
	}
	if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
		t.Fatal(err)
	}
	again, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
	if again.Version != settled.Version || again.BudgetUsage != settled.BudgetUsage || len(launcher.byKey) != 2 {
		t.Fatal("successful recovery replay changed usage or started work")
	}
	remaining, err := repos.WorkflowExecutions.ListRecoverable(t.Context(), 100)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("settled success stayed recoverable: %+v %v", remaining, err)
	}
}

func TestOrdinaryWorkflowInspectionUsesTerminalOwnerReceipt(t *testing.T) {
	o, repos := newRelayOrchestrator(t, newFakeRunLauncher())
	events := mocks.NewFakeEventStore()
	WithEvents(events)(o)
	task := &domain.Task{ID: uuid.New(), Title: "terminal receipt", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusCancelled, Phase: domain.RunPhaseCompleted, CreatedAt: now.Add(-time.Minute), EndedAt: &now, Summary: &domain.RunSummary{TokensUsed: 900, TurnsUsed: 8}}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	launcher := workflowChildLauncher{o: o}
	unknown, err := launcher.Inspect(t.Context(), run.ID)
	if err != nil || !unknown.Terminal || unknown.TokensKnown || unknown.ChargeMeasured {
		t.Fatalf("summary substituted for absent final owner receipt: %+v %v", unknown, err)
	}
	zero := int64(0)
	if err := events.Append(t.Context(), run.ID,
		&domain.RunEvent{ID: uuid.New(), RunID: run.ID, EventType: domain.EventTypeMetric, Timestamp: now, Data: &domain.UsageEventData{InputTokens: 12, OutputTokens: 9, Turns: 2, ReconciliationAuthority: true}},
		&domain.RunEvent{ID: uuid.New(), RunID: run.ID, EventType: domain.EventTypeMetric, Timestamp: now, Data: &domain.ChargeEventData{Basis: domain.ChargeBasisMetered, AmountMicroUSD: &zero}},
	); err != nil {
		t.Fatal(err)
	}
	known, err := launcher.Inspect(t.Context(), run.ID)
	if err != nil || !known.Terminal || !known.TokensKnown || !known.ChargeMeasured || known.Tokens != 21 || known.Turns != 2 || known.ChargeMicroUSD != 0 {
		t.Fatalf("late terminal receipt not visible on default inspection: %+v %v", known, err)
	}
}

func TestWorkflowRecoverySettlesLateCancellationAccountingWithoutRedispatch(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		launcher := newFakeRunLauncher()
		o, repos := newRelayOrchestrator(t, launcher)
		if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{relayDefinition()}); err != nil {
			t.Fatal(err)
		}
		x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "cancel-recovery"})
		if err != nil {
			t.Fatal(err)
		}
		runID := runIDForNode(t, repos.WorkflowExecutions, x.ID, "a")
		if _, err := o.CancelWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: x.ID, IdempotencyKey: "cancel-original", Reason: "operator"}); err == nil {
			t.Fatal("stop acknowledgement published complete cancellation")
		}
		pending, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
		if pending.Status != domain.WorkflowExecutionCancelling || pending.EndedAt != nil {
			t.Fatalf("unknown tail became terminal: %+v", pending)
		}
		var oldEnd time.Time
		if legacy {
			// Old owner wrote cleanup and failed the attempt before final usage.
			// The periodic repository query must still select that original row.
			oldEnd = time.Now().UTC()
			pending.Status, pending.EndedAt = domain.WorkflowExecutionCancelled, &oldEnd
			pending.BudgetUsage.AccountingComplete = false
			pending.BudgetUsage.Tokens = 7
			pending.Version++
			attempts, _ := repos.WorkflowExecutions.ListAttempts(t.Context(), x.ID)
			attempts[0].Status, attempts[0].Version = domain.WorkflowAttemptFailed, attempts[0].Version+1
			journal, _ := repos.WorkflowExecutions.ListJournal(t.Context(), x.ID, 0, 0)
			entry := &domain.WorkflowJournalEntry{ID: uuid.New(), ExecutionID: x.ID, Sequence: int64(len(journal) + 1), Kind: domain.WorkflowJournalCleanup, Payload: json.RawMessage(`{"retry":0,"stoppedRuns":1}`), CreatedAt: oldEnd}
			if ok, err := repos.WorkflowExecutions.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: pending.Version - 1, Execution: pending, Attempt: attempts[0], Journal: []*domain.WorkflowJournalEntry{entry}}); err != nil || !ok {
				t.Fatalf("legacy fixture: %t %v", ok, err)
			}
		}
		restarted, _ := reopenRelayOrchestrator(t, repos, launcher)
		if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
			t.Fatal(err)
		}
		unknown, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
		if unknown.Status.Terminal() && unknown.BudgetUsage.AccountingComplete {
			t.Fatal("recovery replaced unknown final usage with zero")
		}
		launcher.mu.Lock()
		state := launcher.states[runID]
		state.TokensKnown, state.ChargeMeasured, state.Tokens, state.Turns = true, true, 23, 2
		launcher.states[runID] = state
		launcher.mu.Unlock()
		// Invoke the real periodic/startup owner entry, not the engine helper.
		if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
			t.Fatal(err)
		}
		settled, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
		if settled.Status != domain.WorkflowExecutionCancelled || !settled.BudgetUsage.AccountingComplete || settled.BudgetUsage.Tokens != 23 || settled.BudgetUsage.Turns != 2 || settled.BudgetUsage.Children != 1 {
			t.Fatalf("legacy=%t: late owner accounting not recovered: %+v", legacy, settled)
		}
		if legacy && !settled.EndedAt.Equal(oldEnd) {
			t.Fatal("accounting recovery rewrote original terminal time")
		}
		if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
			t.Fatal(err)
		}
		again, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
		if again.Version != settled.Version || again.BudgetUsage != settled.BudgetUsage || len(launcher.byKey) != 1 {
			t.Fatal("periodic recovery duplicated receipt or dispatched another node")
		}
		remaining, err := repos.WorkflowExecutions.ListRecoverable(t.Context(), 100)
		if err != nil || len(remaining) != 0 {
			t.Fatalf("settled execution stayed in recovery: %+v %v", remaining, err)
		}
	}
}

func TestWorkflowRecoverySettlesNestedOrdinaryCancellationAccounting(t *testing.T) {
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	child := relayDefinition()
	child.ID, child.Key, child.Definition.Key, child.Digest = uuid.New(), "owner/child", "owner/child", "sha256:child"
	parent := relayDefinition()
	parent.Definition.Nodes = []domain.WorkflowNode{{ID: "review", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "owner/child", Version: "1.0.0", MaxDepth: 2}}}
	parent.Definition.EntryNode, parent.Definition.Edges = "review", nil
	if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{parent, child}); err != nil {
		t.Fatal(err)
	}
	o.workflowEngine.Subworkflows = workflowSubworkflowLauncher{o: o}
	x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: json.RawMessage(`{}`), IdempotencyKey: "nested-cancel"})
	if err != nil {
		t.Fatal(err)
	}
	parentAttempts, _ := repos.WorkflowExecutions.ListAttempts(t.Context(), x.ID)
	if len(parentAttempts) != 1 || parentAttempts[0].ChildExecutionID == nil {
		t.Fatal("nested owner fixture did not dispatch")
	}
	childID := *parentAttempts[0].ChildExecutionID
	runID := runIDForNode(t, repos.WorkflowExecutions, childID, "a")
	if _, err := o.CancelWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: x.ID, IdempotencyKey: "cancel-nested", Reason: "operator"}); err == nil {
		t.Fatal("nested stop acknowledgement settled unknown child usage")
	}
	launcher.mu.Lock()
	state := launcher.states[runID]
	state.TokensKnown, state.ChargeMeasured, state.Tokens, state.Turns = true, true, 31, 3
	launcher.states[runID] = state
	launcher.mu.Unlock()
	restarted, _ := reopenRelayOrchestrator(t, repos, launcher)
	restarted.workflowEngine.Subworkflows = workflowSubworkflowLauncher{o: restarted}
	if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
		t.Fatal(err)
	}
	settled, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
	if settled.Status != domain.WorkflowExecutionCancelled || !settled.BudgetUsage.AccountingComplete || settled.BudgetUsage.Tokens != 31 || settled.BudgetUsage.Children != 2 || settled.BudgetUsage.NodeAttempts != 2 {
		t.Fatalf("nested final accounting was not collected: %+v", settled)
	}
	if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
		t.Fatal(err)
	}
	again, _ := repos.WorkflowExecutions.Get(t.Context(), x.ID)
	if again.Version != settled.Version || again.BudgetUsage != settled.BudgetUsage || len(launcher.byKey) != 1 {
		t.Fatal("nested recovery repeated charges or child dispatch")
	}
}
