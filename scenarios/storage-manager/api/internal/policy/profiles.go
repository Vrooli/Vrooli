package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"storage-manager/internal/cleanup"
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
			if meta.ID == "go-build-cache" {
				defaultPolicy.Enabled = true
				defaultPolicy.MinAge = 7 * 24 * time.Hour
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

// Reconcile adds providers that were registered after a persisted policy was
// written. Existing entries are copied verbatim: an operator's explicit
// disable or approval choice is never silently changed. The returned provider
// IDs are sorted so the audit event and tests are deterministic.
func Reconcile(name ProfileName, existing cleanupPolicy, providers []cleanup.ProviderMetadata) (cleanupPolicy, []string, error) {
	profile, err := BuildProfile(name, providers)
	if err != nil {
		return cleanupPolicy{}, nil, err
	}
	out := existing
	providersCopy := make(map[string]cleanup.ProviderPolicy, len(existing.Providers)+len(providers))
	for id, providerPolicy := range existing.Providers {
		providersCopy[id] = providerPolicy
	}
	out.Providers = providersCopy
	added := make([]string, 0)
	for _, meta := range providers {
		if _, ok := out.Providers[meta.ID]; ok {
			if err := ValidateProviderPolicy(meta, out.Providers[meta.ID]); err != nil {
				return cleanupPolicy{}, nil, err
			}
			continue
		}
		defaultPolicy, ok := profile.Defaults[meta.ID]
		if !ok {
			return cleanupPolicy{}, nil, fmt.Errorf("profile %q has no default for provider %q", name, meta.ID)
		}
		if err := ValidateProviderPolicy(meta, defaultPolicy); err != nil {
			return cleanupPolicy{}, nil, err
		}
		out.Providers[meta.ID] = defaultPolicy
		added = append(added, meta.ID)
	}
	if len(added) > 0 {
		sort.Strings(added)
		out.Version = StableVersion(name, out.Providers)
	}
	return out, added, nil
}

// cleanupPolicy is the small policy shape needed by reconciliation. Keeping
// this package independent of orchestrator avoids an import cycle.
type cleanupPolicy struct {
	Version   string
	Profile   ProfileName
	Providers map[string]cleanup.ProviderPolicy
	CreatedAt time.Time
}

// ReconcilePolicy adapts the orchestrator policy without coupling the policy
// package to the orchestrator package.
func ReconcilePolicy(name ProfileName, version string, existing map[string]cleanup.ProviderPolicy, createdAt time.Time, providers []cleanup.ProviderMetadata) (string, map[string]cleanup.ProviderPolicy, []string, error) {
	policy, added, err := Reconcile(name, cleanupPolicy{Version: version, Profile: name, Providers: existing, CreatedAt: createdAt}, providers)
	return policy.Version, policy.Providers, added, err
}

// StableVersion is the canonical policy fingerprint used for replay safety.
func StableVersion(name ProfileName, policies map[string]cleanup.ProviderPolicy) string {
	raw, _ := json.Marshal(struct {
		Name     ProfileName
		Policies map[string]cleanup.ProviderPolicy
	}{Name: name, Policies: policies})
	sum := sha256.Sum256(raw)
	return "policy-" + hex.EncodeToString(sum[:])[:16]
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
