package routing

import (
	"fmt"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
)

// policyReviewEvent is an observation, never a command to the catalog owner.
// Every admission observes a fresh catalog revision; no local state is stored.
type policyReviewEvent string

const (
	policyObservedProposed   policyReviewEvent = "catalog_proposed"
	policyObservedPending    policyReviewEvent = "catalog_pending_review"
	policyObservedReopened   policyReviewEvent = "catalog_reopened"
	policyObservedExpired    policyReviewEvent = "catalog_expired"
	policyObservedDenied     policyReviewEvent = "catalog_denied"
	policyApprovalVerified   policyReviewEvent = "approval_verified"
	policyApprovalUnverified policyReviewEvent = "approval_unverified"
	policyReviewDue          policyReviewEvent = "review_due"
	policyWindowExpired      policyReviewEvent = "window_expired"
)

// transitionPolicyReview projects the next owner observation. All declared
// states can be superseded by another catalog revision. Approval events must
// come from observePolicyReview, which enforces the existing verification seam.
func transitionPolicyReview(state policy.ReviewStatus, event policyReviewEvent) (policy.ReviewStatus, error) {
	if err := checkPolicyReviewState(state); err != nil {
		return state, err
	}
	switch event {
	case policyObservedProposed:
		return policy.ReviewProposed, nil
	case policyObservedPending:
		return policy.ReviewPending, nil
	case policyObservedReopened, policyApprovalUnverified, policyReviewDue:
		return policy.ReviewReopened, nil
	case policyObservedExpired, policyWindowExpired:
		return policy.ReviewExpired, nil
	case policyObservedDenied:
		return policy.ReviewDenied, nil
	case policyApprovalVerified:
		return policy.ReviewApproved, nil
	default:
		return state, fmt.Errorf("unknown provider review observation %q", event)
	}
}

func checkPolicyReviewState(state policy.ReviewStatus) error {
	switch state {
	case policy.ReviewProposed, policy.ReviewPending, policy.ReviewApproved, policy.ReviewReopened, policy.ReviewExpired, policy.ReviewDenied:
		return nil
	default:
		return fmt.Errorf("unknown provider review status %q", state)
	}
}

// observePolicyReview never promotes a proposed/pending/reopened/expired/denied
// catalog, even if a previously approved decision remains available in Hub.
func observePolicyReview(terms policy.ProviderTerms, now time.Time, verify func(string) bool) (policyReviewEvent, error) {
	switch terms.Review.Status {
	case policy.ReviewProposed:
		return policyObservedProposed, nil
	case policy.ReviewPending:
		return policyObservedPending, nil
	case policy.ReviewReopened:
		return policyObservedReopened, nil
	case policy.ReviewExpired:
		return policyObservedExpired, nil
	case policy.ReviewDenied:
		return policyObservedDenied, nil
	case policy.ReviewApproved:
		if terms.Review.DecisionRef == "" || verify == nil || !verify(terms.Review.DecisionRef) {
			return policyApprovalUnverified, nil
		}
		if policyReviewDeadlineElapsed(terms.ExpiresAt, now) {
			return policyWindowExpired, nil
		}
		if policyReviewDeadlineElapsed(terms.NextReviewAt, now) {
			return policyReviewDue, nil
		}
		return policyApprovalVerified, nil
	default:
		return "", checkPolicyReviewState(terms.Review.Status)
	}
}

// Catalog date windows include the whole declared UTC day, matching policy's
// eligibility boundary. Invalid dates fail closed. Hub timestamps are verified
// separately at their exact instant by VerifyProviderDecisionReceipt.
func policyReviewDeadlineElapsed(date string, now time.Time) bool {
	if date == "" {
		return false
	}
	deadline, err := time.Parse("2006-01-02", date)
	return err != nil || !now.Before(deadline.Add(24*time.Hour))
}
