package orchestrator

import (
	"context"
	"fmt"
	"storage-manager/internal/cleanup"
	"time"
)

// ConfigureTemporaryRetention changes only tmp's standing authority. Enabling
// requires a positive, bounded age and never grants authority to cache/model,
// database, owner-data or conditional providers.
func (s *Service) ConfigureTemporaryRetention(ctx context.Context, automatic bool, minAge time.Duration) (Policy, error) {
	if minAge < 0 || (automatic && minAge < 24*time.Hour) {
		return Policy{}, fmt.Errorf("automatic temporary retention requires an age of at least 24 hours")
	}
	current, err := s.CurrentPolicy(ctx)
	if err != nil {
		return Policy{}, err
	}
	provider, ok := s.registry.Get("tmp")
	if !ok || provider.Metadata().SafetyTier != cleanup.SafetyTierSafe {
		return Policy{}, fmt.Errorf("safe temporary provider is unavailable")
	}
	p := current.Providers["tmp"]
	if minAge > 0 {
		p.MinAge = minAge
	}
	if automatic {
		p.Enabled = true
		p.ApprovalMode = cleanup.ApprovalModeNone
	} else {
		p.ApprovalMode = cleanup.ApprovalModeOperator
	}
	current.Providers["tmp"] = p
	current.Version = stablePolicyVersion(current.Profile, current.Providers)
	current.CreatedAt = s.now()
	if err := s.store.SavePolicy(ctx, current); err != nil {
		return Policy{}, err
	}
	_ = s.audit(ctx, AuditEvent{Type: "temporary_retention.configured", ProviderID: "tmp", Message: fmt.Sprintf("automatic=%t min_age=%s", automatic, p.MinAge)})
	return current, nil
}

// StartTemporaryRetention uses the same server-owned controller, lock, safety
// checks and receipts as pressure recovery, but selects only the temporary
// provider and respects explicit policy disable. Its target is one normal
// batch, allowing a single fully measured indivisible entry to exceed it.
func (s *Service) StartTemporaryRetention(ctx context.Context) (RecoveryRun, error) {
	if !s.AutonomousApplyEnabled() {
		return RecoveryRun{}, nil
	}
	policy, err := s.CurrentPolicy(ctx)
	if err != nil {
		return RecoveryRun{}, err
	}
	p := policy.Providers["tmp"]
	if !p.Enabled || p.ApprovalMode != cleanup.ApprovalModeNone {
		return RecoveryRun{}, nil
	}
	if p.MinAge <= 0 {
		return RecoveryRun{}, fmt.Errorf("scheduled temporary cleanup requires a positive retention age")
	}
	return s.StartRecovery(ctx, temporaryRetentionTrigger, "/", 0, 0, 0, false)
}
