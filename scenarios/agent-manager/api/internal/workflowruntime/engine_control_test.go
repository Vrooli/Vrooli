// Responsibility: retain engine test declarations within their original package.
package workflowruntime

import (
	"agent-manager/internal/domain"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"
)

func TestWaitSignalIsTypedDurableAndIdempotent(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", PayloadSchema: json.RawMessage(`{"type":"object","required":["actor"],"properties":{"actor":{"type":"string"}},"additionalProperties":false}`), TimeoutSeconds: 30}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "wait-1")
	waiting := mustAdvance(t, engine, execution.ID)
	if waiting.Status != domain.WorkflowExecutionWaiting || waiting.BudgetUsage.Turns != 0 {
		t.Fatalf("waiting execution=%+v", waiting)
	}
	if _, _, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{"wrong":true}`), "signal-bad", waiting.Version); err == nil {
		t.Fatal("wrong signal payload accepted")
	}
	signalled, duplicate, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{"actor":"operator"}`), "signal-1", waiting.Version)
	if err != nil || duplicate || signalled.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("signal execution=%+v duplicate=%t err=%v", signalled, duplicate, err)
	}
	if _, duplicate, err = engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{"actor":"operator"}`), "signal-1", 0); err != nil || !duplicate {
		t.Fatalf("duplicate signal duplicate=%t err=%v", duplicate, err)
	}
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("final=%+v", final)
	}
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	if len(journal) < 3 || journal[1].Kind != domain.WorkflowJournalWait || journal[2].Kind != domain.WorkflowJournalSignal {
		t.Fatalf("journal=%+v", journal)
	}
}

func TestSignalBuffersBeforeWaitArmsAndConsumesOnArrival(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "work"}},
		{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", PayloadSchema: json.RawMessage(`{"type":"object","required":["actor"],"properties":{"actor":{"type":"string"}},"additionalProperties":false}`), TimeoutSeconds: 30}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "run"
	definition.Edges = []domain.WorkflowEdge{{From: "run", To: "approval"}, {From: "approval", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "signal-before-wait")
	mustAdvance(t, engine, execution.ID) // persist dispatch intent
	mustAdvance(t, engine, execution.ID) // dispatch run
	children.complete(children.requests[0].runID, "complete")

	// The run has completed, but the nudge has not yet driven the execution to
	// its wait node. The declared contract still accepts and durably buffers the
	// signal.
	buffered, duplicate, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{"actor":"operator"}`), "approval-before-arm", 0)
	if err != nil || duplicate || buffered.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("buffered signal execution=%+v duplicate=%t err=%v", buffered, duplicate, err)
	}

	// Completion progression arms and consumes the buffer without a caller-side
	// advance between the terminal run and Signal.
	advanced := mustAdvance(t, engine, execution.ID)
	if advanced.CurrentNodeID != "approval" || advanced.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("wait did not consume buffered signal: %+v", advanced)
	}
	consumed := mustAdvance(t, engine, execution.ID)
	if consumed.CurrentNodeID != "done" || consumed.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("buffered signal was not consumed: %+v", consumed)
	}
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("final=%+v", final)
	}
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	var signalEntry *domain.WorkflowJournalEntry
	for _, entry := range journal {
		if entry.Kind == domain.WorkflowJournalSignal {
			signalEntry = entry
			break
		}
	}
	if signalEntry == nil || signalEntry.NodeID != "approval" {
		t.Fatalf("buffered signal journal=%+v", journal)
	}
}

func TestBoundedWaitRoutesTimeoutAndPreservesReason(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 10, OnTimeout: "timed-out"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
		{ID: "timed-out", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "blocked"}},
	}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	engine.Now = func() time.Time { return now }
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "wait-timeout")
	if waiting := mustAdvance(t, engine, execution.ID); waiting.Status != domain.WorkflowExecutionWaiting {
		t.Fatalf("waiting=%+v", waiting)
	}
	now = now.Add(11 * time.Second)
	routed := mustAdvance(t, engine, execution.ID)
	if routed.CurrentNodeID != "timed-out" || routed.Status != domain.WorkflowExecutionRunning || routed.TerminalReason == nil || routed.TerminalReason.Code != "wait_timeout" {
		t.Fatalf("routed=%+v", routed)
	}
	terminal := mustAdvance(t, engine, execution.ID)
	if terminal.Status != domain.WorkflowExecutionBlocked || terminal.TerminalReason == nil || terminal.TerminalReason.Code != "wait_timeout" {
		t.Fatalf("terminal=%+v", terminal)
	}
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	if journal[len(journal)-1].Kind != domain.WorkflowJournalWaitTimeout {
		t.Fatalf("timeout journal missing: %+v", journal)
	}
}

func TestIndefiniteWaitAndWallTimeExcludePausedDuration(t *testing.T) {
	definition := baseDefinition()
	definition.Budgets.WallTimeSeconds = 5
	definition.Nodes = []domain.WorkflowNode{
		{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 0}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "done"}}
	engine, _, _ := testEngine(t, definition)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	engine.Now = func() time.Time { return now }
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "indefinite-wait")
	mustAdvance(t, engine, execution.ID)
	now = now.Add(24 * time.Hour)
	stillWaiting := mustAdvance(t, engine, execution.ID)
	if stillWaiting.Status != domain.WorkflowExecutionWaiting {
		t.Fatalf("indefinite wait exhausted wall time: %+v", stillWaiting)
	}
	if _, _, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{}`), "after-day", 0); err != nil {
		t.Fatal(err)
	}
	now = now.Add(6 * time.Second)
	mustAdvance(t, engine, execution.ID)
	terminal := mustAdvance(t, engine, execution.ID)
	if terminal.Status != domain.WorkflowExecutionBudgetExhausted || terminal.TerminalReason == nil || terminal.TerminalReason.BudgetName != "wall_time" {
		t.Fatalf("active time after wait did not exhaust wall budget: %+v", terminal)
	}
}

func TestRevisitedWaitRequiresVisitScopedSignal(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 30}},
		{ID: "loop", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}},
	}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "loop", MaxTraversals: 2}, {From: "loop", To: "approval", MaxTraversals: 1}}
	engine, store, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "wait-revisit")
	firstWait := mustAdvance(t, engine, execution.ID)
	if _, _, err := engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{}`), "approval-1", firstWait.Version); err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID) // consume first wait
	mustAdvance(t, engine, execution.ID) // loop back to approval
	secondWait := mustAdvance(t, engine, execution.ID)
	if secondWait.Status != domain.WorkflowExecutionWaiting {
		t.Fatalf("revisited wait reused prior signal: %#v", secondWait)
	}
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	correlations := []string{}
	for _, entry := range journal {
		if entry.Kind != domain.WorkflowJournalWait {
			continue
		}
		var intent waitIntent
		if json.Unmarshal(entry.Payload, &intent) == nil {
			correlations = append(correlations, intent.CorrelationKey)
		}
	}
	if len(correlations) != 2 || correlations[0] == correlations[1] {
		t.Fatalf("wait correlations are not visit scoped: %v", correlations)
	}
}

func TestCancelWinsAgainstLateSignalAndIsIdempotent(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 30}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "approval"
	definition.Edges = []domain.WorkflowEdge{{From: "approval", To: "done"}}
	engine, _, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "cancel-1")
	waiting := mustAdvance(t, engine, execution.ID)
	cancelled, duplicate, err := engine.Cancel(context.Background(), execution.ID, "cancel-op", "operator request", waiting.Version)
	if err != nil || duplicate || cancelled.Status != domain.WorkflowExecutionCancelling {
		t.Fatalf("cancelled=%+v duplicate=%t err=%v", cancelled, duplicate, err)
	}
	if advanced, err := engine.Advance(context.Background(), execution.ID); err != nil || advanced.Status != domain.WorkflowExecutionCancelling {
		t.Fatalf("advance after cancellation=%+v err=%v; cancelling must be a hard barrier until cleanup settles it", advanced, err)
	}
	cancelled, err = engine.RecordCancellationDisposition(context.Background(), execution.ID, 0, 0, nil)
	if err != nil || cancelled.Status != domain.WorkflowExecutionCancelled {
		t.Fatalf("cancellation cleanup=%+v err=%v", cancelled, err)
	}
	if _, duplicate, err = engine.Cancel(context.Background(), execution.ID, "cancel-op", "operator request", 0); err != nil || !duplicate {
		t.Fatalf("duplicate cancel duplicate=%t err=%v", duplicate, err)
	}
	if _, _, err = engine.Signal(context.Background(), execution.ID, "approved", json.RawMessage(`{}`), "late-signal", 0); err == nil {
		t.Fatal("late signal changed cancelled execution")
	}
}

func TestCancellationRemainsRecoverableUntilChildCleanupSucceeds(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approved", TimeoutSeconds: 30}}}
	definition.EntryNode = "approval"
	engine, store, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "cancel-cleanup")
	waiting := mustAdvance(t, engine, execution.ID)
	if _, _, err := engine.Cancel(context.Background(), execution.ID, "cancel-cleanup", "operator request", waiting.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.RecordCleanupDisposition(context.Background(), execution.ID, 0, 0, []string{"child still running"}); err == nil {
		t.Fatal("incomplete cleanup was accepted")
	}
	persisted, _ := store.Get(context.Background(), execution.ID)
	if persisted.Status != domain.WorkflowExecutionCancelling || persisted.EndedAt != nil || persisted.BudgetUsage.AccountingComplete {
		t.Fatalf("incomplete cancellation became terminal: %#v", persisted)
	}
	completed, err := engine.RecordCleanupDisposition(context.Background(), execution.ID, 1, 0, nil)
	if err != nil || completed.Status != domain.WorkflowExecutionCancelled || completed.EndedAt == nil {
		t.Fatalf("completed cancellation cleanup=%#v err=%v", completed, err)
	}
}

func TestChildWorkflowPersistsIdentityAndAggregatesBudget(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "child", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "example/child", MaxDepth: 2}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "child"
	definition.Edges = []domain.WorkflowEdge{{From: "child", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	children := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = children
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parent-1")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.starts) != 1 {
		t.Fatalf("child starts=%d", len(children.starts))
	}
	childID := children.byKey[children.starts[0].IdempotencyKey]
	children.states[childID] = SubworkflowState{ExecutionID: childID, Terminal: true, Output: json.RawMessage(`{"ok":true}`), BudgetUsage: domain.WorkflowBudgetUsage{Turns: 2, Tokens: 30, NodeAttempts: 1}}
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded || final.BudgetUsage.Turns != 2 || final.BudgetUsage.Tokens != 30 {
		t.Fatalf("final=%+v", final)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 1 || attempts[0].ChildExecutionID == nil || *attempts[0].ChildExecutionID != childID {
		t.Fatalf("attempts=%+v", attempts)
	}
}

func TestChildWorkflowPreservesDistinctTerminalStatus(t *testing.T) { // [REQ:REQ-P2-001]
	for _, want := range []domain.WorkflowExecutionStatus{domain.WorkflowExecutionBlocked, domain.WorkflowExecutionAbstained} {
		t.Run(string(want), func(t *testing.T) {
			definition := baseDefinition()
			definition.Nodes = []domain.WorkflowNode{{ID: "child", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "example/child", MaxDepth: 2}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
			definition.EntryNode = "child"
			definition.Edges = []domain.WorkflowEdge{{From: "child", To: "done"}}
			engine, _, _ := testEngine(t, definition)
			children := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
			engine.Subworkflows = children
			execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "child-terminal-"+string(want))
			if err != nil {
				t.Fatal(err)
			}
			mustAdvance(t, engine, execution.ID)
			mustAdvance(t, engine, execution.ID)
			childID := children.byKey[children.starts[0].IdempotencyKey]
			children.states[childID] = SubworkflowState{ExecutionID: childID, Terminal: true, Status: want, Output: json.RawMessage(`{"reason":"child stopped"}`), TerminalReason: &domain.WorkflowTerminalReason{Code: string(want)}}
			final := mustAdvance(t, engine, execution.ID)
			if final.Status != want || string(final.Output) != `{"reason":"child stopped"}` {
				t.Fatalf("parent terminal=%+v", final)
			}
		})
	}
}

func TestParallelBranchDispatchesDistinctProfilesAndJoins(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "research", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{ProfileKey: "researcher", PromptTemplate: "research"}},
		{ID: "review", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{ProfileKey: "reviewer", PromptTemplate: "review"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "all"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{{From: "fanout", To: "research"}, {From: "fanout", To: "review"}, {From: "research", To: "joined"}, {From: "review", To: "joined"}, {From: "joined", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-1")
	mustAdvance(t, engine, execution.ID) // atomically persist membership and intents
	mustAdvance(t, engine, execution.ID) // dispatch member 1
	mustAdvance(t, engine, execution.ID) // dispatch member 2
	if len(children.requests) != 2 || children.requests[0].runID == children.requests[1].runID {
		t.Fatalf("parallel runs=%+v", children.requests)
	}
	profiles := map[string]bool{children.requests[0].profile: true, children.requests[1].profile: true}
	if !profiles["researcher"] || !profiles["reviewer"] {
		t.Fatalf("profiles=%v", profiles)
	}
	children.complete(children.requests[0].runID, "research handoff")
	children.complete(children.requests[1].runID, "review handoff")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	final := mustAdvance(t, engine, execution.ID)
	if final.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("final=%+v", final)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 2 {
		t.Fatalf("attempts=%+v", attempts)
	}
}

func TestParallelFanoutWiderThanConcurrencyDispatchesInBatches(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "one", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "one"}},
		{ID: "two", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "two"}},
		{ID: "three", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "three"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "all"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{{From: "fanout", To: "one"}, {From: "fanout", To: "two"}, {From: "fanout", To: "three"}, {From: "one", To: "joined"}, {From: "two", To: "joined"}, {From: "three", To: "joined"}, {From: "joined", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-batches")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 {
		t.Fatalf("first batch size=%d, want maxConcurrency=2", len(children.requests))
	}
	children.complete(children.requests[0].runID, "done")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 3 {
		t.Fatalf("third member was not dispatched after capacity opened: %d", len(children.requests))
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	if len(attempts) != 3 {
		t.Fatalf("fanout membership=%d, want 3", len(attempts))
	}
}

func TestParallelAnyJoinDoesNotWaitForHungLoser(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "winner", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "winner"}},
		{ID: "loser", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "loser"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "any"}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{{From: "fanout", To: "winner"}, {From: "fanout", To: "loser"}, {From: "winner", To: "joined"}, {From: "loser", To: "joined"}, {From: "joined", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-any")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	children.complete(children.requests[0].runID, "winner")
	mustAdvance(t, engine, execution.ID) // persist winner
	joined := mustAdvance(t, engine, execution.ID)
	if joined.CurrentNodeID != "joined" {
		t.Fatalf("any join waited for loser: %#v", joined)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	shortCircuited := false
	for _, attempt := range attempts {
		shortCircuited = shortCircuited || attempt.ErrorCode == "parallel_join_short_circuit"
	}
	if !shortCircuited {
		t.Fatalf("loser was not durably short-circuited: %+v", attempts)
	}
}

func TestParallelQuorumJoinDoesNotWaitForRemainingMember(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "one", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "one"}},
		{ID: "two", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "two"}},
		{ID: "three", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "three"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "quorum", Quorum: 2}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{{From: "fanout", To: "one"}, {From: "fanout", To: "two"}, {From: "fanout", To: "three"}, {From: "one", To: "joined"}, {From: "two", To: "joined"}, {From: "three", To: "joined"}, {From: "joined", To: "done"}}
	engine, store, children := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-quorum")
	mustAdvance(t, engine, execution.ID) // persist all three members
	mustAdvance(t, engine, execution.ID) // dispatch first member
	mustAdvance(t, engine, execution.ID) // dispatch second member; third remains pending
	children.complete(children.requests[0].runID, "one")
	children.complete(children.requests[1].runID, "two")
	mustAdvance(t, engine, execution.ID)           // persist first completion
	mustAdvance(t, engine, execution.ID)           // use the newly free slot before the second completion is observed
	mustAdvance(t, engine, execution.ID)           // persist second completion
	joined := mustAdvance(t, engine, execution.ID) // short-circuit the remaining member
	if joined.CurrentNodeID != "joined" {
		t.Fatalf("quorum join waited for remaining member: %#v", joined)
	}
	attempts, _ := store.ListAttempts(context.Background(), execution.ID)
	shortCircuited := false
	for _, attempt := range attempts {
		shortCircuited = shortCircuited || attempt.ErrorCode == "parallel_join_short_circuit"
	}
	if !shortCircuited {
		t.Fatalf("remaining member was not durably short-circuited: %+v", attempts)
	}
}

func TestParallelBranchRevisitCreatesFreshVisitAttempts(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "fanout", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "left", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "left"}},
		{ID: "right", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "right"}},
		{ID: "joined", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "all"}},
		{ID: "again", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "fanout"
	definition.Edges = []domain.WorkflowEdge{
		{From: "fanout", To: "left", MaxTraversals: 2},
		{From: "fanout", To: "right", MaxTraversals: 2},
		{From: "left", To: "joined", MaxTraversals: 2},
		{From: "right", To: "joined", MaxTraversals: 2},
		{From: "joined", To: "again", MaxTraversals: 2},
		{From: "again", To: "fanout", Condition: "iteration < 4", MaxTraversals: 1},
		{From: "again", To: "done"},
	}
	engine, store, children := testEngine(t, definition)
	execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "parallel-revisit")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	children.complete(children.requests[0].runID, "left-1")
	children.complete(children.requests[1].runID, "right-1")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	expressions, err := NewExpressionEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	engine = &Engine{Store: store, Catalog: fakeCatalog{revision(definition)}, Children: children, Expressions: expressions}
	mustAdvance(t, engine, execution.ID)

	attempts, err := store.ListAttempts(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 4 {
		t.Fatalf("parallel revisit attempts=%d, want 4: %+v", len(attempts), attempts)
	}
	ordinals := map[string][]int{}
	keys := map[string]bool{}
	for _, attempt := range attempts {
		ordinals[attempt.NodeID] = append(ordinals[attempt.NodeID], attempt.Ordinal)
		if keys[attempt.IdempotencyKey] {
			t.Fatalf("parallel revisit reused idempotency key %q", attempt.IdempotencyKey)
		}
		keys[attempt.IdempotencyKey] = true
	}
	for _, nodeID := range []string{"left", "right"} {
		if got := ordinals[nodeID]; len(got) != 2 || got[0] != 1 || got[1] != 2 {
			t.Fatalf("%s ordinals=%v, want [1 2]", nodeID, got)
		}
	}
}

func TestEnginePreservesAuthoredTerminalOutcome(t *testing.T) { // [REQ:REQ-P2-001]
	for _, tc := range []struct {
		authored string
		want     domain.WorkflowExecutionStatus
	}{
		{authored: "blocked", want: domain.WorkflowExecutionBlocked},
		{authored: "abstained", want: domain.WorkflowExecutionAbstained},
	} {
		t.Run(tc.authored, func(t *testing.T) {
			definition := baseDefinition()
			definition.Nodes = []domain.WorkflowNode{{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: tc.authored}}}
			definition.EntryNode = "done"
			engine, _, _ := testEngine(t, definition)
			execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "terminal-"+tc.authored)
			if err != nil {
				t.Fatal(err)
			}
			final := mustAdvance(t, engine, execution.ID)
			if final.Status != tc.want {
				t.Fatalf("status=%s, want %s", final.Status, tc.want)
			}
		})
	}
}

func TestAgentNodeLimitsReachFreshAndContinuationRequests(t *testing.T) { // [REQ:REQ-P2-001]
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{
		{ID: "initial", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{RoleRef: "code.default", PromptTemplate: "initial", MaxTurns: 7, TimeoutSeconds: 90}},
		{ID: "followup", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "initial", PromptTemplate: "follow up", MaxTurns: 2, TimeoutSeconds: 30}},
		{ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	definition.EntryNode = "initial"
	definition.Edges = []domain.WorkflowEdge{{From: "initial", To: "followup"}, {From: "followup", To: "done"}}
	engine, _, children := testEngine(t, definition)
	execution, err := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "node-limits")
	if err != nil {
		t.Fatal(err)
	}
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	children.complete(children.requests[0].runID, "initial")
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	mustAdvance(t, engine, execution.ID)
	if len(children.requests) != 2 {
		t.Fatalf("requests=%d, want 2", len(children.requests))
	}
	if got := children.requests[0]; got.maxTurns != 7 || got.timeout != 90*time.Second {
		t.Fatalf("fresh limits=(%d,%s), want (7,1m30s)", got.maxTurns, got.timeout)
	}
	if got := children.requests[1]; got.maxTurns != 2 || got.timeout != 30*time.Second {
		t.Fatalf("continuation limits=(%d,%s), want (2,30s)", got.maxTurns, got.timeout)
	}
}

func TestConcurrentSignalAndCancelCommitExactlyOneOperation(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "gate", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "go", TimeoutSeconds: 30}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "gate"
	definition.Edges = []domain.WorkflowEdge{{From: "gate", To: "done"}}
	engine, store, _ := testEngine(t, definition)
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "race-1")
	waiting := mustAdvance(t, engine, execution.ID)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, _, _ = engine.Signal(context.Background(), execution.ID, "go", json.RawMessage(`{}`), "race-signal", waiting.Version)
	}()
	go func() {
		defer wg.Done()
		<-start
		_, _, _ = engine.Cancel(context.Background(), execution.ID, "race-cancel", "race", waiting.Version)
	}()
	close(start)
	wg.Wait()
	journal, _ := store.ListJournal(context.Background(), execution.ID, 0, 0)
	operations := 0
	for _, entry := range journal {
		if entry.Kind == domain.WorkflowJournalSignal || entry.Kind == domain.WorkflowJournalCancel {
			operations++
		}
	}
	if operations != 1 {
		t.Fatalf("operations=%d journal=%+v", operations, journal)
	}
}

func TestSubworkflowRecoveryReusesPersistedChildIntent(t *testing.T) {
	definition := baseDefinition()
	definition.Nodes = []domain.WorkflowNode{{ID: "child", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "example/child", MaxDepth: 2}}, {ID: "done", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}}}
	definition.EntryNode = "child"
	definition.Edges = []domain.WorkflowEdge{{From: "child", To: "done"}}
	engine, store, runs := testEngine(t, definition)
	children := &fakeSubworkflows{states: map[uuid.UUID]SubworkflowState{}, byKey: map[string]uuid.UUID{}}
	engine.Subworkflows = children
	execution, _ := engine.Start(context.Background(), revision(definition), json.RawMessage(`{}`), "child-recovery")
	mustAdvance(t, engine, execution.ID)
	expressions, _ := NewExpressionEvaluator()
	restarted := &Engine{Store: store, Catalog: fakeCatalog{revision(definition)}, Children: runs, Subworkflows: children, Expressions: expressions}
	mustAdvance(t, restarted, execution.ID)
	mustAdvance(t, restarted, execution.ID)
	if len(children.starts) != 1 {
		t.Fatalf("child dispatches=%d", len(children.starts))
	}
}

func TestBindingsAreBoundedAndDoNotExposeTranscript(t *testing.T) { // [REQ:REQ-P2-001]
	ctx := BindingContext{Input: json.RawMessage(`{"topic":"x"}`)}
	binding := inputBinding("topic", "$.topic")
	values, err := EvaluateBindings([]domain.WorkflowInputBinding{binding}, ctx)
	if err != nil || values["topic"] != "x" {
		t.Fatalf("values=%v err=%v", values, err)
	}
	binding.Source = domain.WorkflowBindingSource("transcript")
	if _, err := EvaluateBindings([]domain.WorkflowInputBinding{binding}, ctx); err == nil {
		t.Fatal("transcript source accepted")
	}
	binding = inputBinding("topic", "$.topic")
	binding.MaxBytes = 1
	if _, err := EvaluateBindings([]domain.WorkflowInputBinding{binding}, ctx); err == nil {
		t.Fatal("oversized binding accepted")
	}
}

func TestBindingsSelectBoundedChildWorkflowOutput(t *testing.T) { // [REQ:REQ-P2-001]
	payload := json.RawMessage(`{"childExecutionId":"child-1","status":"succeeded","output":{"review":{"accepted":false,"note":"tighten the handoff"}}}`)
	ctx := BindingContext{Journal: []*domain.WorkflowJournalEntry{{Kind: domain.WorkflowJournalChild, NodeID: "review", Sequence: 1, Payload: payload}}}
	binding := domain.WorkflowInputBinding{Name: "review_note", Source: domain.WorkflowBindingChild, Selector: "node=review;$.output.review.note", Order: "desc", Limit: 1, MaxBytes: 128, RenderAs: "text", MissingPolicy: "error"}
	values, err := EvaluateBindings([]domain.WorkflowInputBinding{binding}, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if values["review_note"] != "tighten the handoff" {
		t.Fatalf("child output binding=%#v", values["review_note"])
	}
}
