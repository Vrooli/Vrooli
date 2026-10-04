// Responsibility: validate current runner and model policy before run creation.
package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"
	"context"
	"strings"
)

// validateExecutionModel is the retained-run admission fence. It evaluates
// only the immutable policy evidence copied into a run; it never reloads or
// rewrites historical configuration. This is used before continuation,
// recovery, and replacement admission can touch a session, sandbox, or
// executor.
func validateExecutionModel(cfg *domain.RunConfig) error {
	if cfg == nil || cfg.PolicySnapshot == nil || strings.TrimSpace(cfg.Model) == "" {
		return nil
	}
	candidate := cfg.PolicySnapshot.SelectedCandidate
	if candidate.RunnerType != cfg.RunnerType && cfg.PolicySnapshot.SelectedIndex >= 0 && cfg.PolicySnapshot.SelectedIndex < len(cfg.PolicySnapshot.Candidates) {
		candidate = cfg.PolicySnapshot.Candidates[cfg.PolicySnapshot.SelectedIndex]
	}
	canonical := ""
	if strings.EqualFold(strings.TrimSpace(candidate.Model), strings.TrimSpace(cfg.Model)) {
		canonical = candidate.CanonicalModel
	}
	if domain.IsModelExcluded(cfg.Model, canonical, candidate.ExcludedModels) {
		return domain.NewValidationErrorWithHint("model", "retained model is excluded by resource policy", "stop the old run or use an allowed model in a new run")
	}
	return nil
}

// currentModelExclusions reads the current resource-owned deny overlay for a
// retained config. It is deliberately separate from the immutable run
// snapshot: policy tightening must fence future work without rewriting the
// historical execution contract or receipts.
func currentModelExclusions(ctx context.Context, cfg *domain.RunConfig, state *rolepolicy.State, resolver rolepolicy.Resolver) (map[domain.RunnerType][]string, error) {
	if cfg == nil {
		return nil, domain.NewValidationErrorWithHint("runConfig", "retained run configuration is unavailable", "stop the old run and create a new run with a complete execution policy")
	}
	if state == nil || resolver == nil {
		return nil, domain.NewValidationErrorWithHint("rolePolicyCatalog", "current resource model policy is unavailable", "restore the required resource policy before continuing this run")
	}
	role := strings.TrimSpace(cfg.RoleRef)
	if role == "" && cfg.PolicySnapshot != nil {
		role = strings.TrimSpace(cfg.PolicySnapshot.SelectedCandidate.ResourceRole)
	}
	if role == "" {
		if active := state.Active(); active != nil && active.Catalog() != nil {
			role = strings.TrimSpace(active.Catalog().DefaultRole)
		}
		if role == "" {
			return nil, domain.NewValidationErrorWithHint("rolePolicyCatalog", "retained run has no resource policy role", "stop the old run and create a new run with a portable role")
		}
	}
	resolution, err := state.ResolvePreferred(ctx, resolver, role, string(cfg.RunnerType))
	if err != nil {
		return nil, domain.NewValidationErrorWithHint("rolePolicyCatalog", "current resource model policy could not be read", err.Error())
	}
	exclusions := make(map[domain.RunnerType][]string)
	observed := make(map[string]bool)
	addExclusions := func(runnerType domain.RunnerType, values []string) {
		// Keep an explicit empty entry for an observed runner. The map is also
		// admission evidence: a missing key means that this fallback's current
		// policy was never read, whereas an empty slice means it was read and
		// currently permits every model.
		if _, exists := exclusions[runnerType]; !exists {
			exclusions[runnerType] = []string{}
		}
		seen := make(map[string]bool, len(exclusions[runnerType]))
		for _, value := range exclusions[runnerType] {
			seen[strings.ToLower(strings.TrimSpace(value))] = true
		}
		for _, value := range values {
			key := strings.ToLower(strings.TrimSpace(value))
			if key != "" && !seen[key] {
				exclusions[runnerType] = append(exclusions[runnerType], strings.TrimSpace(value))
				seen[key] = true
			}
		}
	}
	found := false
	for _, candidate := range resolution.Candidates {
		if !candidate.Available {
			// Keep the selected/current candidate usable when its own policy was
			// read successfully, but fence this candidate from fallback execution
			// until its resource policy becomes readable again. The immutable run
			// snapshot remains historical evidence; this marker is only a current
			// admission overlay and is never persisted into that snapshot.
			addExclusions(candidate.Runner, []string{domain.ModelPolicyUnavailable})
			continue
		}
		if candidate.Runner == cfg.RunnerType {
			found = true
		}
		observed[string(candidate.Runner)+"\x00"+candidate.ResourceRole] = true
		addExclusions(candidate.Runner, candidate.ExcludedModels)
	}
	// A retained snapshot can contain a runner/resource role that a later AM
	// catalog revision removed. Resolve those historical fallback identities
	// directly so an omitted current catalog entry cannot become an un-fenced
	// executable fallback.
	if cfg.PolicySnapshot != nil {
		for _, historical := range cfg.PolicySnapshot.Candidates {
			if !historical.RunnerType.IsValid() {
				return nil, domain.NewValidationErrorWithHint("rolePolicyCatalog", "retained fallback runner identity is invalid", "stop the old run and create a new run with a current execution policy")
			}
			historicalRole := strings.TrimSpace(historical.ResourceRole)
			if historicalRole == "" {
				historicalRole = role
			}
			if historicalRole == "" {
				return nil, domain.NewValidationErrorWithHint("rolePolicyCatalog", "retained fallback resource role is unavailable", "stop the old run and create a new run with a current execution policy")
			}
			identity := string(historical.RunnerType) + "\x00" + historicalRole
			if observed[identity] {
				continue
			}
			candidate, resolveErr := resolver.Resolve(ctx, historical.RunnerType, historicalRole)
			if resolveErr != nil {
				addExclusions(historical.RunnerType, []string{domain.ModelPolicyUnavailable})
				observed[identity] = true
				continue
			}
			observed[identity] = true
			addExclusions(historical.RunnerType, candidate.ExcludedModels)
		}
	}
	if strings.TrimSpace(string(cfg.RunnerType)) != "" && !found {
		// Historical runs can retain a runner that a later AM catalog revision no
		// longer lists. The resource policy is still authoritative for its own
		// vocabulary, so resolve that runner directly with the retained role.
		candidate, resolveErr := resolver.Resolve(ctx, cfg.RunnerType, role)
		if resolveErr != nil {
			addExclusions(cfg.RunnerType, []string{domain.ModelPolicyUnavailable})
			return exclusions, nil
		}
		addExclusions(cfg.RunnerType, candidate.ExcludedModels)
	}
	return exclusions, nil
}

func (o *Orchestrator) validateCurrentExecutionModel(ctx context.Context, cfg *domain.RunConfig) error {
	exclusions, err := currentModelExclusions(ctx, cfg, o.rolePolicy, o.roleResolver)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(cfg.RunnerType)) == "" {
		return domain.NewValidationErrorWithHint("runnerType", "retained run has no effective runner", "stop the old run and create a new run with a complete execution policy")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		for _, denied := range exclusions {
			if len(denied) > 0 {
				return domain.NewValidationErrorWithHint("model", "retained run has an unknown native model and current resource policy contains exclusions", "stop the old run and create a new run with an explicit allowed model")
			}
		}
		return nil
	}
	var candidate domain.ExecutionCandidate
	if cfg.PolicySnapshot != nil {
		candidate = cfg.PolicySnapshot.SelectedCandidate
	}
	canonical := ""
	if strings.EqualFold(strings.TrimSpace(candidate.Model), strings.TrimSpace(cfg.Model)) {
		canonical = candidate.CanonicalModel
	}
	if domain.IsModelExcluded(cfg.Model, canonical, exclusions[cfg.RunnerType]) {
		return domain.NewValidationErrorWithHint("model", "model is excluded by current resource policy", "stop the old run or use an allowed model in a new run")
	}
	return nil
}

// currentCandidateAllowed refreshes the resource-owned deny overlay at the
// final fallback launch boundary. It does not mutate the retained snapshot.
func (o *Orchestrator) currentCandidateAllowed(ctx context.Context, cfg *domain.RunConfig, candidate domain.ExecutionCandidate, model string) (bool, error) {
	exclusions, err := currentModelExclusions(ctx, cfg, o.rolePolicy, o.roleResolver)
	if err != nil {
		return false, err
	}
	canonical := ""
	if strings.EqualFold(strings.TrimSpace(candidate.Model), strings.TrimSpace(model)) {
		canonical = candidate.CanonicalModel
	}
	if strings.TrimSpace(model) == "" {
		return len(exclusions[candidate.RunnerType]) == 0, nil
	}
	return !domain.IsModelExcluded(model, canonical, exclusions[candidate.RunnerType]), nil
}
