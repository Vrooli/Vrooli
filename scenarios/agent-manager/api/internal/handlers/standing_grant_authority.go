package handlers

import (
	"context"
	"net/http"
	"time"

	"agent-manager/internal/domain"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// RunCreationTarget is the authority target a run-creation standing grant must
// cover. Create-run names no specific run id, so the target is the domain-wide
// agent-manager object; a grant's Targets of "agent-manager:*" matches it.
const RunCreationTarget = "agent-manager:*"

// StandingGrantUsageReader reports the live, owner-attributed usage a standing
// grant is measured against. It is a seam over existing run reads — never new
// accounting — so the authority stays a thin evaluator. owner is the grant's
// accountable approver; the reader counts that owner's non-terminal runs (and,
// best-effort, their committed spend).
type StandingGrantUsageReader interface {
	StandingGrantUsage(ctx context.Context, owner string) (authn.StandingGrantUsage, error)
}

// StandingGrantAuthority is the domain-owned RequestOperationAuthority for a
// verified service principal. It sources the operator-approved grant durably by
// the caller's verified subject and delegates the decision to
// authn.AuthorizeStandingGrant — the one evaluator every carrier shares. It
// holds no inline grant logic, so Phase 3 folds it into the unified decision
// without unwinding a parallel check.
type StandingGrantAuthority struct {
	grants domain.StandingGrantRepository
	usage  StandingGrantUsageReader
	now    func() time.Time
}

// NewStandingGrantAuthority builds the authority from the durable store and a
// usage reader. now may be nil.
func NewStandingGrantAuthority(grants domain.StandingGrantRepository, usage StandingGrantUsageReader, now func() time.Time) *StandingGrantAuthority {
	if now == nil {
		now = time.Now
	}
	return &StandingGrantAuthority{grants: grants, usage: usage, now: now}
}

// AuthorizeOperation admits the operation when some currently-authoritative
// grant for the verified subject covers it within limits. It returns
// authn.ErrOperation when no grant covers the operation/target and
// authn.ErrStandingGrantLimit when one covers it but a bound is exceeded.
func (a *StandingGrantAuthority) AuthorizeOperation(ctx context.Context, p identity.Principal, _, operation, target string) error {
	if a == nil || a.grants == nil || p.Kind != identity.ActorService || !p.IsVerified() {
		return authn.ErrOperation
	}
	now := a.now()
	candidates, err := a.grants.ActiveForSubject(ctx, p.Subject, now)
	if err != nil {
		return err
	}
	// The first grant that covers the operation decides the request. Limits are
	// measured against the grant's accountable owner, not the service subject, so
	// concurrency/budget bound the operator's attributed runs.
	var lastErr error = authn.ErrOperation
	for _, g := range candidates {
		if g.Grant == nil {
			continue
		}
		var usage authn.StandingGrantUsage
		if a.usage != nil {
			usage, err = a.usage.StandingGrantUsage(ctx, g.Grant.Approver)
			if err != nil {
				return err
			}
		}
		if decideErr := authn.AuthorizeStandingGrant(g.Grant, g.Limits, usage, operation, target, now); decideErr == nil {
			return nil
		} else {
			lastErr = decideErr
		}
	}
	return lastErr
}

// StandingGrantMiddleware installs the standing-grant authority into the request
// context for a verified service principal, before AgentOperationMiddleware and
// the handlers decide. A non-service caller is untouched, so human and agent
// authorization are unchanged. The authority is a trusted, immutable ingress
// object, exactly like a verified principal.
func StandingGrantMiddleware(authority *StandingGrantAuthority) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authority != nil {
				if p, ok := identity.PrincipalFromContext(r.Context()); ok && p.Kind == identity.ActorService && p.IsVerified() {
					next.ServeHTTP(w, r.WithContext(authn.WithRequestOperationAuthority(r.Context(), authority)))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
