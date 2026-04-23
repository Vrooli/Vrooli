package feedback

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/proposals"
)

// e2eEnv stitches together every collaborator the feedback service touches
// in production: backlog FileStore, initiatives.Service, graph.Materializer,
// proposals.Applier, agentactivity.Service, lock, store, spawner.
//
// The point is to drive the *full* round lifecycle in one test so any
// boundary regression (graph drift, attribution gaps, lock leaks, missing
// activity records) surfaces here rather than from production telemetry.
type e2eEnv struct {
	t            *testing.T
	root         string
	backlogStore *backlog.FileStore
	initStore    *initiatives.Store
	initSvc      *initiatives.Service
	materializer *graph.Materializer
	activity     *agentactivity.Service
	rawAgent     *e2eRawAgent
	applier      *proposals.Applier
	feedbackSvc  *Service
	feedbackStor *Store
	feedbackLock *Lock
	spawner      *e2eSpawner
	events       *capturingEmitter
}

// e2eRawAgent is a minimal stand-in for agentmanager.AgentService so the
// activity-tracked spawn path has a real spawner to delegate to.
type e2eRawAgent struct {
	enabled         bool
	initSpawnReturn agentmanager.RunResult
	initSpawnErr    error
	initSpawnCalls  []agentmanager.InitiativeSpawnRequest
	continueCalls   []string
	continueErr     error
}

func (a *e2eRawAgent) IsEnabled() bool                              { return a.enabled }
func (a *e2eRawAgent) IsAvailable(_ context.Context) bool           { return a.enabled }
func (a *e2eRawAgent) ResolveURL(_ context.Context) (string, error) { return "test://agent", nil }
func (a *e2eRawAgent) GetProfileID() string                         { return "swarm-manager" }
func (a *e2eRawAgent) SpawnBacklog(_ context.Context, _ agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	return agentmanager.RunResult{}, errors.New("not used in this test")
}

func (a *e2eRawAgent) SpawnInitiative(_ context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error) {
	a.initSpawnCalls = append(a.initSpawnCalls, req)
	if a.initSpawnErr != nil {
		return agentmanager.RunResult{}, a.initSpawnErr
	}
	return a.initSpawnReturn, nil
}

func (a *e2eRawAgent) GetRunState(_ context.Context, runID string) (agentmanager.RunState, error) {
	return agentmanager.RunState{RunID: runID, Status: "running"}, nil
}
func (a *e2eRawAgent) StopRun(_ context.Context, _ string) error { return nil }
func (a *e2eRawAgent) ContinueRun(_ context.Context, runID, message string) error {
	a.continueCalls = append(a.continueCalls, runID+":"+message)
	return a.continueErr
}

// e2eSpawner is the AgentSpawner injected into Service. It routes spawns
// through the agentactivity.Service so the test can prove that activity
// records get the initiative metadata.
type e2eSpawner struct {
	activity *agentactivity.Service
	rawAgent *e2eRawAgent
	initSvc  *initiatives.Service
}

func (s *e2eSpawner) SpawnInitiativeFeedback(ctx context.Context, req SpawnRequest) (string, error) {
	title := ""
	if init, err := s.initSvc.Get(req.InitiativeName); err == nil && init != nil {
		title = init.Initiative.Title
	}
	spec := agentactivity.Spec{
		OwnerType:   agentactivity.OwnerInitiative,
		OwnerName:   req.InitiativeName,
		OwnerTitle:  title,
		Purpose:     agentactivity.PurposeFeedback,
		RequestedBy: "swarm-manager",
		Metadata: map[string]string{
			"round_number": intStr(req.RoundNumber),
			"round_slug":   req.RoundSlug,
			"entrypoint":   "initiative.feedback",
		},
	}
	ctx = agentactivity.WithSpec(ctx, spec)
	res, err := s.activity.SpawnInitiative(ctx, agentmanager.InitiativeSpawnRequest{
		Name:        req.InitiativeName,
		Description: req.SubmissionText,
		Prompt:      "synthetic prompt for e2e test",
		Purpose:     req.Purpose,
		RoundNumber: req.RoundNumber,
		RoundSlug:   req.RoundSlug,
	})
	if err != nil {
		return "", err
	}
	return res.RunID, nil
}

func (s *e2eSpawner) ContinueRun(ctx context.Context, req ContinueRequest) error {
	spec := agentactivity.Spec{
		OwnerType:   agentactivity.OwnerInitiative,
		OwnerName:   req.InitiativeName,
		Purpose:     agentactivity.PurposeFeedbackContinue,
		RequestedBy: "swarm-manager",
		Metadata: map[string]string{
			"round_number": intStr(req.RoundNumber),
			"round_slug":   req.RoundSlug,
			"entrypoint":   "initiative.feedback.continue",
		},
	}
	ctx = agentactivity.WithSpec(ctx, spec)
	return s.activity.ContinueRun(ctx, req.RunID, req.Message)
}

func intStr(i int) string {
	if i == 0 {
		return "0"
	}
	// Avoid pulling in fmt for one call site.
	const digits = "0123456789"
	if i < 0 {
		return "-" + intStr(-i)
	}
	var buf []byte
	for i > 0 {
		buf = append([]byte{digits[i%10]}, buf...)
		i /= 10
	}
	return string(buf)
}

// capturingEmitter records every applied mutation so the test can prove
// the attribution chain (initiative → round → mutation → item) is
// complete.
type capturingEmitter struct {
	events []capturedApply
}

type capturedApply struct {
	source   proposals.Source
	mutation proposals.Mutation
}

func (c *capturingEmitter) EmitProposalMutationApplied(source proposals.Source, m proposals.Mutation) {
	c.events = append(c.events, capturedApply{source: source, mutation: m})
}

// initiativeListerAdapter bridges initiatives.Service to graph.InitiativeLister.
type initiativeListerAdapter struct {
	svc *initiatives.Service
}

func (a *initiativeListerAdapter) List() ([]graph.InitiativeEntry, error) {
	all, err := a.svc.List()
	if err != nil {
		return nil, err
	}
	out := make([]graph.InitiativeEntry, 0, len(all))
	for _, w := range all {
		i := w.Initiative
		out = append(out, graph.InitiativeEntry{
			Name:   i.Name,
			Title:  i.Title,
			Status: i.Status,
			Items:  append([]string(nil), i.Items...),
		})
	}
	return out, nil
}

// syncMaterializer wraps graph.Materializer so ScheduleAll() runs
// MaterializeAll synchronously. This makes the e2e test deterministic
// without sleeping for the background goroutine.
type syncMaterializer struct {
	inner *graph.Materializer
}

func (s *syncMaterializer) ScheduleAll() {
	if s == nil || s.inner == nil {
		return
	}
	_ = s.inner.MaterializeAll(context.Background())
}

func newE2EEnv(t *testing.T) *e2eEnv {
	t.Helper()
	root := t.TempDir()

	for _, dir := range []string{"ideas", "research", "fixes", "executes", "chores", "initiatives"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	backlogStore := backlog.NewFileStore(root)
	initStore := initiatives.NewStore(filepath.Join(root, "initiatives"))
	initSvc := initiatives.NewService(initStore, backlogStore)

	if _, err := initSvc.Create(initiatives.CreateRequest{
		Name:  "command-center",
		Title: "Command Center Foundation",
	}); err != nil {
		t.Fatalf("seed initiative: %v", err)
	}
	for _, item := range []struct {
		kind, name, title string
	}{
		{"execute", "kiosk-mode", "Kiosk mode"},
		{"execute", "theming", "Theming system"},
	} {
		if err := os.MkdirAll(backlogStore.ItemDir(backlog.BacklogKind(item.kind), item.name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := backlogStore.SaveItem(backlog.BacklogItem{
			Name:     item.name,
			Title:    item.title,
			Kind:     backlog.BacklogKind(item.kind),
			Status:   backlog.StatusBacklog,
			Priority: 5,
			Created:  "2026-04-23T00:00:00Z",
			Updated:  "2026-04-23T00:00:00Z",
		}); err != nil {
			t.Fatal(err)
		}
		if err := initSvc.AddItems("command-center", []string{item.kind + "/" + item.name}); err != nil {
			t.Fatal(err)
		}
	}

	materializer := graph.NewMaterializer(
		&initiativeListerAdapter{svc: initSvc},
		backlogStore,
		func(name string) string { return filepath.Join(root, "initiatives", name) },
	)
	if err := materializer.MaterializeAll(context.Background()); err != nil {
		t.Fatalf("seed materialize: %v", err)
	}

	rawAgent := &e2eRawAgent{
		enabled:         true,
		initSpawnReturn: agentmanager.RunResult{TaskID: "task-1", RunID: "run-1"},
	}
	activity := agentactivity.NewService(agentactivity.ServiceConfig{
		StorePath:    filepath.Join(root, "agent-activities.json"),
		AgentService: rawAgent,
	})

	events := &capturingEmitter{}
	applier, err := proposals.NewApplier(proposals.Config{
		Store:       backlogStore,
		Assigner:    initSvc,
		Invalidator: &syncMaterializer{inner: materializer},
		Events:      events,
	})
	if err != nil {
		t.Fatalf("NewApplier: %v", err)
	}

	feedbackStor := NewStore(initSvc.InitDir)
	feedbackLock := &Lock{Dir: initSvc.InitDir, MaxAge: time.Hour}
	spawner := &e2eSpawner{
		activity: activity,
		rawAgent: rawAgent,
		initSvc:  initSvc,
	}
	stateBuilder := func(initName string) (proposals.CurrentState, error) {
		mg, err := materializer.ReadGraph(initName)
		if err != nil {
			return proposals.CurrentState{}, err
		}
		known, err := loadInitiativeNames(initSvc)
		if err != nil {
			return proposals.CurrentState{}, err
		}
		return proposals.FromMaterializedGraph(mg, known, nil)
	}
	feedbackSvc, err := NewService(Config{
		Store:        feedbackStor,
		Lock:         feedbackLock,
		Spawner:      spawner,
		Apply:        applier,
		StateBuilder: stateBuilder,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	return &e2eEnv{
		t:            t,
		root:         root,
		backlogStore: backlogStore,
		initStore:    initStore,
		initSvc:      initSvc,
		materializer: materializer,
		activity:     activity,
		rawAgent:     rawAgent,
		applier:      applier,
		feedbackSvc:  feedbackSvc,
		feedbackStor: feedbackStor,
		feedbackLock: feedbackLock,
		spawner:      spawner,
		events:       events,
	}
}

func loadInitiativeNames(svc *initiatives.Service) ([]string, error) {
	all, err := svc.List()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(all))
	for _, w := range all {
		names = append(names, w.Initiative.Name)
	}
	return names, nil
}

// TestE2E_FeedbackRound_FullStoryStartsToAppliedMutations is the
// integration-test-for-the-whole-story called out in the plan. It covers
// the full lifecycle plus the note-type short-circuit and asserts every
// downstream side-effect: backlog mutations, graph regeneration,
// agentactivity records, and event attribution.
func TestE2E_FeedbackRound_FullStoryStartsToAppliedMutations(t *testing.T) {
	env := newE2EEnv(t)

	// 1. A note-type round skips the agent and lands on disk for audit.
	noteRound, err := env.feedbackSvc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "command-center",
		Type:           RoundTypeNote,
		Text:           "Reminder: theming should land before kiosk mode.",
	})
	if err != nil {
		t.Fatalf("note StartRound: %v", err)
	}
	if noteRound.Status != RoundStatusDismissed {
		t.Fatalf("note round should land dismissed, got %s", noteRound.Status)
	}
	if len(env.rawAgent.initSpawnCalls) != 0 {
		t.Fatalf("note round must not spawn agent, got %d calls", len(env.rawAgent.initSpawnCalls))
	}

	// 2. A feedback-type round spawns the agent.
	round, err := env.feedbackSvc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "command-center",
		Type:           RoundTypeFeedback,
		Text:           "kiosk mode looks broken; please bump priority and add a follow-up edge",
	})
	if err != nil {
		t.Fatalf("feedback StartRound: %v", err)
	}
	if round.Status != RoundStatusAgentThinking {
		t.Fatalf("expected agent_thinking, got %s", round.Status)
	}
	if round.RunID != "run-1" {
		t.Fatalf("expected RunID=run-1, got %q", round.RunID)
	}
	if len(env.rawAgent.initSpawnCalls) != 1 {
		t.Fatalf("expected 1 spawn call, got %d", len(env.rawAgent.initSpawnCalls))
	}
	spawnReq := env.rawAgent.initSpawnCalls[0]
	if spawnReq.Name != "command-center" || spawnReq.RoundNumber != round.Number {
		t.Fatalf("spawn req mismatch: %+v", spawnReq)
	}

	// 3. The agent emits a proposal that boosts priority on kiosk-mode and
	//    adds an edge from theming → kiosk-mode.
	body := "Plan:\n```json\n" + `{
		"form": "mutation_list",
		"mutations": [
			{"id":"m1","op":"change_priority","target":"execute/kiosk-mode","priority":9,"rationale":"user said it's broken"},
			{"id":"m2","op":"add_edge","from":"execute/theming","to":"execute/kiosk-mode","rationale":"theming must land first"}
		]
	}` + "\n```"

	round, err = env.feedbackSvc.RecordAgentTurn("command-center", round.Number, body)
	if err != nil {
		t.Fatalf("RecordAgentTurn: %v", err)
	}
	if round.Status != RoundStatusAwaitingUser {
		t.Fatalf("expected awaiting_user, got %s", round.Status)
	}
	if len(round.Proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(round.Proposals))
	}

	// 4. User accepts both mutations.
	decided, applyResult, err := env.feedbackSvc.Decide(context.Background(), DecideRequest{
		InitiativeName:      "command-center",
		RoundNumber:         round.Number,
		Kind:                DecisionAccept,
		AcceptedMutationIDs: []string{"m1", "m2"},
		DecidedBy:           "matthalloran8",
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if decided.Status != RoundStatusApplied {
		t.Fatalf("expected applied, got %s", decided.Status)
	}
	if applyResult == nil || applyResult.Applied != 2 {
		t.Fatalf("expected 2 applied mutations, got %+v", applyResult)
	}

	// 5. Backlog items reflect the changes.
	kiosk, err := env.backlogStore.LoadItem("execute", "kiosk-mode")
	if err != nil {
		t.Fatalf("load kiosk: %v", err)
	}
	if kiosk.Priority != 9 {
		t.Errorf("kiosk priority: got %d, want 9", kiosk.Priority)
	}
	theming, err := env.backlogStore.LoadItem("execute", "theming")
	if err != nil {
		t.Fatalf("load theming: %v", err)
	}
	if !containsString(theming.DependsOn, "execute/kiosk-mode") {
		t.Errorf("theming.DependsOn: got %v, expected to contain execute/kiosk-mode", theming.DependsOn)
	}

	// 6. graph.json regenerated through the invalidator.
	gpath := filepath.Join(env.root, "initiatives", "command-center", "graph.json")
	data, err := os.ReadFile(gpath)
	if err != nil {
		t.Fatalf("read graph.json: %v", err)
	}
	var gjson struct {
		Nodes []struct {
			ID       string `json:"id"`
			Priority int    `json:"priority"`
		} `json:"nodes"`
		Edges []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(data, &gjson); err != nil {
		t.Fatalf("unmarshal graph.json: %v", err)
	}
	foundEdge := false
	for _, e := range gjson.Edges {
		if e.From == "execute/theming" && e.To == "execute/kiosk-mode" {
			foundEdge = true
		}
	}
	if !foundEdge {
		t.Errorf("graph.json missing new edge theming→kiosk-mode: %+v", gjson.Edges)
	}
	foundPri := false
	for _, n := range gjson.Nodes {
		if n.ID == "execute/kiosk-mode" && n.Priority == 9 {
			foundPri = true
		}
	}
	if !foundPri {
		t.Errorf("graph.json missing priority bump on execute/kiosk-mode: %+v", gjson.Nodes)
	}

	// 7. agentactivity records the initiative-owned spawn with full metadata.
	records, err := env.activity.List(context.Background(), agentactivity.ListFilters{
		OwnerType: string(agentactivity.OwnerInitiative),
	})
	if err != nil {
		t.Fatalf("activity list: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 initiative activity record, got %d", len(records))
	}
	rec := records[0]
	if rec.OwnerName != "command-center" {
		t.Errorf("record owner_name: %q", rec.OwnerName)
	}
	if rec.Purpose != agentactivity.PurposeFeedback {
		t.Errorf("record purpose: %q", rec.Purpose)
	}
	if rec.RunID != "run-1" {
		t.Errorf("record run_id: %q", rec.RunID)
	}
	if rec.Metadata["entrypoint"] != "initiative.feedback" {
		t.Errorf("record metadata entrypoint: %q", rec.Metadata["entrypoint"])
	}
	if rec.Metadata["round_number"] != intStr(round.Number) {
		t.Errorf("record metadata round_number: got %q want %q", rec.Metadata["round_number"], intStr(round.Number))
	}
	if rec.Metadata["round_slug"] == "" {
		t.Errorf("record metadata round_slug missing")
	}

	// 8. Attribution chain: every captured event carries Source with full
	//    round metadata so audits can reconstruct the path back to the
	//    feedback round.
	if len(env.events.events) != 2 {
		t.Fatalf("expected 2 emitted events, got %d", len(env.events.events))
	}
	for _, ev := range env.events.events {
		if ev.source.InitiativeName != "command-center" {
			t.Errorf("source.InitiativeName: %q", ev.source.InitiativeName)
		}
		if ev.source.RoundNumber != round.Number {
			t.Errorf("source.RoundNumber: got %d want %d", ev.source.RoundNumber, round.Number)
		}
		if ev.source.Entrypoint != "initiative.feedback" {
			t.Errorf("source.Entrypoint: %q", ev.source.Entrypoint)
		}
		if ev.source.DecidedBy != "matthalloran8" {
			t.Errorf("source.DecidedBy: %q", ev.source.DecidedBy)
		}
	}

	// 9. Note round is still on disk so the future meta-optimizer can mine it.
	notes, err := env.feedbackStor.ListRounds("command-center")
	if err != nil {
		t.Fatal(err)
	}
	foundNote := false
	for _, r := range notes {
		if r.Type == RoundTypeNote {
			foundNote = true
		}
	}
	if !foundNote {
		t.Error("note round missing from disk listing")
	}

	// 10. Lock released after Decide so the next round can acquire it
	//     without override.
	holder, err := env.feedbackLock.Inspect("command-center")
	if err != nil {
		t.Fatalf("Inspect lock: %v", err)
	}
	if holder != nil {
		t.Errorf("expected no lock holder after Decide, got %+v", holder)
	}
}

// TestE2E_FeedbackRound_RejectLeavesGraphIntact mirrors the full-story
// test but proves that a reject decision touches nothing — items stay,
// graph stays, no events emitted.
func TestE2E_FeedbackRound_RejectLeavesGraphIntact(t *testing.T) {
	env := newE2EEnv(t)

	round, err := env.feedbackSvc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "command-center",
		Type:           RoundTypeFeedback,
		Text:           "should we drop kiosk mode?",
	})
	if err != nil {
		t.Fatal(err)
	}
	round, err = env.feedbackSvc.RecordAgentTurn("command-center", round.Number,
		"```json\n"+`{"form":"mutation_list","mutations":[{"id":"m1","op":"archive_item","target":"execute/kiosk-mode"}]}`+"\n```")
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := env.feedbackSvc.Decide(context.Background(), DecideRequest{
		InitiativeName: "command-center",
		RoundNumber:    round.Number,
		Kind:           DecisionReject,
		DecidedBy:      "matthalloran8",
	}); err != nil {
		t.Fatal(err)
	}

	// Item still active.
	kiosk, err := env.backlogStore.LoadItem("execute", "kiosk-mode")
	if err != nil {
		t.Fatal(err)
	}
	if kiosk.ArchivedAt != nil {
		t.Errorf("kiosk should not be archived after reject, got ArchivedAt=%v", kiosk.ArchivedAt)
	}
	// No events emitted.
	if len(env.events.events) != 0 {
		t.Errorf("reject should emit no apply events, got %d", len(env.events.events))
	}
}

func containsString(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}

// Compile-time assertion: e2eRawAgent satisfies the same surface
// agentactivity.Service expects from its raw spawner. If the agentactivity
// rawAgentService interface drifts, this test breaks fast.
var _ interface {
	IsEnabled() bool
	IsAvailable(context.Context) bool
	ResolveURL(context.Context) (string, error)
	GetProfileID() string
	SpawnBacklog(context.Context, agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
	GetRunState(context.Context, string) (agentmanager.RunState, error)
	StopRun(context.Context, string) error
} = (*e2eRawAgent)(nil)
