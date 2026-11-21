package outcomeledger

import (
	"testing"
	"time"

	"github.com/vrooli/cli-core/usagemode"
)

var armDay = time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)

func learnPeriod(optIn bool) ModePeriod {
	return ModePeriod{ID: "um-l", Kind: usagemode.KindLearn, Runner: "codex", Role: "code.economy.delivery", Split: 0.5, SessionOptIn: optIn, BlockDays: 1, StartedAt: armDay.Add(-30 * 24 * time.Hour)}
}

func TestManagedRunArmIsTheRecordedCanaryArmUnderItsLearnMode(t *testing.T) {
	for _, arm := range []string{usagemode.ArmIncumbent, usagemode.ArmChallenger} {
		run := RunFacts{RunID: "r", Path: PathHeadless, Role: "code.economy.delivery", CanaryArm: arm, SelectionReason: "role_default;usage_mode_learn:um-l"}
		got := ResolveArm(run, nil, nil)
		if got.Arm != arm || got.ModeID != "um-l" || got.Basis != ArmBasisCanary {
			t.Fatalf("%s: %+v", arm, got)
		}
	}
	// A run on a policy canary that no learn mode touched is not in a comparison.
	policy := RunFacts{RunID: "p", Path: PathHeadless, Role: "code.default", CanaryArm: usagemode.ArmChallenger, SelectionReason: "role_default"}
	if got := ResolveArm(policy, nil, nil); got.Arm != "" {
		t.Fatalf("a policy canary is not a learn arm: %+v", got)
	}
	// A learn reason with no recorded arm (canary inert) is not an arm either.
	inert := RunFacts{RunID: "i", Path: PathHeadless, Role: "code.economy.delivery", SelectionReason: "usage_mode_learn:um-l"}
	if got := ResolveArm(inert, nil, nil); got.Arm != "" {
		t.Fatalf("no recorded arm, no assignment: %+v", got)
	}
}

func TestSessionArmComesFromTheDayBlockCalendarOnlyWhenOptedIn(t *testing.T) {
	session := RunFacts{RunID: "s", Path: PathImported, Harness: "codex", SessionStart: armDay}
	want := usagemode.SessionArm("um-l", armDay, 0.5, 1)
	got := ResolveArm(session, []ModePeriod{learnPeriod(true)}, nil)
	if got.Arm != want || got.Basis != ArmBasisCalendar || got.ModeID != "um-l" {
		t.Fatalf("opted in: %+v want arm %s", got, want)
	}
	if got := ResolveArm(session, []ModePeriod{learnPeriod(false)}, nil); got.Arm != "" {
		t.Fatalf("sessions that did not opt in are never in the comparison: %+v", got)
	}
	outside := session
	outside.SessionStart = armDay.Add(-90 * 24 * time.Hour)
	if got := ResolveArm(outside, []ModePeriod{learnPeriod(true)}, nil); got.Arm != "" {
		t.Fatalf("a session before the period is not in it: %+v", got)
	}
	managed := RunFacts{RunID: "m", Path: PathHeadless, Role: "code.default", SessionStart: armDay}
	if got := ResolveArm(managed, []ModePeriod{learnPeriod(true)}, nil); got.Arm != "" {
		t.Fatalf("a managed run never takes a session arm: %+v", got)
	}
}

func TestLauncherMarkerConfirmsTheArmAndRecordsTheException(t *testing.T) {
	session := RunFacts{RunID: "s", Path: PathImportedWebConsole, Harness: "codex", SessionStart: armDay}
	arm := usagemode.SessionArm("um-l", armDay, 0.5, 1)
	marker := usagemode.LaunchMarker{At: armDay.Add(time.Minute), ModeID: "um-l", Runner: "codex", Arm: arm, Exception: usagemode.ExceptionFullPower}
	got := ResolveArm(session, []ModePeriod{learnPeriod(true)}, []usagemode.LaunchMarker{marker})
	if got.Basis != ArmBasisMarker || got.Exception != usagemode.ExceptionFullPower || got.Arm != arm {
		t.Fatalf("marker: %+v", got)
	}
	far := marker
	far.At = armDay.Add(3 * time.Hour)
	if got := ResolveArm(session, []ModePeriod{learnPeriod(true)}, []usagemode.LaunchMarker{far}); got.Basis != ArmBasisCalendar || got.Exception != "" {
		t.Fatalf("a marker hours away is not this session's: %+v", got)
	}
}

func TestDeriveRecordsArmsAndLearnPeriodsDoNotLookLikeASavingMode(t *testing.T) {
	period := learnPeriod(true)
	period.Targets = []string{"gpt-6-luna"}
	session := RunFacts{RunID: "s", Path: PathImported, Harness: "codex", SessionStart: armDay, FirstActivity: armDay, LastActivity: armDay.Add(time.Hour), ActualModel: "gpt-6-luna", Requests: 1}
	managed := RunFacts{RunID: "m", Path: PathHeadless, Harness: "codex", Role: "code.economy.delivery", CanaryArm: usagemode.ArmChallenger, SelectionReason: "usage_mode_learn:um-l", FirstActivity: armDay, LastActivity: armDay.Add(time.Hour), ActualModel: "gpt-6-luna", Requests: 1}
	rows := []RunFacts{session, managed}
	Derive(rows, []ModePeriod{period}, nil)
	if rows[0].ArmMode != "um-l" || rows[0].Arm == "" || rows[1].Arm != usagemode.ArmChallenger || rows[1].ArmBasis != ArmBasisCanary {
		t.Fatalf("arms not recorded in facts: %+v / %+v", rows[0], rows[1])
	}
	for _, row := range rows {
		if row.ModeRelation != ModeNone {
			t.Fatalf("a learn period must not make a run 'ran during' a mode: %s %s", row.RunID, row.ModeRelation)
		}
	}
	counts := CountArms(rows, "um-l")
	if counts.Incumbent+counts.Challenger != 2 {
		t.Fatalf("counts = %+v", counts)
	}
}
