package readinesshandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"deployment-manager/readiness"

	"github.com/vrooli/api-core/operatordecision"
)

// ReleaseDecisionSource / Kind compose the producer event type this effect owns:
// deployment-manager.release.decision_requested.v1. It mirrors the Raiser wired
// in server.go so a release ask resolves back to exactly this effect.
const (
	ReleaseDecisionSource = "deployment-manager"
	ReleaseDecisionKind   = "release"
)

// PendingResolver is the subset of the pending-ask store the resolution effect
// needs: look an ask up by its correlation id and record its terminal state.
// internal/releasedecision.SQLStore satisfies it.
type PendingResolver interface {
	Get(ctx context.Context, id string) (*operatordecision.PendingAsk, error)
	Put(ctx context.Context, ask *operatordecision.PendingAsk) error
}

// ApprovalLedger is the optional deployment-approval ledger mirror. When the
// resolution carries an approval id, the resolver records the same verdict under
// the human subject so the bespoke approvals view stays consistent. Satisfied by
// deployments.ApprovalsRepository (UpdateDecision).
type ApprovalLedger interface {
	UpdateDecision(ctx context.Context, id, decision, reviewer, notes string) error
}

// ReleaseDecisionResolver applies a notification-hub ask resolution for a
// release decision raised by the ask-don't-block path. It re-validates the
// durable review before recording any approval, binds the approval to the
// verified human (answered_by), and is single-use + artifact-bound through the
// pending-ask store and the canonical review key.
type ReleaseDecisionResolver struct {
	handler *ConnectHandler
	pending PendingResolver
	ledger  ApprovalLedger
	now     func() time.Time
	log     func(string, map[string]any)
}

// NewReleaseDecisionResolver builds the resolver over the connect handler (whose
// approveReviewCore + revalidate it reuses) and the pending-ask store. ledger and
// log are optional.
func NewReleaseDecisionResolver(handler *ConnectHandler, pending PendingResolver, ledger ApprovalLedger, log func(string, map[string]any)) *ReleaseDecisionResolver {
	now := time.Now
	if handler != nil && handler.now != nil {
		now = handler.now
	}
	if log == nil {
		log = func(string, map[string]any) {}
	}
	return &ReleaseDecisionResolver{handler: handler, pending: pending, ledger: ledger, now: now, log: log}
}

// Handles reports whether this effect owns the producer event type.
func (r *ReleaseDecisionResolver) Handles(sourceEventType string) bool {
	return sourceEventType == operatordecision.DecisionRequestedEventType(ReleaseDecisionSource, ReleaseDecisionKind)
}

// Apply records the operator's decision. It is deliberately permissive about
// unknown/already-decided asks (benign handled=true no-op) so a redelivery never
// triggers a retry storm, and it NEVER approves without re-validating the
// durable review first.
func (r *ReleaseDecisionResolver) Apply(ctx context.Context, resolution operatordecision.DecisionResolution) (bool, error) {
	if r == nil || r.handler == nil || r.pending == nil {
		return false, nil
	}
	reviewKey := strings.TrimSpace(resolution.Correlation["review_key"])
	if reviewKey == "" {
		return false, nil
	}

	// Locate the pending ask. Subject is the review key; the ask id is the
	// artifact-bound correlation id, which the resolution does not carry back, so
	// reconstruct it from the correlation's artifact identity.
	askID := operatordecision.CorrelationID("release",
		resolution.Correlation["profile_id"],
		resolution.Correlation["candidate_commit"],
		resolution.Correlation["artifact_digest"],
		strings.TrimSpace(resolution.Correlation["targets"]))
	ask, err := r.pending.Get(ctx, askID)
	if err != nil {
		return true, err
	}
	// SINGLE-USE: an already-resolved (or unknown) ask is a benign no-op. This is
	// what refuses a replay of a prior resolution.
	if ask == nil || ask.State != operatordecision.PendingStatePending {
		r.log("release decision resolution ignored (unknown or already resolved ask)", map[string]any{"ask_id": askID, "review_key": reviewKey})
		return true, nil
	}

	approvedBy := strings.TrimSpace(resolution.AnsweredBy)
	answer := strings.TrimSpace(resolution.Answer)
	approvalID := strings.TrimSpace(resolution.Correlation["approval_id"])

	// A rejection (explicit, or an expired/defaulted non-approve resolution)
	// records the refusal and ships nothing.
	if answer != "approve" {
		// Best-effort on the reject path: nothing was shipped, so a failed store
		// write only risks the operator re-seeing a rejected ask, not a double ship.
		_ = r.markResolved(ctx, ask, operatordecision.PendingStateRejected, approvedBy)
		if r.ledger != nil && approvalID != "" {
			if lerr := r.ledger.UpdateDecision(ctx, approvalID, "rejected", approvedBy, resolution.Note); lerr != nil {
				r.log("release decision ledger reject failed", map[string]any{"approval_id": approvalID, "error": lerr.Error()})
			}
		}
		return true, nil
	}

	// ARTIFACT-BOUND re-validation: recompute the canonical review key from the
	// correlation's identity fields and refuse if it does not match the stored
	// review (a replay against a different artifact cannot ship). approveReviewCore
	// repeats this check against the stored review and re-runs ValidateCurrentEvidence
	// + goal closure, so a regressed-evidence approval is refused here.
	requestIdentity := r.identityFromCorrelation(resolution.Correlation)
	recomputedKey, keyErr := requestIdentity.Key()
	if keyErr != nil || recomputedKey != reviewKey {
		r.log("release decision refused: identity does not match review key", map[string]any{"ask_id": askID, "review_key": reviewKey, "error": errString(keyErr)})
		_ = r.markResolved(ctx, ask, operatordecision.PendingStateRejected, approvedBy)
		return true, nil
	}

	if err := r.handler.approveReviewCore(ctx, reviewKey, requestIdentity, approvedBy); err != nil {
		// A regressed-evidence / mismatched approval is refused WITHOUT recording an
		// approval. Mark the ask resolved-rejected so no retry storm or replay, and
		// return handled=true (the operator can re-raise once evidence is restored).
		r.log("release decision approval refused at re-validation", map[string]any{"ask_id": askID, "review_key": reviewKey, "error": err.Error()})
		_ = r.markResolved(ctx, ask, operatordecision.PendingStateRejected, approvedBy)
		return true, nil
	}

	// Durable approval recorded under the HUMAN subject. The pending ask MUST flip
	// to resolved now, or the single-use guard is defeated: the review is approved
	// yet the ask stays pending (re-raisable, re-approvable). So on the success
	// path a failed store write is surfaced (handled=false, err) — vrooli-events
	// retries the delivery, the already-approved review makes approveReviewCore a
	// safe no-op, and the retry reaches the Put again instead of leaving the ask
	// stuck pending forever.
	if err := r.markResolved(ctx, ask, operatordecision.PendingStateApproved, approvedBy); err != nil {
		r.log("release decision approved but pending store update failed; will retry on redelivery", map[string]any{"ask_id": askID, "review_key": reviewKey, "error": err.Error()})
		return false, fmt.Errorf("record release decision resolution: %w", err)
	}
	// Mirror the verdict onto the bespoke approvals ledger when an approval id is
	// carried. A ledger failure is non-fatal: the authoritative approval already
	// stands on the readiness review.
	if r.ledger != nil && approvalID != "" {
		if lerr := r.ledger.UpdateDecision(ctx, approvalID, "approved", approvedBy, resolution.Note); lerr != nil {
			r.log("release decision ledger approve failed", map[string]any{"approval_id": approvalID, "error": lerr.Error()})
		}
	}
	return true, nil
}

// markResolved flips the pending ask to a terminal state and persists it. It
// returns the store error so the caller can decide whether a failed write is
// fatal: on the approve-success path it MUST be surfaced (the approval already
// stands, so the ask must not linger pending), while the reject/refusal paths
// treat it as best-effort and only log.
func (r *ReleaseDecisionResolver) markResolved(ctx context.Context, ask *operatordecision.PendingAsk, state, by string) error {
	now := r.now().UTC()
	ask.State = state
	ask.ResolvedAt = &now
	ask.ResolvedBy = by
	if err := r.pending.Put(ctx, ask); err != nil {
		r.log("release decision pending store update failed", map[string]any{"ask_id": ask.ID, "error": err.Error()})
		return err
	}
	return nil
}

func (r *ReleaseDecisionResolver) identityFromCorrelation(c map[string]string) readiness.ReviewIdentity {
	targets := []string{}
	if raw := strings.TrimSpace(c["targets"]); raw != "" {
		targets = strings.Split(raw, ",")
	}
	epoch, _ := strconv.ParseUint(strings.TrimSpace(c["authorization_epoch"]), 10, 64)
	policyVersion, _ := strconv.Atoi(strings.TrimSpace(c["policy_version"]))
	return readiness.ReviewIdentity{
		Scenario:              c["scenario"],
		ProfileID:             c["profile_id"],
		CandidateCommit:       c["candidate_commit"],
		ArtifactDigest:        c["artifact_digest"],
		Targets:               targets,
		Channel:               c["channel"],
		PolicyVersion:         policyVersion,
		CandidateID:           c["candidate_id"],
		DestinationRevisionID: c["destination_revision_id"],
		AuthorizationEpoch:    epoch,
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

var _ operatordecision.DomainEffect = (*ReleaseDecisionResolver)(nil)
