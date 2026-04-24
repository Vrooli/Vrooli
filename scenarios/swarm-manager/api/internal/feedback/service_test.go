package feedback

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/proposals"
)

// fakeSpawner lets service_test exercise the full lifecycle without an
// actual agent-manager. It records spawn/continue calls and simulates a
// persistent RunID per round.
type fakeSpawner struct {
	spawnCalls    []SpawnRequest
	continueCalls []ContinueRequest
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

func (f *fakeSpawner) ContinueRun(_ context.Context, req ContinueRequest) error {
	f.continueCalls = append(f.continueCalls, req)
	return nil
}

type serviceEnv struct {
	t       *testing.T
	root    string
	store   *Store
	lock    *initiativelock.Lock
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

	creator, err := backlog.NewService(backlog.ServiceConfig{Store: bStore, Assigner: iSvc})
	if err != nil {
		t.Fatal(err)
	}
	applier, err := proposals.NewApplier(proposals.Config{
		Store:    bStore,
		Assigner: iSvc,
		Creator:  creator,
	})
	if err != nil {
		t.Fatal(err)
	}

	store := NewStore(iSvc.InitDir)
	lock := &initiativelock.Lock{Dir: iSvc.InitDir, MaxAge: time.Hour}
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
	// Auto-populated rationale distinguishes a note from a user-driven
	// dismiss on a live feedback round — the meta-optimizer reader needs
	// this to filter out notes without inspecting Type.
	if round.Decision.Rationale != "note" {
		t.Fatalf("expected auto rationale=note, got %q", round.Decision.Rationale)
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
	if err := env.lock.Acquire("ui-rewrite", initiativelock.Holder{RunID: "prior", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}
	_, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "hi",
	})
	if err == nil || !errors.Is(err, initiativelock.ErrLocked) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
}

func TestService_StartRound_OverridePreempts(t *testing.T) {
	env := newServiceEnv(t)
	if err := env.lock.Acquire("ui-rewrite", initiativelock.Holder{RunID: "prior", Purpose: "feedback"}); err != nil {
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

// captureEvents records Source/Mutation pairs from proposals.Applier so the
// feedback service test can assert that round metadata reaches the
// attribution surface (proposals.Source fields).
type captureEvents struct {
	sources   []proposals.Source
	mutations []proposals.Mutation
}

func (c *captureEvents) EmitProposalMutationApplied(s proposals.Source, m proposals.Mutation) {
	c.sources = append(c.sources, s)
	c.mutations = append(c.mutations, m)
}

func TestService_Decide_PopulatesProposalSourceWithRoundMetadata(t *testing.T) {
	t.Parallel()
	env := newServiceEnv(t)

	// Rebuild applier + service with a capturing emitter.
	cap := &captureEvents{}
	creator, err := backlog.NewService(backlog.ServiceConfig{Store: env.bStore, Assigner: env.iSvc})
	if err != nil {
		t.Fatal(err)
	}
	applier, err := proposals.NewApplier(proposals.Config{
		Store:    env.bStore,
		Assigner: env.iSvc,
		Creator:  creator,
		Events:   cap,
	})
	if err != nil {
		t.Fatal(err)
	}
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
		Store:        env.store,
		Lock:         env.lock,
		Spawner:      env.spawner,
		Apply:        applier,
		StateBuilder: stateBuilder,
	})
	if err != nil {
		t.Fatal(err)
	}

	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "investigate priority",
		SlugHint:       "boost-foo",
	})
	if err != nil {
		t.Fatal(err)
	}
	round, err = svc.RecordAgentTurn("ui-rewrite", round.Number,
		"```json\n"+
			`{"form":"mutation_list","mutations":[{"id":"m1","op":"change_priority","target":"execute/foo","priority":9}]}`+
			"\n```",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Decide(context.Background(), DecideRequest{
		InitiativeName:      "ui-rewrite",
		RoundNumber:         round.Number,
		Kind:                DecisionAccept,
		AcceptedMutationIDs: []string{"m1"},
		DecidedBy:           "matthalloran8",
	}); err != nil {
		t.Fatalf("Decide: %v", err)
	}

	if len(cap.sources) != 1 {
		t.Fatalf("expected 1 captured source, got %d", len(cap.sources))
	}
	src := cap.sources[0]
	if src.InitiativeName != "ui-rewrite" {
		t.Errorf("InitiativeName: got %q", src.InitiativeName)
	}
	if src.RoundNumber != round.Number {
		t.Errorf("RoundNumber: got %d, want %d", src.RoundNumber, round.Number)
	}
	if src.RoundSlug != round.Slug {
		t.Errorf("RoundSlug: got %q, want %q", src.RoundSlug, round.Slug)
	}
	if src.Entrypoint != "initiative.feedback" {
		t.Errorf("Entrypoint: got %q, want %q", src.Entrypoint, "initiative.feedback")
	}
	if src.DecidedBy != "matthalloran8" {
		t.Errorf("DecidedBy: got %q", src.DecidedBy)
	}
	if src.FeedbackRoundID == "" {
		t.Error("FeedbackRoundID empty")
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
	got := env.spawner.continueCalls[0]
	if !strings.Contains(got.Message, "revise please") {
		t.Fatalf("expected continue call to carry message, got %q", got.Message)
	}
	if got.InitiativeName != "ui-rewrite" {
		t.Fatalf("expected InitiativeName=ui-rewrite, got %q", got.InitiativeName)
	}
	if got.RoundNumber != round.Number {
		t.Fatalf("expected RoundNumber=%d, got %d", round.Number, got.RoundNumber)
	}
	if got.RunID == "" {
		t.Fatal("expected non-empty RunID in continue request")
	}
}

func TestService_StartRound_SpawnFailureRecordsErrorAndReleasesLock(t *testing.T) {
	env := newServiceEnv(t)
	env.spawner.spawnErr = errors.New("agent-manager unreachable")

	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "kick off",
	})
	if err == nil {
		t.Fatalf("expected spawn error to surface")
	}
	if round.Status != RoundStatusAwaitingUser {
		t.Fatalf("expected round to land in awaiting_user after spawn fail, got %s", round.Status)
	}
	// Last thread message should describe the spawn failure so the user
	// knows why the round flipped to awaiting_user without a proposal.
	if len(round.Thread) < 2 {
		t.Fatalf("expected user + agent-error messages in thread, got %+v", round.Thread)
	}
	last := round.Thread[len(round.Thread)-1]
	if last.Role != "agent" || !strings.Contains(last.Content, "agent spawn failed") {
		t.Fatalf("expected trailing agent error message, got %+v", last)
	}
	// Lock must be released — otherwise the next submit gets a stale
	// 409 even though the agent never actually started.
	if holder, _ := env.lock.Inspect("ui-rewrite"); holder != nil {
		t.Fatalf("expected lock released after spawn failure, got %+v", holder)
	}
	// Round was persisted so a subsequent GET surfaces the error to UI.
	persisted, loadErr := env.store.LoadRound("ui-rewrite", round.Number)
	if loadErr != nil {
		t.Fatalf("load round after spawn fail: %v", loadErr)
	}
	if persisted.Status != RoundStatusAwaitingUser {
		t.Fatalf("persisted status = %s, want awaiting_user", persisted.Status)
	}
}

// stubActivityChecker lets service_test assert that BusyError carries
// multi-item blocker details, which the UI uses to render the override
// dialog with a per-item explanation.
type stubActivityChecker struct {
	activities []ItemActivity
	err        error
}

func (s *stubActivityChecker) ActiveRunsForInitiative(_ string) ([]ItemActivity, error) {
	return s.activities, s.err
}

func TestService_StartRound_BusyError_CarriesAllBlockers(t *testing.T) {
	env := newServiceEnv(t)
	activity := &stubActivityChecker{
		activities: []ItemActivity{
			{Ref: "execute/foo", RunID: "run-foo", Purpose: "execute"},
			{Ref: "execute/bar", RunID: "run-bar", Purpose: "workshop"},
			{Ref: "research/baz", RunID: "run-baz", Purpose: "research"},
		},
	}
	svc, err := NewService(Config{
		Store:    env.store,
		Lock:     env.lock,
		Spawner:  env.spawner,
		Apply:    env.applier,
		Activity: activity,
		StateBuilder: func(string) (proposals.CurrentState, error) {
			return proposals.CurrentState{InitiativeName: "ui-rewrite"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "kick off",
	})
	if err == nil {
		t.Fatal("expected BusyError, got nil")
	}
	if !errors.Is(err, ErrInitiativeBusy) {
		t.Fatalf("expected ErrInitiativeBusy wrap, got %v", err)
	}
	var busy *BusyError
	if !errors.As(err, &busy) {
		t.Fatalf("expected *BusyError, got %T", err)
	}
	if len(busy.Activities) != 3 {
		t.Fatalf("expected 3 blockers surfaced, got %d (%+v)", len(busy.Activities), busy.Activities)
	}
	// Order-preserving: the UI renders blockers in the order the
	// activity checker returned them, so the override dialog stays
	// deterministic across refreshes.
	wantRefs := []string{"execute/foo", "execute/bar", "research/baz"}
	for i, want := range wantRefs {
		if busy.Activities[i].Ref != want {
			t.Errorf("blocker[%d].Ref: got %q, want %q", i, busy.Activities[i].Ref, want)
		}
	}
}

func TestService_StartRound_BusyError_OverrideBypassesCheck(t *testing.T) {
	env := newServiceEnv(t)
	activity := &stubActivityChecker{
		activities: []ItemActivity{{Ref: "execute/foo", RunID: "run-foo", Purpose: "execute"}},
	}
	svc, err := NewService(Config{
		Store:    env.store,
		Lock:     env.lock,
		Spawner:  env.spawner,
		Apply:    env.applier,
		Activity: activity,
		StateBuilder: func(string) (proposals.CurrentState, error) {
			return proposals.CurrentState{InitiativeName: "ui-rewrite"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	round, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "preempt",
		Override:       true,
	})
	if err != nil {
		t.Fatalf("expected override to bypass busy check: %v", err)
	}
	if round.Status != RoundStatusAgentThinking {
		t.Fatalf("expected agent_thinking, got %s", round.Status)
	}
}

// recordingCanceller captures every StopRun call so override-path tests
// can assert exactly which agent runs were preempted. "Single agent per
// initiative" only holds if override actually cancels — otherwise the
// lock file is overwritten but the previous agent keeps running in the
// background.
type recordingCanceller struct {
	stopped []string
	err     error
}

func (c *recordingCanceller) StopRun(_ context.Context, runID string) error {
	c.stopped = append(c.stopped, runID)
	return c.err
}

// Override must cancel the previous feedback round's agent-manager run
// *and* mark the preempted round dismissed so the audit log explains why
// it never completed. This is what turns "override" from a lock-file
// overwrite into actual single-agent-per-initiative enforcement.
func TestService_StartRound_Override_CancelsPriorFeedbackRun(t *testing.T) {
	env := newServiceEnv(t)
	canceller := &recordingCanceller{}
	svc, err := NewService(Config{
		Store:     env.store,
		Lock:      env.lock,
		Spawner:   env.spawner,
		Apply:     env.applier,
		Canceller: canceller,
		StateBuilder: func(string) (proposals.CurrentState, error) {
			return proposals.CurrentState{InitiativeName: "ui-rewrite"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// First round — reaches agent_thinking and takes the lock with
	// spawner's run-42.
	first, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "first",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.RunID == "" || first.Status != RoundStatusAgentThinking {
		t.Fatalf("setup: expected agent_thinking round with RunID, got %+v", first)
	}

	// Second round overrides the first. Spawner is shared but reuses
	// returnRunID=run-42, so the preempted and new RunIDs differ only
	// in sequence — what we really care about is that Cancel was called
	// with the first round's RunID before the new lock was acquired.
	env.spawner.returnRunID = "run-43"
	second, err := svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "second via override",
		Override:       true,
	})
	if err != nil {
		t.Fatalf("override start: %v", err)
	}
	if second.Number == first.Number {
		t.Fatal("expected a distinct round number for the overriding round")
	}

	if len(canceller.stopped) != 1 || canceller.stopped[0] != first.RunID {
		t.Fatalf("expected StopRun(%q), got %+v", first.RunID, canceller.stopped)
	}

	// Preempted round should now be dismissed with a clear rationale
	// so the feedback log explains why round N stopped short.
	preempted, err := env.store.LoadRound("ui-rewrite", first.Number)
	if err != nil {
		t.Fatal(err)
	}
	if preempted.Status != RoundStatusDismissed {
		t.Fatalf("expected preempted round dismissed, got %s", preempted.Status)
	}
	if preempted.Decision == nil || preempted.Decision.Kind != DecisionDismiss {
		t.Fatalf("expected dismiss decision on preempted round, got %+v", preempted.Decision)
	}
	if !strings.Contains(preempted.Decision.Rationale, "preempted") {
		t.Fatalf("expected preemption rationale, got %q", preempted.Decision.Rationale)
	}
}

// Override paired with busy member items cancels each item run too — the
// single-agent guarantee covers both initiative-level and item-level
// agents. Without this the override dialog would be technically correct
// (new lock acquired) but practically broken (the workshopping agent on
// execute/foo would keep mutating state underneath the new feedback run).
func TestService_StartRound_Override_CancelsBusyItemRuns(t *testing.T) {
	env := newServiceEnv(t)
	canceller := &recordingCanceller{}
	activity := &stubActivityChecker{
		activities: []ItemActivity{
			{Ref: "execute/foo", RunID: "run-foo", Purpose: "execute"},
			{Ref: "research/bar", RunID: "run-bar", Purpose: "workshop"},
			{Ref: "idea/baz", Purpose: "classify"}, // no RunID — must skip
		},
	}
	svc, err := NewService(Config{
		Store:     env.store,
		Lock:      env.lock,
		Spawner:   env.spawner,
		Apply:     env.applier,
		Activity:  activity,
		Canceller: canceller,
		StateBuilder: func(string) (proposals.CurrentState, error) {
			return proposals.CurrentState{InitiativeName: "ui-rewrite"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "preempt everything",
		Override:       true,
	})
	if err != nil {
		t.Fatalf("override start: %v", err)
	}
	want := []string{"run-foo", "run-bar"}
	if len(canceller.stopped) != len(want) {
		t.Fatalf("expected %d StopRun calls, got %+v", len(want), canceller.stopped)
	}
	for i, runID := range want {
		if canceller.stopped[i] != runID {
			t.Errorf("StopRun[%d]: got %q, want %q", i, canceller.stopped[i], runID)
		}
	}
}

// When no canceller is wired (degraded mode), override must still succeed —
// we don't want a misconfigured deployment to block feedback outright.
// The zombie-run risk is documented on the interface; the service itself
// is resilient.
func TestService_StartRound_Override_NoCancellerStillStarts(t *testing.T) {
	env := newServiceEnv(t)
	// Pre-populate a round in agent_thinking so there's a holder to
	// preempt. Then call StartRound with override=true but no canceller.
	first, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "first",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != RoundStatusAgentThinking {
		t.Fatalf("setup: want agent_thinking, got %s", first.Status)
	}
	env.spawner.returnRunID = "run-43"
	_, err = env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "second",
		Override:       true,
	})
	if err != nil {
		t.Fatalf("override without canceller: %v", err)
	}
}

func TestService_RecordAgentTurn_ParseWarningsSurfaceOnSuccessfulExtract(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "start",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Two fenced blocks: the first is malformed JSON, the second is
	// valid. The lenient extractor tries the first, records the parse
	// error as a warning, then succeeds on the second. The revision
	// must surface the recorded warning so the UI/meta-optimizer can
	// see the agent produced noise before landing on a valid proposal.
	body := "Attempt one:\n```json\n{not json}\n```\n\nBetter:\n```json\n" +
		`{"form":"mutation_list","mutations":[{"id":"m1","op":"change_priority","target":"execute/foo","priority":8}]}` +
		"\n```\n"
	round, err = env.svc.RecordAgentTurn("ui-rewrite", round.Number, body)
	if err != nil {
		t.Fatalf("RecordAgentTurn: %v", err)
	}
	if len(round.Proposals) != 1 {
		t.Fatalf("expected 1 proposal attached, got %d", len(round.Proposals))
	}
	warnings := round.Proposals[0].ParseWarnings
	if len(warnings) == 0 {
		t.Fatal("expected ParseWarnings populated by the failed first extraction")
	}
	joined := strings.Join(warnings, "|")
	if !strings.Contains(joined, "parse proposal block") {
		t.Fatalf("expected parse-warning message, got %v", warnings)
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

// TestService_ContinueRound_PreservesRunID asserts the clarification-style
// multi-turn contract: a follow-up ContinueRound reuses the same agent-manager
// RunID assigned at StartRound, so downstream activity tracking and event
// attribution can group turns under the same run.
func TestService_ContinueRound_PreservesRunID(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "start",
	})
	if err != nil {
		t.Fatal(err)
	}
	initialRunID := round.RunID
	if initialRunID == "" {
		t.Fatal("expected RunID after StartRound")
	}

	round, err = env.svc.RecordAgentTurn("ui-rewrite", round.Number, "no proposal")
	if err != nil {
		t.Fatal(err)
	}

	continued, err := env.svc.ContinueRound(context.Background(), ContinueRoundRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
		Text:           "revise please",
	})
	if err != nil {
		t.Fatalf("ContinueRound: %v", err)
	}
	if continued.RunID != initialRunID {
		t.Fatalf("expected ContinueRound to preserve RunID %q, got %q", initialRunID, continued.RunID)
	}
	if len(env.spawner.continueCalls) != 1 {
		t.Fatalf("expected 1 continue call, got %d", len(env.spawner.continueCalls))
	}
	if env.spawner.continueCalls[0].RunID != initialRunID {
		t.Fatalf("expected ContinueRun to use RunID %q, got %q",
			initialRunID, env.spawner.continueCalls[0].RunID)
	}
}

// TestService_Decide_PartialAccept_RecordsRejectedMutations asserts that
// when a user accepts a subset of mutation IDs, the decision on disk
// records the rejected IDs so auditors can later see what was dropped.
func TestService_Decide_PartialAccept_RecordsRejectedMutations(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "start",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Agent emits a proposal with three mutations; user accepts only the first.
	body := "Plan:\n```json\n" + `{
  "form": "mutation_list",
  "mutations": [
    {"id":"m1","op":"change_priority","target":"execute/foo","priority":7},
    {"id":"m2","op":"change_priority","target":"execute/foo","priority":8},
    {"id":"m3","op":"change_priority","target":"execute/foo","priority":9}
  ]
}` + "\n```"
	round, err = env.svc.RecordAgentTurn("ui-rewrite", round.Number, body)
	if err != nil {
		t.Fatal(err)
	}

	decided, _, err := env.svc.Decide(context.Background(), DecideRequest{
		InitiativeName:      "ui-rewrite",
		RoundNumber:         round.Number,
		Kind:                DecisionPartialAccept,
		AcceptedMutationIDs: []string{"m1"},
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if decided.Decision == nil {
		t.Fatal("expected decision recorded")
	}
	got := append([]string(nil), decided.Decision.RejectedMutationIDs...)
	if len(got) != 2 {
		t.Fatalf("expected 2 rejected IDs, got %v", got)
	}
	sort.Strings(got)
	if got[0] != "m2" || got[1] != "m3" {
		t.Fatalf("expected rejected [m2 m3], got %v", got)
	}

	// Re-load from disk to confirm persistence.
	reloaded, err := env.store.LoadRound("ui-rewrite", round.Number)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Decision == nil || len(reloaded.Decision.RejectedMutationIDs) != 2 {
		t.Fatalf("rejected ids not persisted: %+v", reloaded.Decision)
	}
}

// TestService_RecordAgentTurn_UnparsableFlagsNeedsRevision asserts the
// structured parse-error signal: when the agent's output lacks a
// parseable proposal, the round lands in awaiting_user with
// NeedsRevision=true so the UI can render "ask for revision" without
// scanning the thread for warnings.
func TestService_RecordAgentTurn_UnparsableFlagsNeedsRevision(t *testing.T) {
	env := newServiceEnv(t)
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "start",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := "Here's some prose.\n```json\n{broken json}\n```"
	round, err = env.svc.RecordAgentTurn("ui-rewrite", round.Number, body)
	if err != nil {
		t.Fatal(err)
	}
	if !round.NeedsRevision {
		t.Fatal("expected NeedsRevision=true after unparsable turn")
	}
	if len(round.LastParseWarnings) == 0 {
		t.Fatal("expected LastParseWarnings populated")
	}
	if round.CurrentProposalID != "" {
		t.Fatalf("expected no current proposal, got %q", round.CurrentProposalID)
	}

	// ContinueRound clears the signal so the next turn starts clean.
	continued, err := env.svc.ContinueRound(context.Background(), ContinueRoundRequest{
		InitiativeName: "ui-rewrite",
		RoundNumber:    round.Number,
		Text:           "try again",
	})
	if err != nil {
		t.Fatal(err)
	}
	if continued.NeedsRevision {
		t.Fatal("ContinueRound should clear NeedsRevision")
	}
	if len(continued.LastParseWarnings) != 0 {
		t.Fatalf("ContinueRound should clear LastParseWarnings, got %v", continued.LastParseWarnings)
	}
}

// TestService_StartRound_SpawnFailure_ReleasesProvisionalLock covers the
// lock run-id swap fix: on spawn error the provisional holder is released
// so a follow-up StartRound succeeds without waiting for the stale-lock
// sweep.
func TestService_StartRound_SpawnFailure_ReleasesProvisionalLock(t *testing.T) {
	env := newServiceEnv(t)
	env.spawner.spawnErr = errors.New("agent-manager unreachable")

	_, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "attempt",
	})
	if err == nil {
		t.Fatal("expected spawn error")
	}
	// Lock should be released — a fresh StartRound succeeds.
	env.spawner.spawnErr = nil
	round, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeFeedback,
		Text:           "retry",
	})
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if round.RunID == "" {
		t.Fatal("expected retry to carry RunID")
	}
}
