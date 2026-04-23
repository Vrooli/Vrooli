package feedback

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/proposals"
)

// fakeSpawner lets service_test exercise the full lifecycle without an
// actual agent-manager. It records spawn/continue calls and simulates a
// persistent RunID per round.
type fakeSpawner struct {
	spawnCalls    []SpawnRequest
	continueCalls []string
	returnRunID   string
	spawnErr      error
}

func (f *fakeSpawner) SpawnInitiativeFeedback(_ context.Context, req SpawnRequest) (string, error) {
	f.spawnCalls = append(f.spawnCalls, req)
	if f.spawnErr != nil {
		return "", f.spawnErr
	}
	if f.returnRunID == "" {
		return "run-1", nil
	}
	return f.returnRunID, nil
}

func (f *fakeSpawner) ContinueRun(_ context.Context, runID, message string, _ []string) error {
	f.continueCalls = append(f.continueCalls, runID+":"+message)
	return nil
}

type serviceEnv struct {
	t       *testing.T
	root    string
	store   *Store
	lock    *Lock
	applier *proposals.Applier
	svc     *Service
	bStore  *backlog.FileStore
	iSvc    *initiatives.Service
	spawner *fakeSpawner
}

func newServiceEnv(t *testing.T) *serviceEnv {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"ideas", "research", "fixes", "executes", "chores", "initiatives"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	bStore := backlog.NewFileStore(root)
	iStore := initiatives.NewStore(filepath.Join(root, "initiatives"))
	iSvc := initiatives.NewService(iStore, bStore)

	if _, err := iSvc.Create(initiatives.CreateRequest{Name: "ui-rewrite", Title: "UI Rewrite"}); err != nil {
		t.Fatal(err)
	}
	if err := seedBacklogItem(bStore, "execute", "foo", "Foo"); err != nil {
		t.Fatal(err)
	}
	if err := iSvc.AddItems("ui-rewrite", []string{"execute/foo"}); err != nil {
		t.Fatal(err)
	}

	applier, err := proposals.NewApplier(proposals.Config{
		Store:    bStore,
		Assigner: iSvc,
	})
	if err != nil {
		t.Fatal(err)
	}

	store := NewStore(iSvc.InitDir)
	lock := &Lock{Dir: iSvc.InitDir, MaxAge: time.Hour}
	spawner := &fakeSpawner{returnRunID: "run-42"}

	stateBuilder := func(name string) (proposals.CurrentState, error) {
		return proposals.CurrentState{
			InitiativeName: name,
			Nodes: map[string]proposals.GraphNode{
				"execute/foo": {ID: "execute/foo", Kind: "execute", Name: "foo", Title: "Foo", Priority: 5},
			},
			KnownInitiatives: map[string]struct{}{"ui-rewrite": {}},
		}, nil
	}
	svc, err := NewService(Config{
		Store:        store,
		Lock:         lock,
		Spawner:      spawner,
		Apply:        applier,
		StateBuilder: stateBuilder,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &serviceEnv{
		t: t, root: root,
		store: store, lock: lock, applier: applier, svc: svc,
		bStore: bStore, iSvc: iSvc, spawner: spawner,
	}
}

func seedBacklogItem(store *backlog.FileStore, kind, name, title string) error {
	if err := os.MkdirAll(store.ItemDir(backlog.BacklogKind(kind), name), 0o755); err != nil {
		return err
	}
	return store.SaveItem(backlog.BacklogItem{
		Name:     name,
		Title:    title,
		Kind:     backlog.BacklogKind(kind),
		Status:   backlog.StatusBacklog,
		Priority: 5,
		Created:  "2026-04-23T00:00:00Z",
		Updated:  "2026-04-23T00:00:00Z",
	})
}

func TestService_StartRound_Feedback_SpawnsAgentAndPersists(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "Fix the UI",
		DecidedBy:      "tester",
	})
	if err != nil {
		t.Fatalf("StartRound: %v", err)
	}
	if round.Number != 1 {
		t.Fatalf("expected round 1, got %d", round.Number)
	}
	if round.Status != RoundStatusAgentThinking {
		t.Fatalf("expected agent_thinking, got %s", round.Status)
	}
	if round.RunID != "run-42" {
		t.Fatalf("expected run id run-42, got %q", round.RunID)
	}
	if len(env.spawner.spawnCalls) != 1 {
		t.Fatalf("expected 1 spawn call, got %d", len(env.spawner.spawnCalls))
	}
	holder, _ := env.lock.Inspect("ui-rewrite")
	if holder == nil || holder.RunID != "run-42" {
		t.Fatalf("expected lock held by run-42, got %+v", holder)
	}
}

func TestService_StartRound_Note_SkipsAgent(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeNote,
		Text:           "Just a note",
	})
	if err != nil {
		t.Fatalf("StartRound(note): %v", err)
	}
	if round.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", round.Status)
	}
	if len(env.spawner.spawnCalls) != 0 {
		t.Fatalf("expected no spawn for note, got %d", len(env.spawner.spawnCalls))
	}
	if round.Decision == nil || round.Decision.Kind != DecisionDismiss {
		t.Fatalf("expected dismiss decision, got %+v", round.Decision)
	}
	holder, _ := env.lock.Inspect("ui-rewrite")
	if holder != nil {
		t.Fatalf("expected no lock for note round, got %+v", holder)
	}
}

func TestService_StartRound_RejectsResearchType(t *testing.T) {
	env := newServiceEnv(t)
	_, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeResearch,
		Text:           "go deep",
	})
	if err == nil || !strings.Contains(err.Error(), "research-type rounds are not implemented") {
		t.Fatalf("expected research-not-implemented error, got %v", err)
	}
}

func TestService_StartRound_RejectsIfLocked(t *testing.T) {
	env := newServiceEnv(t)
	if err := env.lock.Acquire("ui-rewrite", Holder{RunID: "prior", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}
	_, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "hi",
	})
	if err == nil || !errors.Is(err, ErrLocked) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
}

func TestService_StartRound_OverridePreempts(t *testing.T) {
	env := newServiceEnv(t)
	if err := env.lock.Acquire("ui-rewrite", Holder{RunID: "prior", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "preempt",
		Override:       true,
	})
	if err != nil {
		t.Fatalf("override StartRound: %v", err)
	}
	if round.RunID != "run-42" {
		t.Fatalf("expected new run id, got %q", round.RunID)
	}
}

func TestService_RecordAgentTurn_ExtractsProposal(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "start",
	})
	if err != nil {
		t.Fatal(err)
	}

	agentBody := "Here's my plan.\n\n```json\n" +
		`{"form":"mutation_list","mutations":[` +
		`{"id":"m1","op":"change_priority","target":"execute/foo","priority":8}` +
		`]}` +
		"\n```\n\nReview and accept what looks right."

	round, err = env.svc.RecordAgentTurn("ui-rewrite", round.Number, agentBody)
	if err != nil {
		t.Fatalf("RecordAgentTurn: %v", err)
	}
	if round.Status != RoundStatusAwaitingUser {
		t.Fatalf("expected awaiting_user, got %s", round.Status)
	}
	if len(round.Proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(round.Proposals))
	}
	if round.CurrentProposalID != "p1" {
		t.Fatalf("expected current=p1, got %s", round.CurrentProposalID)
	}
	holder, _ := env.lock.Inspect("ui-rewrite")
	if holder != nil {
		t.Fatalf("expected lock released after agent turn, got %+v", holder)
	}
}

func TestService_RecordAgentTurn_HandlesUnparsableProposal(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "start",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Malformed JSON — the turn still lands, but no proposal attaches and
	// a parse warning is surfaced on the round.
	body := "No proposal today.\n```json\n{this is not json}\n```\n"
	round, err = env.svc.RecordAgentTurn("ui-rewrite", round.Number, body)
	if err != nil {
		t.Fatalf("RecordAgentTurn: %v", err)
	}
	if len(round.Proposals) != 0 {
		t.Fatalf("expected no proposal, got %d", len(round.Proposals))
	}
	if round.Status != RoundStatusAwaitingUser {
		t.Fatalf("expected awaiting_user, got %s", round.Status)
	}
}

func TestService_Decide_AcceptAppliesProposal(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "start",
	})
	if err != nil {
		t.Fatal(err)
	}
	round, err = env.svc.RecordAgentTurn("ui-rewrite", round.Number,
		"Plan:\n```json\n"+
			`{"form":"mutation_list","mutations":[{"id":"m1","op":"change_priority","target":"execute/foo","priority":9}]}`+
			"\n```",
	)
	if err != nil {
		t.Fatal(err)
	}

	decided, result, err := env.svc.Decide(context.Background(), DecideRequest{
		InitiativeName:      "ui-rewrite",
		RoundNumber:         round.Number,
		Kind:                DecisionAccept,
		AcceptedMutationIDs: []string{"m1"},
		DecidedBy:           "tester",
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if decided.Status != RoundStatusApplied {
		t.Fatalf("expected applied, got %s", decided.Status)
	}
	if result == nil || result.Applied != 1 {
		t.Fatalf("expected 1 mutation applied, got %+v", result)
	}

	item, err := env.bStore.LoadItem("execute", "foo")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if item.Priority != 9 {
		t.Fatalf("expected priority updated to 9, got %d", item.Priority)
	}
}

func TestService_Decide_RejectDoesNotApply(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "start",
	})
	if err != nil {
		t.Fatal(err)
	}
	round, err = env.svc.RecordAgentTurn("ui-rewrite", round.Number,
		"No:\n```json\n"+
			`{"form":"mutation_list","mutations":[{"id":"m1","op":"change_priority","target":"execute/foo","priority":3}]}`+
			"\n```",
	)
	if err != nil {
		t.Fatal(err)
	}
	decided, _, err := env.svc.Decide(context.Background(), DecideRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
		Kind:           DecisionReject,
	})
	if err != nil {
		t.Fatalf("Decide reject: %v", err)
	}
	if decided.Status != RoundStatusRejected {
		t.Fatalf("expected rejected, got %s", decided.Status)
	}
	item, _ := env.bStore.LoadItem("execute", "foo")
	if item.Priority != 5 {
		t.Fatalf("expected priority unchanged at 5, got %d", item.Priority)
	}
}

func TestService_ContinueRound_AppendsAndFlips(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "start",
	})
	if err != nil {
		t.Fatal(err)
	}
	round, err = env.svc.RecordAgentTurn("ui-rewrite", round.Number, "no proposal")
	if err != nil {
		t.Fatal(err)
	}

	round, err = env.svc.ContinueRound(context.Background(), ContinueRoundRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
		Text:           "revise please",
	})
	if err != nil {
		t.Fatalf("ContinueRound: %v", err)
	}
	if round.Status != RoundStatusAgentThinking {
		t.Fatalf("expected agent_thinking, got %s", round.Status)
	}
	if len(env.spawner.continueCalls) != 1 {
		t.Fatalf("expected 1 continue call, got %d", len(env.spawner.continueCalls))
	}
	if !strings.Contains(env.spawner.continueCalls[0], "revise please") {
		t.Fatalf("expected continue call to carry message, got %q", env.spawner.continueCalls[0])
	}
}

func TestService_NextRoundNumber_Increments(t *testing.T) {
	env := newServiceEnv(t)
	for i := 1; i <= 3; i++ {
		round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
			InitiativeName: "ui-rewrite",
			Type:           RoundTypeNote, // note doesn't take the lock
			Text:           "n",
		})
		if err != nil {
			t.Fatalf("start %d: %v", i, err)
		}
		if round.Number != i {
			t.Fatalf("expected round %d, got %d", i, round.Number)
		}
	}
}
