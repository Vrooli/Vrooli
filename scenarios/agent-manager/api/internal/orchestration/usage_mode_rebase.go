// Responsibility: let an operator explicitly move managed runs to a usage
// mode's lower model at their next resume, with a per-run result.
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/usagemode"
)

// Rebase result statuses.
const (
	RebaseSwitchedAtNextResume = "switched_at_next_resume"
	RebaseAppliesNextLaunch    = "applies_on_next_launch"
	RebaseUnable               = "unable"
)

// UsageModeRebaseResult is one run's outcome. Nothing changes for a run unless
// the operator asked for it, and an unable run changes nothing.
type UsageModeRebaseResult struct {
	RunID     uuid.UUID
	Status    string
	Reason    string
	FromModel string
	ToModel   string
	// EstimatedRereadTokens is the retained context the next resume re-sends on
	// the new model (a model switch invalidates the prompt cache). Zero means
	// the run has no context size on record.
	EstimatedRereadTokens int64
}

// RebaseRunsToUsageMode prepares the explicit "also switch running work" step
// of a usage mode. For a managed (codec pipe) run it records a pending change
// that the next resume applies (applyPendingModelRebase); interactive sessions
// are never touched (typing /model would persist as the operator's default and
// the switch forces a full re-read) and report that the mode applies on their
// next launch. dryRun computes the same results without recording anything.
// With no run ids it considers every non-terminal run.
func (o *Orchestrator) RebaseRunsToUsageMode(ctx context.Context, modeID string, runIDs []uuid.UUID, dryRun bool) ([]UsageModeRebaseResult, error) {
	if o.runs == nil {
		return nil, errors.New("run repository is unavailable")
	}
	mode, ok := activeUsageModeByID(modeID, o.now())
	if !ok {
		return nil, domain.NewValidationError("modeId", fmt.Sprintf("usage mode %q is not active", modeID))
	}
	if len(runIDs) == 0 {
		rows, err := o.runs.List(ctx, repository.RunListFilter{NonTerminalOnly: true})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			runIDs = append(runIDs, row.ID)
		}
	}
	results := make([]UsageModeRebaseResult, 0, len(runIDs))
	for _, id := range runIDs {
		results = append(results, o.rebaseOneRun(ctx, mode, id, dryRun))
	}
	return results, nil
}

func activeUsageModeByID(id string, now time.Time) (usagemode.Mode, bool) {
	for _, mode := range usagemode.Read(now) {
		if mode.ID == strings.TrimSpace(id) {
			return mode, true
		}
	}
	return usagemode.Mode{}, false
}

func unableRebase(id uuid.UUID, from, reason string) UsageModeRebaseResult {
	return UsageModeRebaseResult{RunID: id, Status: RebaseUnable, Reason: reason, FromModel: from}
}

// rebaseOneRun decides and (unless dryRun) records one run's rebase under the
// wake lock, so it cannot race a wake or continuation of the same run.
func (o *Orchestrator) rebaseOneRun(ctx context.Context, mode usagemode.Mode, id uuid.UUID, dryRun bool) UsageModeRebaseResult {
	o.wakeMu.Lock()
	defer o.wakeMu.Unlock()
	run, err := o.runs.Get(ctx, id)
	if err != nil || run == nil {
		return unableRebase(id, "", "run not found")
	}
	cfg := run.ResolvedConfig
	if cfg == nil {
		return unableRebase(id, "", "run has no resolved configuration")
	}
	from := cfg.Model
	switch run.ExecutionMode.Normalized() {
	case domain.ExecutionModeInteractive:
		return UsageModeRebaseResult{RunID: id, Status: RebaseAppliesNextLaunch, FromModel: from,
			Reason: "an interactive session keeps its model; switching it would force a full re-read and typing /model would be saved as a default. The mode applies on its next launch"}
	case domain.ExecutionModeCodecPipe:
	default:
		return unableRebase(id, from, "not a managed run: "+string(run.ExecutionMode.Normalized())+" runs are not started by Agent Manager")
	}
	switch run.Status {
	case domain.RunStatusParked, domain.RunStatusComplete, domain.RunStatusNeedsReview:
	case domain.RunStatusRunning, domain.RunStatusStarting, domain.RunStatusPending:
		return unableRebase(id, from, "the run is mid-turn; its model is fixed for this turn. Ask again once it parks or finishes")
	default:
		return unableRebase(id, from, "the run has ended ("+string(run.Status)+") and cannot be resumed")
	}
	if strings.TrimSpace(run.SessionID) == "" {
		return unableRebase(id, from, "the run has no retained session to resume")
	}
	snapshot := cfg.PolicySnapshot
	if snapshot == nil || snapshot.SelectedIndex < 0 || snapshot.SelectedIndex >= len(snapshot.Candidates) {
		return unableRebase(id, from, "the run has no recorded model selection")
	}
	candidate := snapshot.Candidates[snapshot.SelectedIndex]
	candidate.Model = from
	runnerName := string(cfg.RunnerType)
	settings, has := mode.SettingsFor(runnerName)
	if !has {
		return unableRebase(id, from, "the mode has no settings for "+runnerName)
	}
	if mode.Scope.Runner != "" && usagemode.NormalizeRunner(mode.Scope.Runner) != usagemode.NormalizeRunner(runnerName) {
		return unableRebase(id, from, "the mode is scoped to "+mode.Scope.Runner)
	}
	// A new run is only lowered when the mode's role and profile match it, so a
	// rebase must apply the same scope, not every run it is pointed at.
	if want := mode.Scope.Role; want != "" && candidate.ResourceRole != want && cfg.RoleRef != want {
		return unableRebase(id, from, "the mode is scoped to role "+want+", not this run's role")
	}
	if want := mode.Scope.Profile; want != "" {
		have, known := o.runProfileKey(ctx, run)
		if !known {
			return unableRebase(id, from, "the mode is scoped to profile "+want+" and this run's profile is not known")
		}
		if have != want {
			return unableRebase(id, from, "the mode is scoped to profile "+want+", not this run's profile")
		}
	}
	if why := comparisonArm(run, snapshot, candidate); why != "" {
		return unableRebase(id, from, why)
	}
	if explicitRunModel(cfg, snapshot) {
		return unableRebase(id, from, "the run was created with an explicit model, which a usage mode never overrides")
	}
	for _, role := range []string{candidate.ResourceRole, cfg.RoleRef} {
		if modeExempts(mode, role) {
			return unableRebase(id, from, "role "+role+" is exempt from usage modes")
		}
	}
	target := mappedModel(settings, candidate)
	if target == "" {
		return unableRebase(id, from, "the mode does not lower "+from)
	}
	if why := o.usageModeTargetRefusal(ctx, cfg, candidate, target); why != "" {
		return unableRebase(id, from, "the lower model is refused: "+why)
	}
	var contextTokens int64
	if run.Summary != nil {
		contextTokens = int64(run.Summary.ContextTokens)
	}
	result := UsageModeRebaseResult{RunID: id, Status: RebaseSwitchedAtNextResume, FromModel: from, ToModel: target, EstimatedRereadTokens: contextTokens,
		Reason: "applies at the run's next resume; that resume re-reads the whole conversation on the new model"}
	if dryRun {
		return result
	}
	updater, ok := o.runs.(repository.RunConfigUpdater)
	if !ok {
		return unableRebase(id, from, "the run store cannot record a pending change")
	}
	updated := *cfg
	updated.PendingModelRebase = &domain.ModelRebase{ModeID: mode.ID, Reason: "usage_mode_rebase:" + mode.ID, FromModel: from, ToModel: target, ContextTokens: contextTokens, RequestedAt: o.now()}
	saved, err := updater.UpdateResolvedConfig(ctx, id, &updated, run.Status)
	if err != nil {
		return unableRebase(id, from, "recording the change failed: "+err.Error())
	}
	if !saved {
		return unableRebase(id, from, "the run changed state while recording; ask again")
	}
	return result
}

// applyPendingModelRebase runs at the start of a resume (continue or wake). It
// moves the retained model to the pending target, clears the pending request,
// appends to ModelRebases and notes usage_mode_rebase:<id> in the selection
// reason. PolicySnapshot candidates are not rewritten, so the creation-time
// selection stays truthful. A target that is no longer allowed leaves the run on
// its current model and says so in the run log; a rebase never fails a resume.
func (o *Orchestrator) applyPendingModelRebase(ctx context.Context, run *domain.Run) {
	if run == nil || run.ResolvedConfig == nil || run.ResolvedConfig.PendingModelRebase == nil {
		return
	}
	cfg := run.ResolvedConfig
	pending := *cfg.PendingModelRebase
	cfg.PendingModelRebase = nil
	why := ""
	if _, active := activeUsageModeByID(pending.ModeID, o.now()); !active {
		why = "usage mode " + pending.ModeID + " is no longer active"
	} else if snapshot := cfg.PolicySnapshot; snapshot != nil && snapshot.SelectedIndex >= 0 && snapshot.SelectedIndex < len(snapshot.Candidates) {
		candidate := snapshot.Candidates[snapshot.SelectedIndex]
		candidate.Model = cfg.Model
		why = o.usageModeTargetRefusal(ctx, cfg, candidate, pending.ToModel)
	}
	message := ""
	if why != "" {
		message = fmt.Sprintf("usage mode rebase to %s skipped: %s; the run keeps %s", pending.ToModel, why, cfg.Model)
	} else {
		at := o.now()
		pending.AppliedAt = &at
		pending.FromModel = cfg.Model
		cfg.Model = pending.ToModel
		cfg.ModelRebases = append(cfg.ModelRebases, pending)
		if cfg.PolicySnapshot != nil {
			appendSelectionReason(cfg.PolicySnapshot, pending.Reason)
		}
		message = fmt.Sprintf("%s: model moved from %s to %s at resume (about %d tokens of context re-read)", pending.Reason, pending.FromModel, pending.ToModel, pending.ContextTokens)
	}
	if updater, ok := o.runs.(repository.RunConfigUpdater); ok {
		if _, err := updater.UpdateResolvedConfig(ctx, run.ID, cfg, run.Status); err != nil {
			message += " (not persisted: " + err.Error() + ")"
		}
	}
	o.appendRunLogEvent(ctx, run.ID, "info", message)
}

// runProfileKey is the profile key of the run's agent profile.
func (o *Orchestrator) runProfileKey(ctx context.Context, run *domain.Run) (string, bool) {
	if run.AgentProfileID == nil || o.profiles == nil {
		return "", false
	}
	profile, err := o.profiles.Get(ctx, *run.AgentProfileID)
	if err != nil || profile == nil {
		return "", false
	}
	return profile.ProfileKey, true
}

// comparisonArm explains why a run in a Learn comparison keeps its model: a
// switch would change what its assigned arm measures.
func comparisonArm(run *domain.Run, snapshot *domain.ExecutionPolicySnapshot, candidate domain.ExecutionCandidate) string {
	arm := firstNonEmpty(run.CanaryArm, snapshot.CanaryArm, candidate.CanaryArm)
	switch {
	case arm != "":
		return "the run is in a model comparison as the " + arm + " arm; switching it would corrupt the comparison"
	case candidate.ChallengerModel != "":
		return "the run belongs to a model comparison; switching it would corrupt the comparison"
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// explicitRunModel reports a run whose model came from the caller. The
// creation-time preflight keeps the model the role resolved to; a different
// current model that no usage mode or rebase explains was the caller's choice.
func explicitRunModel(cfg *domain.RunConfig, snapshot *domain.ExecutionPolicySnapshot) bool {
	if len(cfg.ModelRebases) > 0 || strings.Contains(snapshot.SelectionReason, "usage_mode:") {
		return false
	}
	for _, check := range snapshot.Explanation.Preflight {
		if check.Index == snapshot.SelectedIndex {
			return check.Candidate.Model != "" && !strings.EqualFold(check.Candidate.Model, cfg.Model)
		}
	}
	return false
}
