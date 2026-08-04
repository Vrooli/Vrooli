package rolepolicy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

// ResolvedCandidate is immutable resource-owned evidence captured for one
// portable catalog candidate. Model and fallbacks are output evidence only.
type ResolvedCandidate struct {
	Runner         domain.RunnerType        `json:"runner"`
	ResourceRole   string                   `json:"resourceRole"`
	Model          string                   `json:"model,omitempty"`
	CanonicalModel string                   `json:"canonicalModel,omitempty"`
	Fallbacks      []string                 `json:"fallbacks,omitempty"`
	Available      bool                     `json:"available"`
	FailureCode    string                   `json:"failureCode,omitempty"`
	Failure        string                   `json:"failure,omitempty"`
	Provenance     ResourceProvenance       `json:"provenance,omitempty"`
	Enforcement    EnforcementPosture       `json:"enforcement,omitempty"`
	PolicyPath     string                   `json:"policyPath,omitempty"`
	PolicyDigest   string                   `json:"policyDigest,omitempty"`
	Billing        domain.BillingSnapshot   `json:"billing,omitempty"`
	Challenger     *domain.ChallengerConfig `json:"challenger,omitempty"`
}

// Resolution is a run-creation-time immutable result. It is intentionally
// separate from mutable State and can be persisted without rereading resource
// policies during fallback or resume.
type Resolution struct {
	CatalogDigest string              `json:"catalogDigest"`
	RoleRef       string              `json:"roleRef"`
	Candidates    []ResolvedCandidate `json:"candidates"`
}

// Snapshot converts resource-owned resolution evidence into the domain's
// run-owned immutable execution contract. Failed candidates remain recorded
// so a later selection is auditable.
func (r *Resolution) Snapshot() *domain.ExecutionPolicySnapshot {
	if r == nil {
		return nil
	}
	candidates := make([]domain.ExecutionCandidate, 0, len(r.Candidates))
	for _, candidate := range r.Candidates {
		selection := domain.ModelSelectionTypeModel
		if candidate.Model == "" {
			selection = ""
		}
		candidates = append(candidates, domain.ExecutionCandidate{
			RunnerType: candidate.Runner, SelectionType: selection, Model: candidate.Model,
			CanonicalModel: candidate.CanonicalModel,
			ResourceRole:   candidate.ResourceRole, Fallbacks: append([]string(nil), candidate.Fallbacks...),
			Available: candidate.Available, FailureCode: candidate.FailureCode, Failure: candidate.Failure,
			Provenance:  domain.ResourceProvenance{Source: candidate.Provenance.Source, ObservedAt: candidate.Provenance.ObservedAt},
			Enforcement: domain.PermissionEnforcement{Permissions: candidate.Enforcement.Permissions, Caveats: append([]string(nil), candidate.Enforcement.Caveats...)},
			PolicyPath:  candidate.PolicyPath, PolicyDigest: candidate.PolicyDigest,
			Billing:         candidate.Billing,
			ChallengerModel: challengerModel(candidate.Challenger), ChallengerSampleRate: challengerRate(candidate.Challenger),
		})
	}
	return &domain.ExecutionPolicySnapshot{
		CatalogDigest: r.CatalogDigest, RoleRef: r.RoleRef, Candidates: candidates,
		Explanation: domain.PolicyResolutionExplanation{Source: "portable_role", Summary: fmt.Sprintf("portable role %q resolved through resource-owned policy catalogs", r.RoleRef), RequestedRoleRef: r.RoleRef},
	}
}

func challengerModel(value *domain.ChallengerConfig) string {
	if value == nil {
		return ""
	}
	return value.Model
}

func challengerRate(value *domain.ChallengerConfig) float64 {
	if value == nil {
		return 0
	}
	return value.SampleRate
}

// Resolve resolves every configured candidate in deterministic catalog order.
// Unavailable/unknown resources remain represented as evidence so callers can
// surface truthful diagnostics and choose a later available candidate.
func (s *State) Resolve(ctx context.Context, resolver Resolver, roleRef string) (*Resolution, error) {
	roleRef = strings.TrimSpace(roleRef)
	if roleRef == "" {
		return nil, domain.NewValidationError("roleRef", "field is required")
	}
	if resolver == nil {
		return nil, domain.NewValidationError("resourceRoleResolver", "resolver is not configured")
	}
	revision, catalog, err := s.activeCatalog()
	if err != nil {
		return nil, err
	}
	role, ok := catalog.Roles[roleRef]
	if !ok {
		return nil, domain.NewValidationError("roleRef", fmt.Sprintf("role %q is not declared in active catalog %s", roleRef, revision.Digest()))
	}

	result := &Resolution{CatalogDigest: revision.Digest(), RoleRef: roleRef, Candidates: make([]ResolvedCandidate, 0, len(role.Candidates))}
	for _, candidate := range role.Candidates {
		resolved := ResolvedCandidate{Runner: candidate.Runner, ResourceRole: candidate.ResourceRole}
		evidence, resolveErr := resolver.Resolve(ctx, candidate.Runner, candidate.ResourceRole)
		if resolveErr != nil {
			resolved.Failure = resolveErr.Error()
			switch {
			case errors.Is(resolveErr, ErrResourceUnavailable):
				resolved.FailureCode = "resource_unavailable"
			case errors.Is(resolveErr, ErrUnknownResourceRole):
				resolved.FailureCode = "resource_role_unknown"
			default:
				resolved.FailureCode = "resource_response_invalid"
			}
			result.Candidates = append(result.Candidates, resolved)
			continue
		}
		resolved.Model = evidence.Model
		resolved.CanonicalModel = evidence.CanonicalModel
		resolved.Fallbacks = append([]string(nil), evidence.Fallbacks...)
		resolved.Available = true
		resolved.Provenance = evidence.Provenance
		resolved.Enforcement = EnforcementPosture{Permissions: evidence.Enforcement.Permissions, Caveats: append([]string(nil), evidence.Enforcement.Caveats...)}
		resolved.PolicyPath = evidence.PolicyPath
		resolved.PolicyDigest = evidence.PolicyDigest
		resolved.Billing = evidence.Billing
		resolved.Challenger = evidence.Challenger
		if resolved.Billing.Basis == "" {
			resolved.Billing.Basis = resolved.Billing.EffectiveBasis()
		}
		if resolved.Billing.ObservedAt.IsZero() {
			resolved.Billing.ObservedAt = time.Now().UTC()
		}
		result.Candidates = append(result.Candidates, resolved)
	}
	return result, nil
}

func (s *State) activeCatalog() (*Revision, *Catalog, error) {
	if s == nil {
		return nil, nil, domain.NewValidationError("rolePolicyCatalog", "state is not configured")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.active == nil || s.active.catalog == nil {
		message := "no validated catalog revision is active"
		if s.lastAttempt != nil && s.lastAttempt.Diagnostic != nil {
			message = s.lastAttempt.Diagnostic.Message
		}
		return nil, nil, domain.NewValidationError("rolePolicyCatalog", message)
	}
	return s.active, s.active.catalog.Clone(), nil
}
