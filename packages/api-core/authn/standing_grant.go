package authn

import (
	"errors"
	"time"

	"github.com/vrooli/api-core/identity"
)

// This file holds the carrier-agnostic authority decision for a standing grant:
// an operator-approved, bounded authority that lets a verified service principal
// (or an unattended agent) perform a costly operation — run creation first —
// without a human present for each call.
//
// The decision is deliberately pure and source-independent. The durable,
// subject-keyed store that holds approved grants, and the live usage it is
// measured against, belong to the domain (agent-manager); this package only
// evaluates a grant the domain has already looked up. That keeps one evaluator
// for every carrier: an agent carries its grant in a verified identity token,
// while a service presents identity only and the domain sources the grant from
// its store, but both converge here.

// ErrStandingGrantLimit is the refusal when a grant is valid for the
// operation and target but the caller is already at or beyond a declared
// bound. It is distinct from ErrOperation so a domain can tell "no authority"
// apart from "authority exhausted" in its audit stream and operator messaging.
var ErrStandingGrantLimit = errors.New("standing grant limit reached")

// ErrNeedsApproval is the "ask, don't block" outcome: a verified, KNOWN caller
// (a service principal, or a verified agent) presents no covering grant for a
// costly operation and no live human is present to authorize it synchronously.
// It is deliberately distinct from ErrOperation (hard refuse for an anonymous,
// forged, or absent identity) and from a human's missing-capability 403: only a
// caller the system already trusts-as-identity but not-yet-for-this-operation
// may trigger an operator approval. The domain translates it to a typed
// "approval pending" (202) plus a notification-hub decision request, never a
// hard block. An over-limit grant is still ErrStandingGrantLimit, not approval.
var ErrNeedsApproval = errors.New("operation requires operator approval")

// StandingGrantLimits are the bounds an operator attaches to a standing grant.
// They live on the grant record, never in a proto message or a token claim, so
// tightening a bound is an operator edit to the record with no contract change.
// A zero value means "no bound declared" for that dimension, so an operator who
// omits a limit gets an unbounded one explicitly rather than by a silent default.
type StandingGrantLimits struct {
	// MaxConcurrentRuns caps how many non-terminal runs attributed to the grant's
	// owner may exist at once. 0 means unbounded.
	MaxConcurrentRuns int `json:"max_concurrent_runs,omitempty"`
	// MaxRunBudgetUSD caps the projected spend of the runs attributed to the
	// grant's owner. 0 means unbounded.
	MaxRunBudgetUSD float64 `json:"max_run_budget_usd,omitempty"`
}

// StandingGrantUsage is the live, owner-attributed usage the domain measures at
// admission time. It is injected rather than read here so this decision stays
// pure and unit-testable without a database.
type StandingGrantUsage struct {
	// ConcurrentRuns is the current count of non-terminal runs attributed to the
	// grant's owner (the runs this grant is accountable for).
	ConcurrentRuns int
	// CommittedBudgetUSD is the projected spend already committed to those runs.
	CommittedBudgetUSD float64
	// RequestBudgetUSD is the projected spend this new request would add. When a
	// request's spend is not knowable up front the domain passes 0 and the budget
	// bound is enforced purely on already-committed spend.
	RequestBudgetUSD float64
}

// StandingGrantLimitDimension names a declared grant bound reached by the
// supplied usage. It is used for a read-only diagnosis of the same admission
// decision; it does not alter that decision or inspect owner data.
type StandingGrantLimitDimension string

const (
	StandingGrantConcurrentRunsLimit StandingGrantLimitDimension = "max_concurrent_runs"
	StandingGrantBudgetLimit         StandingGrantLimitDimension = "max_run_budget_usd"
)

// StandingGrantLimit reports the current aggregate and configured bound for a
// single exceeded dimension. Exactly one numeric pair is populated.
type StandingGrantLimit struct {
	Dimension        StandingGrantLimitDimension
	CurrentRuns      int
	BoundRuns        int
	CurrentBudgetUSD float64
	BoundBudgetUSD   float64
}

// StandingGrantEvaluation is the pure result of evaluating one grant. Covered
// distinguishes a grant that applies to the requested operation/target from an
// absent, expired, or non-covering grant. Exceeded contains only active bounds.
type StandingGrantEvaluation struct {
	Covered  bool
	Admitted bool
	Exceeded []StandingGrantLimit
}

// EvaluateStandingGrant evaluates one grant against already-read usage. It is
// pure: it neither reads state nor mutates authority, and is shared by normal
// admission and the bounded read-only diagnostic.
func EvaluateStandingGrant(grant *identity.OperationGrant, limits StandingGrantLimits, usage StandingGrantUsage, operation, target string, now time.Time) StandingGrantEvaluation {
	if grant == nil || !grant.Allows(operation, target, now) {
		return StandingGrantEvaluation{}
	}
	evaluation := StandingGrantEvaluation{Covered: true}
	if limits.MaxConcurrentRuns > 0 && usage.ConcurrentRuns >= limits.MaxConcurrentRuns {
		evaluation.Exceeded = append(evaluation.Exceeded, StandingGrantLimit{
			Dimension:   StandingGrantConcurrentRunsLimit,
			CurrentRuns: usage.ConcurrentRuns,
			BoundRuns:   limits.MaxConcurrentRuns,
		})
	}
	currentBudget := usage.CommittedBudgetUSD + usage.RequestBudgetUSD
	if limits.MaxRunBudgetUSD > 0 && currentBudget > limits.MaxRunBudgetUSD {
		evaluation.Exceeded = append(evaluation.Exceeded, StandingGrantLimit{
			Dimension:        StandingGrantBudgetLimit,
			CurrentBudgetUSD: currentBudget,
			BoundBudgetUSD:   limits.MaxRunBudgetUSD,
		})
	}
	evaluation.Admitted = len(evaluation.Exceeded) == 0
	return evaluation
}

// AuthorizeStandingGrant decides whether a standing grant authorizes one costly
// operation against one target for one request, within its declared limits. It
// returns nil on admission, ErrOperation when the grant does not cover the
// operation/target (or is missing/expired), and ErrStandingGrantLimit when the
// grant covers the request but a bound would be exceeded.
//
// It never mutates state and never reads a store; the domain supplies the
// already-looked-up grant, its limits, and the live usage. The owner the run is
// attributed to is the grant's Approver — the accountable human — which the
// caller reads from grant.Approver after a nil return.
func AuthorizeStandingGrant(grant *identity.OperationGrant, limits StandingGrantLimits, usage StandingGrantUsage, operation, target string, now time.Time) error {
	evaluation := EvaluateStandingGrant(grant, limits, usage, operation, target, now)
	if !evaluation.Covered {
		return ErrOperation
	}
	if !evaluation.Admitted {
		return ErrStandingGrantLimit
	}
	return nil
}
