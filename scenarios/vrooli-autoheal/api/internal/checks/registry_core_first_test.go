package checks

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

const fenceOutput = "vrooli: root update initialization fence: root update conflict: candidate command must wait for native verification"

func healableFailing(id string, action string, result ActionResult) *mockHealableCheck {
	return &mockHealableCheck{
		id:            id,
		result:        Result{CheckID: id, Status: StatusCritical, Message: id + " down"},
		actions:       []RecoveryAction{{ID: action, Available: true}},
		executeResult: result,
	}
}

func incidentOpened(spy *healIncidentReporterSpy, checkID, actionID string) bool {
	for _, record := range spy.opened {
		if record.checkID == checkID && record.actionID == actionID {
			return true
		}
	}
	return false
}

func registerAutoHeal(reg *Registry, checks ...*mockHealableCheck) []Result {
	enabled := map[string]bool{}
	results := make([]Result, 0, len(checks))
	for _, check := range checks {
		reg.Register(check)
		enabled[check.id] = true
		results = append(results, check.result)
	}
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: enabled})
	return results
}

func TestHealPriorityRanksCoreAheadOfResourcesAndScenarios(t *testing.T) {
	order := []string{"scenario-web-console", "resource-cloudflared", "resource-whisper", "scenario-visited-tracker", "vrooli-orphans"}
	for i := 1; i < len(order); i++ {
		if getHealPriority(order[i-1]) > getHealPriority(order[i]) {
			t.Fatalf("%s must not rank behind %s", order[i-1], order[i])
		}
	}
	for _, core := range []string{"scenario-web-console", "scenario-agent-manager", "scenario-prompt-manager", "scenario-vrooli-autoheal", "resource-cloudflared"} {
		if getHealPriority(core) != 0 {
			t.Fatalf("%s is core and must rank first, got %d", core, getHealPriority(core))
		}
	}
	if getHealPriority("scenario-visited-tracker") != 2 || getHealPriority("resource-whisper") != 1 {
		t.Fatal("non-core ranking changed")
	}
}

// Replays 2026-11-20 23:56Z: web-console was cut by the per-tick cap while
// resource starts ran. Core must be inside the cap however many resources fail.
func TestRunAutoHealCapNeverCutsCoreBehindResources(t *testing.T) {
	reg := newTestRegistry()
	var checks []*mockHealableCheck
	for i := 0; i < MaxAutoHealActionsPerTick+2; i++ {
		checks = append(checks, healableFailing(fmt.Sprintf("resource-r%d", i), "start", ActionResult{Success: true}))
	}
	core := []*mockHealableCheck{
		healableFailing("scenario-web-console", "restart", ActionResult{Success: true}),
		healableFailing("scenario-prompt-manager", "restart", ActionResult{Success: true}),
		healableFailing("scenario-agent-manager", "restart", ActionResult{Success: true}),
	}
	// Core registered last: registration and result order must not matter.
	results := registerAutoHeal(reg, append(checks, core...)...)

	heals := reg.RunAutoHeal(context.Background(), results)

	for _, c := range core {
		if len(c.executedActions) != 1 {
			t.Fatalf("%s was not healed within the cap: %v", c.id, c.executedActions)
		}
	}
	attempted, deferred := 0, 0
	for _, h := range heals {
		switch {
		case h.Attempted:
			attempted++
		case strings.Contains(h.Reason, "deferred to next tick"):
			deferred++
			if !strings.HasPrefix(h.CheckID, "resource-") {
				t.Fatalf("cap cut a non-resource check: %s", h.CheckID)
			}
		}
	}
	if want := len(checks) + len(core) - MaxAutoHealActionsPerTick; attempted != MaxAutoHealActionsPerTick || deferred != want {
		t.Fatalf("attempted=%d deferred=%d, want %d and %d", attempted, deferred, MaxAutoHealActionsPerTick, want)
	}
}

// A deferral by the cap is a held restart on the check, not a quiet skip, and
// is dropped when the check finally runs.
func TestTickCapDeferralIsHeldThenCriticalThenCleared(t *testing.T) {
	reg := newTestRegistry()
	clock := &fixedClock{current: time.Unix(1000, 0)}
	reg.SetClock(clock)
	reporter := &healIncidentReporterSpy{}
	reg.SetHealIncidentReporter(reporter)
	var checks []*mockHealableCheck
	for i := 0; i < MaxAutoHealActionsPerTick+1; i++ {
		checks = append(checks, healableFailing(fmt.Sprintf("resource-r%d", i), "start", ActionResult{Success: true}))
	}
	results := registerAutoHeal(reg, checks...)
	cut := "resource-r5"

	reg.RunAutoHeal(context.Background(), results)
	result := reg.overlayHealState(Result{CheckID: cut, Status: StatusWarning}, clock.Now())
	if held, _ := result.Details["healHeld"].(bool); !held || result.Details["healHeldClass"] != GateClassTickCap {
		t.Fatalf("cap deferral is not carried on the check: %v", result.Details)
	}
	if result.Status != StatusWarning {
		t.Fatalf("a fresh deferral must not be critical: %s", result.Status)
	}

	clock.Advance(TickCapCriticalAfter + time.Second)
	reg.RunAutoHeal(context.Background(), results)
	result = reg.overlayHealState(Result{CheckID: cut, Status: StatusWarning}, clock.Now())
	if result.Status != StatusCritical {
		t.Fatalf("a deferral held past its bound must be critical: %s", result.Status)
	}
	if !incidentOpened(reporter, cut, holdIncidentAction) {
		t.Fatal("held deferral opened no incident")
	}

	reg.RunAutoHeal(context.Background(), []Result{{CheckID: cut, Status: StatusCritical}})
	if h := reg.overlayHealState(Result{CheckID: cut, Status: StatusWarning}, clock.Now()); h.Details["healHeld"] != nil {
		t.Fatalf("hold survived the heal: %v", h.Details)
	}
}

// Replays 2026-11-20 23:04:34Z: five restarts refused by the root-update fence
// in 55 ms were counted as failures. A refusal is not a failure; the heal is
// retried when the fence clears.
func TestFenceRefusedHealIsRetriedWhenFenceClearsAndNeverSuspends(t *testing.T) {
	reg := newTestRegistry()
	clock := &fixedClock{current: time.Unix(1000, 0)}
	reg.SetClock(clock)
	refused := ActionResult{Success: false, Error: "exit status 1", Output: fenceOutput}
	ids := []string{"scenario-web-console", "scenario-agent-manager", "scenario-prompt-manager", "scenario-vrooli-autoheal", "resource-cloudflared"}
	var checks []*mockHealableCheck
	for _, id := range ids {
		checks = append(checks, healableFailing(id, "restart", refused))
	}
	results := registerAutoHeal(reg, checks...)

	for tick := 0; tick < 2*3; tick++ { // twice MaxRestartAttempts
		heals := reg.RunAutoHeal(context.Background(), results)
		for _, h := range heals {
			if h.Attempted || !strings.Contains(h.Reason, "update initialization fence") {
				t.Fatalf("tick %d: fence refusal must be a deferral naming the fence: %+v", tick, h)
			}
		}
		clock.Advance(time.Minute)
	}
	for _, id := range ids {
		if tracker, ok := reg.GetHealTracker(id); ok && (tracker.ConsecutiveFailures > 0 || tracker.IsSuspended() || tracker.TotalAttempts > 0) {
			t.Fatalf("%s: a refused heal was counted as an attempt: %+v", id, tracker)
		}
		held := reg.overlayHealState(Result{CheckID: id, Status: StatusCritical}, clock.Now())
		if held.Details["healHeldClass"] != GateClassUpdateFence {
			t.Fatalf("%s: refusal is not carried on the check: %v", id, held.Details)
		}
	}

	for _, c := range checks {
		c.mu.Lock()
		c.executeResult = ActionResult{Success: true}
		c.mu.Unlock()
	}
	for _, h := range reg.RunAutoHeal(context.Background(), results) {
		if !h.Attempted || !h.ActionResult.Success {
			t.Fatalf("fence cleared but the heal was not retried: %+v", h)
		}
	}
	for _, id := range ids {
		if held := reg.overlayHealState(Result{CheckID: id, Status: StatusCritical}, clock.Now()); held.Details["healHeld"] != nil {
			t.Fatalf("%s: hold survived the successful retry", id)
		}
	}
}

func TestFenceHoldTurnsCriticalAndOpensIncident(t *testing.T) {
	reg := newTestRegistry()
	clock := &fixedClock{current: time.Unix(1000, 0)}
	reg.SetClock(clock)
	reporter := &healIncidentReporterSpy{}
	reg.SetHealIncidentReporter(reporter)
	check := healableFailing("scenario-web-console", "restart", ActionResult{Output: fenceOutput, Error: "exit status 1"})
	results := registerAutoHeal(reg, check)

	reg.RunAutoHeal(context.Background(), results)
	if incidentOpened(reporter, check.id, holdIncidentAction) {
		t.Fatal("incident opened before the bound")
	}
	clock.Advance(UpdateFenceCriticalAfter + time.Second)
	reg.RunAutoHeal(context.Background(), results)
	if !incidentOpened(reporter, check.id, holdIncidentAction) {
		t.Fatal("a fence held past its bound opened no incident")
	}
	if got := reg.overlayHealState(Result{CheckID: check.id, Status: StatusWarning}, clock.Now()).Status; got != StatusCritical {
		t.Fatalf("fence held past its bound is %s, want critical", got)
	}
}

type timedCheck struct {
	id    string
	delay time.Duration
	start *atomic.Int64
	seq   *atomic.Int64
}

func (c *timedCheck) ID() string                 { return c.id }
func (c *timedCheck) Title() string              { return c.id }
func (c *timedCheck) Description() string        { return c.id }
func (c *timedCheck) Importance() string         { return "test" }
func (c *timedCheck) IntervalSeconds() int       { return 60 }
func (c *timedCheck) Platforms() []platform.Type { return nil }
func (c *timedCheck) Category() Category         { return CategoryInfrastructure }
func (c *timedCheck) Run(ctx context.Context) Result {
	c.start.Store(c.seq.Add(1))
	select {
	case <-time.After(c.delay):
	case <-ctx.Done():
	}
	return Result{CheckID: c.id, Status: StatusOK}
}

// Replays the 96 s tick: slow resource probes ran one after another ahead of
// the next core observation. A tick now takes about its slowest check, core
// starts first, and results keep dispatch order.
func TestRunAllRunsSlowChecksConcurrentlyCoreFirst(t *testing.T) {
	reg := newTestRegistry()
	seq := &atomic.Int64{}
	slow := 3 * CheckConcurrency
	const delay = 80 * time.Millisecond
	starts := map[string]*atomic.Int64{}
	for i := 0; i < slow; i++ {
		id := fmt.Sprintf("resource-slow%02d", i)
		starts[id] = &atomic.Int64{}
		reg.Register(&timedCheck{id: id, delay: delay, start: starts[id], seq: seq})
	}
	starts["scenario-web-console"] = &atomic.Int64{}
	reg.Register(&timedCheck{id: "scenario-web-console", delay: delay, start: starts["scenario-web-console"], seq: seq})

	began := time.Now()
	results := reg.RunAll(context.Background(), true)
	elapsed := time.Since(began)

	if len(results) != slow+1 {
		t.Fatalf("got %d results, want %d", len(results), slow+1)
	}
	if results[0].CheckID != "scenario-web-console" {
		t.Fatalf("core must be dispatched and reported first, got %s", results[0].CheckID)
	}
	if starts["scenario-web-console"].Load() > int64(CheckConcurrency) {
		t.Fatalf("core started in position %d, behind a full pool of slow checks", starts["scenario-web-console"].Load())
	}
	serial := time.Duration(slow+1) * delay
	if elapsed > serial/2 {
		t.Fatalf("tick took %s; sequential would be %s", elapsed, serial)
	}
}

func TestRunAllStopsDispatchingWhenContextEnds(t *testing.T) {
	reg := newTestRegistry()
	var ran sync.Map
	seq := &atomic.Int64{}
	for i := 0; i < 4*CheckConcurrency; i++ {
		id := fmt.Sprintf("resource-c%02d", i)
		start := &atomic.Int64{}
		ran.Store(id, start)
		reg.Register(&timedCheck{id: id, delay: 200 * time.Millisecond, start: start, seq: seq})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	results := reg.RunAll(ctx, true)
	if len(results) >= 4*CheckConcurrency {
		t.Fatalf("an ended context still ran all %d checks", len(results))
	}
}

// A critical with no automatic action says who owns it in the heal skip reason,
// not only on the check, so the steward reading skips hears the owner too.
func TestSkipReasonNamesTheRepairOwner(t *testing.T) {
	reg := newTestRegistry()
	result := OwnedElsewhere(Result{CheckID: "system-swap", Status: StatusCritical, Message: "swap paging"}, "portfolio steward free-host lane")
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: map[string]bool{}})

	heals := reg.RunAutoHeal(context.Background(), []Result{result})

	if len(heals) != 1 || heals[0].Attempted || heals[0].SkipClass != SkipNotEnabled ||
		!strings.Contains(heals[0].Reason, "repair owner: portfolio steward free-host lane") {
		t.Fatalf("skip does not name the owner: %+v", heals)
	}
}

// gatedHeal blocks its action until released, standing in for a 45-77 s restart.
type gatedHeal struct {
	*mockHealableCheck
	started chan struct{}
	release chan struct{}
}

func newGatedHeal(id, action string) *gatedHeal {
	return &gatedHeal{
		mockHealableCheck: healableFailing(id, action, ActionResult{Success: true}),
		started:           make(chan struct{}, 4),
		release:           make(chan struct{}),
	}
}

func (g *gatedHeal) ExecuteAction(ctx context.Context, actionID string) ActionResult {
	g.started <- struct{}{}
	select {
	case <-g.release:
	case <-ctx.Done():
	}
	return g.mockHealableCheck.ExecuteAction(ctx, actionID)
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// Replays the 2-minute tick: heals ran inside the tick one after another, so a
// restart held the next observation of the core. StartAutoHeal returns at once,
// reports the heal when it finishes, and never doubles a heal in flight.
func TestStartAutoHealReturnsAtOnceAndNeverDoublesAHealInFlight(t *testing.T) {
	reg := newTestRegistry()
	slow := newGatedHeal("resource-whisper", "start")
	reg.Register(slow)
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: map[string]bool{slow.id: true}})
	results := []Result{slow.result}
	var mu sync.Mutex
	var finished []AutoHealResult
	done := func(result AutoHealResult) { mu.Lock(); finished = append(finished, result); mu.Unlock() }

	began := time.Now()
	planned := reg.StartAutoHeal(context.Background(), results, done)
	if time.Since(began) > time.Second || len(planned) != 1 || !planned[0].Started {
		t.Fatalf("StartAutoHeal blocked or reported nothing: %v %+v", time.Since(began), planned)
	}
	<-slow.started

	again := reg.StartAutoHeal(context.Background(), results, done)
	if len(again) != 1 || again[0].Started || again[0].SkipClass != SkipInFlight {
		t.Fatalf("a heal in flight must be skipped, not doubled: %+v", again)
	}
	select {
	case <-slow.started:
		t.Fatal("the action was started a second time")
	case <-time.After(50 * time.Millisecond):
	}

	close(slow.release)
	reg.WaitForHeals()
	mu.Lock()
	defer mu.Unlock()
	if len(finished) != 1 || !finished[0].Attempted || !finished[0].ActionResult.Success {
		t.Fatalf("the finished heal was not reported: %+v", finished)
	}
	if reg.healInFlightCount() != 0 {
		t.Fatal("a finished heal still holds its slot")
	}
}

// A core heal is never queued behind slow non-core heals: it starts in its own
// round while they are still running, even with every slot taken.
func TestCoreHealStartsWhileSlowHealsHoldEverySlot(t *testing.T) {
	reg := newTestRegistry()
	var slow []*gatedHeal
	for i := 0; i < MaxAutoHealActionsPerTick; i++ {
		slow = append(slow, newGatedHeal(fmt.Sprintf("resource-slow%d", i), "start"))
	}
	var checks []*mockHealableCheck
	for _, g := range slow {
		checks = append(checks, g.mockHealableCheck)
		reg.Register(g)
	}
	enabled := map[string]bool{}
	for _, g := range slow {
		enabled[g.id] = true
	}
	core := healableFailing("scenario-web-console", "restart", ActionResult{Success: true})
	reg.Register(core)
	enabled[core.id] = true
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: enabled})

	var mu sync.Mutex
	finished := map[string]bool{}
	done := func(result AutoHealResult) { mu.Lock(); finished[result.CheckID] = true; mu.Unlock() }
	var slowResults []Result
	for _, c := range checks {
		slowResults = append(slowResults, c.result)
	}
	reg.StartAutoHeal(context.Background(), slowResults, done)
	<-slow[0].started // the round is running its first heal; all five hold slots

	planned := reg.StartAutoHeal(context.Background(), []Result{core.result}, done)
	if len(planned) != 1 || !planned[0].Started {
		t.Fatalf("a core heal found no slot: %+v", planned)
	}
	waitFor(t, "the core heal to finish while the slow round is still running", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return finished[core.id]
	})
	for _, g := range slow {
		select {
		case <-g.release:
		default:
			close(g.release)
		}
	}
	reg.WaitForHeals()
}

// Non-core candidates see the slots held by heals still running.
func TestSlotsHeldByRunningHealsDeferNonCoreWork(t *testing.T) {
	reg := newTestRegistry()
	var running []*gatedHeal
	enabled := map[string]bool{}
	var runningResults []Result
	for i := 0; i < MaxAutoHealActionsPerTick; i++ {
		g := newGatedHeal(fmt.Sprintf("resource-busy%d", i), "start")
		reg.Register(g)
		enabled[g.id] = true
		running = append(running, g)
		runningResults = append(runningResults, g.result)
	}
	waiting := healableFailing("resource-waiting", "start", ActionResult{Success: true})
	reg.Register(waiting)
	enabled[waiting.id] = true
	reg.SetConfigProvider(&mockConfigProvider{autoHealChecks: enabled})
	reg.StartAutoHeal(context.Background(), runningResults, func(AutoHealResult) {})
	<-running[0].started

	planned := reg.StartAutoHeal(context.Background(), []Result{waiting.result}, func(AutoHealResult) {})
	if len(planned) != 1 || planned[0].Started || planned[0].SkipClass != GateClassTickCap {
		t.Fatalf("a non-core heal must wait for a slot: %+v", planned)
	}
	for _, g := range running {
		close(g.release)
	}
	reg.WaitForHeals()
}

// A 60 s check on a 60 s tick runs every tick: its jittered next run (up to 6 s
// late) must not push it to the following tick.
func TestSixtySecondCheckRunsOnEverySixtySecondTick(t *testing.T) {
	reg := newTestRegistry()
	clock := &fixedClock{current: time.Unix(10_000, 0)}
	reg.SetClock(clock)
	check := &mockCheck{id: "scenario-web-console", interval: 60, result: Result{CheckID: "scenario-web-console", Status: StatusOK}}
	reg.Register(check)

	for tick := 0; tick < 50; tick++ {
		if got := len(reg.RunAll(context.Background(), false)); got != 1 {
			t.Fatalf("tick %d ran %d checks, want 1: a 60 s check must run on every 60 s tick", tick, got)
		}
		clock.Advance(60 * time.Second)
	}
	clock.Advance(-30 * time.Second) // half an interval after the last run
	if got := len(reg.RunAll(context.Background(), false)); got != 0 {
		t.Fatalf("a check half an interval after its last run must not be due, ran %d", got)
	}
}
