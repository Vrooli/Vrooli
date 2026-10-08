package heartbeat

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	_ "modernc.org/sqlite"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type pmEffortOwners struct{ enabled bool }

func (o *pmEffortOwners) CurrentEffortOwner(context.Context, string) (effortauthority.OwnerState, error) {
	return effortauthority.OwnerState{Subject: "fixture-owner", Enabled: o.enabled, Scopes: []string{"agent-manager:*"}}, nil
}
func (o *pmEffortOwners) ApproveEffortOwner(context.Context, string) (effortauthority.OwnerState, error) {
	return o.CurrentEffortOwner(context.Background(), "")
}

type pmEffortScope struct{}

func (pmEffortScope) CheckEffortScope(context.Context, effortauthority.Policy) error { return nil }
func finiteAuthorityFixture(t *testing.T, keepAlive bool) (*finiteFixture, *effortauthority.Authority, *pmEffortOwners, effortauthority.Policy, ed25519.PrivateKey, *time.Time) {
	t.Helper()
	f := newFiniteFixtureWithKeepAlive(t, keepAlive)
	ctx := context.Background()
	cfg, e := f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
	if e != nil {
		t.Fatal(e)
	}
	db, e := sql.Open("sqlite", filepath.Join(t.TempDir(), "authority.db"))
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	ledger := &effortauthority.SQLStore{DB: db}
	if e = ledger.Ensure(ctx); e != nil {
		t.Fatal(e)
	}
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC().Truncate(time.Second)
	owner := &pmEffortOwners{true}
	a := &effortauthority.Authority{Store: ledger, Owners: owner, Approver: owner, Scope: pmEffortScope{}, Now: func() time.Time { return now }}
	p := effortauthority.Policy{ID: "fixture-policy", Owner: "fixture-owner", Client: "fixture-client", ClientKey: pub, Epoch: 1, Repository: t.TempDir(), Effort: cfg.FiniteLeader.EffortRef, Revision: cfg.FiniteLeader.AcceptedRevision, ContentDigest: effortauthority.Digest("accepted"), Team: "committee", TeamDigest: effortauthority.Digest("team"), BindingDigest: effortauthority.Digest(cfg.FiniteLeader), Members: []string{"lead"}, Profiles: map[string]string{cfg.ProfileKey: effortauthority.Digest("native profile")}, Scopes: []string{"agent-manager:write", "agent-manager:orchestrate"}, Effects: []string{"run.create"}, NotBefore: now, Deadline: now.Add(8 * time.Hour), MaxStarts: 1, MaxConcurrent: 1, MaxTurns: 10, MaxToolCalls: 20, MaxRunSeconds: 3600, TotalTurns: 10, TotalToolCalls: 20, TotalRunSeconds: 3600, SurviveLogout: true}
	if e = a.Approve(ctx, "disposable-human-fixture", p); e != nil {
		t.Fatal(e)
	}
	seed := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: cfg.ProfileKey, ProfileDigest: p.Profiles[cfg.ProfileKey], Effect: "run.create", IdempotencyKey: "fixture", InputDigest: effortauthority.Digest("fixture"), Turns: 10, ToolCalls: 20, RunSeconds: 3600}
	proof, _ := effortauthority.Sign(effortauthority.Proof{Intent: seed, Nonce: "fixture-initial-nonce", IssuedAt: now}, key)
	b, e := a.CheckProof(ctx, proof)
	if e != nil {
		t.Fatal(e)
	}
	f.runtime.Efforts = &FiniteEffortAuthority{Authority: a, Signer: &effortauthority.CustodySigner{Clients: map[string]crypto.Signer{p.Client: key}}, Bindings: map[string]effortauthority.Binding{"committee/lead": b}, Now: func() time.Time { return now }}
	return f, a, owner, p, key, &now
}
func fillFiniteBudget(t *testing.T, a *effortauthority.Authority, p effortauthority.Policy, key ed25519.PrivateKey) {
	t.Helper()
	i := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: "qualified-profile", ProfileDigest: p.Profiles["qualified-profile"], Effect: "run.create", IdempotencyKey: "already-occupied", InputDigest: effortauthority.Digest("old input"), Turns: 10, ToolCalls: 20, RunSeconds: 3600}
	proof, e := effortauthority.Sign(effortauthority.Proof{Intent: i, Nonce: "existing-disposable-nonce", IssuedAt: a.Now()}, key)
	if e != nil {
		t.Fatal(e)
	}
	b, e := a.CheckProof(context.Background(), proof)
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = a.Reserve(context.Background(), b, i, &proof); e != nil {
		t.Fatal(e)
	}
}
func TestFiniteEffortTickAndDispatchExhaustionHaveZeroEffects(t *testing.T) {
	for _, operation := range []string{"tick", "dispatch"} {
		t.Run(operation, func(t *testing.T) {
			f, a, _, p, key, _ := finiteAuthorityFixture(t, false)
			fillFiniteBudget(t, a, p, key)
			before, e := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
			if e != nil {
				t.Fatal(e)
			}
			cfgBefore, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
			r, _ := a.Store.Get(context.Background(), p.ID)
			if operation == "tick" {
				_, e = f.runtime.Tick(context.Background(), "committee", "lead")
			} else {
				_, e = f.runtime.Dispatch(context.Background(), "committee", "lead")
			}
			if !errors.Is(e, effortauthority.ErrRefused) {
				t.Fatalf("exhausted admitted %v", e)
			}
			after, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
			cfgAfter, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
			r2, _ := a.Store.Get(context.Background(), p.ID)
			if effortauthority.Digest(before) != effortauthority.Digest(after) || effortauthority.Digest(cfgBefore) != effortauthority.Digest(cfgAfter) || effortauthority.Digest(r) != effortauthority.Digest(r2) || f.queue.enqueues != 0 || len(f.agent.createTaskCalls) != 0 || len(f.agent.createRunCalls) != 0 {
				t.Fatal("refusal wrote leader/history/config/queue/task/run/ledger")
			}
		})
	}
}
func TestFiniteEffortLateTickSamePreparationAndExternalIngress(t *testing.T) {
	f, a, _, p, key, now := finiteAuthorityFixture(t, false)
	*now = now.Add(5 * time.Hour)
	id := uuid.NewString()
	i := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: EffortStartEndpoint("committee", "lead"), PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: "qualified-profile", ProfileDigest: p.Profiles["qualified-profile"], Effect: "run.create", IdempotencyKey: "finite-leader-" + id, InputDigest: effortauthority.Digest(struct{ Team, Member string }{"committee", "lead"}), Turns: 10, ToolCalls: 20, RunSeconds: 3600}
	proof, e := effortauthority.Sign(effortauthority.Proof{Intent: i, Nonce: "external-late-nonce-123", IssuedAt: *now}, key)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(proof)
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	if _, e = f.runtime.StartEffort(context.Background(), "committee", "lead", encoded); e != nil {
		t.Fatal(e)
	}
	state, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if state.ID != id || f.queue.enqueues != 1 {
		t.Fatal("external identity not retained")
	}
	if _, e = f.runtime.StartEffort(context.Background(), "committee", "lead", encoded); e != nil {
		t.Fatal(e)
	}
	r, _ := a.Store.Get(context.Background(), p.ID)
	if len(r.Reservations) != 1 || len(r.Ingress) != 1 || f.queue.enqueues != 1 {
		t.Fatal("same ingress duplicated or renewed")
	}
	before := effortauthority.Digest(state)
	*now = p.Deadline
	if _, e = f.runtime.Tick(context.Background(), "committee", "lead"); e == nil {
		t.Fatal("expired admitted")
	}
	after, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if before != effortauthority.Digest(after) {
		t.Fatal("expired tick changed leader")
	}
}

func TestFiniteEffortTerminalExhaustionDoesNotWriteHistoryOrCompleteQueue(t *testing.T) {
	f, a, _, p, _, _ := finiteAuthorityFixture(t, true)
	state := f.dispatch(t)
	f.agent.getRuns[state.RunID] = &Run{ID: state.RunID, TaskID: state.TaskID, Status: "complete", StartedAt: time.Now().Add(-time.Hour).Format(time.RFC3339Nano), EndedAt: time.Now().Format(time.RFC3339Nano)}
	before := effortauthority.Digest(state)
	cfgBefore, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	record, _ := a.Store.Get(context.Background(), p.ID)
	if _, e := f.runtime.Tick(context.Background(), "committee", "lead"); !errors.Is(e, effortauthority.ErrRefused) {
		t.Fatalf("exhausted relaunch: %v", e)
	}
	after, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	cfgAfter, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	recordAfter, _ := a.Store.Get(context.Background(), p.ID)
	if before != effortauthority.Digest(after) || effortauthority.Digest(cfgBefore) != effortauthority.Digest(cfgAfter) || effortauthority.Digest(record) != effortauthority.Digest(recordAfter) || f.queue.enqueues != 1 || len(f.queue.queued) != 1 {
		t.Fatal("refused relaunch changed terminal bookkeeping/history/queue/budget")
	}
}
func TestFiniteEffortPositiveTerminalRelaunchRecordsAndEnqueuesOnce(t *testing.T) {
	f, a, _, p, key, _ := finiteAuthorityFixture(t, true)
	p.ID = "two-start-policy"
	p.MaxStarts = 2
	p.MaxConcurrent = 2
	p.TotalTurns = 20
	p.TotalToolCalls = 40
	p.TotalRunSeconds = 7200
	if e := a.Approve(context.Background(), "disposable-human", p); e != nil {
		t.Fatal(e)
	}
	i := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: "qualified-profile", ProfileDigest: p.Profiles["qualified-profile"], Effect: "run.create", IdempotencyKey: "setup", InputDigest: effortauthority.Digest("fixture"), Turns: 10, ToolCalls: 20, RunSeconds: 3600}
	proof, _ := effortauthority.Sign(effortauthority.Proof{Intent: i, Nonce: "disposable-policy-check", IssuedAt: a.Now()}, key)
	b, e := a.CheckProof(context.Background(), proof)
	if e != nil {
		t.Fatal(e)
	}
	f.runtime.Efforts.Bindings["committee/lead"] = b
	state := f.dispatch(t)
	terminal := &Run{ID: state.RunID, TaskID: state.TaskID, Status: "complete", StartedAt: time.Now().Add(-time.Hour).Format(time.RFC3339Nano), EndedAt: time.Now().Format(time.RFC3339Nano)}
	f.agent.getRuns[state.RunID] = terminal
	next, e := f.runtime.Tick(context.Background(), "committee", "lead")
	if e != nil {
		t.Fatal(e)
	}
	cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	r, _ := a.Store.Get(context.Background(), p.ID)
	if next.ID == state.ID || len(next.RestartHistory) != 1 || cfg.LastExecution == nil || cfg.LastExecution.RunID != state.RunID || cfg.LastExecution.Status != "completed" || f.queue.enqueues != 2 || len(f.queue.queued) != 1 || len(r.Reservations) != 2 {
		t.Fatal("positive relaunch lost terminal evidence/completion/successor")
	}
	if _, e = f.runtime.Tick(context.Background(), "committee", "lead"); e != nil {
		t.Fatal(e)
	}
	if f.queue.enqueues != 2 {
		t.Fatal("successor duplicated")
	}
}

func TestFiniteEffortConcurrentTerminalTicksKeepOneSuccessorObligation(t *testing.T) {
	// Two runtime instances share the supported single-process owner store.
	f, a, _, p, key, _ := finiteAuthorityFixture(t, true)
	p.ID = "concurrent-successor-policy"
	p.MaxStarts = 2
	p.MaxConcurrent = 2
	p.TotalTurns = 20
	p.TotalToolCalls = 40
	p.TotalRunSeconds = 7200
	if e := a.Approve(context.Background(), "fixture-human", p); e != nil {
		t.Fatal(e)
	}
	i := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: "qualified-profile", ProfileDigest: p.Profiles["qualified-profile"], Effect: "run.create", IdempotencyKey: "setup", InputDigest: effortauthority.Digest("fixture"), Turns: 10, ToolCalls: 20, RunSeconds: 3600}
	proof, _ := effortauthority.Sign(effortauthority.Proof{Intent: i, Nonce: "disposable-concurrent-setup", IssuedAt: a.Now()}, key)
	b, e := a.CheckProof(context.Background(), proof)
	if e != nil {
		t.Fatal(e)
	}
	f.runtime.Efforts.Bindings["committee/lead"] = b
	state := f.dispatch(t)
	f.agent.getRuns[state.RunID] = &Run{ID: state.RunID, TaskID: state.TaskID, Status: "complete", StartedAt: time.Now().Add(-time.Hour).Format(time.RFC3339Nano), EndedAt: time.Now().Format(time.RFC3339Nano)}
	other := &FiniteLeaderRuntime{Executor: f.runtime.Executor, Queue: f.runtime.Queue, Efforts: f.runtime.Efforts}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, runtime := range []*FiniteLeaderRuntime{f.runtime, other} {
		wg.Add(1)
		go func(r *FiniteLeaderRuntime) {
			defer wg.Done()
			<-start
			if _, e := r.Tick(context.Background(), "committee", "lead"); e != nil {
				t.Error(e)
			}
		}(runtime)
	}
	close(start)
	wg.Wait()
	after, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	r, _ := a.Store.Get(context.Background(), p.ID)
	if after.ID == state.ID || len(after.RestartHistory) != 1 || len(r.Reservations) != 2 || f.queue.enqueues != 2 || len(f.queue.queued) != 1 {
		t.Fatal("concurrent terminal completion removed or duplicated successor")
	}
}

type blockingFiniteCompletion struct {
	delegate         *finiteQueueFake
	entered, release chan struct{}
	once             sync.Once
}

func (q *blockingFiniteCompletion) Enqueue(c context.Context, t, m, p string) (*EnqueueResult, error) {
	return q.delegate.Enqueue(c, t, m, p)
}
func (q *blockingFiniteCompletion) Status(t string) TeamExecutionStatus { return q.delegate.Status(t) }
func (q *blockingFiniteCompletion) OnComplete(t, m string) {
	q.once.Do(func() { close(q.entered); <-q.release })
	q.delegate.OnComplete(t, m)
}
func TestFiniteEffortMixedDispatchTickCompletionCannotClearSuccessor(t *testing.T) {
	f, a, _, p, key, _ := finiteAuthorityFixture(t, true)
	p.ID = "mixed-policy"
	p.MaxStarts = 2
	p.MaxConcurrent = 2
	p.TotalTurns = 20
	p.TotalToolCalls = 40
	p.TotalRunSeconds = 7200
	if e := a.Approve(context.Background(), "fixture-human", p); e != nil {
		t.Fatal(e)
	}
	i := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: "qualified-profile", ProfileDigest: p.Profiles["qualified-profile"], Effect: "run.create", IdempotencyKey: "setup", InputDigest: effortauthority.Digest("fixture"), Turns: 10, ToolCalls: 20, RunSeconds: 3600}
	proof, _ := effortauthority.Sign(effortauthority.Proof{Intent: i, Nonce: "disposable-mixed-setup", IssuedAt: a.Now()}, key)
	b, e := a.CheckProof(context.Background(), proof)
	if e != nil {
		t.Fatal(e)
	}
	f.runtime.Efforts.Bindings["committee/lead"] = b
	state := f.dispatch(t)
	f.agent.getRuns[state.RunID] = &Run{ID: state.RunID, TaskID: state.TaskID, Status: "complete", StartedAt: time.Now().Add(-time.Hour).Format(time.RFC3339Nano), EndedAt: time.Now().Format(time.RFC3339Nano)}
	q := &blockingFiniteCompletion{delegate: f.queue, entered: make(chan struct{}), release: make(chan struct{})}
	f.runtime.Queue = q
	dispatchDone := make(chan error, 1)
	tickDone := make(chan error, 1)
	go func() { _, e := f.runtime.Dispatch(context.Background(), "committee", "lead"); dispatchDone <- e }()
	select {
	case <-q.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("dispatch never reached terminal completion")
	}
	go func() { _, e := f.runtime.Tick(context.Background(), "committee", "lead"); tickDone <- e }()
	// A delayed terminal callback owns the fence. No successor may be admitted
	// into a queue which that callback could subsequently clear.
	select {
	case e := <-tickDone:
		t.Fatalf("tick crossed delayed completion fence: %v", e)
	case <-time.After(50 * time.Millisecond):
	}
	close(q.release)
	if e := <-dispatchDone; e != nil {
		t.Fatal(e)
	}
	if e := <-tickDone; e != nil {
		t.Fatal(e)
	}
	after, _ := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	r, _ := a.Store.Get(context.Background(), p.ID)
	if after.ID == state.ID || len(after.RestartHistory) != 1 || len(r.Reservations) != 2 || f.queue.enqueues != 2 || len(f.queue.queued) != 1 {
		t.Fatal("mixed delayed completion cleared successor obligation")
	}
}
