package authn

import (
	"context"
	"errors"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/scopecatalog"
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

// ErrOperation is the sentinel for a refused operation. RequireOperation returns
// an *OperationRefusal for an agent, which wraps it and adds the request
// command; test with errors.Is, never ==.
var ErrOperation = errors.New("operation or target is outside the approved agent grant")

// RequireOperation admits a capable human, an agent with an explicitly approved
// live operation grant, or a verified service principal whose operator-approved
// standing grant covers the operation. Domain owners retain their object rules.
//
// The grant for an agent rides inside its verified identity token (p.Grant); the
// grant for a service is held durably by the domain and consulted through the
// RequestOperationAuthority ingress (never a caller-supplied scope). Both resolve
// to one OperationGrant evaluated by identity.OperationGrant.Allows.
func RequireOperation(ctx context.Context, capability, operation, target string) (identity.Principal, error) {
	if a, present := RequestOperationAuthorityFromContext(ctx); present {
		p, ok := identity.PrincipalFromContext(ctx)
		if !ok || (p.Kind != identity.ActorAgent && p.Kind != identity.ActorService) {
			return identity.Principal{}, ErrOperation
		}
		if err := a.AuthorizeOperation(ctx, p, capability, operation, target); err != nil {
			if p.Kind == identity.ActorAgent && err == ErrOperation {
				return identity.Principal{}, NewOperationRefusal(operation, target)
			}
			return identity.Principal{}, err
		}
		return p, nil
	}
	p, present := identity.PrincipalFromContext(ctx)
	if !present || !p.IsVerified() {
		return identity.Principal{}, ErrCapability
	}
	if !p.ExpiresAt.IsZero() && !time.Now().Before(p.ExpiresAt) {
		return identity.Principal{}, identity.NewFailure(identity.FailureExpired, p.Source)
	}
	if p.IsHuman() {
		if !scopecatalog.Resolve(p.Scopes, capability) {
			return identity.Principal{}, ErrCapability
		}
		return p, nil
	}
	if p.Kind == identity.ActorAgent {
		// A paired operation proof remains distinct from the ordinary principal.
		// It can satisfy only this exact operation check; its scopes are never
		// copied into p.Scopes or used by the bounded-route classifier.
		verified, verifyErr := provenance.ReverifyOperationProof(ctx)
		if verifyErr == nil && verified != nil && time.Now().Before(verified.ExpiresAt) &&
			scopecatalog.Resolve(verified.Scopes, capability) && verified.Grant.Allows(operation, target, time.Now()) {
			return p, nil
		}
		// Preserve the existing grant-only credential contract. Its token scopes
		// and grant remain one bounded principal as before.
		if scopecatalog.Resolve(p.Scopes, capability) && p.Grant.Allows(operation, target, time.Now()) {
			return p, nil
		}
		// The refusal names the exact request that would let the operator approve
		// this operation on this target. errors.Is(err, ErrOperation) still holds.
		return identity.Principal{}, NewOperationRefusal(operation, target)
	}
	if !scopecatalog.Resolve(p.Scopes, capability) {
		return identity.Principal{}, ErrCapability
	}
	return identity.Principal{}, ErrOperation
}

// RequireIndependentHuman retains a human decision even when a forwarded
// request also carries valid agent attribution. The approver cannot become
// the actor merely because both credentials verify.
func RequireIndependentHuman(ctx context.Context) (identity.Principal, error) {
	p, err := RequireHuman(ctx)
	if err != nil {
		return p, err
	}
	if provenance.FromContext(ctx).IsAgent() && p.Source != identity.SourceUIFailOpen {
		return identity.Principal{}, ErrHumanRequired
	}
	return p, nil
}
