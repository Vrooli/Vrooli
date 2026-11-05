package orchestration

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/vrooli/cli-core/usagemode"
)

func learnFixtureMode(role string, mutate func(*usagemode.Mode)) usagemode.Mode {
	now := time.Now()
	expires := now.Add(2 * time.Hour)
	mode := usagemode.Mode{
		ID: "um-learn-aaaaaa", Kind: usagemode.KindLearn, StartedAt: now.Add(-time.Minute), Source: "agent-manager",
		Scope: usagemode.Scope{Runner: "codex", Role: role}, Until: usagemode.Until{Kind: usagemode.UntilTime}, ExpiresAt: &expires,
		Runners: map[string]usagemode.RunnerSettings{},
		Learn: &usagemode.Learn{SplitPercent: 50, Runners: map[string]usagemode.RunnerSettings{
			"codex": {ModelMap: map[string]string{"gpt-6.1-terra": "gpt-6-luna"}, SessionModel: "gpt-6-luna"}}},
	}
	if mutate != nil {
		mutate(&mode)
	}
	return mode
}

func writeModes(t *testing.T, modes ...usagemode.Mode) {
	t.Helper()
	if err := usagemode.Write(usagemode.File{Modes: modes}); err != nil {
		t.Fatal(err)
	}
}

func assignArms(cfg *domain.RunConfig, runs int) (challenger int, snapshots []*domain.ExecutionPolicySnapshot) {
	for i := 0; i < runs; i++ {
		snapshot := *cfg.PolicySnapshot
		snapshot.Candidates = append([]domain.ExecutionCandidate(nil), cfg.PolicySnapshot.Candidates...)
		applyCanary(&snapshot, fmt.Sprintf("run-%d", i), cfg.Model)
		if snapshot.CanaryArm == CanaryArmChallenger {
			challenger++
		}
		snapshots = append(snapshots, &snapshot)
	}
	return challenger, snapshots
}

func TestLearnModeDeclaresTheChallengerAndTheExistingCanaryAssignsAndRecordsTheArm(t *testing.T) {
	f := newModeFixture(t, nil)
	mode := learnFixtureMode("code.default", nil)
	writeModes(t, mode)
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	selected := cfg.PolicySnapshot.SelectedCandidate
	if cfg.Model != "gpt-6.1-terra" || selected.ChallengerModel != "gpt-6-luna" || selected.ChallengerSampleRate != 0.5 {
		t.Fatalf("model %q challenger %q rate %v: the overlay declares the challenger and leaves the incumbent in place", cfg.Model, selected.ChallengerModel, selected.ChallengerSampleRate)
	}
	if !strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode_learn:"+mode.ID) || strings.Contains(cfg.PolicySnapshot.SelectionReason, ";usage_mode:") {
		t.Fatalf("selection reason %q", cfg.PolicySnapshot.SelectionReason)
	}
	challenger, snapshots := assignArms(cfg, 400)
	if challenger < 160 || challenger > 240 {
		t.Fatalf("challenger runs = %d of 400 at a 50/50 split", challenger)
	}
	for _, snapshot := range snapshots {
		switch snapshot.CanaryArm {
		case CanaryArmChallenger:
			if snapshot.SelectedCandidate.Model != "gpt-6-luna" || snapshot.SelectedCandidate.CanaryArm != CanaryArmChallenger {
				t.Fatalf("challenger snapshot: %+v", snapshot.SelectedCandidate)
			}
		case CanaryArmIncumbent:
			if snapshot.SelectedCandidate.Model != "gpt-6.1-terra" {
				t.Fatalf("incumbent snapshot ran %q", snapshot.SelectedCandidate.Model)
			}
		default:
			t.Fatalf("no arm recorded: %+v", snapshot)
		}
	}
}

func TestLearnSplitSetsTheSampleRate(t *testing.T) {
	f := newModeFixture(t, nil)
	writeModes(t, learnFixtureMode("code.default", func(m *usagemode.Mode) { m.Learn.SplitPercent = 20 }))
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	challenger, _ := assignArms(cfg, 1000)
	if rate := cfg.PolicySnapshot.SelectedCandidate.ChallengerSampleRate; rate != 0.2 || challenger < 150 || challenger > 250 {
		t.Fatalf("rate %v, %d challenger runs of 1000", rate, challenger)
	}
}

func requireNoComparison(t *testing.T, cfg *domain.RunConfig, why string) {
	t.Helper()
	selected := cfg.PolicySnapshot.SelectedCandidate
	if selected.ChallengerModel != "" || selected.ChallengerSampleRate != 0 || strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode_learn:") {
		t.Fatalf("%s: run joined a comparison: challenger %q reason %q", why, selected.ChallengerModel, cfg.PolicySnapshot.SelectionReason)
	}
	if _, snapshots := assignArms(cfg, 50); snapshots[0].CanaryArm != "" {
		t.Fatalf("%s: canary recorded an arm %q", why, snapshots[0].CanaryArm)
	}
}

func TestNoModeLeavesTheCanaryInert(t *testing.T) {
	f := newModeFixture(t, nil)
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	requireNoComparison(t, cfg, "no mode")
	if cfg.Model != "gpt-6.1-terra" {
		t.Fatalf("model changed to %q", cfg.Model)
	}
}

func TestExemptRoleIsNeverRandomizedEvenWhenTheModeIsScopedToIt(t *testing.T) {
	f := newModeFixture(t, nil)
	writeModes(t, learnFixtureMode("code.review", nil))
	requireNoComparison(t, f.create(t, "code.review", CreateRunRequest{}, nil), "exempt role scoped explicitly")
	f = newModeFixture(t, nil)
	writeModes(t, learnFixtureMode("code.default", func(m *usagemode.Mode) { m.ExemptRoles = []string{"code.default"} }))
	requireNoComparison(t, f.create(t, "code.default", CreateRunRequest{}, nil), "role the mode itself exempts")
}

func TestLearnModeCoversOnlyItsRole(t *testing.T) {
	f := newModeFixture(t, nil)
	writeModes(t, learnFixtureMode("code.default", nil))
	requireNoComparison(t, f.create(t, "code.review", CreateRunRequest{}, nil), "another role")
}

func TestSaveModeTakesPriorityOverLearnAtRunCreation(t *testing.T) {
	f := newModeFixture(t, nil)
	save := usagemode.Mode{
		ID: "um-save-bbbbbb", StartedAt: time.Now().Add(-time.Minute), Until: usagemode.Until{Kind: usagemode.UntilTime}, ExpiresAt: ptrTime(time.Now().Add(time.Hour)),
		Runners: map[string]usagemode.RunnerSettings{"codex": {ModelMap: map[string]string{"gpt-6.1-terra": "gpt-6-luna"}, SessionModel: "gpt-6-luna"}},
	}
	writeModes(t, learnFixtureMode("code.default", nil), save)
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	requireLowered(t, cfg, save)
	requireNoComparison(t, cfg, "save active")
}

func TestPausedLearnModeAssignsNothing(t *testing.T) {
	f := newModeFixture(t, nil)
	writeModes(t, learnFixtureMode("code.default", func(m *usagemode.Mode) { m.Learn.Paused = true }))
	requireNoComparison(t, f.create(t, "code.default", CreateRunRequest{}, nil), "paused")
}

func TestExplicitModelStaysOutOfTheComparison(t *testing.T) {
	f := newModeFixture(t, nil)
	writeModes(t, learnFixtureMode("code.default", nil))
	model := "gpt-6.1-terra"
	cfg := f.create(t, "code.default", CreateRunRequest{Model: &model}, nil)
	requireNoComparison(t, cfg, "explicit model")
}

func TestLearnModeNeverFailsRunCreationWhenTheChallengerIsRefused(t *testing.T) {
	f := newModeFixture(t, func(model string) error {
		if model == "gpt-6-luna" {
			return errors.New("not served")
		}
		return nil
	})
	writeModes(t, learnFixtureMode("code.default", nil))
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	if cfg.Model != "gpt-6.1-terra" || cfg.PolicySnapshot.SelectedCandidate.ChallengerModel != "" || !strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode_learn_skipped:probe_failed") {
		t.Fatalf("model %q reason %q", cfg.Model, cfg.PolicySnapshot.SelectionReason)
	}
}

func TestResumeIsNotEligibleForALearnComparison(t *testing.T) {
	f := newModeFixture(t, nil)
	writeModes(t, learnFixtureMode("code.default", nil))
	cfg := domain.DefaultRunConfig()
	cfg.RoleRef = "code.default"
	if err := f.o.resolveExecutionPolicy(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	f.o.applyUsageMode(t.Context(), cfg, nil, CreateRunRequest{}) // usageModeEligible is false for a recovery
	requireNoComparison(t, cfg, "recovery")
}
