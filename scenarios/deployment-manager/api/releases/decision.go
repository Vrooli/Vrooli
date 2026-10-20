package releases

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/operatordecision"
)

// ReleaseDecisionRaiser is the seam the release handler uses to raise an
// operator decision when an agent asks to ship a release that has no approved
// readiness review yet ("ask, don't block"). operatordecision.Raiser (wired in
// server.go over the pending-ask store) satisfies it. Keeping it an interface
// avoids pulling the substrate's concrete storage wiring into this package's
// direct tests.
type ReleaseDecisionRaiser interface {
	Raise(ctx context.Context, ask operatordecision.PendingAsk, facts operatordecision.Facts) (bool, error)
}

// ReleaseGateLookup resolves the human-readable gate verdict for a profile and
// commit so the operator Ask renders exactly what is being approved. It is a
// closure injected from server.go (over the approvals repository) to avoid a
// releases -> deployments import cycle; perTarget is already flattened into the
// string map shape the Ask note uses.
type ReleaseGateLookup func(ctx context.Context, profileID, commit string) (ready bool, reason string, perTarget []map[string]string, err error)

// WithReleaseDecisionRaiser installs the ask-don't-block raiser. Without it, an
// agent lacking an approved review keeps the prior hard-refuse behavior (no
// regression for isolated tests or a deployment that opts out).
func (h *Handler) WithReleaseDecisionRaiser(raiser ReleaseDecisionRaiser) *Handler {
	h.decisionRaiser = raiser
	return h
}

// WithReleaseGateLookup installs the gate-verdict lookup used to build the
// operator Ask. It is optional: without it the Ask still renders the release
// identity, just without the gate reason / per-target evidence note.
func (h *Handler) WithReleaseGateLookup(lookup ReleaseGateLookup) *Handler {
	h.gateLookup = lookup
	return h
}

// releaseDecisionOutcome is the result of an ask-don't-block attempt in the
// agent Start path. raised=true means the handler already answered (202 pending
// or 429 flood); the caller returns. raised=false means the raiser is not
// configured, so the caller falls through to the prior hard-refuse.
type releaseDecisionOutcome struct {
	raised        bool
	correlationID string
}

// releaseDecisionCorrelationID binds the ask to exactly one artifact and target
// set. A different commit, digest, or target set hashes to a different id, so an
// approved "yes" authorizes only that artifact (single-use, artifact-bound) and
// an agent retry for the SAME (profile, commit, digest, targets) reuses the one
// pending ask instead of raising a duplicate.
func releaseDecisionCorrelationID(profileID, commit, artifactDigest string, targets []string) string {
	sorted := append([]string(nil), targets...)
	sort.Strings(sorted)
	return operatordecision.CorrelationID("release", profileID, commit, artifactDigest, strings.Join(sorted, ","))
}

// maybeRaiseReleaseDecision raises an operator decision for an AGENT caller that
// asked to ship a release whose readiness review exists but is not yet
// approved. It returns raised=false (so the caller keeps the existing 412) when
// no raiser is configured or the caller is not an agent. The decision never
// grants authority: only the resolution webhook does, after re-validating.
func (h *Handler) maybeRaiseReleaseDecision(ctx context.Context, profileID string, body StartRequest, channel string, platforms []string, review *ReadinessApproval) (releaseDecisionOutcome, int, string) {
	if h.decisionRaiser == nil {
		return releaseDecisionOutcome{}, 0, ""
	}
	principal, _ := identity.PrincipalFromContext(ctx)
	if principal.Kind != identity.ActorAgent {
		return releaseDecisionOutcome{}, 0, ""
	}

	scenario, policyVersion := "", 0
	if review != nil {
		scenario = review.Scenario
		policyVersion = review.PolicyVersion
	}
	correlationID := releaseDecisionCorrelationID(profileID, body.GitCommitHash, body.ArtifactDigest, platforms)

	gateReady, gateReason, perTarget := false, "", []map[string]string(nil)
	if h.gateLookup != nil {
		ready, reason, targets, err := h.gateLookup(ctx, profileID, body.GitCommitHash)
		if err == nil {
			gateReady, gateReason, perTarget = ready, reason, targets
		} else {
			gateReason = fmt.Sprintf("gate status unavailable: %v", err)
		}
	}

	facts := buildReleaseDecisionFacts(releaseDecisionInputs{
		Scenario:        scenario,
		ProfileID:       profileID,
		Commit:          body.GitCommitHash,
		ArtifactDigest:  body.ArtifactDigest,
		CandidateID:     body.CandidateID,
		DestinationRev:  body.DestinationRevisionID,
		Channel:         channel,
		Targets:         platforms,
		Epoch:           body.AuthorizationEpoch,
		ReviewKey:       body.ReadinessReviewKey,
		PolicyVersion:   policyVersion,
		RequestingAgent: principal.Subject,
		GateReady:       gateReady,
		GateReason:      gateReason,
		PerTarget:       perTarget,
	})
	if err := facts.Validate(); err != nil {
		return releaseDecisionOutcome{}, 0, fmt.Sprintf("build release decision: %v", err)
	}

	ask := operatordecision.PendingAsk{
		ID:      correlationID,
		Subject: body.ReadinessReviewKey,
		Extra: map[string]string{
			"review_key":              body.ReadinessReviewKey,
			"scenario":                scenario,
			"profile_id":              profileID,
			"candidate_commit":        body.GitCommitHash,
			"artifact_digest":         body.ArtifactDigest,
			"candidate_id":            body.CandidateID,
			"destination_revision_id": body.DestinationRevisionID,
			"channel":                 channel,
			"targets":                 strings.Join(sortedCopy(platforms), ","),
			"authorization_epoch":     strconv.FormatUint(body.AuthorizationEpoch, 10),
			"requesting_agent":        principal.Subject,
		},
	}

	raised, err := h.decisionRaiser.Raise(ctx, ask, facts)
	if err != nil {
		if err == operatordecision.ErrFlood {
			return releaseDecisionOutcome{raised: true}, 429, "too many outstanding release approval requests for this review"
		}
		// A best-effort publish failure still leaves the durable ask standing;
		// surface it so the operator can act from the inbox and a retry
		// re-attempts the notification.
		if raised {
			return releaseDecisionOutcome{raised: true, correlationID: correlationID}, 0, ""
		}
		return releaseDecisionOutcome{}, 0, fmt.Sprintf("raise release decision: %v", err)
	}
	if !raised {
		// No store configured: degrade to the prior hard-refuse.
		return releaseDecisionOutcome{}, 0, ""
	}
	return releaseDecisionOutcome{raised: true, correlationID: correlationID}, 0, ""
}

type releaseDecisionInputs struct {
	Scenario        string
	ProfileID       string
	Commit          string
	ArtifactDigest  string
	CandidateID     string
	DestinationRev  string
	Channel         string
	Targets         []string
	Epoch           uint64
	ReviewKey       string
	PolicyVersion   int
	RequestingAgent string
	GateReady       bool
	GateReason      string
	PerTarget       []map[string]string
}

// buildReleaseDecisionFacts renders the operator Ask from the readiness-review
// identity and the gate verdict. A release is NOT reversible, so there is no
// DefaultAnswer (no silent auto-apply at a deadline). The requesting agent is
// carried in the correlation DISTINCT from the eventual approver (the resolution
// carries answered_by), preserving independence.
func buildReleaseDecisionFacts(in releaseDecisionInputs) operatordecision.Facts {
	commitShort := in.Commit
	if len(commitShort) > 12 {
		commitShort = commitShort[:12]
	}
	scenarioLabel := in.Scenario
	if scenarioLabel == "" {
		scenarioLabel = in.ProfileID
	}
	recommended := "reject"
	reason := "the release gate is not ready"
	if in.GateReason != "" {
		reason = in.GateReason
	}
	if in.GateReady {
		recommended = "approve"
		if in.GateReason == "" {
			reason = "all required readiness evidence and target approvals are satisfied"
		}
	}

	note := strings.Builder{}
	fmt.Fprintf(&note, "Requesting agent: %s\n", in.RequestingAgent)
	fmt.Fprintf(&note, "Channel: %s · Targets: %s\n", in.Channel, strings.Join(sortedCopy(in.Targets), ", "))
	fmt.Fprintf(&note, "Artifact digest: %s\n", in.ArtifactDigest)
	if in.CandidateID != "" {
		fmt.Fprintf(&note, "Candidate: %s · Destination revision: %s · Authorization epoch: %d\n", in.CandidateID, in.DestinationRev, in.Epoch)
	}
	fmt.Fprintf(&note, "Readiness review: %s\n", in.ReviewKey)
	if len(in.PerTarget) > 0 {
		note.WriteString("Per-target evidence:\n")
		for _, t := range in.PerTarget {
			fmt.Fprintf(&note, "  - %s: evidence=%s approval=%s\n", t["target"], t["evidence_disposition"], t["approval_status"])
		}
	}

	noteText := note.String()
	if len(noteText) > operatordecision.MaxQuestion {
		noteText = noteText[:operatordecision.MaxQuestion]
	}
	gateReason := reason
	if len(gateReason) > operatordecision.MaxReason {
		gateReason = gateReason[:operatordecision.MaxReason]
	}

	return operatordecision.Facts{
		Subject:  fmt.Sprintf("Release %s %s needs approval", scenarioLabel, commitShort),
		Question: fmt.Sprintf("Approve shipping %s at commit %s to %s (targets: %s)?", scenarioLabel, commitShort, in.Channel, strings.Join(sortedCopy(in.Targets), ", ")),
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Approve and ship"},
			{Key: "reject", Label: "Do not ship"},
		},
		Recommended:          recommended,
		RecommendationReason: gateReason,
		// A release cannot be undone: no DefaultAnswer, Reversible stays false so
		// the hub never auto-applies at a deadline.
		Reversible: false,
		Urgency:    "high",
		Note:       noteText,
		Correlation: map[string]string{
			"review_key":              in.ReviewKey,
			"scenario":                in.Scenario,
			"profile_id":              in.ProfileID,
			"candidate_commit":        in.Commit,
			"artifact_digest":         in.ArtifactDigest,
			"candidate_id":            in.CandidateID,
			"destination_revision_id": in.DestinationRev,
			"channel":                 in.Channel,
			"targets":                 strings.Join(sortedCopy(in.Targets), ","),
			"authorization_epoch":     strconv.FormatUint(in.Epoch, 10),
			"policy_version":          strconv.Itoa(in.PolicyVersion),
			"requesting_agent":        in.RequestingAgent,
		},
	}
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// pendingResponse is the 202 body an agent receives while an operator decides.
func pendingResponse(correlationID, reviewKey string) map[string]interface{} {
	return map[string]interface{}{
		"status":                  "pending",
		"decision_correlation_id": correlationID,
		"readiness_review_key":    reviewKey,
		"message":                 "release approval requested from an operator",
	}
}
