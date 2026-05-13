package initiativereview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

// fakeExecutionLookup feeds deterministic ItemFinalization records into
// buildContextAttachments so tests can pin the GCT attachments without
// dragging in the execution store. Keyed by "kind/name" to mirror the
// initiative item-ref format.
type fakeExecutionLookup struct {
	finalizations map[string]*ItemFinalization
}

func (f *fakeExecutionLookup) LatestFinalizationFor(kind backlog.BacklogKind, name string) (*ItemFinalization, error) {
	if f == nil || f.finalizations == nil {
		return nil, nil
	}
	return f.finalizations[string(kind)+"/"+name], nil
}

// TestStartReview_AttachmentKeys locks down the exact attachment keys the
// review agent receives. Keys match internal/review's backlog review
// vocabulary so skill authors see a single contract across owner types.
// Failing this test means the agent's expected inputs drifted — either
// update the skill to match the new keys, or restore the contract.
func TestStartReview_AttachmentKeys(t *testing.T) {
	e := newEnv(t)

	// Seed two items in terminal states so the initiative is review-ready,
	// and wire a fake execution lookup that returns GCT verdicts for both
	// (one with overlapping scenarios to exercise the union path).
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.seedItem("fix", "bravo", "Bravo", backlog.StatusCompleted)

	e.svc.executionLookup = &fakeExecutionLookup{
		finalizations: map[string]*ItemFinalization{
			"execute/alpha": {
				AffectedScenarios: []string{"web-console", "swarm-manager"},
			},
			"fix/bravo": {
				AffectedScenarios: []string{"swarm-manager"},
			},
		},
	}
	// Wire a fake GCT client so the fresh-GCT fan-out lands real
	// verdicts in the gct-review-results attachment — the historical
	// aggregation path has been removed.
	e.svc.gctClient = &fakeGCTClient{
		results: map[string]*GCTResult{
			"web-console": {
				ScenarioName:   "web-console",
				Classification: "ready",
				Summary:        "Web console checks passed.",
			},
			"swarm-manager": {
				ScenarioName:   "swarm-manager",
				Classification: "ready_with_notes",
				Summary:        "One warning, nothing blocking.",
			},
		},
	}

	e.createInitiative("attach-init", "Attach", "execute/alpha", "fix/bravo")
	e.setItemInitiative("execute", "alpha", "attach-init")
	e.setItemInitiative("fix", "bravo", "attach-init")

	// Materialize the graph so the initiative-graph attachment fires.
	if err := e.mat.MaterializeInitiative(context.Background(), "attach-init"); err != nil {
		t.Fatalf("materialize: %v", err)
	}

	// Write a review round for the first item so the per-item review-
	// snapshots attachment has content.
	itemDir := e.bStore.ItemDir(backlog.BacklogKind("execute"), "alpha")
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := review.SaveRound(itemDir, review.Round{
		RoundNum:        1,
		GeneratedAt:     "2026-04-23T00:00:00Z",
		Status:          review.RoundStatusComplete,
		AgentAssessment: "Item alpha met its goal.",
		Classification:  "delivered",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := e.svc.TriggerIfReady(context.Background(), "attach-init")
	if err != nil {
		t.Fatalf("TriggerIfReady: %v", err)
	}
	if !result.Started {
		t.Fatalf("expected Started=true, got %+v", result)
	}
	if len(e.spawner.spawnCalls) != 1 {
		t.Fatalf("expected 1 spawn, got %d", len(e.spawner.spawnCalls))
	}

	got := make(map[string]bool, len(e.spawner.spawnCalls[0].ContextAttachments))
	for _, att := range e.spawner.spawnCalls[0].ContextAttachments {
		got[att.Key] = true
	}

	// Required keys: initiative identity + graph + per-item evidence.
	// GCT keys are required because we wired executionLookup above.
	required := []string{
		"initiative-summary",
		"initiative-graph",
		"item-summaries",
		"item-review-snapshots",
		"affected-scenarios",
		"gct-review-results",
	}
	for _, key := range required {
		if !got[key] {
			t.Errorf("missing required attachment key %q; got keys: %v", key, keysOf(got))
		}
	}

	// Verify the affected-scenarios attachment contains the union (sorted).
	var affected, gct string
	for _, att := range e.spawner.spawnCalls[0].ContextAttachments {
		switch att.Key {
		case "affected-scenarios":
			affected = att.Content
		case "gct-review-results":
			gct = att.Content
		}
	}
	if !strings.Contains(affected, "swarm-manager") || !strings.Contains(affected, "web-console") {
		t.Errorf("affected-scenarios missing union members; got %q", affected)
	}
	if !strings.Contains(gct, "web-console") || !strings.Contains(gct, "swarm-manager") {
		t.Errorf("gct-review-results missing per-scenario verdicts; got %q", gct)
	}
}

// TestStartReview_AttachmentKeys_NoExecutionLookup verifies the GCT
// attachments are silently omitted when no execution lookup is wired,
// rather than failing the spawn. This is the degraded-mode contract:
// reviews still run, just without integration evidence.
func TestStartReview_AttachmentKeys_NoExecutionLookup(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "solo", "Solo", backlog.StatusCompleted)
	e.createInitiative("no-gct-init", "No GCT", "execute/solo")
	e.setItemInitiative("execute", "solo", "no-gct-init")
	if err := e.mat.MaterializeInitiative(context.Background(), "no-gct-init"); err != nil {
		t.Fatalf("materialize: %v", err)
	}

	// executionLookup stays nil (default from newEnv) — so GCT attachments
	// should be absent from the spawn call.
	result, err := e.svc.TriggerIfReady(context.Background(), "no-gct-init")
	if err != nil || !result.Started {
		t.Fatalf("trigger: err=%v started=%v", err, result.Started)
	}

	got := make(map[string]bool)
	for _, att := range e.spawner.spawnCalls[0].ContextAttachments {
		got[att.Key] = true
	}
	if got["affected-scenarios"] {
		t.Errorf("affected-scenarios should be absent when executionLookup is nil")
	}
	if got["gct-review-results"] {
		t.Errorf("gct-review-results should be absent when executionLookup is nil")
	}
	// Base attachments must still be present.
	for _, key := range []string{"initiative-summary", "initiative-graph", "item-summaries"} {
		if !got[key] {
			t.Errorf("base attachment %q missing; got keys: %v", key, keysOf(got))
		}
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
	if len(e.spawner.spawnCalls) != 0 {
		t.Fatalf("expected 0 spawn calls under conflict, got %d", len(e.spawner.spawnCalls))
	}
}

// TestStartReview_LockReleasedOnTerminal verifies that when a review round
// reaches terminal status the lock is released so subsequent feedback
// rounds can proceed. Without release, the initiative would wedge: every
// feedback submission post-review would 409.
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
	// Lock should be held right after spawn.
	if h, _ := e.svc.lock.Inspect("release-init"); h == nil {
		t.Fatalf("expected lock held after spawn")
	}

	// Drive the round to terminal via the inline poller.
	e.spawner.runStateQueue[result.RunID] = agentmanager.RunState{RunID: result.RunID, Status: "complete"}
	itemDir := e.initStore.InitDir("release-init")
	round, _ := review.LoadRound(itemDir, 1)
	round.AgentAssessment = "Delivered."
	round.Classification = "delivered"
	_ = review.SaveRound(itemDir, *round)
	if _, err := e.svc.ListRounds("release-init"); err != nil {
		t.Fatal(err)
	}

	if h, _ := e.svc.lock.Inspect("release-init"); h != nil {
		t.Fatalf("expected lock released after terminal round, got holder %+v", h)
	}
}

// TestStartReview_AgentRunFailsMidRound pins the failure-path contract:
// when the agent run itself errors (vs. a validation failure on a complete
// round), the round lands in RoundStatusFailed with the agent's ErrorMsg,
// the per-initiative lock is released, and the initiative still advances to
// review_pending so the user can Decide() their way out via fail/followup.
// Keeping the initiative wedged in in_review on agent failure would block
// every subsequent feedback round on the lock.
func TestStartReview_AgentRunFailsMidRound(t *testing.T) {
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
		t.Fatalf("expected lock held after spawn")
	}

	// Agent run dies — not a validation failure, an actual run-level error.
	e.spawner.runStateQueue[result.RunID] = agentmanager.RunState{
		RunID:    result.RunID,
		Status:   "failed",
		ErrorMsg: "prompt timeout after 10m",
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
	if !strings.Contains(rounds[0].FailureReason, "prompt timeout") {
		t.Errorf("expected agent ErrorMsg surfaced in FailureReason, got %q", rounds[0].FailureReason)
	}

	if h, _ := e.svc.lock.Inspect("crash-init"); h != nil {
		t.Fatalf("expected lock released after agent failure, got holder %+v", h)
	}

	init, err := e.initStore.Load("crash-init")
	if err != nil {
		t.Fatal(err)
	}
	if init.Status != initiatives.InitiativeStatusReviewPending {
		t.Fatalf("expected review_pending so user can decide fail/followup, got %s", init.Status)
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
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

	e.svc.RecoverActiveRounds()
	e.svc.mu.Lock()
	_, ok := e.svc.activeRounds[result.RunID]
	e.svc.mu.Unlock()
	if !ok {
		t.Fatalf("expected RunID %q to be recovered into activeRounds", result.RunID)
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
	firstResult, _ := e.svc.TriggerIfReady(context.Background(), "first")

	e.seedItem("execute", "beta", "Beta", backlog.StatusCompleted)
	e.createInitiative("second", "Second", "execute/beta")
	e.setItemInitiative("execute", "beta", "second")
	secondResult, _ := e.svc.TriggerIfReady(context.Background(), "second")

	e.svc.mu.Lock()
	delete(e.svc.activeRounds, firstResult.RunID)
	delete(e.svc.activeRounds, secondResult.RunID)
	e.svc.mu.Unlock()

	e.svc.RecoverActiveRounds()

	e.svc.mu.Lock()
	_, firstOK := e.svc.activeRounds[firstResult.RunID]
	_, secondOK := e.svc.activeRounds[secondResult.RunID]
	e.svc.mu.Unlock()
	if !firstOK || !secondOK {
		t.Fatalf("expected both initiatives' rounds recovered (first=%v second=%v)", firstOK, secondOK)
	}
}

// fakeGCTClient is a deterministic stand-in for initiativereview.GCTClient
// used by the fresh-GCT test family. Results is keyed by scenario name and
// returned on the first Poll call (done=true). TriggerErrors / PollErrors
// let individual tests inject failures for one scenario while letting
// others succeed — the service's per-scenario error surface requires that
// a single failure not abort the fan-out.
type fakeGCTClient struct {
	mu            sync.Mutex
	results       map[string]*GCTResult
	triggerErrors map[string]error
	pollErrors    map[string]error
	triggerCalls  []string
	jobToScenario map[string]string
}

func (f *fakeGCTClient) TriggerReview(_ context.Context, scenarioName string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.triggerCalls = append(f.triggerCalls, scenarioName)
	if err := f.triggerErrors[scenarioName]; err != nil {
		return "", err
	}
	if f.jobToScenario == nil {
		f.jobToScenario = make(map[string]string)
	}
	jobID := "job-" + scenarioName
	f.jobToScenario[jobID] = scenarioName
	return jobID, nil
}

func (f *fakeGCTClient) PollReview(_ context.Context, jobID string) (*GCTResult, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	scenario := f.jobToScenario[jobID]
	if err := f.pollErrors[scenario]; err != nil {
		return nil, false, err
	}
	if f.results == nil {
		return &GCTResult{ScenarioName: scenario, Classification: "ready"}, true, nil
	}
	if res, ok := f.results[scenario]; ok {
		return res, true, nil
	}
	return &GCTResult{ScenarioName: scenario, Classification: "ready"}, true, nil
}

// TestStartReview_FreshGCT_RunsPerScenario verifies the fresh-GCT
// pass fans out one TriggerReview per affected scenario and serializes
// every verdict into the gct-review-results attachment. This is the
// "is the whole thing still working together" integration check the
// initiative review is designed around.
func TestStartReview_FreshGCT_RunsPerScenario(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.seedItem("fix", "bravo", "Bravo", backlog.StatusCompleted)
	e.createInitiative("fresh-gct-init", "Fresh GCT", "execute/alpha", "fix/bravo")
	e.setItemInitiative("execute", "alpha", "fresh-gct-init")
	e.setItemInitiative("fix", "bravo", "fresh-gct-init")

	e.svc.executionLookup = &fakeExecutionLookup{
		finalizations: map[string]*ItemFinalization{
			"execute/alpha": {AffectedScenarios: []string{"web-console", "swarm-manager"}},
			"fix/bravo":     {AffectedScenarios: []string{"swarm-manager", "prompt-manager"}},
		},
	}
	gct := &fakeGCTClient{
		results: map[string]*GCTResult{
			"web-console":    {ScenarioName: "web-console", Classification: "ready", Summary: "green"},
			"swarm-manager":  {ScenarioName: "swarm-manager", Classification: "ready", Summary: "green"},
			"prompt-manager": {ScenarioName: "prompt-manager", Classification: "ready_with_notes", Summary: "yellow"},
		},
	}
	e.svc.gctClient = gct

	result, err := e.svc.TriggerIfReady(context.Background(), "fresh-gct-init")
	if err != nil || !result.Started {
		t.Fatalf("TriggerIfReady: err=%v started=%v", err, result.Started)
	}

	// Every scenario in the union must have been triggered exactly once.
	gct.mu.Lock()
	triggered := append([]string(nil), gct.triggerCalls...)
	gct.mu.Unlock()
	sort.Strings(triggered)
	want := []string{"prompt-manager", "swarm-manager", "web-console"}
	if !reflect.DeepEqual(triggered, want) {
		t.Errorf("fresh GCT triggered = %v, want %v", triggered, want)
	}

	// The gct-review-results attachment must carry the fresh verdicts.
	var gctBody string
	for _, att := range e.spawner.spawnCalls[0].ContextAttachments {
		if att.Key == "gct-review-results" {
			gctBody = att.Content
			break
		}
	}
	if gctBody == "" {
		t.Fatalf("gct-review-results attachment missing")
	}
	for _, scenario := range want {
		if !strings.Contains(gctBody, scenario) {
			t.Errorf("gct-review-results missing %q; got %q", scenario, gctBody)
		}
	}
	// "yellow" is the prompt-manager summary — confirms we're serializing
	// fresh verdicts, not a historical snapshot.
	if !strings.Contains(gctBody, "yellow") {
		t.Errorf("gct-review-results missing fresh summary; got %q", gctBody)
	}
}

// TestStartReview_FreshGCT_PerScenarioFailure verifies one flaky scenario
// does not abort the review — the failure is captured on GCTResult.Error
// and surfaced alongside the healthy verdicts. Contract: initiative
// review must degrade gracefully, not hard-fail, when GCT hiccups for
// a single scenario.
func TestStartReview_FreshGCT_PerScenarioFailure(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("flaky-gct-init", "Flaky", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "flaky-gct-init")

	e.svc.executionLookup = &fakeExecutionLookup{
		finalizations: map[string]*ItemFinalization{
			"execute/alpha": {AffectedScenarios: []string{"web-console", "swarm-manager"}},
		},
	}
	e.svc.gctClient = &fakeGCTClient{
		results: map[string]*GCTResult{
			"swarm-manager": {ScenarioName: "swarm-manager", Classification: "ready", Summary: "green"},
		},
		triggerErrors: map[string]error{
			"web-console": errors.New("connection refused"),
		},
	}

	result, err := e.svc.TriggerIfReady(context.Background(), "flaky-gct-init")
	if err != nil || !result.Started {
		t.Fatalf("TriggerIfReady: err=%v started=%v", err, result.Started)
	}

	var gctBody string
	for _, att := range e.spawner.spawnCalls[0].ContextAttachments {
		if att.Key == "gct-review-results" {
			gctBody = att.Content
			break
		}
	}
	if !strings.Contains(gctBody, "connection refused") {
		t.Errorf("per-scenario error should appear in gct-review-results; got %q", gctBody)
	}
	if !strings.Contains(gctBody, "swarm-manager") || !strings.Contains(gctBody, "green") {
		t.Errorf("healthy verdict should still appear; got %q", gctBody)
	}
}

// TestStartReview_FreshGCT_NoClient verifies the review still spawns
// (with an affected-scenarios attachment but no gct-review-results)
// when no GCT client is wired — degraded mode contract mirroring the
// no-executionLookup case.
func TestStartReview_FreshGCT_NoClient(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("no-client-init", "No client", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "no-client-init")

	e.svc.executionLookup = &fakeExecutionLookup{
		finalizations: map[string]*ItemFinalization{
			"execute/alpha": {AffectedScenarios: []string{"web-console"}},
		},
	}
	// gctClient stays nil.

	result, err := e.svc.TriggerIfReady(context.Background(), "no-client-init")
	if err != nil || !result.Started {
		t.Fatalf("TriggerIfReady: err=%v started=%v", err, result.Started)
	}

	got := make(map[string]bool)
	for _, att := range e.spawner.spawnCalls[0].ContextAttachments {
		got[att.Key] = true
	}
	if !got["affected-scenarios"] {
		t.Errorf("affected-scenarios should still appear with no GCT client")
	}
	if got["gct-review-results"] {
		t.Errorf("gct-review-results must be omitted when no GCT client is wired")
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
	if got := len(e.spawner.spawnCalls); got != 1 {
		t.Errorf("expected exactly 1 spawn, got %d", got)
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
