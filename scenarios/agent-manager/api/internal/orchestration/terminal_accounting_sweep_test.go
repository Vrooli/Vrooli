package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

// The rollout of a codex goal run interrupted mid-turn: codex's own
// turn_aborted closes the turn after two cumulative token counts.
const standaloneCodexAborted = `{"type":"session_meta","payload":{"id":"codex-session"}}
{"type":"event_msg","payload":{"type":"task_started","turn_id":"t1"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":15211,"cached_input_tokens":10880,"output_tokens":205}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":30666,"cached_input_tokens":21760,"output_tokens":415}}}}
{"type":"event_msg","payload":{"type":"turn_aborted","reason":"interrupted"}}
`

// The same turn killed right after a tool result was written: the next request
// may already have been sent, so its usage cannot be known from the transcript.
const standaloneCodexRequestInFlight = `{"type":"session_meta","payload":{"id":"codex-session"}}
{"type":"event_msg","payload":{"type":"task_started","turn_id":"t1"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":15211,"cached_input_tokens":10880,"output_tokens":205}}}}
{"type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"c1","output":"ok"}}
`

// A claude session that never reached the model: Claude's explicit zero cost
// snapshot is its own receipt (the shape of the zero-turn qualification runs).
const standaloneClaudeZero = `{"type":"mode","mode":"normal","sessionId":"claude-session"}
{"type":"cost-state","sessionId":"claude-session","totalCostUSD":0,"totalAPIDuration":0,"totalAPIDurationWithoutRetries":0,"modelUsage":{},"hasUnknownModelCost":false}
`

const standaloneClaudeToolRunning = `{"type":"mode","mode":"normal","sessionId":"claude-session"}
{"type":"assistant","sessionId":"claude-session","message":{"model":"claude-fable-5","id":"msg_1","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"Read","input":{"file_path":"/x"}}],"stop_reason":"tool_use","usage":{"input_tokens":120,"cache_creation_input_tokens":30,"cache_read_input_tokens":5000,"output_tokens":40}}}
`

type standaloneRunFixture struct {
	kind       domain.RunnerType
	sessionID  string
	transcript string
	basis      domain.ChargeBasis
	retained   []*domain.RunEvent
}

func seedStandaloneRun(t *testing.T, fx standaloneRunFixture, events event.Store) (*Orchestrator, uuid.UUID) {
	t.Helper()
	o, repos := newRelayOrchestrator(t, newFakeRunLauncher())
	task := &domain.Task{ID: uuid.New(), Title: "standalone accounting", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	ended := time.Now().UTC().Add(-time.Minute)
	started := ended.Add(-10 * time.Minute)
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusFailed, Phase: domain.RunPhaseCompleted, RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeInteractive, SessionID: fx.sessionID, StartedAt: &started, EndedAt: &ended, FinalizationStatus: domain.RunFinalizationStatusNone, ResolvedConfig: &domain.RunConfig{RunnerType: fx.kind, Model: "retained-model"}, Billing: domain.BillingSnapshot{Basis: fx.basis}, TranscriptPath: filepath.Join(t.TempDir(), "transcript.jsonl")}
	if err := os.WriteFile(run.TranscriptPath, []byte(fx.transcript), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	retained := append([]*domain.RunEvent{domain.NewStatusEvent(run.ID, "starting", "running", "original invocation")}, fx.retained...)
	for _, item := range retained {
		item.RunID = run.ID
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
	}
	if err := events.Append(t.Context(), run.ID, retained...); err != nil {
		t.Fatal(err)
	}
	WithEvents(events)(o)
	registry := runner.NewRegistry()
	codec := core.NewRunner(codecs.NewCodexForTest(), nil, nil)
	if fx.kind == domain.RunnerTypeClaudeCode {
		codec = core.NewRunner(codecs.NewClaudeForTest(), nil, nil)
	}
	if err := registry.Register(codec); err != nil {
		t.Fatal(err)
	}
	o.runners = registry
	o.terminalAccounting.exclude = func(_ context.Context, actual *domain.Run) error {
		if actual.ID != run.ID {
			t.Fatalf("executor proof used run %s", actual.ID)
		}
		return nil
	}
	return o, run.ID
}

func liveUsage(input, output, cached int) *domain.RunEvent {
	return &domain.RunEvent{EventType: domain.EventTypeMetric, Data: &domain.UsageEventData{PayloadKind: domain.PayloadKindUsage, InputTokens: input, OutputTokens: output, CacheReadTokens: cached, TurnIndex: 1, RunnerType: "codex"}}
}

// The shape of the stuck goal-session workflow child 800ae00c: the live stream
// kept two readings but never the abort's receipt. The sweep reads it back.
func TestStandaloneSweepSettlesInterruptedCodexRunFromItsRollout(t *testing.T) {
	events := mocks.NewFakeEventStore()
	o, runID := seedStandaloneRun(t, standaloneRunFixture{kind: domain.RunnerTypeCodex, sessionID: "codex-session", transcript: standaloneCodexAborted, basis: domain.ChargeBasisSubscription, retained: []*domain.RunEvent{liveUsage(4331, 205, 10880), liveUsage(8906, 415, 21760)}}, events)
	before, err := o.RunAccounting(t.Context(), runID)
	if err != nil || !before.Terminal || before.TokensKnown {
		t.Fatalf("fixture accounting = %+v %v, want terminal and unknown", before, err)
	}
	if err := o.RecoverStandaloneTerminalAccounting(t.Context()); err != nil {
		t.Fatal(err)
	}
	after, err := o.RunAccounting(t.Context(), runID)
	if err != nil || !after.TokensKnown || !after.ChargeMeasured || after.Tokens != 31081 || after.ChargeMicroUSD != 0 {
		t.Fatalf("recovered accounting = %+v %v, want known 31081 tokens at a measured zero charge", after, err)
	}
	count, _ := events.Count(t.Context(), runID)
	// A direct replay must neither append nor re-read into a second receipt.
	if err := o.recoverTerminalAccounting(t.Context(), runID, o.terminalAccounting.exclusion()); err != nil {
		t.Fatal(err)
	}
	if err := o.RecoverStandaloneTerminalAccounting(t.Context()); err != nil {
		t.Fatal(err)
	}
	if replay, _ := events.Count(t.Context(), runID); replay != count {
		t.Fatalf("replay appended accounting: %d -> %d", count, replay)
	}
}

func TestStandaloneSweepSettlesZeroTurnClaudeRunAsExplicitZero(t *testing.T) {
	events := mocks.NewFakeEventStore()
	o, runID := seedStandaloneRun(t, standaloneRunFixture{kind: domain.RunnerTypeClaudeCode, sessionID: "claude-session", transcript: standaloneClaudeZero, basis: domain.ChargeBasisSubscription}, events)
	if err := o.RecoverStandaloneTerminalAccounting(t.Context()); err != nil {
		t.Fatal(err)
	}
	got, err := o.RunAccounting(t.Context(), runID)
	if err != nil || !got.TokensKnown || !got.ChargeMeasured || got.Tokens != 0 {
		t.Fatalf("zero-turn accounting = %+v %v, want a known, measured zero", got, err)
	}
}

// Killed while a tool ran: no request was outstanding, so the last response's
// usage closes the turn. Without a pricing service the charge stays unmeasured.
func TestStandaloneSweepClosesClaudeTurnKilledDuringToolUse(t *testing.T) {
	events := mocks.NewFakeEventStore()
	o, runID := seedStandaloneRun(t, standaloneRunFixture{kind: domain.RunnerTypeClaudeCode, sessionID: "claude-session", transcript: standaloneClaudeToolRunning, basis: domain.ChargeBasisUnpriced}, events)
	if err := o.RecoverStandaloneTerminalAccounting(t.Context()); err != nil {
		t.Fatal(err)
	}
	got, err := o.RunAccounting(t.Context(), runID)
	if err != nil || !got.TokensKnown || got.ChargeMeasured || got.Tokens != 5190 {
		t.Fatalf("interrupted-turn accounting = %+v %v, want 5190 known tokens and an unmeasured charge", got, err)
	}
	all, _ := events.Get(t.Context(), runID, event.GetOptions{AfterSequence: -1})
	found := false
	for _, item := range all {
		if usage, ok := item.Data.(*domain.UsageEventData); ok && usage.ReconciliationSource == domain.ReconciliationSourceInterruptedTurn {
			found = true
		}
	}
	if !found {
		t.Fatal("recovered receipt does not record its provenance")
	}
	count, _ := events.Count(t.Context(), runID)
	o.terminalAccounting = terminalAccountingSweep{exclude: o.terminalAccounting.exclude}
	if err := o.RecoverStandaloneTerminalAccounting(t.Context()); err != nil {
		t.Fatal(err)
	}
	if replay, _ := events.Count(t.Context(), runID); replay != count {
		t.Fatalf("a fresh sweep re-recovered an unmeasurable charge: %d -> %d", count, replay)
	}
}

// The stuck goal-session workflows were cancelling forever because their
// interactive child runs never settled: receipt recovery refused interactive
// runs. With interactive recovery, workflow recovery settles the cancellation.
func TestWorkflowRecoverySettlesInteractiveChildFromItsTranscript(t *testing.T) {
	for _, tc := range []struct {
		name, sessionID, transcript string
		kind                        domain.RunnerType
		tokens                      int
	}{
		{"codex-interrupted", "codex-session", standaloneCodexAborted, domain.RunnerTypeCodex, 31081},
		{"claude-zero-turn", "claude-session", standaloneClaudeZero, domain.RunnerTypeClaudeCode, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeRunLauncher()
			o, repos := newRelayOrchestrator(t, fake)
			if err := repos.Workflows.ActivateBatch(t.Context(), []*domain.WorkflowRevision{relayDefinition()}); err != nil {
				t.Fatal(err)
			}
			x, err := o.StartWorkflowExecution(t.Context(), StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: "owner/relay", Input: []byte(`{}`), IdempotencyKey: tc.name})
			if err != nil {
				t.Fatal(err)
			}
			runID := runIDForNode(t, repos.WorkflowExecutions, x.ID, "a")
			if _, err := o.CancelWorkflowExecution(t.Context(), WorkflowExecutionOperationRequest{ExecutionID: x.ID, IdempotencyKey: "cancel", Reason: "operator"}); err == nil {
				t.Fatal("missing receipt settled cancellation")
			}
			task := &domain.Task{ID: uuid.New(), Title: "interactive child", ScopePath: ".", Status: domain.TaskStatusQueued}
			if err := repos.Tasks.Create(t.Context(), task); err != nil {
				t.Fatal(err)
			}
			ended := time.Now().UTC().Add(-time.Minute)
			run := &domain.Run{ID: runID, TaskID: task.ID, Status: domain.RunStatusCancelled, Phase: domain.RunPhaseCompleted, RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeInteractive, SessionID: tc.sessionID, EndedAt: &ended, FinalizationStatus: domain.RunFinalizationStatusNone, ResolvedConfig: &domain.RunConfig{RunnerType: tc.kind, Model: "retained-model"}, Billing: domain.BillingSnapshot{Basis: domain.ChargeBasisSubscription}, TranscriptPath: filepath.Join(t.TempDir(), "transcript.jsonl")}
			if err := os.WriteFile(run.TranscriptPath, []byte(tc.transcript), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := repos.Runs.Create(t.Context(), run); err != nil {
				t.Fatal(err)
			}
			events := mocks.NewFakeEventStore()
			if err := events.Append(t.Context(), runID, domain.NewStatusEvent(runID, "starting", "running", "original invocation")); err != nil {
				t.Fatal(err)
			}
			restarted, _ := reopenRelayOrchestrator(t, repos, fake)
			WithEvents(events)(restarted)
			registry := runner.NewRegistry()
			codec := core.NewRunner(codecs.NewCodexForTest(), nil, nil)
			if tc.kind == domain.RunnerTypeClaudeCode {
				codec = core.NewRunner(codecs.NewClaudeForTest(), nil, nil)
			}
			if err := registry.Register(codec); err != nil {
				t.Fatal(err)
			}
			restarted.runners = registry
			restarted.workflowEngine.Children = terminalReceiptRecoveryLauncher{workflowChildLauncher: workflowChildLauncher{o: restarted}, exclude: func(context.Context, *domain.Run) error { return nil }}
			if err := restarted.RecoverWorkflowExecutions(t.Context()); err != nil {
				t.Fatal(err)
			}
			got, err := repos.WorkflowExecutions.Get(t.Context(), x.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != domain.WorkflowExecutionCancelled || !got.BudgetUsage.AccountingComplete || !got.BudgetUsage.ChargeMeasured || got.BudgetUsage.Tokens != tc.tokens {
				t.Fatalf("interactive child receipt not reconciled: %+v", got)
			}
		})
	}
}

func TestStandaloneSweepLeavesRequestInFlightUnknownAndRetriesLater(t *testing.T) {
	events := mocks.NewFakeEventStore()
	o, runID := seedStandaloneRun(t, standaloneRunFixture{kind: domain.RunnerTypeCodex, sessionID: "codex-session", transcript: standaloneCodexRequestInFlight, basis: domain.ChargeBasisSubscription, retained: []*domain.RunEvent{liveUsage(4331, 205, 10880)}}, events)
	count, _ := events.Count(t.Context(), runID)
	if err := o.RecoverStandaloneTerminalAccounting(t.Context()); err != nil {
		t.Fatal(err)
	}
	got, err := o.RunAccounting(t.Context(), runID)
	if err != nil || got.TokensKnown {
		t.Fatalf("in-flight accounting = %+v %v, want unknown", got, err)
	}
	if after, _ := events.Count(t.Context(), runID); after != count {
		t.Fatalf("unknown usage was fabricated: %d -> %d events", count, after)
	}
	if o.terminalAccounting.due(runID, time.Now()) {
		t.Fatal("an unknown run was not scheduled for a later retry")
	}
}
