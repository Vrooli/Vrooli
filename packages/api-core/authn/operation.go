package authn

import (
	"context"
	"errors"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

// OwnerBinding assigns operator policy to an exact verified provider subject.
// Tunnel reachability, email and local UID never create this binding.
type OwnerBinding struct {
	Source           identity.AuthSource `json:"source"`
	Subject          string              `json:"subject"`
	CanonicalSubject string              `json:"canonical_subject"`
	Scopes           []string            `json:"scopes"`
	Resources        []string            `json:"resources,omitempty"`
}

var ErrOperation = errors.New("operation or target is outside the approved agent grant")

// RequireOperation admits a capable human, or an agent with an explicitly
// approved live operation grant. Domain owners retain their object rules.
func RequireOperation(ctx context.Context, capability, operation, target string) (identity.Principal, error) {
	if a, present := RequestOperationAuthorityFromContext(ctx); present {
		p, ok := identity.PrincipalFromContext(ctx)
		if !ok || p.Kind != identity.ActorAgent {
			return identity.Principal{}, ErrOperation
		}
		if err := a.AuthorizeOperation(ctx, p, capability, operation, target); err != nil {
			return identity.Principal{}, err
		}
		return p, nil
	}
	p, err := RequireCapability(ctx, capability)
	if err != nil {
		return identity.Principal{}, err
	}
	if !p.ExpiresAt.IsZero() && !time.Now().Before(p.ExpiresAt) {
		return identity.Principal{}, identity.NewFailure(identity.FailureExpired, p.Source)
	}
	if p.IsHuman() {
		return p, nil
	}
	if p.Kind == identity.ActorAgent && p.Grant.Allows(operation, target, time.Now()) {
		return p, nil
	}
	return identity.Principal{}, ErrOperation
}

// Policy is the reviewed, finite operation set available to an interactive
// grant. It cannot include retained human decisions or arbitrary capabilities.
type Policy struct{ Scopes, Operations, Domains []string }

func InteractivePolicy(name string) (Policy, bool) {
	switch name {
	case "swarm-bookkeeping":
		return Policy{[]string{"swarm-manager:read", "swarm-manager:write", "swarm-manager:destructive"}, []string{"swarm.bookkeeping"}, []string{"swarm-manager"}}, true
	case "agent-investigation":
		return Policy{[]string{"agent-manager:read", "agent-manager:write"}, []string{"agent.investigate"}, []string{"agent-manager"}}, true
	case "agent-management":
		return Policy{[]string{"agent-manager:read", "agent-manager:write"}, []string{"agent.investigate", "agent.manage"}, []string{"agent-manager"}}, true
	case "effort-metadata":
		// Owner-facing effort metadata only (goal home, display name, revisions).
		// Agent Manager refuses any change to authority, subjects or withdrawal.
		return Policy{[]string{"agent-manager:read", "agent-manager:write"}, []string{"effort.reconcile"}, []string{"agent-manager"}}, true
	case "deployment-operations":
		return Policy{[]string{"deployment-manager:read", "deployment-manager:write", "deployment-manager:destructive", "scenario-to-cloud:read", "scenario-to-cloud:write", "scenario-to-cloud:destructive"}, []string{"deployment.manage"}, []string{"deployment-manager", "scenario-to-cloud"}}, true
	case "nutrition-management":
		return Policy{[]string{"nutrition-planner:read", "nutrition-planner:write", "nutrition-planner:destructive"}, []string{"nutrition.manage"}, []string{"nutrition-planner"}}, true
	default:
		return Policy{}, false
	}
}

// RequireIndependentHuman retains a human decision even when a forwarded
// request also carries valid agent attribution. The approver cannot become
// the actor merely because both credentials verify.
func RequireIndependentHuman(ctx context.Context) (identity.Principal, error) {
	p, err := RequireHuman(ctx)
	if err != nil {
		return p, err
	}
	if provenance.FromContext(ctx).IsAgent() {
		return identity.Principal{}, ErrHumanRequired
	}
	return p, nil
}
