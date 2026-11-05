package orchestration

import (
	"context"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/usagemode"
)

// rebaseRuns is an in-memory run store that supports only what a rebase uses.
type rebaseRuns struct {
	repository.RunRepository
	runs    map[uuid.UUID]*domain.Run
	updates int
}

func (r *rebaseRuns) Get(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	run := r.runs[id]
	if run == nil {
		return nil, nil
	}
	copied := *run
	cfg := *run.ResolvedConfig
	copied.ResolvedConfig = &cfg
	return &copied, nil
}

func (r *rebaseRuns) UpdateResolvedConfig(_ context.Context, id uuid.UUID, cfg *domain.RunConfig, expected domain.RunStatus) (bool, error) {
	run := r.runs[id]
	if run == nil || run.Status != expected {
		return false, nil
	}
	copied := *cfg
	run.ResolvedConfig = &copied
	r.updates++
	return true, nil
}

func newRebaseFixture(t *testing.T, mutate func(*domain.Run)) (*modeFixture, *rebaseRuns, *domain.Run, usagemode.Mode) {
	t.Helper()
	f := newModeFixture(t, nil)
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil) // no mode yet: the run is created on gpt-6.1-terra
	cfg.RunnerType = domain.RunnerTypeCodex
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusParked, ExecutionMode: domain.ExecutionModeCodecPipe, SessionID: "thread-1",
		ResolvedConfig: cfg, Summary: &domain.RunSummary{ContextTokens: 123000}}
	if mutate != nil {
		mutate(run)
	}
	store := &rebaseRuns{runs: map[uuid.UUID]*domain.Run{run.ID: run}}
	f.o.runs = store
	return f, store, run, writeMode(t, nil)
}

func rebase(t *testing.T, f *modeFixture, mode usagemode.Mode, dry bool, ids ...uuid.UUID) UsageModeRebaseResult {
	t.Helper()
	results, err := f.o.RebaseRunsToUsageMode(context.Background(), mode.ID, ids, dry)
	if err != nil || len(results) != 1 {
		t.Fatalf("results=%v err=%v", results, err)
	}
	return results[0]
}

func TestRebaseRecordsAPendingChangeForAParkedManagedRun(t *testing.T) {
	f, store, run, mode := newRebaseFixture(t, nil)
	got := rebase(t, f, mode, false, run.ID)
	if got.Status != RebaseSwitchedAtNextResume || got.FromModel != "gpt-6.1-terra" || got.ToModel != "gpt-6-luna" || got.EstimatedRereadTokens != 123000 {
		t.Fatalf("result = %+v", got)
	}
	stored := store.runs[run.ID].ResolvedConfig
	if stored.Model != "gpt-6.1-terra" {
		t.Fatalf("the model must not change before the resume: %q", stored.Model)
	}
	if p := stored.PendingModelRebase; p == nil || p.Reason != "usage_mode_rebase:"+mode.ID || p.ToModel != "gpt-6-luna" {
		t.Fatalf("pending = %+v", p)
	}
}

func TestRebaseDryRunChangesNothing(t *testing.T) {
	f, store, run, mode := newRebaseFixture(t, nil)
	got := rebase(t, f, mode, true, run.ID)
	if got.Status != RebaseSwitchedAtNextResume || got.EstimatedRereadTokens != 123000 {
		t.Fatalf("preview = %+v, want the switch and its re-read cost", got)
	}
	if store.updates != 0 || store.runs[run.ID].ResolvedConfig.PendingModelRebase != nil {
		t.Fatal("a preview must not record anything")
	}
}

func TestRebaseNeverTouchesInteractiveOrUnmanagedRuns(t *testing.T) {
	cases := map[string]struct {
		mutate func(*domain.Run)
		status string
		why    string
	}{
		"interactive": {func(r *domain.Run) { r.ExecutionMode = domain.ExecutionModeInteractive }, RebaseAppliesNextLaunch, "next launch"},
		"imported":    {func(r *domain.Run) { r.ExecutionMode = domain.ExecutionModeImported }, RebaseUnable, "not a managed run"},
		"attached":    {func(r *domain.Run) { r.ExecutionMode = domain.ExecutionModeAttached }, RebaseUnable, "not a managed run"},
		"mid turn":    {func(r *domain.Run) { r.Status = domain.RunStatusRunning }, RebaseUnable, "mid-turn"},
		"ended":       {func(r *domain.Run) { r.Status = domain.RunStatusFailed }, RebaseUnable, "ended"},
		"no session":  {func(r *domain.Run) { r.SessionID = "" }, RebaseUnable, "no retained session"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f, store, run, mode := newRebaseFixture(t, tc.mutate)
			got := rebase(t, f, mode, false, run.ID)
			if got.Status != tc.status || !strings.Contains(got.Reason, tc.why) {
				t.Fatalf("result = %+v, want %s mentioning %q", got, tc.status, tc.why)
			}
			if store.updates != 0 || store.runs[run.ID].ResolvedConfig.PendingModelRebase != nil || store.runs[run.ID].ResolvedConfig.Model != "gpt-6.1-terra" {
				t.Fatal("a run that cannot switch must be left exactly as it was")
			}
		})
	}
}

func TestRebaseReportsUnableWhenTheModeDoesNotLowerTheModel(t *testing.T) {
	f, _, run, mode := newRebaseFixture(t, func(r *domain.Run) {
		// The role itself resolved to the lower model, so it is not an explicit choice.
		r.ResolvedConfig.Model = "gpt-6-luna"
		for i := range r.ResolvedConfig.PolicySnapshot.Explanation.Preflight {
			r.ResolvedConfig.PolicySnapshot.Explanation.Preflight[i].Candidate.Model = "gpt-6-luna"
		}
	})
	got := rebase(t, f, mode, false, run.ID)
	if got.Status != RebaseUnable || !strings.Contains(got.Reason, "does not lower gpt-6-luna") {
		t.Fatalf("result = %+v", got)
	}
}

func TestRebaseIsRefusedForExemptRolesAndInactiveModes(t *testing.T) {
	f, _, run, mode := newRebaseFixture(t, func(r *domain.Run) { r.ResolvedConfig.RoleRef = "code.review" })
	if got := rebase(t, f, mode, false, run.ID); got.Status != RebaseUnable || !strings.Contains(got.Reason, "exempt") {
		t.Fatalf("exempt role result = %+v", got)
	}
	if _, err := f.o.RebaseRunsToUsageMode(context.Background(), "um-gone", []uuid.UUID{run.ID}, false); err == nil {
		t.Fatal("an unknown mode must be an error, not a silent no-op")
	}
}

func TestResumeAppliesThePendingRebaseAndKeepsTruthfulHistory(t *testing.T) {
	f, store, run, mode := newRebaseFixture(t, nil)
	rebase(t, f, mode, false, run.ID)
	resumed, _ := store.Get(context.Background(), run.ID)
	originalCandidate := resumed.ResolvedConfig.PolicySnapshot.SelectedCandidate.Model
	f.o.applyPendingModelRebase(context.Background(), resumed)
	cfg := resumed.ResolvedConfig
	if cfg.Model != "gpt-6-luna" || cfg.PendingModelRebase != nil {
		t.Fatalf("model=%q pending=%+v", cfg.Model, cfg.PendingModelRebase)
	}
	if len(cfg.ModelRebases) != 1 || cfg.ModelRebases[0].FromModel != "gpt-6.1-terra" || cfg.ModelRebases[0].AppliedAt == nil || cfg.ModelRebases[0].Reason != "usage_mode_rebase:"+mode.ID {
		t.Fatalf("history = %+v", cfg.ModelRebases)
	}
	if !strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode_rebase:"+mode.ID) {
		t.Fatalf("selection reason %q must record the rebase", cfg.PolicySnapshot.SelectionReason)
	}
	if cfg.PolicySnapshot.SelectedCandidate.Model != originalCandidate {
		t.Fatalf("the creation-time candidate was rewritten to %q; prior history must stay truthful", cfg.PolicySnapshot.SelectedCandidate.Model)
	}
	if stored := store.runs[run.ID].ResolvedConfig; stored.Model != "gpt-6-luna" || len(stored.ModelRebases) != 1 {
		t.Fatalf("the applied change was not persisted: %+v", stored)
	}
}

func TestResumeSkipsARebaseWhoseModeEndedAndKeepsTheModel(t *testing.T) {
	f, store, run, mode := newRebaseFixture(t, nil)
	rebase(t, f, mode, false, run.ID)
	if err := usagemode.Write(usagemode.File{}); err != nil { // the operator ended the mode
		t.Fatal(err)
	}
	resumed, _ := store.Get(context.Background(), run.ID)
	f.o.applyPendingModelRebase(context.Background(), resumed)
	if cfg := resumed.ResolvedConfig; cfg.Model != "gpt-6.1-terra" || cfg.PendingModelRebase != nil || len(cfg.ModelRebases) != 0 {
		t.Fatalf("an ended mode must not move the model: %+v", cfg)
	}
}

func TestNothingChangesUnlessRebaseIsRequested(t *testing.T) {
	f, store, run, _ := newRebaseFixture(t, nil) // a mode is active, but nobody asked
	resumed, _ := store.Get(context.Background(), run.ID)
	f.o.applyPendingModelRebase(context.Background(), resumed)
	if resumed.ResolvedConfig.Model != "gpt-6.1-terra" || store.updates != 0 {
		t.Fatal("an active mode alone must not move a running run")
	}
}

func TestRebaseHonoursTheModesRoleAndProfileScope(t *testing.T) {
	f, store, run, _ := newRebaseFixture(t, nil)
	otherRole := writeMode(t, func(m *usagemode.Mode) { m.Scope.Role = "code.review.other" })
	got := rebase(t, f, otherRole, false, run.ID)
	if got.Status != RebaseUnable || !strings.Contains(got.Reason, "role code.review.other") {
		t.Fatalf("a role-scoped mode must not move a run of another role: %+v", got)
	}
	sameRole := writeMode(t, func(m *usagemode.Mode) { m.Scope.Role = "code.default" })
	if got := rebase(t, f, sameRole, true, run.ID); got.Status != RebaseSwitchedAtNextResume {
		t.Fatalf("a run of the mode's role is moved: %+v", got)
	}
	profiled := writeMode(t, func(m *usagemode.Mode) { m.Scope.Profile = "reviewers" })
	got = rebase(t, f, profiled, false, run.ID)
	if got.Status != RebaseUnable || !strings.Contains(got.Reason, "profile reviewers") {
		t.Fatalf("a profile-scoped mode must not move a run with no matching profile: %+v", got)
	}
	if store.updates != 0 || store.runs[run.ID].ResolvedConfig.PendingModelRebase != nil {
		t.Fatal("refused runs must be left exactly as they were")
	}
}

func TestRebaseRefusesExplicitModelAndComparisonRunsWithAReason(t *testing.T) {
	cases := map[string]struct {
		mutate func(*domain.Run)
		why    string
	}{
		"explicit model": {func(r *domain.Run) {
			// The role resolved to the preflight's model; the caller then chose another.
			r.ResolvedConfig.Model = "gpt-6.1-sol"
			r.ResolvedConfig.PolicySnapshot.Explanation.Preflight = []domain.CandidatePreflight{{Index: r.ResolvedConfig.PolicySnapshot.SelectedIndex, Candidate: domain.ExecutionCandidate{Model: "gpt-6.1-terra"}, Available: true}}
		}, "explicit model"},
		"canary arm": {func(r *domain.Run) { r.ResolvedConfig.PolicySnapshot.CanaryArm = CanaryArmChallenger }, "challenger arm"},
		"run canary": {func(r *domain.Run) { r.CanaryArm = CanaryArmIncumbent }, "incumbent arm"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f, store, run, mode := newRebaseFixture(t, tc.mutate)
			got := rebase(t, f, mode, false, run.ID)
			if got.Status != RebaseUnable || !strings.Contains(got.Reason, tc.why) {
				t.Fatalf("result = %+v, want unable mentioning %q", got, tc.why)
			}
			if store.updates != 0 || store.runs[run.ID].ResolvedConfig.PendingModelRebase != nil {
				t.Fatal("a refused run must not be changed")
			}
		})
	}
}
