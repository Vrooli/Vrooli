package review

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitionrunner"
	"swarm-manager/internal/transitions"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

// capturingSpawner supplies agent availability and run inspection for workflow tests.
type capturingSpawner struct {
	enabled  bool
	runState agentmanager.RunState
	stateErr error
}

func (s *capturingSpawner) IsEnabled() bool { return s.enabled }

func (s *capturingSpawner) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return s.runState, s.stateErr
}

type fakeReviewWorkflow struct {
	calls      int
	invocation agentmanager.Invocation
	completion agentmanager.InvocationCompletion
	collects   int
	start      agentmanager.WorkflowStart
}

func (f *fakeReviewWorkflow) StartWorkflow(_ context.Context, invocation agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	f.calls++
	f.invocation = invocation
	if f.start.ExecutionID != "" {
		return f.start, nil
	}
	return agentmanager.WorkflowStart{ExecutionID: "review-workflow", RunID: "test-run-id", DefinitionDigest: "sha256:review"}, nil
}

func (f *fakeReviewWorkflow) CollectWorkflow(context.Context, string) (agentmanager.InvocationCompletion, error) {
	f.collects++
	return f.completion, nil
}

func newTestService(spawner *capturingSpawner, promptResult string) *Service {
	svc := &Service{
		dataRoot:      "/tmp/test-backlog",
		inspector:     spawner,
		promptClient:  &promptmanager.MockClient{Result: promptResult},
		itemDirFn:     func(_, _ string) string { return "/tmp/test-backlog/tasks/test-item" },
		loadItemTitle: func(_, _ string) (string, error) { return "Loaded Title", nil },
		planContentResolver: func(_ context.Context, kind, _, itemDir string) (string, error) {
			if kind == "research" {
				return "", nil
			}
			return "# Test Plan\nDo the thing from plan-manager.", nil
		},
		// Disable the abandoned-round age backstop by default so tests that use
		// fixed placeholder timestamps aren't force-failed by it. Tests that
		// exercise the backstop set roundMaxAge + clock explicitly.
		roundMaxAge: 100 * 365 * 24 * time.Hour,
	}
	setTestReviewRunner(nil, svc, &fakeReviewWorkflow{})
	return svc
}

func TestListRoundsUsesAuthoritativeVerificationProjection(t *testing.T) {
	dir := t.TempDir()
	svc := newTestService(nil, "")
	svc.itemDirFn = func(_, _ string) string { return dir }
	if err := saveRound(dir, Round{RoundNum: 1, Status: RoundStatusComplete, Evidence: []EvidenceItem{{ID: "proof", Verified: true}, {ID: "other", Verified: false}}}); err != nil {
		t.Fatal(err)
	}
	svc.SetEvidenceVerificationProjection(func(context.Context, string, string, Round) (map[string]bool, bool, error) {
		return map[string]bool{"other": true}, true, nil
	})
	rounds, err := svc.ListRounds("execute", "item")
	if err != nil {
		t.Fatal(err)
	}
	if rounds[0].Evidence[0].Verified || !rounds[0].Evidence[1].Verified {
		t.Fatalf("verification projection = %#v", rounds[0].Evidence)
	}
}

func TestVerifyEvidenceRecordsAppendOnlyVerificationAndRevocation(t *testing.T) {
	dir := t.TempDir()
	svc := newTestService(nil, "")
	svc.itemDirFn = func(_, _ string) string { return dir }
	if err := saveRound(dir, Round{RoundNum: 1, Status: RoundStatusComplete, Evidence: []EvidenceItem{{ID: "proof", Title: "Proof"}}}); err != nil {
		t.Fatal(err)
	}
	var events []bool
	svc.SetEvidenceVerificationRecorder(func(_ context.Context, _, _ string, _ Round, evidence EvidenceItem, verified bool, actor, reason string) error {
		if evidence.ID != "proof" || actor != "operator@example.test" || reason != "Rechecked evidence." {
			t.Fatalf("unexpected recorder input: evidence=%+v verified=%v actor=%q reason=%q", evidence, verified, actor, reason)
		}
		events = append(events, verified)
		return nil
	})
	if err := svc.VerifyEvidenceWithActor(context.Background(), "execute", "item", 1, "proof", true, "exec-1", "operator@example.test", "Rechecked evidence."); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := svc.VerifyEvidenceWithActor(context.Background(), "execute", "item", 1, "proof", false, "exec-1", "operator@example.test", "Rechecked evidence."); err != nil {
		t.Fatalf("unverify: %v", err)
	}
	if got, want := fmt.Sprint(events), "[true false]"; got != want {
		t.Fatalf("verification events = %s, want %s", got, want)
	}
	if err := svc.VerifyEvidenceWithActor(context.Background(), "execute", "item", 1, "proof", true, "exec-1", "operator@example.test", ""); err == nil {
		t.Fatal("expected actor/reason validation")
	}
}

func TestValidateRoundEvidenceRejectsUnboundAndDishonestGaps(t *testing.T) {
	criteria := map[string]struct{}{"criterion-1": {}}
	for _, tc := range []struct {
		name     string
		evidence EvidenceItem
	}{
		{name: "unknown criterion", evidence: EvidenceItem{ID: "e1", CriterionID: "missing", Type: EvidenceTypeCLIOutput, Title: "output", Producer: "command", Trust: "observed", Settlement: "settled"}},
		{name: "unavailable without reason", evidence: EvidenceItem{ID: "e1", CriterionID: "criterion-1", Type: EvidenceTypeCLIOutput, Title: "output", Producer: "command", Trust: "reported", Settlement: "unavailable", AttemptedProducer: "command"}},
		{name: "unavailable without attempted producer", evidence: EvidenceItem{ID: "e1", CriterionID: "criterion-1", Type: EvidenceTypeCLIOutput, Title: "output", Producer: "command", Trust: "reported", Settlement: "unavailable", UnavailableReason: "runner unavailable"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateRoundEvidence([]EvidenceItem{tc.evidence}, criteria); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	valid := EvidenceItem{ID: "e1", CriterionID: "criterion-1", Type: EvidenceTypeCLIOutput, Title: "output", Producer: "command", Trust: "reported", Settlement: "unavailable", UnavailableReason: "runner unavailable", AttemptedProducer: "command"}
	if err := validateRoundEvidence([]EvidenceItem{valid}, criteria); err != nil {
		t.Fatalf("valid unavailable evidence rejected: %v", err)
	}
}

func TestValidateCriterionVerdictsRequireCompleteBoundEvidence(t *testing.T) {
	criteria := map[string]struct{}{"criterion-1": {}, "criterion-2": {}}
	evidence := []EvidenceItem{{ID: "e1"}, {ID: "e2"}}
	if err := validateCriterionVerdicts([]CriterionVerdict{{CriterionID: "criterion-1", Settlement: "settled", EvidenceIDs: []string{"e1"}}}, criteria, evidence); err == nil {
		t.Fatal("expected incomplete criterion verdict rejection")
	}
	if err := validateCriterionVerdicts([]CriterionVerdict{{CriterionID: "criterion-1", Settlement: "settled", EvidenceIDs: []string{"e1"}}, {CriterionID: "criterion-2", Settlement: "unavailable", EvidenceIDs: []string{"missing"}}}, criteria, evidence); err == nil {
		t.Fatal("expected unknown evidence rejection")
	}
	valid := []CriterionVerdict{{CriterionID: "criterion-1", Settlement: "settled", EvidenceIDs: []string{"e1"}}, {CriterionID: "criterion-2", Settlement: "unavailable", EvidenceIDs: []string{"e2"}}}
	if err := validateCriterionVerdicts(valid, criteria, evidence); err != nil {
		t.Fatalf("valid criterion verdicts rejected: %v", err)
	}
}

func TestSettledEvidenceForExecutionCarriesOnlyFulfilledRequestEvidence(t *testing.T) {
	dir := t.TempDir()
	if err := saveRound(dir, Round{
		RoundNum:              1,
		AgentWorkflowSnapshot: json.RawMessage(`{"executionId":"exec-1"}`),
		Evidence: []EvidenceItem{
			{ID: "review-derived", Settlement: "settled"},
			{ID: "requested", Settlement: "settled"},
			{ID: "unavailable", Settlement: "unavailable"},
		},
		RequestThreads: []RequestThread{{
			ID:     "thread-1",
			Status: "fulfilled",
			Messages: []RequestMessage{{
				Role:             "assistant",
				AddedEvidenceIDs: []string{"requested", "unavailable"},
			}},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := saveRound(dir, Round{
		RoundNum:              2,
		AgentWorkflowSnapshot: json.RawMessage(`{"executionId":"exec-2"}`),
		Evidence:              []EvidenceItem{{ID: "other-execution", Settlement: "settled"}},
	}); err != nil {
		t.Fatal(err)
	}
	evidence, err := settledEvidenceForExecution(dir, "exec-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence) != 1 || evidence[0].ID != "requested" {
		t.Fatalf("settled evidence = %#v", evidence)
	}
}

func setTestReviewRunner(t *testing.T, svc *Service, workflow *fakeReviewWorkflow) {
	if t != nil {
		t.Helper()
	}
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		panic(err)
	}
	runner := transitionrunner.New(registry, workflow, transitionrun.NewFileStore(filepath.Join(os.TempDir(), "review-transition-tests", fmt.Sprintf("%d", time.Now().UnixNano()))), nil)
	svc.RegisterTransitionAdapter(runner)
	svc.SetTransitionRunner(runner)
}

// TestRefreshGatheringRounds_InvokesOnRoundTerminal verifies that when a
// gathering round transitions to a terminal status, the OnRoundTerminal
// callback is fired with the item's kind, name, and final round state.
// This is the hook the backlog layer uses to flip items from in_review to
// review_pending.
// setupItemDir creates a temporary backlog item directory. Every backlog kind
// resolves its actionable plan through planContentResolver.
func setupItemDir(t *testing.T, kind string) string {
	t.Helper()
	return t.TempDir()
}

func TestStartReview_StartsDeclaredWorkflowAndWritesRound(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "review instructions here")
	itemDir := setupItemDir(t, "task")
	svc.itemDirFn = func(_, _ string) string { return itemDir }
	workflow := &fakeReviewWorkflow{}
	setTestReviewRunner(t, svc, workflow)

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID: "exec-123",
		BacklogKind: "task",
		BacklogName: "test-item",
		ItemTitle:   "Test Item",
		ItemDir:     itemDir,
	})
	if err != nil {
		t.Fatalf("startReview failed: %v", err)
	}

	if workflow.calls != 1 || workflow.invocation.WorkflowKey != "swarm-manager/independent-review" {
		t.Fatalf("expected one independent-review workflow invocation, got calls=%d key=%q", workflow.calls, workflow.invocation.WorkflowKey)
	}

	round, err := LoadRound(itemDir, 1)
	if err != nil || round == nil {
		t.Fatalf("expected round 1 on disk: err=%v round=%v", err, round)
	}
	if round.Status != RoundStatusGathering {
		t.Errorf("round status = %q, want gathering", round.Status)
	}
	if round.RunID != "test-run-id" {
		t.Errorf("round RunID = %q, want test-run-id", round.RunID)
	}
}

func TestStartReviewPersistsTypedCriteriaInImmutableSnapshot(t *testing.T) {
	svc := newTestService(&capturingSpawner{enabled: true}, "instructions")
	itemDir := t.TempDir()
	svc.itemDirFn = func(_, _ string) string { return itemDir }
	svc.loadReviewContract = func(_, _ string) (ReviewContract, error) {
		return ReviewContract{Description: "Outcome statement", Criteria: []map[string]any{{"id": "criterion-1", "gherkin": "Given work When reviewed Then evidence exists"}}}, nil
	}
	setTestReviewRunner(t, svc, &fakeReviewWorkflow{})
	if err := svc.startReview(context.Background(), startReviewParams{ExecutionID: "exec-criteria", BacklogKind: "execute", BacklogName: "criteria-item", ItemTitle: "Criteria", ItemDir: itemDir}); err != nil {
		t.Fatal(err)
	}
	round, err := LoadRound(itemDir, 1)
	if err != nil || round == nil {
		t.Fatalf("load round: %v", err)
	}
	var snapshot map[string]any
	if err := json.Unmarshal(round.AgentWorkflowSnapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot["itemDescription"] != "Outcome statement" {
		t.Fatalf("description missing from snapshot: %#v", snapshot)
	}
	if snapshot["planContent"] != "# Test Plan\nDo the thing from plan-manager." {
		t.Fatalf("plan content missing from snapshot: %#v", snapshot)
	}
	criteria, ok := snapshot["acceptanceCriteria"].([]any)
	if !ok || len(criteria) != 1 {
		t.Fatalf("criteria missing from snapshot: %#v", snapshot)
	}
}

func TestStartReview_RequiresWorkflow(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "instructions")
	svc.transitionRunner = nil
	itemDir := setupItemDir(t, "task")
	svc.itemDirFn = func(_, _ string) string { return itemDir }

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID: "exec-x",
		BacklogKind: "task",
		BacklogName: "test-item",
		ItemDir:     itemDir,
	})
	if err == nil || !strings.Contains(err.Error(), "transition runner is not configured") {
		t.Fatalf("expected workflow-unavailable error, got %v", err)
	}
}

// [REQ:SWM-P0-007] independent post-run review verdict projected into terminal round
func TestApplyWorkflowRound_ExactlyOnce(t *testing.T) {
	itemDir := t.TempDir()
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "task", "name": "item", "executionId": "exec-1", "version": "sha256:snapshot"}, "snapshot": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "accepted", "handoff": map[string]any{"verdict": "ready", "agent_assessment": "verified", "evidence": []any{}, "improvement_suggestions": []any{}, "regression_introduced": false, "notes": []any{}, "summary": "ready", "disposition": map[string]any{"kind": "archive", "rationale": "Evidence is sufficient", "confidence": "high"}}}})
	if err != nil {
		t.Fatal(err)
	}
	workflow := &fakeReviewWorkflow{start: agentmanager.WorkflowStart{ExecutionID: "workflow-1", DefinitionDigest: "sha256:def"}, completion: agentmanager.InvocationCompletion{ExecutionID: "workflow-1", DefinitionDigest: "sha256:def", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: input, Output: output}}
	svc := newTestService(&capturingSpawner{enabled: true}, "")
	setTestReviewRunner(t, svc, workflow)
	svc.itemDirFn = func(_, _ string) string { return itemDir }
	// Persist the round with its request snapshot first, then start through the
	// registered builder — the same order production uses, so the version the
	// correlation pins is the version the apply-time rebuild recomputes.
	requestSnapshot := []byte(`{"executionId":"exec-1","backlogKind":"task","backlogName":"item"}`)
	if err := SaveRound(itemDir, Round{RoundNum: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: RoundStatusGathering, Evidence: []EvidenceItem{}, AgentWorkflowSnapshot: requestSnapshot}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.transitionRunner.StartWith(context.Background(), "work.review", reviewSubject("task", "item", 1), transitionrunner.PreparedInput{}); err != nil {
		t.Fatal(err)
	}
	observed := 0
	svc.SetRoundTerminalObserver(func(_ context.Context, kind, name string, round Round) {
		observed++
		if kind != "task" || name != "item" || round.Disposition == nil || round.Disposition.Kind != "archive" {
			t.Fatalf("unexpected terminal projection: %s/%s %#v", kind, name, round)
		}
	})
	if err := SaveRound(itemDir, Round{RoundNum: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: RoundStatusGathering, Evidence: []EvidenceItem{}, AgentWorkflowSnapshot: requestSnapshot}); err != nil {
		t.Fatal(err)
	}
	first, replay, err := svc.ApplyWorkflowRound(context.Background(), "task", "item", 1)
	if err != nil || replay || first.Status != RoundStatusComplete || first.Classification != "ready" {
		t.Fatalf("first apply = %#v replay=%v err=%v", first, replay, err)
	}
	if observed != 1 {
		t.Fatalf("terminal observer calls = %d, want 1", observed)
	}
	second, replay, err := svc.ApplyWorkflowRound(context.Background(), "task", "item", 1)
	if err != nil || !replay || second.Status != RoundStatusComplete || workflow.collects != 1 {
		t.Fatalf("replay = %#v replay=%v collects=%d err=%v", second, replay, workflow.collects, err)
	}
}

func TestApplyEvidenceRequestWorkflow_ExactlyOnce(t *testing.T) {
	itemDir := t.TempDir()
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "task", "name": "item", "executionId": "exec-1/thread-1", "version": "sha256:snapshot"}, "snapshot": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "fulfilled", "summary": "Evidence gathered", "evidence": []any{map[string]any{"id": "evidence-1", "type": "log", "title": "Test log", "description": "The requested evidence."}}}})
	if err != nil {
		t.Fatal(err)
	}
	workflow := &fakeReviewWorkflow{start: agentmanager.WorkflowStart{ExecutionID: "workflow-1", DefinitionDigest: "sha256:def"}, completion: agentmanager.InvocationCompletion{ExecutionID: "workflow-1", DefinitionDigest: "sha256:def", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: input, Output: output}}
	svc := newTestService(&capturingSpawner{enabled: true}, "")
	setTestReviewRunner(t, svc, workflow)
	svc.itemDirFn = func(_, _ string) string { return itemDir }
	// Save the thread before starting so the registered builder can reproject it,
	// then reuse that same projection for the stored version.
	thread := RequestThread{ID: "thread-1", Status: "pending", Messages: []RequestMessage{{Role: "user", Content: "please gather", Timestamp: time.Now().UTC().Format(time.RFC3339)}}}
	if err := SaveRound(itemDir, Round{RoundNum: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Status: RoundStatusGathering, RequestThreads: []RequestThread{thread}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.transitionRunner.StartWith(context.Background(), "review.evidence_request", reviewThreadSubject("task", "item", 1, "thread-1"), transitionrunner.PreparedInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.buildEvidenceRequestInput(context.Background(), reviewThreadSubject("task", "item", 1, "thread-1")); err != nil {
		t.Fatal(err)
	}

	first, replay, err := svc.ApplyEvidenceRequestWorkflow(context.Background(), "task", "item", 1, "thread-1")
	// Two messages: the operator request the thread was created with, and the
	// agent reply the apply appended.
	if err != nil || replay || len(first.Evidence) != 1 || first.RequestThreads[0].Status != "fulfilled" || len(first.RequestThreads[0].Messages) != 2 {
		t.Fatalf("first apply = %#v replay=%v err=%v", first, replay, err)
	}
	second, replay, err := svc.ApplyEvidenceRequestWorkflow(context.Background(), "task", "item", 1, "thread-1")
	if err != nil || !replay || second.RequestThreads[0].Status != "fulfilled" || workflow.collects != 1 {
		t.Fatalf("replay = %#v replay=%v collects=%d err=%v", second, replay, workflow.collects, err)
	}
}

// --- Explicit liveness-recovery tests ---

func writeRound(t *testing.T, itemDir string, round Round) {
	t.Helper()
	reviewDir := filepath.Join(itemDir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reviewDir, RoundFilename(round.RoundNum)), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMapRunStatusToRoundStatus(t *testing.T) {
	tests := []struct {
		input string
		want  RoundStatus
	}{
		{"complete", RoundStatusComplete},
		{"Complete", RoundStatusComplete},
		{"failed", RoundStatusFailed},
		{"cancelled", RoundStatusFailed},
		{"running", ""},
		{"pending", ""},
		{"starting", ""},
	}
	for _, tt := range tests {
		got := mapRunStatusToRoundStatus(tt.input)
		if got != tt.want {
			t.Errorf("mapRunStatusToRoundStatus(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestListRounds_InlineRefresh(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:        1,
		GeneratedAt:     "2026-04-02T00:00:00Z",
		Status:          RoundStatusGathering,
		RunID:           "run-inline",
		AgentAssessment: "Looks good.",
		Classification:  "ready",
		Evidence:        []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled: true,
		runState: agentmanager.RunState{
			RunID:  "run-inline",
			Status: "complete",
		},
	}
	svc := newTestService(spawner, "")
	svc.itemDirFn = func(_, _ string) string { return itemDir }

	rounds, err := svc.ListRounds("task", "test")
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].Status != RoundStatusComplete {
		t.Errorf("round status = %q, want %q (inline refresh should update)", rounds[0].Status, RoundStatusComplete)
	}

	// Verify persisted to disk too.
	diskRound, _ := LoadRound(itemDir, 1)
	if diskRound.Status != RoundStatusComplete {
		t.Errorf("disk round status = %q, want %q", diskRound.Status, RoundStatusComplete)
	}
}

func TestListRounds_ExposesCurrentRunStatusForNeedsReview(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:    1,
		GeneratedAt: "2026-04-02T00:00:00Z",
		Status:      RoundStatusGathering,
		RunID:       "run-awaiting-review",
		Evidence:    []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled: true,
		runState: agentmanager.RunState{
			RunID:  "run-awaiting-review",
			Status: "needs_review",
		},
	}
	svc := newTestService(spawner, "")
	svc.itemDirFn = func(_, _ string) string { return itemDir }

	rounds, err := svc.ListRounds("task", "test")
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].Status != RoundStatusGathering {
		t.Fatalf("round status = %q, want %q", rounds[0].Status, RoundStatusGathering)
	}
	if rounds[0].CurrentRunStatus != "needs_review" {
		t.Fatalf("current run status = %q, want needs_review", rounds[0].CurrentRunStatus)
	}
}

func TestTriggerReviewAgent_RoutesToDeclaredWorkflow(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "review instructions here")
	itemDir := setupItemDir(t, "research")
	svc.itemDirFn = func(_, _ string) string { return itemDir }
	workflow := &fakeReviewWorkflow{}
	setTestReviewRunner(t, svc, workflow)
	svc.loadExecutionContext = func(_ context.Context, executionID string) (*ExecutionContext, error) {
		if executionID != "exec-rebuild" {
			t.Fatalf("unexpected execution id: %s", executionID)
		}
		return &ExecutionContext{
			BacklogKind:       "research",
			BacklogName:       "test-item",
			ItemTitle:         "Research Item",
			AffectedScenarios: []string{"scenario-b", "scenario-a"},
		}, nil
	}

	if err := svc.TriggerReviewAgent(context.Background(), "exec-rebuild"); err != nil {
		t.Fatalf("TriggerReviewAgent failed: %v", err)
	}

	if workflow.calls != 1 || workflow.invocation.WorkflowKey != "swarm-manager/independent-review" {
		t.Fatalf("expected independent-review invocation, got calls=%d key=%q", workflow.calls, workflow.invocation.WorkflowKey)
	}

	round, err := LoadRound(itemDir, 1)
	if err != nil || round == nil {
		t.Fatalf("expected round 1 on disk: err=%v round=%v", err, round)
	}
	if round.RunID != "test-run-id" {
		t.Errorf("round missing workflow association: %#v", round)
	}
}
