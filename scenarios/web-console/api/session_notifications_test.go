package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"web-console/internal/backend"
	"web-console/session"

	"github.com/vrooli/api-core/eventbus"
)

// fakeTimer is a controllable timerHandle: the test fires it by hand and can
// assert whether it was cancelled.
type fakeTimer struct {
	fn      func()
	stopped bool
}

func (t *fakeTimer) Stop() bool {
	was := t.stopped
	t.stopped = true
	return !was
}

// fire runs the armed callback iff the timer was not cancelled, mimicking the
// real scheduler (a Stopped time.Timer never fires).
func (t *fakeTimer) fire() {
	if !t.stopped {
		t.fn()
	}
}

// fakeScheduler records every armed timer so a test can fire or inspect them.
type fakeScheduler struct {
	timers []*fakeTimer
	dwells []time.Duration
}

func (s *fakeScheduler) arm(d time.Duration, fn func()) timerHandle {
	t := &fakeTimer{fn: fn}
	s.timers = append(s.timers, t)
	s.dwells = append(s.dwells, d)
	return t
}

func (s *fakeScheduler) last() *fakeTimer { return s.timers[len(s.timers)-1] }

// testClock is an advanceable clock so tests can simulate how long a session
// stays active (the turn-completed gate measures that stretch).
type testClock struct{ t time.Time }

func (c *testClock) now() time.Time          { return c.t }
func (c *testClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestNotifier() (*sessionNotifier, *fakeScheduler, *testClock) {
	sched := &fakeScheduler{}
	clk := &testClock{t: time.Unix(0, 0).UTC()}
	n := &sessionNotifier{
		now:      clk.now,
		last:     map[string]session.ActivityState{},
		active:   map[string]time.Time{},
		pending:  map[string]timerHandle{},
		schedule: sched.arm,
	}
	return n, sched, clk
}

// workedTurn drives a session through a genuine turn: idle → working for a real
// stretch → idle, so the turn-completed gate is satisfied.
func workedTurn(n *sessionNotifier, clk *testClock, id string) {
	n.observe(id, session.Activity{State: session.ActivityIdle})    // baseline
	n.observe(id, session.Activity{State: session.ActivityWorking}) // leaves idle
	clk.advance(minTurnActive + time.Second)
	n.observe(id, session.Activity{State: session.ActivityIdle, Since: clk.now()})
}

func TestObserveBaselineNeverNotifies(t *testing.T) {
	n, sched, _ := newTestNotifier()
	// The first state we see for a session — even a prompt that is already up —
	// is the state it was already in when we began watching, not a witnessed
	// transition. It must not arm anything. This is the startup-burst guard.
	n.observe("sess-1", session.Activity{State: session.ActivityWaiting})
	if len(sched.timers) != 0 {
		t.Fatalf("baseline observation must not arm a notice, armed %d", len(sched.timers))
	}
}

func TestObserveArmsInputNeededAfterBaseline(t *testing.T) {
	n, sched, _ := newTestNotifier()
	n.observe("sess-1", session.Activity{State: session.ActivityWorking}) // baseline
	n.observe("sess-1", session.Activity{State: session.ActivityWaiting})

	if len(sched.timers) != 1 {
		t.Fatalf("working→waiting should arm exactly one notice, got %d", len(sched.timers))
	}
	if sched.dwells[0] != inputNeededDwell {
		t.Fatalf("input-needed dwell = %v, want %v", sched.dwells[0], inputNeededDwell)
	}

	var got eventbus.DomainEvent
	n.publish = func(_ context.Context, e eventbus.DomainEvent) error { got = e; return nil }
	n.notice = func(_ context.Context, _ string) (string, error) { return "", nil }
	sched.last().fire()

	if got.Payload["category"] != categoryInputNeeded || got.Payload["urgency"] != "high" {
		t.Fatalf("unexpected input-needed facts: %+v", got.Payload)
	}
}

func TestObserveTurnCompletedAfterRealActiveStretch(t *testing.T) {
	n, sched, clk := newTestNotifier()

	// A settle into idle from a baseline (unknown) is not a turn end.
	n.observe("sess-1", session.Activity{State: session.ActivityUnknown}) // baseline
	n.observe("sess-1", session.Activity{State: session.ActivityIdle})
	if len(sched.timers) != 0 {
		t.Fatalf("unknown→idle must not arm a notice, armed %d", len(sched.timers))
	}

	// A genuine turn (idle → working for a real stretch → idle) arms once.
	workedTurn(n, clk, "sess-2")
	if len(sched.timers) != 1 {
		t.Fatalf("a real turn should arm one notice, got %d", len(sched.timers))
	}
	if sched.dwells[0] != turnCompletedDwell {
		t.Fatalf("turn-completed dwell = %v, want %v", sched.dwells[0], turnCompletedDwell)
	}
}

// A once-a-minute status-line redraw of a parked agent reads as a sub-second
// working blip. It must NOT be reported as a completed turn — this is the
// fleet-wide phantom spam the gate exists to kill.
func TestObserveShortActiveBlipIsNotATurn(t *testing.T) {
	n, sched, clk := newTestNotifier()
	n.observe("parked", session.Activity{State: session.ActivityIdle})    // baseline idle
	n.observe("parked", session.Activity{State: session.ActivityWorking}) // redraw blip starts
	clk.advance(1500 * time.Millisecond)                                  // ...and settles fast
	n.observe("parked", session.Activity{State: session.ActivityIdle, Since: clk.now()})

	if len(sched.timers) != 0 {
		t.Fatalf("a brief redraw blip must not arm a turn-completed notice, armed %d", len(sched.timers))
	}
}

// If we never witnessed the session leave idle (it was mid-turn when we began
// watching), a return to idle stays silent rather than guessing.
func TestObserveTurnCompletedSilentWhenActiveStartUnseen(t *testing.T) {
	n, sched, _ := newTestNotifier()
	n.observe("sess", session.Activity{State: session.ActivityWorking}) // baseline mid-turn
	n.observe("sess", session.Activity{State: session.ActivityIdle})
	if len(sched.timers) != 0 {
		t.Fatalf("unwitnessed active start must not arm a notice, armed %d", len(sched.timers))
	}
}

func TestObserveFlappingNeverFires(t *testing.T) {
	n, sched, _ := newTestNotifier()
	n.observe("sess-1", session.Activity{State: session.ActivityWorking}) // baseline

	// An agent that pauses between tool calls flaps working↔idle. Each flap
	// re-arms and cancels; nothing should ever fire while it keeps moving.
	for i := 0; i < 4; i++ {
		n.observe("sess-1", session.Activity{State: session.ActivityIdle, Since: time.Unix(int64(i), 0)})
		n.observe("sess-1", session.Activity{State: session.ActivityWorking})
	}
	// Every armed idle timer must have been cancelled by the following →working.
	for i, timer := range sched.timers {
		timer.fire() // a cancelled timer is a no-op
		if !timer.stopped {
			t.Fatalf("timer %d was not cancelled by the flap", i)
		}
	}

	var fired bool
	n.publish = func(_ context.Context, _ eventbus.DomainEvent) error { fired = true; return nil }
	n.notice = func(_ context.Context, _ string) (string, error) { return "", nil }
	for _, timer := range sched.timers {
		timer.fire()
	}
	if fired {
		t.Fatalf("a flapping session must never publish a notice")
	}
}

func TestObservePromptClearedBeforeDwellDoesNotFire(t *testing.T) {
	n, sched, _ := newTestNotifier()
	n.observe("sess-1", session.Activity{State: session.ActivityWorking}) // baseline
	n.observe("sess-1", session.Activity{State: session.ActivityWaiting}) // arms input-needed
	armed := sched.last()
	// Operator answers the prompt before the dwell elapses → session resumes.
	n.observe("sess-1", session.Activity{State: session.ActivityWorking})

	var fired bool
	n.publish = func(_ context.Context, _ eventbus.DomainEvent) error { fired = true; return nil }
	n.notice = func(_ context.Context, _ string) (string, error) { return "", nil }
	armed.fire()
	if fired {
		t.Fatalf("a prompt cleared before the dwell must not notify")
	}
}

func TestObserveInputNeededUsesPromptText(t *testing.T) {
	prompt := &backend.PendingPrompt{Kind: "permission", Text: "Allow writing to /etc/hosts?"}
	notice, ok := decideNotice("sess-1", session.ActivityWorking, session.Activity{State: session.ActivityWaiting, Prompt: prompt})
	if !ok {
		t.Fatalf("expected notice")
	}
	if notice.Body != "Allow writing to /etc/hosts?" {
		t.Fatalf("expected prompt text as body, got %q", notice.Body)
	}
	if notice.DedupeKey == "sess-1:input_needed" {
		t.Fatalf("expected prompt hash in dedupe key, got %q", notice.DedupeKey)
	}
}

func TestDeliverBuildsNeutralFactsWithScopeAndLink(t *testing.T) {
	n, _, _ := newTestNotifier()
	var got eventbus.DomainEvent
	n.publish = func(_ context.Context, event eventbus.DomainEvent) error {
		got = event
		return nil
	}
	n.notice = func(_ context.Context, scenario string) (string, error) {
		if scenario != notificationSourceScenario {
			t.Fatalf("link resolved for wrong scenario: %q", scenario)
		}
		return "https://web-console.example.com", nil
	}

	n.deliver(context.Background(), sessionNotice{
		SessionID: "sess-1",
		Category:  categoryInputNeeded,
		Urgency:   "high",
		Title:     "Session needs your input",
		Body:      "Allow?",
		DedupeKey: "sess-1:input_needed",
	})

	if got.EventType != SessionNotificationEventType || got.Source != notificationSourceScenario {
		t.Fatalf("unexpected event envelope: %+v", got)
	}
	if got.Payload["scope_key"] != "sess-1" {
		t.Fatalf("scope_key must be the session id, got %v", got.Payload["scope_key"])
	}
	if got.Payload["category"] != categoryInputNeeded || got.Payload["sensitivity"] != "private" {
		t.Fatalf("unexpected facts: %+v", got.Payload)
	}
	if got.Payload["context_url"] != "https://web-console.example.com/?session=sess-1" {
		t.Fatalf("unexpected deep link: %v", got.Payload["context_url"])
	}
	corr, ok := got.Payload["correlation"].(map[string]any)
	if !ok || corr["session_id"] != "sess-1" {
		t.Fatalf("correlation must carry session id, got %v", got.Payload["correlation"])
	}
}

func TestDeliverOmitsLinkWhenUnresolvable(t *testing.T) {
	n, _, _ := newTestNotifier()
	var got eventbus.DomainEvent
	n.publish = func(_ context.Context, event eventbus.DomainEvent) error { got = event; return nil }
	n.notice = func(_ context.Context, _ string) (string, error) { return "", errors.New("not exposed") }

	n.deliver(context.Background(), sessionNotice{SessionID: "sess-1", Category: categoryTurnCompleted})
	if _, present := got.Payload["context_url"]; present {
		t.Fatalf("no link should be set when the base is unresolvable")
	}
}

func TestDeliverSwallowsPublishError(t *testing.T) {
	n, _, _ := newTestNotifier()
	n.publish = func(_ context.Context, _ eventbus.DomainEvent) error { return errors.New("bus down") }
	n.notice = func(_ context.Context, _ string) (string, error) { return "", nil }
	// Must not panic.
	n.deliver(context.Background(), sessionNotice{SessionID: "sess-1", Category: categoryInputNeeded})
}

func TestForgetCancelsPendingNotice(t *testing.T) {
	n, sched, _ := newTestNotifier()
	n.observe("sess-1", session.Activity{State: session.ActivityWorking}) // baseline
	n.observe("sess-1", session.Activity{State: session.ActivityWaiting}) // arms
	n.forget("sess-1")

	if !sched.last().stopped {
		t.Fatalf("forget must cancel the session's settling notice")
	}
	if len(n.pending) != 0 {
		t.Fatalf("forget must drop the pending entry, have %d", len(n.pending))
	}
}
