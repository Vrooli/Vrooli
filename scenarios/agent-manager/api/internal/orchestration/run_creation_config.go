// Responsibility: resolve execution configuration and preserve owner-declared ceilings.
package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/structuredresult"
	"context"
	"fmt"
	"strings"
	"time"
)

// resolveRunConfig resolves the run configuration from profile and/or inline config.
// Returns the resolved config and the profile (if loaded, may be nil for pure inline config).
func (o *Orchestrator) resolveRunConfig(ctx context.Context, req CreateRunRequest) (*domain.RunConfig, *domain.AgentProfile, error) {
	cfg := domain.DefaultRunConfig()
	var profile *domain.AgentProfile
	workTimeoutCeiling, maxTurnsCeiling, hasGlobalCeilings := o.runExecutionCeilings()
	if hasGlobalCeilings {
		// The persisted settings are both the defaults visible in resolved_config
		// and the global ceilings for profile/inline requests.
		cfg.Timeout = workTimeoutCeiling
		cfg.MaxTurns = maxTurnsCeiling
	}

	if req.profileAdmission != nil {
		profile = req.profileAdmission.profile
	}
	// Load profile if provided
	if req.AgentProfileID != nil {
		var err error
		profile, err = o.GetProfile(ctx, *req.AgentProfileID)
		if err != nil {
			return nil, nil, err
		}
	}

	// Resolve profile by key if provided
	if req.ProfileRef != nil && req.profileAdmission == nil {
		if req.ProfileRef.Defaults == nil && !req.ProfileRef.UpdateExisting {
			key := strings.TrimSpace(req.ProfileRef.ProfileKey)
			if key == "" {
				return nil, nil, domain.NewValidationErrorWithHint("profileRef.profileKey", "field is required",
					"Provide a stable profile key or inline profile defaults")
			}
			var err error
			profile, err = o.profiles.GetByKey(ctx, key)
			if err != nil {
				return nil, nil, err
			}
			if profile == nil {
				return nil, nil, domain.NewValidationErrorWithHint("profileRef.profileKey", "profile not found",
					"Start the owning scenario so agent-manager can reconcile its manifest-declared profiles")
			}
		} else {
			result, err := o.EnsureProfile(ctx, EnsureProfileRequest{
				ProfileKey:     req.ProfileRef.ProfileKey,
				Defaults:       req.ProfileRef.Defaults,
				UpdateExisting: req.ProfileRef.UpdateExisting,
			})
			if err != nil {
				return nil, nil, err
			}
			profile = result.Profile
		}
	}

	if profile != nil {
		if hasGlobalCeilings {
			if err := validateRunDurationCeilings("profile", profile.Timeout, profile.MaxTurns, workTimeoutCeiling, maxTurnsCeiling); err != nil {
				return nil, nil, err
			}
		}
		cfg.ApplyProfile(profile)
		// Zero means the profile did not request a value. Persist the effective
		// global default so run reads and the executor enforce the same numbers.
		if hasGlobalCeilings && cfg.Timeout <= 0 {
			cfg.Timeout = workTimeoutCeiling
		}
		if hasGlobalCeilings && cfg.MaxTurns <= 0 {
			cfg.MaxTurns = maxTurnsCeiling
		}
	}

	// Apply inline overrides
	if req.RoleRef != nil {
		cfg.RoleRef = strings.TrimSpace(*req.RoleRef)
	}
	cfg.PreferredRunner = strings.TrimSpace(req.PreferredRunner)
	if req.MaxTurns != nil {
		cfg.MaxTurns = *req.MaxTurns
	}
	if req.MaxToolCalls != nil {
		if *req.MaxToolCalls < 0 {
			return nil, nil, domain.NewValidationError("maxToolCalls", "must be zero or positive")
		}
		cfg.MaxToolCalls = *req.MaxToolCalls
	}
	if req.Timeout != nil {
		cfg.Timeout = *req.Timeout
	}
	if hasGlobalCeilings {
		if err := validateRunDurationCeilings("inline", cfg.Timeout, cfg.MaxTurns, workTimeoutCeiling, maxTurnsCeiling); err != nil {
			return nil, nil, err
		}
	}
	if strings.TrimSpace(req.Until) != "" {
		if len(req.Until) > 2048 {
			return nil, nil, domain.NewValidationError("until", "completion test must be at most 2048 characters")
		}
		cfg.Until = strings.TrimSpace(req.Until)
	}
	if req.Effort != nil {
		cfg.Effort = *req.Effort
	}
	if req.AllowedTools != nil {
		cfg.AllowedTools = req.AllowedTools
	}
	if req.DeniedTools != nil {
		cfg.DeniedTools = req.DeniedTools
	}
	if req.SkipPermissionPrompt != nil {
		cfg.SkipPermissionPrompt = *req.SkipPermissionPrompt
	}
	// Feature flag overrides
	if req.EnableBrowser != nil {
		cfg.Features.EnableBrowser = *req.EnableBrowser
	}
	// Extra flags overrides (replace per runner type)
	if req.ExtraFlags != nil {
		if cfg.ExtraFlags == nil {
			cfg.ExtraFlags = make(domain.RunnerExtraFlags)
		}
		for rt, flags := range req.ExtraFlags {
			cfg.ExtraFlags[rt] = append([]string(nil), flags...)
		}
	}
	if req.NetworkAccess != nil {
		cfg.NetworkAccess = *req.NetworkAccess
	}
	if req.AllowedPaths != nil {
		cfg.AllowedPaths = req.AllowedPaths
	}
	if req.DeniedPaths != nil {
		cfg.DeniedPaths = req.DeniedPaths
	}
	if req.AllowedEffects != nil {
		if err := validateEffectGrant(req.AllowedEffects); err != nil {
			return nil, nil, domain.NewValidationError("allowedEffects", err.Error())
		}
		cfg.AllowedEffects = append([]string(nil), req.AllowedEffects...)
		cfg.RequireEffectContainment = req.RequireEffectContainment || len(req.AllowedEffects) > 0
		for _, effect := range req.AllowedEffects {
			if paths := effectParameter(effect, "paths"); paths != "" && req.AllowedPaths == nil {
				cfg.AllowedPaths = mergeUnique(cfg.AllowedPaths, []string{paths})
			}
		}
	}
	if req.ResultSpec != nil {
		normalized, err := structuredresult.NormalizeSpec(req.ResultSpec)
		if err != nil {
			return nil, nil, domain.NewValidationErrorWithHint("resultSpec", err.Error(),
				"Use result-spec/v1 with the documented bounded JSON Schema subset")
		}
		cfg.ResultSpec = normalized
	}
	// Validate the resolved config
	if strings.TrimSpace(cfg.RoleRef) == "" {
		return nil, nil, domain.NewValidationErrorWithHint("roleRef", "field is required",
			"Select a portable role from the active role-policy catalog")
	}
	if !cfg.Effort.IsValid() {
		return nil, nil, domain.NewValidationErrorWithHint("effort", "invalid effort", "valid values: low, medium, high, xhigh, max")
	}
	if err := domain.ValidateCanonicalToolList("allowedTools", cfg.AllowedTools); err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateCanonicalToolList("deniedTools", cfg.DeniedTools); err != nil {
		return nil, nil, err
	}
	if err := o.resolveExecutionPolicy(ctx, cfg); err != nil {
		return nil, nil, err
	}
	if err := o.applyModelOverride(ctx, cfg, req.Model); err != nil {
		return nil, nil, err
	}
	if err := o.validateToolRestriction(cfg); err != nil {
		return nil, nil, err
	}

	// Validate extra flags against runner allowlists (delegate to seam)
	if o.flagValidator != nil {
		for rt, flags := range cfg.ExtraFlags {
			if err := o.flagValidator.ValidateFlags(rt, flags); err != nil {
				return nil, nil, err
			}
		}
	}

	return cfg, profile, nil
}

// runExecutionCeilings returns the runtime-owned work limits. Health settings
// deliberately do not participate: executor liveness and agent work duration
// are separate control surfaces.
func (o *Orchestrator) runExecutionCeilings() (time.Duration, int, bool) {
	if o.orchestrationSettings == nil {
		return 0, 0, false
	}
	settings := o.orchestrationSettings.Get()
	return time.Duration(settings.RunExecution.RunTimeoutMinutes) * time.Minute, settings.RunExecution.MaxTurns, true
}

func validateRunDurationCeilings(source string, requestedTimeout time.Duration, requestedMaxTurns int, timeoutCeiling time.Duration, maxTurnsCeiling int) error {
	if requestedTimeout > timeoutCeiling {
		return domain.NewValidationErrorWithHint(
			source+".timeout",
			fmt.Sprintf("requested timeout %ds exceeds global work-time ceiling %ds", int64(requestedTimeout/time.Second), int64(timeoutCeiling/time.Second)),
			"Lower the requested timeout or deliberately raise orchestration runExecution.runTimeoutMinutes",
		)
	}
	if requestedMaxTurns > maxTurnsCeiling {
		return domain.NewValidationErrorWithHint(
			source+".maxTurns",
			fmt.Sprintf("requested max turns %d exceeds global turn ceiling %d", requestedMaxTurns, maxTurnsCeiling),
			"Lower the requested turns or deliberately raise orchestration runExecution.maxTurns",
		)
	}
	return nil
}

// applyModelOverride applies an explicit per-run model only after role policy
// resolution. The immutable snapshot remains the execution authority, so its
// selected candidate must be updated alongside the resolved config; otherwise
// a later execution attempt could silently revert the caller's override.
func (o *Orchestrator) applyModelOverride(ctx context.Context, cfg *domain.RunConfig, requested *string) error {
	if requested == nil {
		return nil
	}
	model := strings.TrimSpace(*requested)
	if model == "" {
		return domain.NewValidationError("model", "must not be empty when supplied")
	}
	if cfg == nil || cfg.PolicySnapshot == nil {
		return domain.NewValidationError("model", "cannot override model without a resolved execution policy")
	}
	selected := cfg.PolicySnapshot.SelectedCandidate
	canonical := ""
	if strings.EqualFold(model, strings.TrimSpace(selected.Model)) {
		canonical = selected.CanonicalModel
	}
	if domain.IsModelExcluded(model, canonical, selected.ExcludedModels) {
		return domain.NewValidationErrorWithHint("model", "model is excluded by resource policy", "select an allowed model or omit the override")
	}
	if o.runners != nil {
		runner, err := o.runners.Get(cfg.RunnerType)
		if err != nil {
			return err
		}
		if runner == nil {
			return domain.NewValidationError("model", "selected runner is not registered")
		}
		if err := runner.ProbeModel(ctx, model); err != nil {
			return domain.NewValidationErrorWithHint("model", "model is not available for the selected runner", err.Error())
		}
	}
	cfg.Model = model
	snapshot := cfg.PolicySnapshot
	if snapshot.SelectedIndex < 0 || snapshot.SelectedIndex >= len(snapshot.Candidates) {
		return domain.NewValidationError("model", "resolved execution policy has an invalid selected candidate")
	}
	snapshot.Candidates[snapshot.SelectedIndex].SelectionType = domain.ModelSelectionTypeModel
	snapshot.Candidates[snapshot.SelectedIndex].Model = model
	snapshot.SelectedCandidate = snapshot.Candidates[snapshot.SelectedIndex]
	return nil
}

// validateToolRestriction makes an allowlist fail closed once policy routing
// has selected the actual runner. Advisory is explicit and intentionally does
// not pretend that an unsupported runner enforces the declaration.
func (o *Orchestrator) validateToolRestriction(cfg *domain.RunConfig) error {
	if cfg == nil || (len(cfg.AllowedTools) == 0 && len(cfg.DeniedTools) == 0) || o.runners == nil {
		return nil
	}
	selected, err := o.runners.Get(cfg.RunnerType)
	if err != nil {
		return err
	}
	if selected.Capabilities().SupportsToolRestriction || cfg.ToolRestrictionPolicy.Effective() == domain.ToolRestrictionPolicyAdvisory {
		return nil
	}
	return domain.NewValidationErrorWithCode("toolRestrictionPolicy",
		fmt.Sprintf("runner %q cannot enforce allowedTools or deniedTools", cfg.RunnerType), domain.ErrCodePolicyRunner)
}

func (o *Orchestrator) toolRestrictionIsAdvisory(cfg *domain.RunConfig) bool {
	if cfg == nil || (len(cfg.AllowedTools) == 0 && len(cfg.DeniedTools) == 0) || cfg.ToolRestrictionPolicy.Effective() != domain.ToolRestrictionPolicyAdvisory || o.runners == nil {
		return false
	}
	selected, err := o.runners.Get(cfg.RunnerType)
	return err == nil && selected != nil && !selected.Capabilities().SupportsToolRestriction
}

// resolveExecutionPolicy converts the final profile-plus-override selection
// into a run-owned immutable snapshot. A named policy is resolved once; no
// runtime decision reads mutable catalog state after this function returns.
func (o *Orchestrator) resolveExecutionPolicy(ctx context.Context, cfg *domain.RunConfig) error {
	if cfg == nil {
		return domain.NewValidationError("runConfig", "field is required")
	}
	if strings.TrimSpace(cfg.RoleRef) != "" {
		if o.rolePolicy == nil || o.roleResolver == nil {
			return domain.NewValidationError("rolePolicyCatalog", "role policy state or resource resolver is not configured")
		}
		resolution, err := o.rolePolicy.ResolvePreferred(ctx, o.roleResolver, cfg.RoleRef, cfg.PreferredRunner)
		if err != nil {
			return err
		}
		snapshot := resolution.Snapshot()
		if snapshot == nil || len(snapshot.Candidates) == 0 {
			return domain.NewValidationError("rolePolicyCatalog", "role resolution produced no candidates")
		}
		applyModelExclusions(snapshot)
		selectedIndex, preflight, err := o.selectInitialCandidate(ctx, snapshot.Candidates)
		if err != nil {
			return err
		}
		snapshot.SelectedIndex = selectedIndex
		snapshot.SelectedCandidate = snapshot.Candidates[selectedIndex]
		if preferred := strings.TrimSpace(cfg.PreferredRunner); preferred != "" && string(snapshot.SelectedCandidate.RunnerType) != preferred {
			reason := "runner_not_selected"
			for index, candidate := range snapshot.Candidates {
				if string(candidate.RunnerType) != preferred {
					continue
				}
				reason = "runner preflight failed"
				if index < len(preflight) && strings.TrimSpace(preflight[index].Reason) != "" {
					reason = preflight[index].Reason
				}
				break
			}
			snapshot.SelectionReason = "preferred_unavailable:" + reason
		}
		snapshot.Explanation.Preflight = preflight
		snapshot.Explanation.Summary = fmt.Sprintf(
			"%s; selected candidate %d (%s/%s)",
			snapshot.Explanation.Summary,
			selectedIndex,
			snapshot.SelectedCandidate.RunnerType,
			snapshot.SelectedCandidate.Model,
		)
		cfg.PolicySnapshot = snapshot
		cfg.RunnerType = snapshot.SelectedCandidate.RunnerType
		cfg.Model = snapshot.SelectedCandidate.Model
		return nil
	}
	return domain.NewValidationError("roleRef", "field is required")
}

// applyModelExclusions removes denied model values from a newly resolved
// candidate sequence before runner/model preflight. Resource policy remains
// the source of the exclusion list; this function only materializes the
// admission-safe snapshot. A candidate whose primary is denied may use its
// first permitted same-runner fallback. Candidates with no permitted model
// remain recorded as unavailable for an auditable fail-closed decision.
func applyModelExclusions(snapshot *domain.ExecutionPolicySnapshot) {
	if snapshot == nil {
		return
	}
	for index := range snapshot.Candidates {
		candidate := &snapshot.Candidates[index]
		if candidate.SelectionType == domain.ModelSelectionTypeRunnerDefault {
			continue
		}
		models := make([]string, 0, 1+len(candidate.Fallbacks))
		for modelIndex, model := range append([]string{candidate.Model}, candidate.Fallbacks...) {
			model = strings.TrimSpace(model)
			canonical := ""
			if modelIndex == 0 {
				canonical = candidate.CanonicalModel
			}
			if model == "" || domain.IsModelExcluded(model, canonical, candidate.ExcludedModels) {
				continue
			}
			models = append(models, model)
		}
		if len(models) == 0 {
			candidate.Available = false
			candidate.FailureCode = "model_excluded"
			candidate.Failure = "all models in this candidate are excluded by resource policy"
			continue
		}
		if strings.TrimSpace(candidate.Model) != models[0] {
			// The resource response only provides a canonical identity for its
			// primary. Do not carry that identity onto a fallback and accidentally
			// reapply the primary's exclusion on continuation.
			candidate.CanonicalModel = ""
		}
		candidate.Model = models[0]
		candidate.Fallbacks = append([]string(nil), models[1:]...)
	}
}

func (o *Orchestrator) selectInitialCandidate(ctx context.Context, candidates []domain.ExecutionCandidate) (int, []domain.CandidatePreflight, error) {
	if len(candidates) == 0 {
		return -1, nil, domain.NewValidationError("rolePolicyCatalog", "resolution produced no candidates")
	}
	if o.runners == nil {
		// Minimal unit orchestrators omit adapters. Production always injects
		// the registry before accepting traffic.
		return 0, nil, nil
	}

	checks := make([]domain.CandidatePreflight, 0, len(candidates))
	for index, candidate := range candidates {
		check := domain.CandidatePreflight{Index: index, Candidate: candidate}
		// Availability is resource-resolution evidence for portable roles.
		// Legacy snapshots predate that field, so their zero value must not
		// make every historical/direct candidate unavailable.
		if candidate.FailureCode == "model_excluded" || (candidate.ResourceRole != "" && !candidate.Available) {
			check.Reason = candidate.Failure
			if check.Reason == "" {
				check.Reason = candidate.FailureCode
			}
			if check.Reason == "" {
				check.Reason = "resource role is unavailable"
			}
			checks = append(checks, check)
			continue
		}
		resolvedRunner, err := o.runners.Get(candidate.RunnerType)
		if err != nil || resolvedRunner == nil {
			check.Reason = "runner is not registered"
			checks = append(checks, check)
			continue
		}
		available, message := resolvedRunner.IsAvailable(ctx)
		if !available {
			check.Reason = strings.TrimSpace(message)
			if check.Reason == "" {
				check.Reason = "runner is unavailable"
			}
			checks = append(checks, check)
			continue
		}
		switch candidate.SelectionType {
		case domain.ModelSelectionTypeModel:
			if err := resolvedRunner.ProbeModel(ctx, candidate.Model); err != nil {
				check.Reason = err.Error()
				checks = append(checks, check)
				continue
			}
		case domain.ModelSelectionTypeRunnerDefault:
			// Catalog/codec conformance already proves runner-default support.
		default:
			check.Reason = "candidate selection type is invalid"
			checks = append(checks, check)
			continue
		}
		check.Available = true
		checks = append(checks, check)
		return index, checks, nil
	}

	reasons := make([]string, 0, len(checks))
	for _, check := range checks {
		reasons = append(reasons, fmt.Sprintf("candidate %d %s/%s: %s", check.Index, check.Candidate.RunnerType, check.Candidate.SelectionType, check.Reason))
	}
	return -1, checks, domain.NewValidationErrorWithHint(
		"rolePolicyCatalog",
		"no policy candidate passed runner/model preflight",
		strings.Join(reasons, "; "),
	)
}
