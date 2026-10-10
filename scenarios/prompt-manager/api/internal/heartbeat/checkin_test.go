package heartbeat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"

	"prompt-manager/internal/teamconfig"
)

// fakeEffortSignals is the Agent Manager effort signal log seam.
type fakeEffortSignals struct {
	mu      sync.Mutex
	signals []EffortSignal
	reads   int
	err     error
}

func (s *fakeEffortSignals) ListEffortSignals(_ context.Context, effortRef string, afterSeq int64, limit int32) (*EffortSignalPage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads++
	if s.err != nil {
		return nil, s.err
	}
	page := &EffortSignalPage{}
	for _, signal := range s.signals {
		page.LatestSeq = max(page.LatestSeq, signal.Seq)
		if signal.EffortRef == effortRef && signal.Seq > afterSeq && len(page.Signals) < int(limit) {
			page.Signals = append(page.Signals, signal)
		}
	}
	return page, nil
}

func (s *fakeEffortSignals) emit(kind, runID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.signals = append(s.signals, EffortSignal{
		Seq: int64(len(s.signals) + 1), EffortRef: "arbitrary:new-effort/42", Kind: kind, RunID: runID,
		Detail: kind + " for " + runID, CreatedAt: time.Now().UTC(),
	})
}

// newCheckinFixture is a keepAlive leader in event check-in mode whose first
// check-in is dispatched and running.
func newCheckinFixture(t *testing.T, policy *teamconfig.CheckinPolicy) (*finiteFixture, *fakeEffortSignals) {
	t.Helper()
	f := newFiniteFixtureWithKeepAlive(t, true)
	cfg, err := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	if err != nil {
		t.Fatal(err)
	}
	cfg.FiniteLeader.Checkin = policy
	if err := f.teams.SetHeartbeatConfig(context.Background(), "committee", "lead", cfg); err != nil {
		t.Fatal(err)
	}
	signals := &fakeEffortSignals{}
	f.runtime.Signals = signals
	f.dispatch(t)
	f.queue.OnComplete("committee", "lead")
	return f, signals
}

// endCheckin ends the current check-in run at endedAt after a short run.
func (f *finiteFixture) endCheckin(t *testing.T, status, stopReason string, endedAt time.Time) string {
	t.Helper()
	state, err := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if err != nil || state.RunID == "" {
		t.Fatalf("no dispatched check-in to end: %+v %v", state, err)
	}
	run := f.agent.getRuns[state.RunID]
	run.Status, run.StopReason = status, stopReason
	run.StartedAt, run.EndedAt = endedAt.Add(-3*time.Minute).Format(time.RFC3339Nano), endedAt.Format(time.RFC3339Nano)
	return state.RunID
}

func (f *finiteFixture) lastPrompt(t *testing.T) string {
	t.Helper()
	if len(f.agent.createTaskCalls) == 0 {
		t.Fatal("no leader task created")
	}
	return f.agent.createTaskCalls[len(f.agent.createTaskCalls)-1].Description
}

func TestCheckinStartsOncePerSignalBurstAndNeverConcurrently(t *testing.T) {
	ctx := context.Background()
	f, signals := newCheckinFixture(t, &teamconfig.CheckinPolicy{Mode: teamconfig.CheckinModeEvent})
	if prompt := f.lastPrompt(t); !strings.Contains(prompt, "one event check-in") || strings.Contains(prompt, "Park instead of waiting") {
		t.Fatalf("event check-in received parked-leader guidance:\n%s", prompt)
	}

	// While the check-in runs, signals never start a second one.
	signals.emit(EffortSignalChildRunEnded, "worker-1")
	if state, err := f.runtime.Tick(ctx, "committee", "lead"); err != nil || state.RunID != "leader-1" || f.queue.enqueues != 1 {
		t.Fatalf("a live check-in was replaced: %+v enqueues=%d %v", state, f.queue.enqueues, err)
	}

	f.endCheckin(t, "complete", "", time.Now().UTC())
	signals.emit(EffortSignalStepBack, "worker-1")
	signals.emit(EffortSignalFeedback, "")
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = f.runtime.Tick(ctx, "committee", "lead")
		}()
	}
	wg.Wait()
	state, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if f.queue.enqueues != 2 || state.DispatchStarted || state.CheckinReason != checkinReasonSignal {
		t.Fatalf("signal burst did not admit exactly one check-in: enqueues=%d %+v", f.queue.enqueues, state)
	}
	f.relaunchNext(t, "leader-2")
	prompt := f.lastPrompt(t)
	for _, want := range []string{"Check-in trigger: signal", "seq 1 child_run_ended run worker-1", "seq 3 feedback"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("check-in prompt lacks %q:\n%s", want, prompt)
		}
	}
	if len(f.agent.createRunCalls) != 2 {
		t.Fatalf("signal burst created %d leader runs", len(f.agent.createRunCalls))
	}

	// The consumed burst does not start another check-in.
	f.endCheckin(t, "complete", "", time.Now().UTC())
	if state, err := f.runtime.Tick(ctx, "committee", "lead"); err != nil || state.RunID != "leader-2" || f.queue.enqueues != 2 {
		t.Fatalf("a consumed signal started another check-in: %+v %v", state, err)
	}
}

func TestCheckinShortRunsAreNotThrottledButFailuresBackOff(t *testing.T) {
	ctx := context.Background()
	f, signals := newCheckinFixture(t, &teamconfig.CheckinPolicy{Mode: teamconfig.CheckinModeEvent})
	for i := 2; i <= livenessMaxRelaunches+3; i++ {
		f.endCheckin(t, "complete", "", time.Now().UTC())
		signals.emit(EffortSignalChildRunEnded, "worker")
		state, err := f.runtime.Tick(ctx, "committee", "lead")
		if err != nil || state.DispatchStarted || state.ConsecutiveRelaunches != 1 {
			t.Fatalf("short check-in %d was throttled: %+v %v", i, state, err)
		}
		f.relaunchNext(t, fmt.Sprintf("leader-%d", i))
	}

	// A token-cap stop ends a check-in on purpose: no retry without a signal.
	f.endCheckin(t, "cancelled", tokenCapStopReason, time.Now().UTC())
	enqueues := f.queue.enqueues
	if _, err := f.runtime.Tick(ctx, "committee", "lead"); err != nil || f.queue.enqueues != enqueues {
		t.Fatalf("cap-stopped check-in was retried: enqueues=%d %v", f.queue.enqueues, err)
	}
	signals.emit(EffortSignalCapStop, "leader")
	if _, err := f.runtime.Tick(ctx, "committee", "lead"); err != nil {
		t.Fatal(err)
	}
	f.relaunchNext(t, "leader-capped")

	// A failed check-in is retried without a signal, under the failure backoff.
	f.endCheckin(t, "failed", "", time.Now().UTC())
	enqueues = f.queue.enqueues
	state, err := f.runtime.Tick(ctx, "committee", "lead")
	if err != nil || f.queue.enqueues != enqueues || !strings.Contains(state.Error, "backing off") {
		t.Fatalf("failing check-in was not backed off: %+v %v", state, err)
	}
	f.backdateLastRestart(t)
	state, err = f.runtime.Tick(ctx, "committee", "lead")
	if err != nil || f.queue.enqueues != enqueues+1 || state.CheckinReason != checkinReasonRetry || state.ConsecutiveRelaunches != 2 {
		t.Fatalf("failed check-in was not retried after backoff: %+v %v", state, err)
	}
}

func TestCheckinBackstopFiresWithoutSignals(t *testing.T) {
	ctx := context.Background()
	f, _ := newCheckinFixture(t, &teamconfig.CheckinPolicy{Mode: teamconfig.CheckinModeEvent, Backstop: "2h"})
	f.endCheckin(t, "complete", "", time.Now().UTC().Add(-time.Hour))
	if state, err := f.runtime.Tick(ctx, "committee", "lead"); err != nil || state.RunID != "leader-1" || f.queue.enqueues != 1 {
		t.Fatalf("a quiet effort started a check-in before the backstop: %+v %v", state, err)
	}
	f.endCheckin(t, "complete", "", time.Now().UTC().Add(-3*time.Hour))
	state, err := f.runtime.Tick(ctx, "committee", "lead")
	if err != nil || f.queue.enqueues != 2 || state.CheckinReason != checkinReasonBackstop {
		t.Fatalf("expired backstop did not start one check-in: %+v %v", state, err)
	}
	f.relaunchNext(t, "leader-2")
	if prompt := f.lastPrompt(t); !strings.Contains(prompt, "Check-in trigger: backstop") || !strings.Contains(prompt, "No new effort signals") {
		t.Fatalf("backstop check-in prompt:\n%s", prompt)
	}
}

func TestCheckinCursorSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	f, signals := newCheckinFixture(t, &teamconfig.CheckinPolicy{Mode: teamconfig.CheckinModeEvent})
	f.endCheckin(t, "complete", "", time.Now().UTC())
	signals.emit(EffortSignalChildRunEnded, "worker-1")
	if _, err := f.runtime.Tick(ctx, "committee", "lead"); err != nil {
		t.Fatal(err)
	}
	f.relaunchNext(t, "leader-2")
	f.endCheckin(t, "complete", "", time.Now().UTC())

	// A fresh process reads the persisted cursor: the consumed signal and the
	// ended check-in's own terminal do not start another check-in.
	restarted := &FiniteLeaderRuntime{Executor: f.runtime.Executor, Queue: f.queue, Signals: signals}
	f.runtime.Executor.FiniteLeader = restarted
	persisted, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if !persisted.CheckinCursorSet || persisted.CheckinCursor != 1 {
		t.Fatalf("cursor not persisted: %+v", persisted)
	}
	signals.emit(EffortSignalChildRunEnded, "leader-2")
	if state, err := restarted.Tick(ctx, "committee", "lead"); err != nil || state.RunID != "leader-2" || f.queue.enqueues != 2 {
		t.Fatalf("restart replayed a consumed signal: %+v %v", state, err)
	}
	signals.emit(EffortSignalOperatorAnswer, "")
	if state, err := restarted.Tick(ctx, "committee", "lead"); err != nil || f.queue.enqueues != 3 || state.CheckinReason != checkinReasonSignal {
		t.Fatalf("new signal after restart did not start a check-in: %+v %v", state, err)
	}

	// A truncated page advances the cursor only past the signals handed over;
	// the remainder starts the next check-in instead of being skipped.
	f.relaunchNext(t, "leader-3")
	f.endCheckin(t, "complete", "", time.Now().UTC())
	for range checkinSignalBatchLimit + 1 {
		signals.emit(EffortSignalChildRunEnded, "worker-2")
	}
	if _, err := restarted.Tick(ctx, "committee", "lead"); err != nil {
		t.Fatal(err)
	}
	f.relaunchNext(t, "leader-4")
	persisted, _ = f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if prompt := f.lastPrompt(t); persisted.CheckinCursor != 3+checkinSignalBatchLimit || !strings.Contains(prompt, "truncated") {
		t.Fatalf("truncated page moved the cursor past unseen signals: cursor %d\n%s", persisted.CheckinCursor, prompt)
	}
	f.endCheckin(t, "complete", "", time.Now().UTC())
	if state, err := restarted.Tick(ctx, "committee", "lead"); err != nil || state.CheckinReason != checkinReasonSignal {
		t.Fatalf("signals beyond a truncated page did not start a check-in: %+v %v", state, err)
	}
	f.relaunchNext(t, "leader-5")
	if prompt := f.lastPrompt(t); !strings.Contains(prompt, fmt.Sprintf("seq %d ", 4+checkinSignalBatchLimit)) {
		t.Fatalf("next check-in was not handed the remaining signal:\n%s", prompt)
	}
}

func TestCheckinWithoutSignalSourceKeepsParkedBehavior(t *testing.T) {
	ctx := context.Background()
	f, signals := newCheckinFixture(t, &teamconfig.CheckinPolicy{Mode: teamconfig.CheckinModeEvent})
	signals.err = ErrEffortSignalsUnavailable
	f.endCheckin(t, "complete", "", time.Now().UTC())
	state, err := f.runtime.Tick(ctx, "committee", "lead")
	if err != nil || state.DispatchStarted || f.queue.enqueues != 2 || state.CheckinReason != "" {
		t.Fatalf("unavailable signals did not fall back to the parked relaunch: %+v %v", state, err)
	}
	f.relaunchNext(t, "leader-2")
	if prompt := f.lastPrompt(t); !strings.Contains(prompt, "Park instead of waiting") {
		t.Fatalf("fallback leader lacks park guidance:\n%s", prompt)
	}
}

func TestCheckinSchedulerPollsAtTheCheckinInterval(t *testing.T) {
	f := newFiniteFixtureWithKeepAlive(t, true)
	scheduler := NewScheduler(f.runtime.Executor, f.agent, f.teams, nil)
	if err := scheduler.Schedule("committee", "lead", "@hourly"); err != nil || scheduler.scheduled["committee/lead"].Schedule != "@hourly" {
		t.Fatalf("parked leader lost its cron: %v", err)
	}
	cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	cfg.FiniteLeader.Checkin = &teamconfig.CheckinPolicy{Mode: teamconfig.CheckinModeEvent, PollInterval: "5m"}
	if err := f.teams.SetHeartbeatConfig(context.Background(), "committee", "lead", cfg); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Schedule("committee", "lead", "@hourly"); err != nil || scheduler.scheduled["committee/lead"].Schedule != "@every 5m0s" {
		t.Fatalf("event check-ins do not poll at their interval: %+v %v", scheduler.scheduled["committee/lead"], err)
	}
}

// The check-in policy is not binding identity: switching modes keeps the
// retained reservation.
func TestCheckinPolicyChangeKeepsTheBinding(t *testing.T) {
	f := newFiniteFixtureWithKeepAlive(t, true)
	state := f.dispatch(t)
	cfg, _ := f.teams.GetHeartbeatConfig(context.Background(), "committee", "lead")
	cfg.FiniteLeader.Checkin = &teamconfig.CheckinPolicy{Mode: teamconfig.CheckinModeEvent}
	if err := f.teams.SetHeartbeatConfig(context.Background(), "committee", "lead", cfg); err != nil {
		t.Fatal(err)
	}
	after, err := f.teams.ReadFiniteLeader(context.Background(), "committee", "lead")
	if err != nil || after.ID != state.ID || after.RunID != state.RunID {
		t.Fatalf("mode switch disturbed the reservation: %+v %v", after, err)
	}
}

// signalOwner serves Agent Manager's generated ListEffortSignals route.
type signalOwner struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	got *ampb.ListEffortSignalsRequest
	err error
}

func (o *signalOwner) ListEffortSignals(_ context.Context, req *connect.Request[ampb.ListEffortSignalsRequest]) (*connect.Response[ampb.ListEffortSignalsResponse], error) {
	o.got = req.Msg
	if o.err != nil {
		return nil, o.err
	}
	return connect.NewResponse(&ampb.ListEffortSignalsResponse{LatestSeq: 9, Signals: []*ampb.EffortSignal{
		{Seq: 8, EffortRef: req.Msg.EffortRef, Kind: EffortSignalStepBack, RunId: "worker-run", Detail: "slice too wide", CreatedAt: timestamppb.New(time.Date(2026, 10, 8, 1, 2, 3, 0, time.UTC))},
	}}), nil
}

func TestCheckinAgentManagerSignalClient(t *testing.T) {
	read := func(t *testing.T, handler http.Handler) (*EffortSignalPage, error) {
		t.Helper()
		server := httptest.NewServer(handler)
		defer server.Close()
		client := &AgentManagerClient{httpClient: server.Client(), testBaseURL: server.URL}
		return client.ListEffortSignals(context.Background(), "arbitrary:new-effort/42", 7, 100)
	}
	serve := func(owner apiconnect.AgentManagerServiceHandler) http.Handler {
		mux := http.NewServeMux()
		mux.Handle(apiconnect.NewAgentManagerServiceHandler(owner))
		return mux
	}

	owner := &signalOwner{}
	page, err := read(t, serve(owner))
	if err != nil {
		t.Fatal(err)
	}
	want := EffortSignal{Seq: 8, EffortRef: "arbitrary:new-effort/42", Kind: EffortSignalStepBack, RunID: "worker-run", Detail: "slice too wide", CreatedAt: time.Date(2026, 10, 8, 1, 2, 3, 0, time.UTC)}
	if owner.got.GetEffortRef() != "arbitrary:new-effort/42" || owner.got.GetAfterSeq() != 7 || owner.got.GetLimit() != 100 ||
		page.LatestSeq != 9 || len(page.Signals) != 1 || page.Signals[0] != want {
		t.Fatalf("signal page was not mapped exactly: request %v page %+v", owner.got, page)
	}

	for name, handler := range map[string]http.Handler{
		"unimplemented": serve(&apiconnect.UnimplementedAgentManagerServiceHandler{}),
		"absent route":  http.NotFoundHandler(),
		"not implemented": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		}),
	} {
		if _, err := read(t, handler); !errors.Is(err, ErrEffortSignalsUnavailable) {
			t.Fatalf("%s owner was not reported unavailable: %v", name, err)
		}
	}
	for name, handler := range map[string]http.Handler{
		"owner unavailable": serve(&signalOwner{err: connect.NewError(connect.CodeUnavailable, errors.New("effort supervision unavailable"))}),
		"server error": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	} {
		if _, err := read(t, handler); err == nil || errors.Is(err, ErrEffortSignalsUnavailable) {
			t.Fatalf("%s was not reported as a transient failure: %v", name, err)
		}
	}
}
