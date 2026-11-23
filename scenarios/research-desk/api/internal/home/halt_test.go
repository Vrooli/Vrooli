package home

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Halts are read from real rollout records: the two commissions that died on
// 2026-11-20 and the other turn-ending errors seen that month, kept as
// fixtures by packages/codex-rollout. Only their timestamps are moved to the
// test's clock.

// rolloutFixture returns a codex-rollout fixture with every record's
// timestamp set to at.
func rolloutFixture(t *testing.T, name string, at time.Time) []byte {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		path := filepath.Join(dir, "packages", "codex-rollout", "testdata", name)
		if data, err := os.ReadFile(path); err == nil {
			return regexp.MustCompile(`"timestamp":"[^"]+"`).ReplaceAll(data, []byte(`"timestamp":"`+at.UTC().Format(time.RFC3339Nano)+`"`))
		}
		if parent := filepath.Dir(dir); parent != dir {
			dir = parent
			continue
		}
		t.Fatalf("packages/codex-rollout/testdata/%s not found above the working directory", name)
	}
}

// errorLine is the one record of turn-errors.jsonl whose message holds fragment.
func errorLine(t *testing.T, fragment string, at time.Time) []byte {
	t.Helper()
	for _, line := range bytes.Split(rolloutFixture(t, "turn-errors.jsonl", at), []byte("\n")) {
		if bytes.Contains(line, []byte(fragment)) {
			return append(line, '\n')
		}
	}
	t.Fatalf("no turn error holds %q", fragment)
	return nil
}

// blockedWindow writes programme-three's session with a blocked goal, an
// open window and the thread's rollout.
func (f *fixture) blockedWindow(t *testing.T, status string, rollout []byte) {
	t.Helper()
	dir := f.codexHome(t, "s", []goalRow{{"t", f.programmeThreeGoal(), status, base}}, nil)
	f.procs.window(dir)
	writeFile(t, filepath.Join(dir, "sessions", "2026", "11", "10", "rollout-2026-11-10T12-00-00-t.jsonl"), string(rollout))
}

const p3 = "programme-three-20261111"

func TestABlockedGoalWhoseTurnEndedOnAnErrorIsHaltedNotLimited(t *testing.T) {
	cases := []struct {
		name, status string
		rollout      func(t *testing.T, now time.Time) []byte
		want         State
		wantRule     string
		class        string
	}{
		{"transport disconnect (steward-free-host-20261120)", "blocked", func(t *testing.T, now time.Time) []byte {
			return rolloutFixture(t, "rollout-transport-disconnect.jsonl", now.Add(-time.Minute))
		}, StateHalted, "last turn ended on a transient error", "transient"},
		{"sub-agent records after the error (steward-test-o8-20261120)", "blocked", func(t *testing.T, now time.Time) []byte {
			return rolloutFixture(t, "rollout-subagents-after-error.jsonl", now.Add(-time.Minute))
		}, StateHalted, "stream disconnected before completion", "transient"},
		{"model not supported", "blocked", func(t *testing.T, now time.Time) []byte {
			return errorLine(t, "model is not supported", now.Add(-time.Minute))
		}, StateHalted, "configuration error", "configuration"},
		{"blocked after a turn that ended cleanly", "blocked", func(t *testing.T, now time.Time) []byte {
			// The disconnect fixture's opening record starts a turn; a turn
			// still running is not an error.
			line, _, _ := bytes.Cut(rolloutFixture(t, "rollout-transport-disconnect.jsonl", now), []byte("\n"))
			return append(line, '\n')
		}, StateLimited, "goal blocked (its newest turn boundary is task_started", ""},
		{"a usage limit stays limited", "usage_limited", func(t *testing.T, now time.Time) []byte {
			return rolloutFixture(t, "rollout-transport-disconnect.jsonl", now.Add(-time.Minute))
		}, StateLimited, "goal usage_limited", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.blockedWindow(t, tc.status, tc.rollout(t, f.clock.Now()))
			c := f.snapshot(t, Options{}).Commission(p3)
			if c.State != tc.want || !strings.Contains(c.StateRule, tc.wantRule) {
				t.Fatalf("got %s (%q), want %s (%q)", c.State, c.StateRule, tc.want, tc.wantRule)
			}
			if tc.class == "" {
				if c.Halt != nil {
					t.Fatalf("%s carries a halt: %+v", c.State, c.Halt)
				}
				return
			}
			if c.Halt == nil || c.Halt.Class != tc.class || !c.State.Running() {
				t.Fatalf("halt = %+v", c.Halt)
			}
		})
	}
}

func TestAHaltIsResumedQuietlyOrEscalatedToTheSteward(t *testing.T) {
	launch := func(entries string) string {
		return `{"schema":"research-desk.launch/v1","commission_id":"` + p3 + `","launched_by":"desk","web_console_session_id":"s","thread_id":"t","steps":[],"controls":[],"recoveries":[` + entries + `]}`
	}
	resume := func(at time.Time) string {
		return `{"at":"` + at.UTC().Format(time.RFC3339) + `","turn_id":"x","class":"transient","action":"resume","attempt":1,"outcome":"resumed"}`
	}
	cases := []struct {
		name     string
		age      time.Duration
		fixture  string
		launch   func(now time.Time) string
		action   string
		flag     string
		attempts int
	}{
		{"fresh transient error: resume due in two minutes, no flag", time.Minute, "transport", nil, "resume", "", 0},
		{"resume overdue: the recovery pass is not running", 15 * time.Minute, "transport", nil, "resume", "automatic resume 1 was due", 0},
		{"stale: nobody acted while it was fresh (the 2026-11-20 deaths)", 90 * time.Minute, "transport", nil, "escalate", "past the 1h0m0s within which it is resumed automatically", 0},
		{"cap reached", time.Minute, "transport", func(now time.Time) string {
			return launch(resume(now.Add(-30*time.Minute)) + "," + resume(now.Add(-20*time.Minute)) + "," + resume(now.Add(-10*time.Minute)))
		}, "escalate", "tried 3 of 3 times", 3},
		{"older resumes fall out of the window", time.Minute, "transport", func(now time.Time) string {
			return launch(resume(now.Add(-3*time.Hour)) + "," + resume(now.Add(-2*time.Hour)) + "," + resume(now.Add(-10*time.Minute)))
		}, "resume", "", 1},
		{"a configuration error is never retried", time.Minute, "model is not supported", nil, "escalate", "a configuration error is not resumed automatically", 0},
		{"a flagged turn is never retried", time.Minute, "flagged for possible cybersecurity risk", nil, "escalate", "a policy error is not resumed automatically", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			now := f.clock.Now()
			rollout := rolloutFixture(t, "rollout-transport-disconnect.jsonl", now.Add(-tc.age))
			if tc.fixture != "transport" {
				rollout = errorLine(t, tc.fixture, now.Add(-tc.age))
			}
			f.blockedWindow(t, "blocked", rollout)
			if tc.launch != nil {
				writeFile(t, filepath.Join(f.research, "cohorts", p3, LaunchFile), tc.launch(now))
			}
			c := f.snapshot(t, Options{}).Commission(p3)
			if c.State != StateHalted || c.Halt.Plan.Action != tc.action || c.Halt.Attempts != tc.attempts {
				t.Fatalf("state %s halt %+v", c.State, c.Halt)
			}
			flags := strings.Join(c.Flags, "\n")
			if tc.flag == "" && strings.Contains(flags, "halted") {
				t.Fatalf("a halt about to be resumed is flagged: %q", flags)
			}
			if tc.flag != "" && !strings.Contains(flags, tc.flag) {
				t.Fatalf("flags %q lack %q", flags, tc.flag)
			}
			if tc.action == "escalate" && !strings.Contains(flags, "the steward decides") {
				t.Fatalf("an escalation must reach the steward: %q", flags)
			}
		})
	}
}

func TestAHaltPastItsAllowanceEscalates(t *testing.T) {
	f := newFixture(t)
	f.clock.Advance(25 * time.Hour)
	f.blockedWindow(t, "blocked", rolloutFixture(t, "rollout-transport-disconnect.jsonl", f.clock.Now().Add(-time.Minute)))
	c := f.snapshot(t, Options{}).Commission(p3)
	if c.State != StateHalted || c.Halt.Plan.Action != "escalate" || !strings.Contains(c.Halt.Plan.Escalation, "allowance ended") {
		t.Fatalf("state %s halt %+v", c.State, c.Halt)
	}
}
