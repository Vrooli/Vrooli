package initiativereview

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/review"
)

// fakeSpawner records SpawnInitiative calls and returns a canned RunResult.
// Also satisfies the optional RunInspector interface so the service wires
// polling against it.
type fakeSpawner struct {
	enabled       bool
	spawnErr      error
	returnRunID   string
	spawnCalls    []agentmanager.InitiativeSpawnRequest
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

func (f *fakeSpawner) SpawnInitiative(_ context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error) {
	f.spawnCalls = append(f.spawnCalls, req)
	if f.spawnErr != nil {
		return agentmanager.RunResult{}, f.spawnErr
	}
	return agentmanager.RunResult{
		TaskID:    "task-" + f.returnRunID,
		RunID:     f.returnRunID,
		CreatedAt: "2026-04-23T00:00:00Z",
	}, nil
}

func (f *fakeSpawner) GetRunState(_ context.Context, runID string) (agentmanager.RunState, error) {
	if state, ok := f.runStateQueue[runID]; ok {
		return state, nil
	}
	return agentmanager.RunState{RunID: runID, Status: "running"}, nil
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
	clock := func() time.Time { return time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC) }

	svc, err := NewService(Config{
		InitStore:     initStore,
		BacklogLoader: &backlogAdapter{store: bStore},
		GraphReader:   mat,
		Spawner:       spawner,
		PromptClient:  prompts,
		Clock:         clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &env{
		t: t, root: root,
		initStore: initStore, initSvc: initSvc, bStore: bStore, mat: mat,
		svc: svc, spawner: spawner, prompts: prompts, clock: clock,
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

	init, err := e.initStore.Load("ready-init")
	if err != nil {
		t.Fatal(err)
	}
	if init.Status != initiatives.InitiativeStatusInReview {
		t.Fatalf("expected in_review, got %s", init.Status)
	}

	if len(e.spawner.spawnCalls) != 1 {
		t.Fatalf("expected 1 spawn call, got %d", len(e.spawner.spawnCalls))
	}
	call := e.spawner.spawnCalls[0]
	if call.Name != "ready-init" {
		t.Errorf("spawn Name = %q, want ready-init", call.Name)
	}
	if call.Purpose != "review" {
		t.Errorf("spawn Purpose = %q, want review", call.Purpose)
	}
	if !strings.Contains(call.Prompt, "INSTRUCTIONS FOR swarm-manager-initiative-review") {
		t.Errorf("spawn Prompt missing rendered instructions; got %q", call.Prompt)
	}
	if len(call.ContextAttachments) == 0 {
		t.Error("expected context attachments to be attached")
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
	if len(e.spawner.spawnCalls) != 1 {
		t.Fatalf("expected 1 spawn call total, got %d", len(e.spawner.spawnCalls))
	}
}

func TestListRounds_PollsGatheringRoundToTerminal(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("poll-init", "Poll", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "poll-init")

	result, err := e.svc.TriggerIfReady(context.Background(), "poll-init")
	if err != nil || !result.Started {
		t.Fatalf("trigger: err=%v started=%v", err, result.Started)
	}

	e.spawner.runStateQueue[result.RunID] = agentmanager.RunState{
		RunID:  result.RunID,
		Status: "complete",
	}
	// Simulate the agent having written a valid JSON payload.
	itemDir := e.initStore.InitDir("poll-init")
	round, err := review.LoadRound(itemDir, 1)
	if err != nil || round == nil {
		t.Fatalf("load round: err=%v round=%v", err, round)
	}
	round.AgentAssessment = "The initiative delivered its stated goal."
	round.Classification = "delivered"
	if err := review.SaveRound(itemDir, *round); err != nil {
		t.Fatal(err)
	}

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
	result, _ := e.svc.TriggerIfReady(context.Background(), "bad-init")

	e.spawner.runStateQueue[result.RunID] = agentmanager.RunState{
		RunID:  result.RunID,
		Status: "complete",
	}

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
	if len(e.spawner.spawnCalls) != 1 {
		t.Fatalf("expected 1 spawn call from TriggerForItem, got %d", len(e.spawner.spawnCalls))
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
	result, _ := e.svc.TriggerIfReady(context.Background(), "worker-init")

	itemDir := e.initStore.InitDir("worker-init")
	round, _ := review.LoadRound(itemDir, 1)
	round.AgentAssessment = "Done"
	round.Classification = "delivered"
	_ = review.SaveRound(itemDir, *round)

	e.spawner.runStateQueue[result.RunID] = agentmanager.RunState{RunID: result.RunID, Status: "complete"}
	e.svc.RefreshGatheringRounds(context.Background())

	init, _ := e.initStore.Load("worker-init")
	if init.Status != initiatives.InitiativeStatusReviewPending {
		t.Fatalf("expected review_pending after worker pass, got %s", init.Status)
	}
}

func TestRecoverActiveRounds_RepopulatesPollingMap(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("recover-init", "Recover", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "recover-init")
	result, _ := e.svc.TriggerIfReady(context.Background(), "recover-init")

	// Drop the in-memory tracking to simulate a restart.
	e.svc.mu.Lock()
	delete(e.svc.activeRounds, result.RunID)
	e.svc.mu.Unlock()

	e.svc.RecoverActiveRounds([]string{"recover-init"})
	e.svc.mu.Lock()
	_, ok := e.svc.activeRounds[result.RunID]
	e.svc.mu.Unlock()
	if !ok {
		t.Fatalf("expected RunID %q to be recovered into activeRounds", result.RunID)
	}
}
