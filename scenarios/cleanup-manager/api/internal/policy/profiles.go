package policy

import (
	"fmt"
	"time"

	"cleanup-manager/internal/cleanup"
)

type ProfileName string

const (
	ProfileConservative ProfileName = "conservative"
	ProfileBalanced     ProfileName = "balanced"
	ProfileAggressive   ProfileName = "aggressive"
)

type Profile struct {
	Name     ProfileName
	Defaults map[string]cleanup.ProviderPolicy
}

func DefaultForProvider(meta cleanup.ProviderMetadata) cleanup.ProviderPolicy {
	policy := cleanup.ProviderPolicy{
		Enabled:      meta.DefaultMode == cleanup.ProviderModeEnabled,
		MinAge:       7 * 24 * time.Hour,
		ApprovalMode: meta.DefaultApproval,
	}
	if meta.SafetyTier == cleanup.SafetyTierConditional || meta.SafetyTier == cleanup.SafetyTierForbidden {
		policy.Enabled = false
	}
	if meta.SafetyTier == cleanup.SafetyTierForbidden {
		policy.ApprovalMode = cleanup.ApprovalModeDisabled
	}
	return policy
}

func BuildProfile(name ProfileName, providers []cleanup.ProviderMetadata) (Profile, error) {
	if name != ProfileConservative && name != ProfileBalanced && name != ProfileAggressive {
		return Profile{}, fmt.Errorf("unknown policy profile %q", name)
	}
	out := Profile{Name: name, Defaults: make(map[string]cleanup.ProviderPolicy, len(providers))}
	for _, meta := range providers {
		defaultPolicy := DefaultForProvider(meta)
		switch name {
		case ProfileBalanced:
			if meta.SafetyTier == cleanup.SafetyTierSafe || meta.SafetyTier == cleanup.SafetyTierSafeWithOwner {
				defaultPolicy.Enabled = true
				defaultPolicy.MinAge = 3 * 24 * time.Hour
			}
		case ProfileAggressive:
			if meta.SafetyTier != cleanup.SafetyTierForbidden {
				defaultPolicy.Enabled = true
				defaultPolicy.MinAge = 24 * time.Hour
			}
			if meta.SafetyTier == cleanup.SafetyTierConditional {
				defaultPolicy.ApprovalMode = cleanup.ApprovalModeOperator
			}
		}
		out.Defaults[meta.ID] = defaultPolicy
	}
	return out, nil
}

func ValidateProviderPolicy(meta cleanup.ProviderMetadata, providerPolicy cleanup.ProviderPolicy) error {
	if meta.SafetyTier == cleanup.SafetyTierForbidden && providerPolicy.Enabled {
		return fmt.Errorf("forbidden provider %q cannot be enabled", meta.ID)
	}
	if meta.SafetyTier == cleanup.SafetyTierConditional && providerPolicy.Enabled && providerPolicy.ApprovalMode != cleanup.ApprovalModeOperator {
		return fmt.Errorf("conditional provider %q requires operator approval", meta.ID)
	}
	if providerPolicy.MaxBytes < 0 {
		return fmt.Errorf("provider %q max bytes cannot be negative", meta.ID)
	}
	if providerPolicy.MinAge < 0 {
		return fmt.Errorf("provider %q min age cannot be negative", meta.ID)
	}
	return nil
}
