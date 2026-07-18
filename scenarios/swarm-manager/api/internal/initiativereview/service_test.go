package initiativereview

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/review"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

// fakeSpawner provides availability and run inspection for workflow tests.
type fakeSpawner struct {
	enabled       bool
	spawnErr      error
	returnRunID   string
	spawnCalls    []any
	runStateQueue map[string]agentmanager.RunState
}

func newFakeSpawner() *fakeSpawner {
	return &fakeSpawner{
		enabled:       true,
		returnRunID:   "run-init-1",
		runStateQueue: make(map[string]agentmanager.RunState),
	}
}

func (f *fakeSpawner) IsEnabled() bool { return f.enabled }

func (f *fakeSpawner) GetRunState(_ context.Context, runID string) (agentmanager.RunState, error) {
	if state, ok := f.runStateQueue[runID]; ok {
		return state, nil
	}
	return agentmanager.RunState{RunID: runID, Status: "running"}, nil
}

type fakeInitiativeReviewWorkflow struct {
	calls      int
	invocation agentmanager.Invocation
	completion agentmanager.InvocationCompletion
}

func (f *fakeInitiativeReviewWorkflow) StartWorkflow(_ context.Context, invocation agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	f.calls++
	f.invocation = invocation
	return agentmanager.WorkflowStart{ExecutionID: "workflow-init-1", RunID: "run-init-1", DefinitionDigest: "sha256:initiative-review"}, nil
}
func (f *fakeInitiativeReviewWorkflow) CollectWorkflow(context.Context, string) (agentmanager.InvocationCompletion, error) {
	return f.completion, nil
}

// newCommitInitiativeReviewActionContext assembles the ActionContext the runner
// completion bridge hands the commit-initiative-review handler for a completing
// initiative-review operation, correlated by run id.
func newCommitInitiativeReviewActionContext(initiativeName, executionID, runID, outcome string, result json.RawMessage) legacyCompletionAction {
	return legacyCompletionAction{InitiativeName: initiativeName, ExecutionID: executionID, RunID: runID, Outcome: outcome, Result: result}
}

// initiativeReviewResultJSON builds the reviewResult contract payload the
// completion bridge delivers for an initiative review round.
func initiativeReviewResultJSON(t *testing.T, verdict, assessment string) json.RawMessage {
	t.Helper()
	handoff, err := json.Marshal(map[string]any{
		"verdict":          verdict,
		"agent_assessment": assessment,
		"evidence":         []review.EvidenceItem{{ID: "ev1", Type: review.EvidenceTypeCLIOutput, Title: "t", Description: "d"}},
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	envelope, err := json.Marshal(map[string]any{"verdict": verdict, "handoff": json.RawMessage(handoff)})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return envelope
}

// stubPromptClient implements promptmanager.Client and returns a canned body
// so the service never hits a real prompt-manager over HTTP.
type stubPromptClient struct {
	calls []string
}

func (s *stubPromptClient) ReadSkill(_ context.Context, skillID string, _ map[string]string, _ bool) (string, error) {
	s.calls = append(s.calls, skillID)
	return "INSTRUCTIONS FOR " + skillID, nil
}

func (s *stubPromptClient) ReadSkillWithExperiment(ctx context.Context, skillID string, vars map[string]string, scope bool, _ string) (promptmanager.ReadSkillResult, error) {
	content, err := s.ReadSkill(ctx, skillID, vars, scope)
	return promptmanager.ReadSkillResult{Content: content}, err
}

type backlogAdapter struct{ store *backlog.FileStore }

func (a *backlogAdapter) LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error) {
	return a.store.LoadItem(kind, name)
}

func (a *backlogAdapter) ItemDir(kind backlog.BacklogKind, name string) string {
	return a.store.ItemDir(kind, name)
}

type env struct {
	t         *testing.T
	root      string
	initStore *initiatives.Store
	initSvc   *initiatives.Service
	bStore    *backlog.FileStore
	mat       *graph.Materializer
	svc       *Service
	spawner   *fakeSpawner
	workflow  *fakeInitiativeReviewWorkflow
	prompts   *stubPromptClient
	clock     func() time.Time
}

func newEnv(t *testing.T) *env {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"ideas", "research", "fixes", "executes", "chores", "initiatives"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	bStore := backlog.NewFileStore(root)
	initStore := initiatives.NewStore(root)
	initSvc := initiatives.NewService(initStore, bStore)

	mat := graph.NewMaterializer(graph.NewInitiativeAdapter(initStore), bStore, initStore.InitDir)
	spawner := newFakeSpawner()
	prompts := &stubPromptClient{}
	workflow := &fakeInitiativeReviewWorkflow{}
	clock := func() time.Time { return time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC) }

	svc, err := NewService(Config{
		InitStore:     initStore,
		BacklogLoader: &backlogAdapter{store: bStore},
		GraphReader:   mat,
		RunInspector:  spawner,
		PromptClient:  prompts,
		Clock:         clock,
		Workflow:      workflow,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Wire the operation-runner adapter by default so TriggerIfReady reroutes to
	// the initiative-review operation (the production path) instead of degrading.
	return &env{
		t: t, root: root,
		initStore: initStore, initSvc: initSvc, bStore: bStore, mat: mat,
		svc: svc, spawner: spawner, workflow: workflow, prompts: prompts, clock: clock,
	}
}

func (e *env) createInitiative(name, title string, itemRefs ...string) *initiatives.Initiative {
	e.t.Helper()
	if _, err := e.initSvc.Create(initiatives.CreateRequest{
		Name:  name,
		Title: title,
	}); err != nil {
		e.t.Fatal(err)
	}
	if len(itemRefs) > 0 {
		if err := e.initSvc.AddItems(name, itemRefs); err != nil {
			e.t.Fatal(err)
		}
	}
	init, err := e.initStore.Load(name)
	if err != nil {
		e.t.Fatal(err)
	}
	return init
}

func (e *env) seedItem(kind, name, title string, status backlog.BacklogStatus) {
	e.t.Helper()
	if err := os.MkdirAll(e.bStore.ItemDir(backlog.BacklogKind(kind), name), 0o755); err != nil {
		e.t.Fatal(err)
	}
	if err := e.bStore.SaveItem(backlog.BacklogItem{
		Name:    name,
		Title:   title,
		Kind:    backlog.BacklogKind(kind),
		Status:  status,
		Created: "2026-04-23T00:00:00Z",
		Updated: "2026-04-23T00:00:00Z",
	}); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) setItemInitiative(kind, name, initiative string) {
	e.t.Helper()
	if _, err := e.bStore.SetItemInitiative(backlog.BacklogKind(kind), name, initiative); err != nil {
		e.t.Fatal(err)
	}
}

// -- Tests ------------------------------------------------------------------

func TestNormalizeVerdict(t *testing.T) {
	for raw, want := range map[string]Verdict{
		"accept":   VerdictAccept,
		"ACCEPT":   VerdictAccept,
		"  fail  ": VerdictFail,
		"followup": VerdictFollowup,
	} {
		got, err := NormalizeVerdict(raw)
		if err != nil {
			t.Errorf("NormalizeVerdict(%q): err %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeVerdict(%q) = %q, want %q", raw, got, want)
		}
	}
	if _, err := NormalizeVerdict("ship"); err == nil {
		t.Error("expected error for unknown verdict")
	}
}

func TestTriggerIfReady_EmptyInitiativeReturnsReason(t *testing.T) {
	e := newEnv(t)
	e.createInitiative("empty-init", "Empty")
	result, err := e.svc.TriggerIfReady(context.Background(), "empty-init")
	if err != nil {
		t.Fatalf("empty: err %v", err)
	}
	if result.Started {
		t.Fatalf("empty initiative should not trigger; got %+v", result)
	}
	if !strings.Contains(result.Reason, "no items") {
		t.Fatalf("expected 'no items' reason, got %q", result.Reason)
	}
}

func TestTriggerIfReady_NonTerminalItemsReturnsReason(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "done", "Done", backlog.StatusCompleted)
	e.seedItem("execute", "wip", "In progress", backlog.StatusInProgress)
	e.createInitiative("mixed-init", "Mixed", "execute/done", "execute/wip")
	e.setItemInitiative("execute", "done", "mixed-init")
	e.setItemInitiative("execute", "wip", "mixed-init")

	result, err := e.svc.TriggerIfReady(context.Background(), "mixed-init")
	if err != nil {
		t.Fatalf("mixed: err %v", err)
	}
	if result.Started {
		t.Fatalf("mixed initiative should not trigger; got %+v", result)
	}
	if !strings.Contains(result.Reason, "not yet terminal") {
		t.Fatalf("expected 'not yet terminal' reason, got %q", result.Reason)
	}
	if len(e.spawner.spawnCalls) != 0 {
		t.Fatalf("expected no spawn, got %d calls", len(e.spawner.spawnCalls))
	}
}

func TestTriggerIfReady_StartsWhenAllItemsTerminal(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.seedItem("fix", "bravo", "Bravo", backlog.StatusNeedsFollowup)
	e.createInitiative("ready-init", "Ready", "execute/alpha", "fix/bravo")
	e.setItemInitiative("execute", "alpha", "ready-init")
	e.setItemInitiative("fix", "bravo", "ready-init")

	result, err := e.svc.TriggerIfReady(context.Background(), "ready-init")
	if err != nil {
		t.Fatalf("TriggerIfReady: %v", err)
	}
	if !result.Started {
		t.Fatalf("expected Started=true, got %+v", result)
	}
	if result.Round != 1 {
		t.Fatalf("expected round 1, got %d", result.Round)
	}
	if result.RunID == "" {
		t.Fatalf("expected RunID to be set")
	}

	rounds, err := e.svc.ListRounds("ready-init")
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round on disk, got %d", len(rounds))
	}
	if rounds[0].Status != review.RoundStatusGathering {
		t.Fatalf("expected gathering, got %s", rounds[0].Status)
	}
	// The round must be runner-owned, carrying the run association the
	// commit-initiative-review handler correlates back to.
	if !rounds[0].WorkflowOwned() {
		t.Fatalf("expected workflow-owned round, got %#v", rounds[0])
	}
	if rounds[0].RunID != "run-init-1" || rounds[0].AgentWorkflowExecutionID != "workflow-init-1" {
		t.Errorf("round missing workflow association: %#v", rounds[0])
	}

	init, err := e.initStore.Load("ready-init")
	if err != nil {
		t.Fatal(err)
	}
	if init.Status != initiatives.InitiativeStatusInReview {
		t.Fatalf("expected in_review, got %s", init.Status)
	}

	// The initiative-review operation must have been started against the
	// initiative target — not a direct agent spawn.
	if len(e.spawner.spawnCalls) != 0 {
		t.Errorf("expected no direct spawn, got %d", len(e.spawner.spawnCalls))
	}
	if e.workflow.calls != 1 {
		t.Fatalf("expected 1 workflow start, got %d", e.workflow.calls)
	}
	entity := e.workflow.invocation.Input.AsInterface().(map[string]any)["entity"].(map[string]any)
	if entity["kind"] != "initiative" || entity["name"] != "ready-init" {
		t.Errorf("workflow entity = %#v, want ready-init initiative", entity)
	}
}

func TestTriggerIfReady_Idempotent(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "solo", "Solo", backlog.StatusCompleted)
	e.createInitiative("once-init", "Once", "execute/solo")
	e.setItemInitiative("execute", "solo", "once-init")

	if _, err := e.svc.TriggerIfReady(context.Background(), "once-init"); err != nil {
		t.Fatal(err)
	}
	result, err := e.svc.TriggerIfReady(context.Background(), "once-init")
	if err != nil {
		t.Fatal(err)
	}
	if result.Started {
		t.Fatalf("expected second trigger to be no-op, got %+v", result)
	}
	if !strings.Contains(result.Reason, "in_review") {
		t.Fatalf("expected 'in_review' in reason, got %q", result.Reason)
	}
	if e.workflow.calls != 1 {
		t.Fatalf("expected 1 workflow start total, got %d", e.workflow.calls)
	}
}

func TestApplyWorkflowRound_ExactlyOnce(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "apply-item", "Apply", backlog.StatusCompleted)
	e.createInitiative("apply-init", "Apply", "execute/apply-item")
	e.setItemInitiative("execute", "apply-item", "apply-init")
	triggered, err := e.svc.TriggerIfReady(context.Background(), "apply-init")
	if err != nil || !triggered.Started {
		t.Fatalf("trigger = %#v, err=%v", triggered, err)
	}
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"assessment": "ready", "classification": "delivered"}})
	if err != nil {
		t.Fatal(err)
	}
	e.workflow.completion = agentmanager.InvocationCompletion{ExecutionID: "workflow-init-1", DefinitionDigest: "sha256:initiative-review", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: e.workflow.invocation.Input, Output: output}
	first, idempotent, err := e.svc.ApplyWorkflowRound(context.Background(), "apply-init", triggered.Round)
	if err != nil || idempotent || first.Status != review.RoundStatusComplete {
		t.Fatalf("first apply = %#v idempotent=%v err=%v", first, idempotent, err)
	}
	init, _ := e.initStore.Load("apply-init")
	if init.Status != initiatives.InitiativeStatusReviewPending {
		t.Fatalf("initiative status = %s, want review_pending", init.Status)
	}
	_, idempotent, err = e.svc.ApplyWorkflowRound(context.Background(), "apply-init", triggered.Round)
	if err != nil || !idempotent {
		t.Fatalf("replay idempotent=%v err=%v", idempotent, err)
	}
}

// writeLegacyGatheringRound writes a NON-runner-owned gathering round (empty
// OpExecutionID) directly to disk and puts the initiative in_review. These rounds
// are still driven by the legacy inline/background pollers — the reroute only
// changes ownership of rounds STARTED through the operation runner. Keeping this
// coverage guards the poller path for any round without an operation execution.
func (e *env) writeLegacyGatheringRound(initiativeName string, round review.Round) {
	e.t.Helper()
	itemDir := e.initStore.InitDir(initiativeName)
	if err := review.SaveRound(itemDir, round); err != nil {
		e.t.Fatal(err)
	}
	init, err := e.initStore.Load(initiativeName)
	if err != nil {
		e.t.Fatal(err)
	}
	init.Status = initiatives.InitiativeStatusInReview
	if err := e.initStore.Save(init); err != nil {
		e.t.Fatal(err)
	}
}

func TestListRounds_PollsGatheringRoundToTerminal(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("poll-init", "Poll", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "poll-init")

	e.writeLegacyGatheringRound("poll-init", review.Round{
		RoundNum:        1,
		GeneratedAt:     e.clock().UTC().Format(time.RFC3339),
		Status:          review.RoundStatusGathering,
		RunID:           "run-legacy",
		AgentAssessment: "The initiative delivered its stated goal.",
		Classification:  "delivered",
		Evidence:        []review.EvidenceItem{},
	})
	e.spawner.runStateQueue["run-legacy"] = agentmanager.RunState{RunID: "run-legacy", Status: "complete"}

	rounds, err := e.svc.ListRounds("poll-init")
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].Status != review.RoundStatusComplete {
		t.Fatalf("expected complete, got %s", rounds[0].Status)
	}

	init, err := e.initStore.Load("poll-init")
	if err != nil {
		t.Fatal(err)
	}
	if init.Status != initiatives.InitiativeStatusReviewPending {
		t.Fatalf("expected review_pending, got %s", init.Status)
	}
}

func TestListRounds_InvalidCompletedRoundMarksFailed(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("bad-init", "Bad", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "bad-init")

	// A legacy round that completes without a valid assessment/classification.
	e.writeLegacyGatheringRound("bad-init", review.Round{
		RoundNum:    1,
		GeneratedAt: e.clock().UTC().Format(time.RFC3339),
		Status:      review.RoundStatusGathering,
		RunID:       "run-legacy",
		Evidence:    []review.EvidenceItem{},
	})
	e.spawner.runStateQueue["run-legacy"] = agentmanager.RunState{RunID: "run-legacy", Status: "complete"}

	rounds, err := e.svc.ListRounds("bad-init")
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if rounds[0].Status != review.RoundStatusFailed {
		t.Fatalf("expected failed (missing classification), got %s", rounds[0].Status)
	}
	if !strings.Contains(rounds[0].FailureReason, "classification") && !strings.Contains(rounds[0].FailureReason, "assessment") {
		t.Fatalf("expected classification/assessment in failure reason, got %q", rounds[0].FailureReason)
	}
}

func TestDecide_FlipsInitiativeToCompleted(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("decide-init", "Decide", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "decide-init")
	_, _ = e.svc.TriggerIfReady(context.Background(), "decide-init")

	init, _ := e.initStore.Load("decide-init")
	init.Status = initiatives.InitiativeStatusReviewPending
	_ = e.initStore.Save(init)

	resp, err := e.svc.Decide(context.Background(), "decide-init", VerdictAccept, "looks good", "tester")
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Status != initiatives.InitiativeStatusCompleted {
		t.Fatalf("expected completed, got %s", resp.Status)
	}

	reloaded, _ := e.initStore.Load("decide-init")
	if reloaded.Status != initiatives.InitiativeStatusCompleted {
		t.Fatalf("expected persisted completed, got %s", reloaded.Status)
	}

	decisions, err := e.svc.ListDecisions("decide-init")
	if err != nil {
		t.Fatalf("ListDecisions: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}
	if decisions[0].Verdict != string(VerdictAccept) {
		t.Errorf("verdict = %q, want accept", decisions[0].Verdict)
	}
	if decisions[0].Rationale != "looks good" {
		t.Errorf("rationale mismatch: %q", decisions[0].Rationale)
	}
	if decisions[0].Round != 1 {
		t.Errorf("round = %d, want 1", decisions[0].Round)
	}
}

func TestDecide_Followup_NeedsFollowupStatus(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("fu-init", "Followup", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "fu-init")

	init, _ := e.initStore.Load("fu-init")
	init.Status = initiatives.InitiativeStatusReviewPending
	_ = e.initStore.Save(init)

	resp, err := e.svc.Decide(context.Background(), "fu-init", VerdictFollowup, "", "tester")
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if resp.Status != initiatives.InitiativeStatusNeedsFollowup {
		t.Fatalf("expected needs_followup, got %s", resp.Status)
	}
}

func TestDecide_RejectsUnlessReviewPending(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("guarded", "Guarded", "execute/alpha")

	_, err := e.svc.Decide(context.Background(), "guarded", VerdictAccept, "", "tester")
	if err == nil || !strings.Contains(err.Error(), "review_pending") {
		t.Fatalf("expected 'review_pending' error, got %v", err)
	}
}

func TestTriggerForItem_IgnoresItemsWithoutInitiative(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "orphan", "Orphan", backlog.StatusCompleted)
	e.svc.TriggerForItem(context.Background(), "execute", "orphan")
	if len(e.spawner.spawnCalls) != 0 {
		t.Fatalf("expected no spawn for orphan item, got %d", len(e.spawner.spawnCalls))
	}
}

func TestTriggerForItem_RoutesToInitiativeTrigger(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "beta", "Beta", backlog.StatusCompleted)
	e.createInitiative("auto-init", "Auto", "execute/beta")
	e.setItemInitiative("execute", "beta", "auto-init")

	e.svc.TriggerForItem(context.Background(), "execute", "beta")
	if e.workflow.calls != 1 {
		t.Fatalf("expected 1 workflow start from TriggerForItem, got %d", e.workflow.calls)
	}
	entity := e.workflow.invocation.Input.AsInterface().(map[string]any)["entity"].(map[string]any)
	if entity["name"] != "auto-init" {
		t.Errorf("workflow entity = %#v, want auto-init", entity)
	}
	init, _ := e.initStore.Load("auto-init")
	if init.Status != initiatives.InitiativeStatusInReview {
		t.Fatalf("expected in_review, got %s", init.Status)
	}
}

func TestRefreshGatheringRounds_FlipsToReviewPending(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("worker-init", "Worker", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "worker-init")

	// A legacy (non-runner-owned) round tracked for background polling.
	e.writeLegacyGatheringRound("worker-init", review.Round{
		RoundNum:        1,
		GeneratedAt:     e.clock().UTC().Format(time.RFC3339),
		Status:          review.RoundStatusGathering,
		RunID:           "run-legacy",
		AgentAssessment: "Done",
		Classification:  "delivered",
		Evidence:        []review.EvidenceItem{},
	})
	e.svc.trackActiveRound("worker-init", 1, "run-legacy")

	e.spawner.runStateQueue["run-legacy"] = agentmanager.RunState{RunID: "run-legacy", Status: "complete"}
	e.svc.RefreshGatheringRounds(context.Background())

	init, _ := e.initStore.Load("worker-init")
	if init.Status != initiatives.InitiativeStatusReviewPending {
		t.Fatalf("expected review_pending after worker pass, got %s", init.Status)
	}
}

// TestStartReview_LockConflict_RejectsSpawn verifies initiative review
// refuses to spawn when feedback (or another review) holds the per-
// initiative lock. This is the cross-service single-agent guarantee: the
// lock file is the same `.feedback-lock` the feedback service acquires,
// and the conflict surfaces via *initiativelock.Conflict so the HTTP
// layer can render a 409 with the current holder details.
func TestStartReview_LockConflict_RejectsSpawn(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "held", "Held", backlog.StatusCompleted)
	e.createInitiative("locked-init", "Locked", "execute/held")
	e.setItemInitiative("execute", "held", "locked-init")

	e.svc.lock = &initiativelock.Lock{Dir: e.initStore.InitDir, MaxAge: time.Hour}

	// Simulate feedback already holding the lock.
	if err := e.svc.lock.Acquire("locked-init", initiativelock.Holder{RunID: "feedback-run-99", Purpose: "feedback"}); err != nil {
		t.Fatalf("pre-acquire: %v", err)
	}

	_, err := e.svc.TriggerIfReady(context.Background(), "locked-init")
	if err == nil {
		t.Fatalf("expected lock conflict, got nil")
	}
	var conflict *initiativelock.Conflict
	if !errors.As(err, &conflict) {
		t.Fatalf("expected *initiativelock.Conflict, got %T: %v", err, err)
	}
	if conflict.Holder.RunID != "feedback-run-99" {
		t.Errorf("conflict should surface existing holder; got %+v", conflict.Holder)
	}
	if e.workflow.calls != 0 {
		t.Fatalf("expected 0 workflow starts under conflict, got %d", e.workflow.calls)
	}
}

// TestStartReview_LockReleasedOnTerminal verifies that when a review round
// reaches terminal status via the operation runner's commit handler, the lock is
// released so subsequent feedback rounds can proceed. Without release, the
// initiative would wedge: every feedback submission post-review would 409.
func TestStartReview_LockReleasedOnTerminal(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "finish", "Finish", backlog.StatusCompleted)
	e.createInitiative("release-init", "Release", "execute/finish")
	e.setItemInitiative("execute", "finish", "release-init")

	e.svc.lock = &initiativelock.Lock{Dir: e.initStore.InitDir, MaxAge: time.Hour}

	result, err := e.svc.TriggerIfReady(context.Background(), "release-init")
	if err != nil || !result.Started {
		t.Fatalf("trigger: err=%v started=%v", err, result.Started)
	}
	// Lock should be held right after the operation starts.
	if h, _ := e.svc.lock.Inspect("release-init"); h == nil {
		t.Fatalf("expected lock held after operation start")
	}

	// The completing operation delivers its result through commit-initiative-review.
	ac := newCommitInitiativeReviewActionContext("release-init", "exec-test", result.RunID,
		"accepted", initiativeReviewResultJSON(t, "ready", "Delivered."))
	if err := e.svc.commitInitiativeReview(context.Background(), ac); err != nil {
		t.Fatalf("commitInitiativeReview: %v", err)
	}

	round, _ := review.LoadRound(e.initStore.InitDir("release-init"), 1)
	if round.Status != review.RoundStatusComplete {
		t.Fatalf("expected complete round, got %s", round.Status)
	}
	if h, _ := e.svc.lock.Inspect("release-init"); h != nil {
		t.Fatalf("expected lock released after terminal round, got holder %+v", h)
	}
	init, _ := e.initStore.Load("release-init")
	if init.Status != initiatives.InitiativeStatusReviewPending {
		t.Fatalf("expected review_pending after terminal round, got %s", init.Status)
	}
}

// TestStartReview_AbstainFailsRoundAndReleasesLock pins the failure-path
// contract: when the review operation abstains/fails (Outcome "needs-attention"),
// the round lands in RoundStatusFailed with a reason, the per-initiative lock is
// released, and the initiative still advances to review_pending so the user can
// Decide() their way out via fail/followup. Keeping the initiative wedged in
// in_review on failure would block every subsequent feedback round on the lock.
func TestStartReview_AbstainFailsRoundAndReleasesLock(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "crashy", "Crashy", backlog.StatusCompleted)
	e.createInitiative("crash-init", "Crash", "execute/crashy")
	e.setItemInitiative("execute", "crashy", "crash-init")

	e.svc.lock = &initiativelock.Lock{Dir: e.initStore.InitDir, MaxAge: time.Hour}

	result, err := e.svc.TriggerIfReady(context.Background(), "crash-init")
	if err != nil || !result.Started {
		t.Fatalf("trigger: err=%v started=%v", err, result.Started)
	}
	if h, _ := e.svc.lock.Inspect("crash-init"); h == nil {
		t.Fatalf("expected lock held after operation start")
	}

	// The operation abstains — the agent could not derive an honest verdict.
	ac := newCommitInitiativeReviewActionContext("crash-init", "exec-test", result.RunID,
		"needs-attention", initiativeReviewResultJSON(t, "not_assessable", "Could not assess."))
	if err := e.svc.commitInitiativeReview(context.Background(), ac); err != nil {
		t.Fatalf("commitInitiativeReview: %v", err)
	}

	rounds, err := e.svc.ListRounds("crash-init")
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].Status != review.RoundStatusFailed {
		t.Fatalf("expected failed, got %s", rounds[0].Status)
	}
	if rounds[0].FailureReason == "" {
		t.Error("expected a failure reason on the abstained round")
	}

	if h, _ := e.svc.lock.Inspect("crash-init"); h != nil {
		t.Fatalf("expected lock released after failure, got holder %+v", h)
	}

	init, err := e.initStore.Load("crash-init")
	if err != nil {
		t.Fatal(err)
	}
	if init.Status != initiatives.InitiativeStatusReviewPending {
		t.Fatalf("expected review_pending so user can decide fail/followup, got %s", init.Status)
	}
}

func TestRecoverActiveRounds_RepopulatesPollingMap(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("recover-init", "Recover", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "recover-init")

	// A legacy (non-runner-owned) gathering round left on disk before a restart.
	// Runner-owned rounds are recovered by the operation runner, not this poller.
	e.writeLegacyGatheringRound("recover-init", review.Round{
		RoundNum:    1,
		GeneratedAt: e.clock().UTC().Format(time.RFC3339),
		Status:      review.RoundStatusGathering,
		RunID:       "run-legacy",
		Evidence:    []review.EvidenceItem{},
	})

	e.svc.RecoverActiveRounds()
	e.svc.mu.Lock()
	_, ok := e.svc.activeRounds["run-legacy"]
	e.svc.mu.Unlock()
	if !ok {
		t.Fatalf("expected RunID %q to be recovered into activeRounds", "run-legacy")
	}
}

// TestRecoverActiveRounds_DiscoversInitiativesFromDisk pins the contract
// that RecoverActiveRounds enumerates initiatives itself via the injected
// store — NOT via a caller-supplied list. The previous signature risked
// silently dropping gathering rounds for initiatives created right before
// a crash (and thus missing from any cached name source).
func TestRecoverActiveRounds_DiscoversInitiativesFromDisk(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("first", "First", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "first")
	e.writeLegacyGatheringRound("first", review.Round{
		RoundNum:    1,
		GeneratedAt: e.clock().UTC().Format(time.RFC3339),
		Status:      review.RoundStatusGathering,
		RunID:       "run-first",
		Evidence:    []review.EvidenceItem{},
	})

	e.seedItem("execute", "beta", "Beta", backlog.StatusCompleted)
	e.createInitiative("second", "Second", "execute/beta")
	e.setItemInitiative("execute", "beta", "second")
	e.writeLegacyGatheringRound("second", review.Round{
		RoundNum:    1,
		GeneratedAt: e.clock().UTC().Format(time.RFC3339),
		Status:      review.RoundStatusGathering,
		RunID:       "run-second",
		Evidence:    []review.EvidenceItem{},
	})

	e.svc.RecoverActiveRounds()

	e.svc.mu.Lock()
	_, firstOK := e.svc.activeRounds["run-first"]
	_, secondOK := e.svc.activeRounds["run-second"]
	e.svc.mu.Unlock()
	if !firstOK || !secondOK {
		t.Fatalf("expected both initiatives' rounds recovered (first=%v second=%v)", firstOK, secondOK)
	}
}

// TestTriggerIfReady_ConcurrentCallsOnlySpawnOnce drives two parallel
// TriggerForItem calls and asserts only one spawns. This is the real
// production scenario: two items flip to terminal inside the same
// backlog batch and both hit TriggerForItem concurrently. Without
// serialization the second caller would find status still "active",
// pass the guard, and spawn a duplicate review round.
func TestTriggerIfReady_ConcurrentCallsOnlySpawnOnce(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "one", "One", backlog.StatusCompleted)
	e.seedItem("execute", "two", "Two", backlog.StatusCompleted)
	e.createInitiative("race-init", "Race", "execute/one", "execute/two")
	e.setItemInitiative("execute", "one", "race-init")
	e.setItemInitiative("execute", "two", "race-init")

	var wg sync.WaitGroup
	results := make([]TriggerResult, 2)
	errs := make([]error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		results[0], errs[0] = e.svc.TriggerIfReady(context.Background(), "race-init")
	}()
	go func() {
		defer wg.Done()
		results[1], errs[1] = e.svc.TriggerIfReady(context.Background(), "race-init")
	}()
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("call %d returned error: %v", i, err)
		}
	}
	startedCount := 0
	for _, r := range results {
		if r.Started {
			startedCount++
		}
	}
	if startedCount != 1 {
		t.Errorf("expected exactly 1 Started=true, got %d (results: %+v)", startedCount, results)
	}
	if e.workflow.calls != 1 {
		t.Errorf("expected exactly 1 workflow start, got %d", e.workflow.calls)
	}
	// The loser must report the in_review status so callers can render
	// a correct "already under review" message rather than a silent no-op.
	loserReasons := make([]string, 0, 1)
	for _, r := range results {
		if !r.Started {
			loserReasons = append(loserReasons, r.Reason)
		}
	}
	if len(loserReasons) != 1 || !strings.Contains(loserReasons[0], "in_review") {
		t.Errorf("expected losing caller to report in_review reason; got %v", loserReasons)
	}
}
