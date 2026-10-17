package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

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
// grant for the verified subject covers it within limits. Its three outcomes:
//   - nil: a grant covers the operation within limits → admit.
//   - authn.ErrStandingGrantLimit: a grant covers it but a bound is exceeded →
//     hard refuse (over-limit is never an approval; approving would not help).
//   - authn.ErrNeedsApproval: the verified caller has NO covering grant at all →
//     "ask, don't block" (the middleware raises an operator decision and returns
//     a typed approval-pending). This fires only for a verified KNOWN caller; a
//     non-service/unverified principal is rejected with authn.ErrOperation up
//     front and never reaches the ask path.
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
	// concurrency/budget bound the operator's attributed runs. We distinguish "a
	// grant covered it but a limit was hit" (hard refuse) from "no grant covered
	// it" (needs approval): only the latter is an "ask, don't block".
	limited := false
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
		switch decideErr := authn.AuthorizeStandingGrant(g.Grant, g.Limits, usage, operation, target, now); {
		case decideErr == nil:
			return nil
		case errors.Is(decideErr, authn.ErrStandingGrantLimit):
			limited = true
		}
	}
	if limited {
		// A grant covers this operation but the caller is over a declared bound.
		// Approval cannot fix a limit; this is a hard refuse, not an ask.
		return authn.ErrStandingGrantLimit
	}
	// A verified known caller with no covering grant: ask, don't block.
	return authn.ErrNeedsApproval
}

// ApprovalRaiser records (idempotently) an operator approval request for a
// verified caller that lacks a covering grant, and raises the notification-hub
// decision. It returns raised=true when the ask is outstanding (so the caller
// is told "approval pending"), and an error (ErrAuthorizationFlood) when the
// subject is over its outstanding-ask cap (refuse without notifying). It is
// satisfied by *orchestration.Orchestrator. A nil raiser degrades the
// needs-approval outcome back to a hard refuse (no regression).
type ApprovalRaiser interface {
	RaiseAuthorizationApproval(ctx context.Context, req orchestration.AuthorizationApprovalRequest) (bool, error)
}

type approvalRaiserKey struct{}

// ApprovalRaiserFromContext returns the raiser installed by
// StandingGrantMiddleware, if any.
func ApprovalRaiserFromContext(ctx context.Context) (ApprovalRaiser, bool) {
	r, ok := ctx.Value(approvalRaiserKey{}).(ApprovalRaiser)
	return r, ok && r != nil
}

// StandingGrantMiddleware installs the standing-grant authority into the request
// context for a verified service principal, before AgentOperationMiddleware and
// the handlers decide. A non-service caller is untouched, so human and agent
// authorization are unchanged. The authority (and the approval raiser used for
// the "ask, don't block" path) are trusted, immutable ingress objects, exactly
// like a verified principal.
func StandingGrantMiddleware(authority *StandingGrantAuthority, raiser ApprovalRaiser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authority != nil {
				if p, ok := identity.PrincipalFromContext(r.Context()); ok && p.Kind == identity.ActorService && p.IsVerified() {
					ctx := authn.WithRequestOperationAuthority(r.Context(), authority)
					if raiser != nil {
						ctx = context.WithValue(ctx, approvalRaiserKey{}, raiser)
					}
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
