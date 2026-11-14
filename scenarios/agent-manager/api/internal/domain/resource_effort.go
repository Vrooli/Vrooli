package domain

// ValidateResourceEffort preserves the requested effort. Delivery effort is an
// exact resource-owner constraint, never an implicit downgrade or fallback.
// Other roles retain their previous effort/default semantics. Both checks are
// inert unless DeliveryEffortEnforced.
func ValidateResourceEffort(cfg *RunConfig) error {
	if cfg == nil || !DeliveryEffortEnforced() {
		return nil
	}
	if cfg.PolicySnapshot == nil {
		if cfg.RoleRef == "code.economy.delivery" {
			return NewValidationError("effort", "delivery resource effort evidence is unavailable")
		}
		return nil
	}
	candidate := cfg.PolicySnapshot.SelectedCandidate
	if candidate.RunnerType != cfg.RunnerType && cfg.PolicySnapshot.SelectedIndex >= 0 && cfg.PolicySnapshot.SelectedIndex < len(cfg.PolicySnapshot.Candidates) {
		candidate = cfg.PolicySnapshot.Candidates[cfg.PolicySnapshot.SelectedIndex]
	}
	return ValidateCandidateResourceEffort(cfg, candidate)
}

func ValidateCandidateResourceEffort(cfg *RunConfig, candidate ExecutionCandidate) error {
	if cfg == nil {
		return NewValidationError("runConfig", "run configuration is unavailable")
	}
	if !DeliveryEffortEnforced() {
		return nil
	}
	if candidate.ResourceRole != "code.delivery" && cfg.RoleRef != "code.economy.delivery" {
		return nil
	}
	if candidate.ResourceRole != "code.delivery" || candidate.Model == "" || cfg.Model != candidate.Model || candidate.DeclaredEffort == "" || !candidate.DeclaredEffort.IsValid() || cfg.Effort != candidate.DeclaredEffort {
		return NewValidationError("effort", "requested delivery effort does not match exact resource-owner evidence")
	}
	return nil
}
