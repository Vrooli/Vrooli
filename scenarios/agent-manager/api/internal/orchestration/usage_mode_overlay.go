// Responsibility: lower what a new run executes on while an operator usage mode is active.
package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/vrooli/cli-core/usagemode"
)

// WithUsageModes lets run creation consult the operator's usage mode file
// (docs/configuration/usage-modes.md). Without it creation never reads the file.
func WithUsageModes() Option { return func(o *Orchestrator) { o.usageModes = true } }

// applyUsageMode is the overlay between role resolution and an explicit model
// override. It lowers the selected candidate's model (and effort) when an
// active mode maps it, records usage_mode:<id> in the selection reason, and
// never fails run creation: any problem leaves the resolved candidate in place
// with usage_mode_skipped:<why>. Only new runs reach it; resume and continue
// keep their immutable snapshot.
func (o *Orchestrator) applyUsageMode(ctx context.Context, cfg *domain.RunConfig, profile *domain.AgentProfile, req CreateRunRequest) {
	if !o.usageModes || !req.usageModeEligible || cfg == nil || cfg.PolicySnapshot == nil {
		return
	}
	snapshot := cfg.PolicySnapshot
	if snapshot.SelectedIndex < 0 || snapshot.SelectedIndex >= len(snapshot.Candidates) {
		return
	}
	candidate := snapshot.Candidates[snapshot.SelectedIndex]
	profileKey := ""
	if profile != nil {
		profileKey = profile.ProfileKey
	}
	now := time.Now()
	mode, ok := resolveUsageMode(string(candidate.RunnerType), candidate.ResourceRole, snapshot.RoleRef, profileKey, now)
	if !ok {
		o.applyLearnMode(ctx, cfg, candidate, profileKey, req, now)
		return
	}
	settings, _ := mode.SettingsFor(string(candidate.RunnerType))
	target := ""
	if req.Model == nil { // an explicit model always wins
		target = mappedModel(settings, candidate)
	}
	effort := loweredEffort(settings.Effort, cfg.Effort, req.Effort == nil)
	limit, limitNote := modeContextLimit(candidate.RunnerType, settings.ContextLimitTokens)
	if target == "" && effort == "" && limit == 0 {
		return
	}
	if target != "" {
		if why := o.usageModeTargetRefusal(ctx, cfg, candidate, target); why != "" {
			appendSelectionReason(snapshot, "usage_mode_skipped:"+why)
			return
		}
		candidate.SelectionType = domain.ModelSelectionTypeModel
		candidate.Model, candidate.CanonicalModel = target, ""
		snapshot.Candidates[snapshot.SelectedIndex] = candidate
		snapshot.SelectedCandidate = candidate
		cfg.Model = target
	}
	if effort != "" {
		cfg.Effort = domain.Effort(effort)
	}
	appendSelectionReason(snapshot, "usage_mode:"+mode.ID)
	if limit > 0 {
		cfg.ContextLimitTokens = limit
		appendSelectionReason(snapshot, fmt.Sprintf("usage_mode_context_limit:%d", limit))
		if limitNote != "" {
			appendSelectionReason(snapshot, limitNote)
		}
	}
}

// modeContextLimit is the auto-compact limit a managed run gets from a mode's
// settings. Claude ignores anything under 100k, so a smaller limit is raised to
// that minimum and the returned note says so for the run's selection reason.
// Other runners have no context-limit control here.
func modeContextLimit(runnerType domain.RunnerType, requested int) (int, string) {
	switch usagemode.NormalizeRunner(string(runnerType)) {
	case "claude":
		limit, clamped := usagemode.ClaudeContextLimit(requested)
		if clamped {
			return limit, fmt.Sprintf("usage_mode_context_limit_raised:%d_to_claude_minimum_%d", requested, limit)
		}
		return limit, ""
	case "codex":
		if requested > 0 {
			return requested, ""
		}
	}
	return 0, ""
}

// usageModeTargetRefusal explains why the lowered model must not replace the
// candidate: its role pins an exact effort, the target is excluded, the
// candidate is billed per use, or the runner cannot serve the target.
func (o *Orchestrator) usageModeTargetRefusal(ctx context.Context, cfg *domain.RunConfig, candidate domain.ExecutionCandidate, target string) string {
	switch {
	case candidate.ResourceRole == "code.delivery" && domain.DeliveryEffortEnforced():
		return "delivery_effort_enforced"
	case candidate.Billing.EffectiveBasis() == domain.ChargeBasisMetered:
		return "metered_billing"
	case domain.IsModelExcluded(target, "", candidate.ExcludedModels):
		return "target_excluded"
	}
	if o.runners == nil {
		return ""
	}
	resolved, err := o.runners.Get(candidate.RunnerType)
	if err != nil || resolved == nil {
		return "runner_unavailable"
	}
	if err := resolved.ProbeModel(ctx, target); err != nil {
		return "probe_failed"
	}
	return ""
}

func appendSelectionReason(snapshot *domain.ExecutionPolicySnapshot, reason string) {
	if snapshot.SelectionReason == "" {
		snapshot.SelectionReason = reason
		return
	}
	snapshot.SelectionReason += ";" + reason
}

// mappedModel is the lower model the mode maps the candidate's model to.
func mappedModel(settings usagemode.RunnerSettings, candidate domain.ExecutionCandidate) string {
	canonical := candidate.CanonicalModel
	if _, id, ok := strings.Cut(canonical, "/"); ok {
		canonical = id
	}
	for _, key := range []string{candidate.Model, candidate.CanonicalModel, canonical} {
		if key == "" {
			continue
		}
		if to := settings.ModelMap[key]; to != "" && to != candidate.Model {
			return to
		}
	}
	return ""
}

var usageEffortRank = map[string]int{string(domain.EffortLow): 1, string(domain.EffortMedium): 2, string(domain.EffortHigh): 3, string(domain.EffortXHigh): 4, string(domain.EffortMax): 5}

// loweredEffort returns the mode's effort when it is lower than the run's
// current one (or the run has none) and the caller did not choose an effort.
func loweredEffort(modeEffort string, current domain.Effort, allowed bool) string {
	rank, known := usageEffortRank[modeEffort]
	if !allowed || !known {
		return ""
	}
	if currentRank := usageEffortRank[string(current)]; currentRank != 0 && rank >= currentRank {
		return ""
	}
	return modeEffort
}

// resolveUsageMode finds the active mode for a candidate. The resource role is
// what the shim and resources know; the portable role is what operators scope
// by, so both are tried and the narrower scope wins. A role exempt under either
// name keeps its model.
func resolveUsageMode(runner, resourceRole, roleRef, profile string, now time.Time) (usagemode.Mode, bool) {
	var best usagemode.Mode
	found := false
	for _, role := range []string{resourceRole, roleRef} {
		mode, ok := usagemode.Resolve(usagemode.Query{Runner: runner, Role: role, Profile: profile, Now: now})
		if !ok {
			continue
		}
		if !found || scopeRank(mode.Scope) > scopeRank(best.Scope) {
			best, found = mode, true
		}
	}
	if !found {
		return usagemode.Mode{}, false
	}
	for _, role := range []string{resourceRole, roleRef} {
		if modeExempts(best, role) {
			return usagemode.Mode{}, false
		}
	}
	return best, true
}

func scopeRank(scope usagemode.Scope) int {
	switch {
	case scope.Profile != "":
		return 4
	case scope.Role != "":
		return 3
	case scope.Runner != "":
		return 2
	}
	return 1
}

func modeExempts(mode usagemode.Mode, role string) bool {
	if role == "" || mode.Scope.Role == role {
		return false
	}
	for _, list := range [][]string{mode.ExemptRoles, usagemode.DefaultExemptRoles} {
		for _, exempt := range list {
			if exempt == role {
				return true
			}
		}
	}
	return false
}

// applyLearnMode joins a new managed run to a Learn comparison when an active,
// unpaused learn mode covers its role. It does not choose the arm and it does
// not change the model: it declares the challenger and the sample rate on the
// selected candidate, and the existing SelectCanaryArm assigns and records
// canary_arm once per run. Nothing here can fail run creation. Exempt roles,
// an explicit model, a role that already declares its own challenger and a
// save mode that applies (resolved before this) keep the run out.
func (o *Orchestrator) applyLearnMode(ctx context.Context, cfg *domain.RunConfig, candidate domain.ExecutionCandidate, profile string, req CreateRunRequest, now time.Time) {
	snapshot := cfg.PolicySnapshot
	if req.Model != nil || candidate.ChallengerModel != "" {
		return
	}
	var mode usagemode.Mode
	found := false
	for _, role := range []string{candidate.ResourceRole, snapshot.RoleRef} {
		if role == "" {
			continue
		}
		if m, ok := usagemode.ResolveLearn(usagemode.Query{Runner: string(candidate.RunnerType), Role: role, Profile: profile, Now: now}); ok {
			mode, found = m, true
			break
		}
	}
	if !found {
		return
	}
	for _, role := range []string{candidate.ResourceRole, snapshot.RoleRef} {
		if mode.RoleExempt(role) {
			return
		}
	}
	settings, _ := mode.ChallengerSettings(string(candidate.RunnerType))
	target := mappedModel(settings, candidate)
	if target == "" {
		appendSelectionReason(snapshot, "usage_mode_learn_skipped:no_mapping")
		return
	}
	if why := o.usageModeTargetRefusal(ctx, cfg, candidate, target); why != "" {
		appendSelectionReason(snapshot, "usage_mode_learn_skipped:"+why)
		return
	}
	candidate.ChallengerModel, candidate.ChallengerSampleRate = target, mode.Learn.Split()
	snapshot.Candidates[snapshot.SelectedIndex] = candidate
	snapshot.SelectedCandidate = candidate
	appendSelectionReason(snapshot, "usage_mode_learn:"+mode.ID)
}
