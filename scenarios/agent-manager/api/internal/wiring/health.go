package wiring

import (
	"context"

	"agent-manager/internal/fallback"
	healthstore "agent-manager/internal/health"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/rolepolicy"

	"github.com/vrooli/api-core/health"
)

// NewModelHealthProbe adapts runner failure classification at the composition
// root, leaving the health substrate dependent only on its local contract.
func NewModelHealthProbe(store *healthstore.Store, registry healthstore.RegistrySnapshot, resolve healthstore.RunnerProberLookup, config healthstore.ProbeConfig) *healthstore.Probe {
	return healthstore.NewProbe(store, registry, resolve, fallbackHealthClassifier{classifier: fallback.NewTextClassifier()}, config)
}

type fallbackHealthClassifier struct{ classifier fallback.Classifier }

func (a fallbackHealthClassifier) Classify(in healthstore.FailureInput) *healthstore.ClassifiedFailure {
	classified := a.classifier.Classify(fallback.ClassifyInput{RunnerType: in.RunnerType, Stderr: in.Stderr, Cause: in.Cause})
	if classified == nil {
		return nil
	}
	return &healthstore.ClassifiedFailure{Reason: string(classified.Reason), Message: classified.Message}
}

// RolePolicyHealthChecker keeps portable role authority a readiness
// dependency. A run can never safely resolve a role from an absent catalog.
func RolePolicyHealthChecker(state *rolepolicy.State) health.Checker {
	if state == nil {
		return nil
	}
	return health.CheckerFunc(func(context.Context) health.CheckResult {
		status := state.Status()
		if err := state.ReadinessError(); err != nil {
			detail := health.NewErrorDetail(rolepolicy.DiagnosticCodeCatalogInvalid, err.Error(), "configuration", true)
			detail.Details = map[string]any{"path": status.Path, "required": status.Requirement.Required, "requirement_reason": status.Requirement.Reason, "active_digest": status.ActiveDigest}
			return health.CheckResult{Name: "role_policy_catalog", Connected: false, Error: detail}
		}
		return health.CheckResult{Name: "role_policy_catalog", Connected: true}
	})
}

// PermissionPolicyHealthChecker separates invalid global desired state from
// resource availability. Projection is never an infrastructure-health side
// effect.
func PermissionPolicyHealthChecker(state *permissionpolicy.State, service *permissionpolicy.Service) health.Checker {
	if state == nil {
		return nil
	}
	return health.CheckerFunc(func(ctx context.Context) health.CheckResult {
		status := state.Status()
		if err := state.ReadinessError(); err != nil {
			detail := health.NewErrorDetail(permissionpolicy.DiagnosticCodeCatalogInvalid, err.Error(), "configuration", true)
			detail.Details = map[string]any{"path": status.Path, "required": status.Requirement.Required, "requirement_reason": status.Requirement.Reason, "active_digest": status.ActiveDigest}
			return health.CheckResult{Name: "permission_policy_catalog", Connected: false, Error: detail}
		}
		if service != nil {
			if err := service.ReadinessError(ctx); err != nil {
				detail := health.NewErrorDetail("PERMISSION_POLICY_HARD_ENFORCEMENT_UNREADY", err.Error(), "permission_enforcement", true)
				detail.Details = map[string]any{"path": status.Path, "active_digest": status.ActiveDigest, "requirement_reason": status.Requirement.Reason}
				return health.CheckResult{Name: "permission_policy_enforcement", Connected: false, Error: detail}
			}
		}
		return health.CheckResult{Name: "permission_policy_catalog", Connected: true}
	})
}
