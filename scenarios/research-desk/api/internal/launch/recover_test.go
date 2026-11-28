package launch

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	codexrollout "github.com/vrooli/vrooli/packages/codex-rollout"

	"research-desk/internal/home"
)

// The halts below carry the real error messages of November 2026's
// turn-ending errors; how they are read from a rollout is tested in
// internal/home and packages/codex-rollout.
var (
	transportError = codexrollout.TurnError{Message: "stream disconnected before completion: Transport error: network error: error decoding response body", Info: "other"}
	modelError     = codexrollout.TurnError{Message: "The 'luna' model is not supported when using Codex with a ChatGPT account.", Info: "model_not_found"}
)

// halted makes handRun a halted commission whose goal turn "turn-9" ended on
// e at errorAt, derived through home.HaltOf from the LAUNCH.json the real
// reader read, so recorded recoveries count exactly as in production.
func (r *rig) halted(t *testing.T, e codexrollout.TurnError, errorAt time.Time) {
	t.Helper()
	table, err := codexrollout.DefaultTurnErrorTable()
	if err != nil {
		t.Fatal(err)
	}
	clock := r.service.Clock.(*schedule.Fake)
	r.codex.goals = []home.Goal{{SessionID: "sess-1", ThreadID: "thread-1", Objective: "k", Status: "blocked"}}
	r.reader.tweaks[handRun] = func(c *home.Commission) {
		c.State = home.StateHalted
		c.Session = &home.Match{Kind: home.MatchExact, SessionID: "sess-1", ThreadID: "thread-1", GoalStatus: "blocked", WindowOpen: true}
		c.AllowanceEnd = home.Time{Value: clock.Now().Add(10 * time.Hour), Evidence: home.Evidence{Known: true, Source: "test"}}
		outcome := codexrollout.TurnOutcome{Event: codexrollout.EventTurnComplete, TurnID: "turn-9", At: errorAt, Error: &e}
		c.Halt = home.HaltOf(c, outcome, "rollout", table, clock.Now())
	}
}

func (r *rig) recoveries(t *testing.T) []home.RecoveryEntry {
	t.Helper()
	return r.launchFile(t, handRun).Recoveries
}

func TestAHaltedGoalIsResumedWhenDueOncePerAttemptAndRecorded(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	clock := r.service.Clock.(*schedule.Fake)
	errorAt := clock.Now().Add(-time.Minute)
	r.halted(t, transportError, errorAt)

	// The first resume is due two minutes after the error: not yet.
	if acts := r.service.Recover(ctx); len(acts) != 0 || len(r.codex.typed()) != 0 {
		t.Fatalf("acted before the backoff: %q, typed %q", acts, r.codex.typed())
	}
	clock.Advance(2 * time.Minute)
	acts := r.service.Recover(ctx)
	resume := []string{"type:/goal resume", "key:Enter"}
	if len(acts) != 1 || !slices.Equal(r.codex.typed(), resume) {
		t.Fatalf("acts %q, typed %q", acts, r.codex.typed())
	}
	got := r.recoveries(t)
	if len(got) != 1 || got[0].Action != "resume" || got[0].Attempt != 1 || got[0].TurnID != "turn-9" || got[0].Class != "transient" || got[0].Rule != "stream-disconnected" || got[0].GoalStatus != "active" {
		t.Fatalf("recoveries %+v", got)
	}
	if record := r.launchFile(t, handRun); record.WebConsoleSessionID != "sess-1" || record.ThreadID != "thread-1" || len(record.Controls) != 0 {
		t.Fatalf("a recovery is not an operator control: %+v", record)
	}

	// The goal halts again on the same error shape. The recorded attempt
	// moves the next one to five minutes after the error: the next pass
	// types nothing, and the one after that makes attempt 2.
	r.codex.setStatus("blocked")
	if acts := r.service.Recover(ctx); len(acts) != 0 || len(r.codex.typed()) != 2 {
		t.Fatalf("repeated an attempt: %q, typed %q", acts, r.codex.typed())
	}
	clock.Advance(3 * time.Minute)
	r.service.Recover(ctx)
	if got := r.recoveries(t); len(got) != 2 || got[1].Attempt != 2 || len(r.codex.typed()) != 4 {
		t.Fatalf("recoveries %+v, typed %q", got, r.codex.typed())
	}
}

func TestAHaltThatCannotBeResumedIsEscalatedOnceWithoutTyping(t *testing.T) {
	cases := []struct {
		name string
		err  codexrollout.TurnError
		age  time.Duration
		why  string
	}{
		{"configuration error", modelError, time.Minute, "a configuration error is not resumed automatically"},
		{"stale transient error, as on 2026-11-20", transportError, 20 * time.Hour, "past the 1h0m0s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			ctx := context.Background()
			r.halted(t, tc.err, r.service.Clock.Now().Add(-tc.age))
			if acts := r.service.Recover(ctx); len(acts) != 1 {
				t.Fatalf("acts %q", acts)
			}
			got := r.recoveries(t)
			if len(got) != 1 || got[0].Action != "escalate" || !strings.Contains(got[0].Outcome, tc.why) || got[0].Attempt != 0 {
				t.Fatalf("recoveries %+v", got)
			}
			if acts := r.service.Recover(ctx); len(acts) != 0 || len(r.recoveries(t)) != 1 || len(r.codex.typed()) != 0 {
				t.Fatalf("second pass acted %q or typed %q", acts, r.codex.typed())
			}
		})
	}
}

func TestAResumeThatDoesNotTakeIsRecordedAsSuchAndCountsTowardTheCap(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	clock := r.service.Clock.(*schedule.Fake)
	r.codex.resumeTo = "blocked"
	r.halted(t, transportError, clock.Now().Add(-3*time.Minute))
	r.service.Recover(ctx)
	clock.Advance(3 * time.Minute)
	r.service.Recover(ctx)
	clock.Advance(5 * time.Minute)
	r.service.Recover(ctx)
	clock.Advance(time.Minute)
	r.service.Recover(ctx)
	got := r.recoveries(t)
	if len(got) != 4 {
		t.Fatalf("recoveries %+v", got)
	}
	for _, e := range got[:3] {
		if e.Action != "resume" || !strings.Contains(e.Outcome, "still blocked") {
			t.Fatalf("attempt %+v", e)
		}
	}
	if got[3].Action != "escalate" || !strings.Contains(got[3].Outcome, "tried 3 of 3") {
		t.Fatalf("after the cap: %+v", got[3])
	}
}

func TestTheStewardCanResumeAHaltedGoalButNotAnotherBlockedOne(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	r.halted(t, modelError, r.service.Clock.Now().Add(-time.Minute))
	out, err := r.service.Control(ctx, handRun, ActResume, "")
	if err != nil || out.GoalStatus != "active" || !slices.Equal(r.codex.typed(), []string{"type:/goal resume", "key:Enter"}) {
		t.Fatalf("resume of a halted goal: %+v %v, typed %q", out, err, r.codex.typed())
	}
	// A goal blocked for a reason the desk cannot name is limited, and
	// resume still refuses it.
	r2 := newRig(t)
	r2.codex.goals = []home.Goal{{SessionID: "sess-1", ThreadID: "thread-1", Objective: "k", Status: "blocked"}}
	r2.reader.states[handRun] = home.StateLimited
	r2.reader.sessions[handRun] = "sess-1"
	if _, err := r2.service.Control(ctx, handRun, ActResume, ""); !errors.Is(err, home.ErrRefused) || len(r2.codex.typed()) != 0 {
		t.Fatalf("resume of a limited blocked goal: %v, typed %q", err, r2.codex.typed())
	}
}

// 2026-11-21: Web Console timed out on the read-back after /goal went in;
// the goal was active in goals_1.sqlite two seconds later, but the launch
// was recorded failed at goal_sent.
func TestALaunchWhoseGoalReadBackFailsIsDecidedByTheGoalRow(t *testing.T) {
	notReached := errors.New("web-console is not reachable: web-console was not reached just now (timed out after 3s)")
	codex := newCodex()
	codex.screenErr = notReached
	var steps []string
	err := launcher(codex).Run(context.Background(), SessionSpec{}, Texts{Stance: "Stance.", Kernel: "Kernel."}, settings, func(step, _, _ string) error {
		steps = append(steps, step)
		return nil
	})
	if err != nil || !slices.Equal(steps, []string{StepCreated, StepComposerReady, StepStanceSent, StepReadySeen, StepGoalSent, StepGoalVerified}) {
		t.Fatalf("err %v, steps %v", err, steps)
	}

	// When no goal row appears, the launch still fails at goal_sent and
	// says both what the call and what the read found.
	lost := newCodex()
	lost.typeErr = notReached
	err = launcher(lost).Run(context.Background(), SessionSpec{}, Texts{Stance: "Stance.", Kernel: "Kernel."}, settings, func(string, string, string) error { return nil })
	var stepErr *StepError
	if !errors.As(err, &stepErr) || stepErr.Step != StepGoalSent || !strings.Contains(err.Error(), "not reached") || !strings.Contains(err.Error(), "no goal with the kernel") {
		t.Fatalf("err = %v", err)
	}
}
