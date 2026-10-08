package orchestration

import (
	"context"

	"agent-manager/internal/domain"
)

// The current overlay checks the same retained resource identity. It cannot
// silently select a different role, model or effort to make continuation pass.
func (o *Orchestrator) validateCurrentDeliveryEffort(ctx context.Context, cfg *domain.RunConfig, candidate domain.ExecutionCandidate) error {
	if !domain.DeliveryEffortEnforced() {
		return nil
	}
	if err := domain.ValidateCandidateResourceEffort(cfg, candidate); err != nil {
		return err
	}
	if candidate.ResourceRole != "code.delivery" {
		return nil
	}
	if o.roleResolver == nil {
		return domain.NewValidationError("effort", "current delivery resource owner is unavailable")
	}
	current, err := o.roleResolver.Resolve(ctx, candidate.RunnerType, candidate.ResourceRole)
	if err != nil || current.Runner != candidate.RunnerType || current.Role != candidate.ResourceRole || current.Model != candidate.Model || current.Effort == "" || current.Effort != cfg.Effort {
		return domain.NewValidationError("effort", "current delivery resource effort does not match retained execution")
	}
	return nil
}
