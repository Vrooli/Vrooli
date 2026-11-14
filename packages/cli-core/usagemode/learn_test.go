package usagemode

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

var challenger = RunnerSettings{ModelMap: map[string]string{"gpt-6.1-terra": "gpt-6-luna"}, SessionModel: "gpt-6-luna"}

func learnMode(id, role string, opts ...func(*Mode)) Mode {
	m := Mode{
		ID: id, Kind: KindLearn, Scope: Scope{Runner: "codex", Role: role}, StartedAt: t0.Add(-time.Hour), ExpiresAt: tp(t0.Add(24 * time.Hour)),
		Until: Until{Kind: UntilTime}, Runners: map[string]RunnerSettings{},
		Learn: &Learn{SplitPercent: 50, Runners: map[string]RunnerSettings{"codex": challenger}},
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func optIn(m *Mode) { m.Learn.Sessions = &SessionBlocks{OptIn: true} }

func TestLearnModeNeverActsAsASaveMode(t *testing.T) {
	m := learnMode("um-l", "code.economy.delivery", optIn)
	if _, ok := resolveFrom([]Mode{m}, Query{Runner: "codex", Role: "code.economy.delivery", Now: t0}); ok {
		t.Fatal("a learn mode must not resolve as a save mode")
	}
	// A reader that does not know Learn finds no runner settings at all.
	if _, ok := m.SettingsFor("codex"); ok {
		t.Fatal("learn settings must not sit in Runners, where an older reader would apply them")
	}
}

func TestResolveLearnCoversOnlyItsRoleAndNeverAnExemptRole(t *testing.T) {
	covered := learnMode("um-l", "code.economy.delivery")
	exempt := learnMode("um-x", "code.review")
	cases := []struct {
		name string
		mode Mode
		role string
		want bool
	}{
		{"covered role joins", covered, "code.economy.delivery", true},
		{"other role stays out", covered, "code.default", false},
		{"no role stays out", covered, "", false},
		{"exempt role is never randomized even when scoped to it", exempt, "code.review", false},
	}
	for _, c := range cases {
		_, ok := resolveLearnFrom([]Mode{c.mode}, Query{Runner: "codex", Role: c.role, Now: t0})
		if ok != c.want {
			t.Errorf("%s: joined = %v, want %v", c.name, ok, c.want)
		}
	}
	if _, ok := resolveLearnFrom([]Mode{covered}, Query{Runner: "claude", Role: "code.economy.delivery", Now: t0}); ok {
		t.Error("a codex comparison must not take a claude run")
	}
}

func TestSaveModeTakesPriorityOverLearnAndPauseStopsAssignment(t *testing.T) {
	learn := learnMode("um-l", "code.economy.delivery", optIn)
	save := mode("um-save", Scope{Runner: "codex"}, t0.Add(-time.Minute), tp(t0.Add(time.Hour)), "codex")
	q := Query{Runner: "codex", Role: "code.economy.delivery", Now: t0}
	if _, ok := resolveLearnFrom([]Mode{learn, save}, q); ok {
		t.Fatal("a save mode that applies must take the run out of the comparison")
	}
	if got, ok := resolveFrom([]Mode{learn, save}, q); !ok || got.ID != "um-save" {
		t.Fatalf("save resolution = %+v, %v", got, ok)
	}
	paused := learnMode("um-p", "code.economy.delivery", func(m *Mode) { m.Learn.Paused = true })
	if _, ok := resolveLearnFrom([]Mode{paused}, q); ok {
		t.Fatal("a paused comparison must not assign new work")
	}
}

func TestSessionArmIsDeterministicBalancedAndBlocked(t *testing.T) {
	day := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	if SessionArm("um-a", day, 0.5, 1) != SessionArm("um-a", day.Add(5*time.Hour), 0.5, 1) {
		t.Fatal("the same day and mode must give the same arm")
	}
	challengerDays := 0
	for i := 0; i < 400; i++ {
		if SessionArm("um-a", day.AddDate(0, 0, i), 0.5, 1) == ArmChallenger {
			challengerDays++
		}
	}
	if challengerDays < 160 || challengerDays > 240 {
		t.Fatalf("challenger days = %d of 400 at 50/50", challengerDays)
	}
	if SessionArm("um-a", day, 0.5, 1) == SessionArm("um-b", day, 0.5, 1) && SessionArm("um-a", day.AddDate(0, 0, 1), 0.5, 1) == SessionArm("um-b", day.AddDate(0, 0, 1), 0.5, 1) &&
		SessionArm("um-a", day.AddDate(0, 0, 2), 0.5, 1) == SessionArm("um-b", day.AddDate(0, 0, 2), 0.5, 1) &&
		SessionArm("um-a", day.AddDate(0, 0, 3), 0.5, 1) == SessionArm("um-b", day.AddDate(0, 0, 3), 0.5, 1) &&
		SessionArm("um-a", day.AddDate(0, 0, 4), 0.5, 1) == SessionArm("um-b", day.AddDate(0, 0, 4), 0.5, 1) {
		t.Fatal("two modes should not share a calendar")
	}
	block := SessionArm("um-a", time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), 0.5, 3)
	for d := 0; d < 3; d++ { // 2026-11-02 is day 20394 = 3 * 6798: a block start
		if got := SessionArm("um-a", time.Date(2026, 11, 2+d, 9, 0, 0, 0, time.UTC), 0.5, 3); got != block {
			t.Fatalf("day %d of a three-day block = %s, want %s", d, got, block)
		}
	}
}

func installLearn(t *testing.T, m Mode, extra ...Mode) {
	t.Helper()
	useHome(t)
	if err := Write(File{Modes: append([]Mode{m}, extra...)}); err != nil {
		t.Fatal(err)
	}
}

func TestSessionWithoutOptInIsNeverRandomized(t *testing.T) {
	installLearn(t, learnMode("um-l", "code.economy.delivery"))
	args, env := []string{"exec", "hi"}, []string{"A=1"}
	plan := LaunchPlan("codex", args, env, t0)
	if plan.Applied || plan.Arm != "" || plan.Env != nil || !reflect.DeepEqual(plan.Args, args) {
		t.Fatalf("a session must stay untouched without opt-in: %+v", plan)
	}
}

func TestOptedInSessionsAreLabelledAndOnlyChallengerDaysChangeTheLaunch(t *testing.T) {
	m := learnMode("um-l", "code.economy.delivery", optIn)
	installLearn(t, m)
	var sawChallenger, sawIncumbent bool
	for d := 0; d < 40 && !(sawChallenger && sawIncumbent); d++ {
		day := t0.AddDate(0, 0, d)
		m.StartedAt, m.ExpiresAt = day.Add(-time.Hour), tp(day.Add(time.Hour))
		installLearn(t, m)
		plan := LaunchPlan("codex", []string{"exec", "hi"}, nil, day)
		if !plan.Applied || plan.ModeID != "um-l" || !strings.Contains(plan.Label, "Learn mode um-l") {
			t.Fatalf("an opted-in session must carry its mode and a visible label: %+v", plan)
		}
		want := m.SessionArm(day)
		if plan.Arm != want || !containsEntry(plan.Env, EnvArm+"="+want) || !containsEntry(plan.Env, EnvMode+"=um-l") {
			t.Fatalf("arm %q env %v, want %q", plan.Arm, plan.Env, want)
		}
		if want == ArmChallenger {
			sawChallenger = true
			if !reflect.DeepEqual(plan.Args, []string{"-c", "model=gpt-6-luna", "exec", "hi"}) {
				t.Fatalf("challenger day args = %v", plan.Args)
			}
		} else {
			sawIncumbent = true
			if !reflect.DeepEqual(plan.Args, []string{"exec", "hi"}) {
				t.Fatalf("incumbent day must not change the launch: %v", plan.Args)
			}
		}
	}
	if !sawChallenger || !sawIncumbent {
		t.Fatal("expected both arms across 40 days")
	}
}

func containsEntry(env []string, entry string) bool {
	for _, e := range env {
		if e == entry {
			return true
		}
	}
	return false
}

func challengerDay(t *testing.T, m Mode) time.Time {
	t.Helper()
	for d := 0; d < 60; d++ {
		day := t0.AddDate(0, 0, d)
		if m.SessionArm(day) == ArmChallenger {
			return day
		}
	}
	t.Fatal("no challenger day found")
	return time.Time{}
}

func TestFullPowerAndExplicitModelAreExceptionsKeptInTheirAssignedArm(t *testing.T) {
	m := learnMode("um-l", "code.economy.delivery", optIn)
	day := challengerDay(t, m)
	m.StartedAt, m.ExpiresAt = day.Add(-time.Hour), tp(day.Add(time.Hour))
	installLearn(t, m)

	power := LaunchPlan("codex", []string{"exec", "hi"}, []string{EnvFullPower + "=1"}, day)
	if power.Arm != ArmChallenger || power.Exception != ExceptionFullPower || !reflect.DeepEqual(power.Args, []string{"exec", "hi"}) || !containsEntry(power.Env, ArmExceptionKey+"=full_power") {
		t.Fatalf("full power must keep the model, stay in the challenger arm and be recorded: %+v", power)
	}
	explicit := LaunchPlan("codex", []string{"-m", "gpt-6.1-terra", "exec", "hi"}, nil, day)
	if explicit.Arm != ArmChallenger || explicit.Exception != ExceptionExplicitModel {
		t.Fatalf("explicit model exception = %+v", explicit)
	}
	claude := LaunchPlan("claude", []string{"--model", "opus"}, nil, day)
	if claude.Applied {
		t.Fatalf("no claude settings exist for this codex mode: %+v", claude)
	}
}

func TestSaveModeBeatsAnOptedInLearnSession(t *testing.T) {
	learn := learnMode("um-l", "code.economy.delivery", optIn)
	save := mode("um-save", Scope{Runner: "codex"}, t0.Add(-time.Minute), tp(t0.Add(time.Hour)), "codex")
	installLearn(t, learn, save)
	plan := LaunchPlan("codex", []string{"exec", "hi"}, nil, t0)
	if plan.ModeID != "um-save" || plan.Arm != "" {
		t.Fatalf("save must take priority over learn: %+v", plan)
	}
}

func TestLaunchMarkersRoundTripAndFailOpen(t *testing.T) {
	useHome(t)
	if got := ReadLaunches(time.Time{}); got != nil {
		t.Fatalf("missing file: %v", got)
	}
	RecordLaunch(LaunchMarker{At: t0, ModeID: "um-l", Runner: "codex", Arm: ArmChallenger, Exception: ExceptionFullPower})
	RecordLaunch(LaunchMarker{At: t0.Add(-48 * time.Hour), ModeID: "um-l", Runner: "codex", Arm: ArmIncumbent})
	RecordLaunch(LaunchMarker{})
	got := ReadLaunches(t0.Add(-time.Hour))
	if len(got) != 1 || got[0].Exception != ExceptionFullPower {
		t.Fatalf("markers = %+v", got)
	}
}
