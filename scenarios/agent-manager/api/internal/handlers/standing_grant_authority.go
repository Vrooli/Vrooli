package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// errStandingGrantStateUnavailable marks a grant or owner-usage lookup that
// could not establish authority. HTTP handlers map it to a generic refusal;
// callers can still inspect the wrapped cause internally.
var errStandingGrantStateUnavailable = errors.New("standing grant state unavailable")

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
	diagnostic, err := a.diagnoseOperation(ctx, p.Subject, operation, target)
	if err != nil {
		return err
	}
	switch diagnostic.Outcome {
	case ServiceAdmissionWouldAdmit:
		return nil
	case ServiceAdmissionWouldRefuseLimit:
		return authn.ErrStandingGrantLimit
	case ServiceAdmissionWouldRequireApproval:
		return authn.ErrNeedsApproval
	default:
		return errStandingGrantStateUnavailable
	}
}

const (
	ServiceAdmissionWouldAdmit           = "would_admit"
	ServiceAdmissionWouldRequireApproval = "would_require_approval"
	ServiceAdmissionWouldRefuseLimit     = "would_refuse_limit"
	ServiceAdmissionUnavailable          = "unavailable"
)

// ServiceAdmissionLimit exposes an aggregate usage and its bound without grant
// records or owner identity. Current and Bound represent either a run count or
// a USD amount, as named by Dimension.
type ServiceAdmissionLimit struct {
	Dimension string  `json:"dimension"`
	Current   float64 `json:"current"`
	Bound     float64 `json:"bound"`
}

// ServiceAdmissionDiagnostic is a redacted, read-only projection of the fixed
// Prompt Manager run.create decision. It contains no grant, approver, or owner
// records and never implies that an approval request was raised.
type ServiceAdmissionDiagnostic struct {
	Outcome         string                  `json:"outcome"`
	Reason          string                  `json:"reason"`
	ObservedAt      time.Time               `json:"observed_at"`
	Limits          []ServiceAdmissionLimit `json:"limits,omitempty"`
	LimitsTruncated bool                    `json:"limits_truncated,omitempty"`
}

const serviceAdmissionLimitResultCap = 8

type serviceAdmissionLimitKey struct {
	dimension string
	current   float64
	bound     float64
}

// diagnoseOperation evaluates current active grants and current usage using
// the same pure evaluator as normal service admission. It has no principal
// fabrication, approval-raiser call, or store write.
func (a *StandingGrantAuthority) diagnoseOperation(ctx context.Context, subject, operation, target string) (ServiceAdmissionDiagnostic, error) {
	if a == nil || a.grants == nil || strings.TrimSpace(subject) == "" {
		return ServiceAdmissionDiagnostic{Outcome: ServiceAdmissionUnavailable, Reason: "authority_state_unavailable"}, fmt.Errorf("%w: grant authority is unavailable", errStandingGrantStateUnavailable)
	}
	now := a.now()
	candidates, err := a.grants.ActiveForSubject(ctx, subject, now)
	if err != nil {
		return ServiceAdmissionDiagnostic{Outcome: ServiceAdmissionUnavailable, Reason: "authority_state_unavailable", ObservedAt: now.UTC()}, fmt.Errorf("%w: %w", errStandingGrantStateUnavailable, err)
	}
	diagnostic := ServiceAdmissionDiagnostic{Outcome: ServiceAdmissionWouldRequireApproval, Reason: "no_covering_grant", ObservedAt: now.UTC()}
	covered := false
	seenLimits := make(map[serviceAdmissionLimitKey]struct{})
	for _, candidate := range candidates {
		if candidate == nil || candidate.Grant == nil {
			continue
		}
		var usage authn.StandingGrantUsage
		if a.usage != nil {
			usage, err = a.usage.StandingGrantUsage(ctx, candidate.Grant.Approver)
			if err != nil {
				return ServiceAdmissionDiagnostic{Outcome: ServiceAdmissionUnavailable, Reason: "authority_state_unavailable", ObservedAt: now.UTC()}, fmt.Errorf("%w: %w", errStandingGrantStateUnavailable, err)
			}
		}
		evaluation := authn.EvaluateStandingGrant(candidate.Grant, candidate.Limits, usage, operation, target, now)
		if !evaluation.Covered {
			continue
		}
		covered = true
		if evaluation.Admitted {
			return ServiceAdmissionDiagnostic{Outcome: ServiceAdmissionWouldAdmit, Reason: "grant_covers_operation", ObservedAt: now.UTC()}, nil
		}
		diagnostic.Outcome = ServiceAdmissionWouldRefuseLimit
		diagnostic.Reason = "grant_limit_reached"
		for _, exceeded := range evaluation.Exceeded {
			limit := ServiceAdmissionLimit{Dimension: string(exceeded.Dimension)}
			switch exceeded.Dimension {
			case authn.StandingGrantConcurrentRunsLimit:
				limit.Current = float64(exceeded.CurrentRuns)
				limit.Bound = float64(exceeded.BoundRuns)
			case authn.StandingGrantBudgetLimit:
				limit.Current = exceeded.CurrentBudgetUSD
				limit.Bound = exceeded.BoundBudgetUSD
			default:
				continue
			}
			key := serviceAdmissionLimitKey{dimension: limit.Dimension, current: limit.Current, bound: limit.Bound}
			if _, exists := seenLimits[key]; exists {
				continue
			}
			seenLimits[key] = struct{}{}
			if len(diagnostic.Limits) < serviceAdmissionLimitResultCap {
				diagnostic.Limits = append(diagnostic.Limits, limit)
			} else {
				diagnostic.LimitsTruncated = true
			}
		}
	}
	if covered && diagnostic.Outcome == ServiceAdmissionWouldRequireApproval {
		return ServiceAdmissionDiagnostic{Outcome: ServiceAdmissionUnavailable, Reason: "authority_state_unavailable", ObservedAt: now.UTC()}, fmt.Errorf("%w: covered grant evaluation returned no decision", errStandingGrantStateUnavailable)
	}
	return diagnostic, nil
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
