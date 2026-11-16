package releases

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"deployment-manager/internal/releasedecision"

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
			releasedecision.ExtraSubjectKey: facts.SubjectKey,
			"review_key":                    body.ReadinessReviewKey,
			"scenario":                      scenario,
			"profile_id":                    profileID,
			"candidate_commit":              body.GitCommitHash,
			"artifact_digest":               body.ArtifactDigest,
			"candidate_id":                  body.CandidateID,
			"destination_revision_id":       body.DestinationRevisionID,
			"channel":                       channel,
			"targets":                       strings.Join(sortedCopy(platforms), ","),
			"authorization_epoch":           strconv.FormatUint(body.AuthorizationEpoch, 10),
			"requesting_agent":              principal.Subject,
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
// identity and the gate verdict: a cover a stranger can answer from the push,
// and a page with the release, its readiness and the evidence per target. A
// release is NOT reversible, so there is no DefaultAnswer (no silent
// auto-apply at a deadline). The requesting agent is carried in the
// correlation DISTINCT from the eventual approver (the resolution carries
// answered_by), preserving independence.
func buildReleaseDecisionFacts(in releaseDecisionInputs) operatordecision.Facts {
	commitShort := in.Commit
	if len(commitShort) > 12 {
		commitShort = commitShort[:12]
	}
	app := in.Scenario
	if app == "" {
		app = in.ProfileID
	}
	targets := strings.Join(sortedCopy(in.Targets), ", ")
	recommended, reason, readiness := "reject", "the release gate is not ready", "Its readiness check has not passed."
	if in.GateReason != "" {
		reason = in.GateReason
	}
	if in.GateReady {
		recommended, readiness = "approve", "Its readiness check passed."
		if in.GateReason == "" {
			reason = "all required readiness evidence and target approvals are satisfied"
		}
	}

	var tech strings.Builder
	fmt.Fprintf(&tech, "Requesting agent: %s\nProfile: %s · Commit: %s\nChannel: %s · Targets: %s\nArtifact digest: %s\n", in.RequestingAgent, in.ProfileID, in.Commit, in.Channel, targets, in.ArtifactDigest)
	if in.CandidateID != "" {
		fmt.Fprintf(&tech, "Candidate: %s · Destination revision: %s · Authorization epoch: %d\n", in.CandidateID, in.DestinationRev, in.Epoch)
	}
	fmt.Fprintf(&tech, "Readiness review: %s · Policy version: %d\nGate: ready=%t reason=%s\n", in.ReviewKey, in.PolicyVersion, in.GateReady, in.GateReason)
	for _, t := range in.PerTarget {
		fmt.Fprintf(&tech, "  - %s: evidence=%s approval=%s\n", t["target"], t["evidence_disposition"], t["approval_status"])
	}

	facts := operatordecision.Facts{
		// One open decision per profile, channel and target set: a newer release
		// updates or replaces it, and the reconciler closes it when it lapses or
		// is approved elsewhere.
		SubjectKey: releasedecision.SubjectKey(in.ProfileID, in.Channel, in.Targets),
		Subject:    fmt.Sprintf("Ship %s to %s?", app, in.Channel),
		Question:   fmt.Sprintf("Ship this build of %s to the %s channel for %s? %s", app, in.Channel, targets, readiness),
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Ship it: release this exact build to those targets"},
			{Key: "reject", Label: "Do not ship: the agent is told no and nothing is released"},
		},
		Recommended:          recommended,
		RecommendationReason: operatordecision.Clip(reason, operatordecision.MaxReason),
		// A release cannot be undone: no DefaultAnswer, Reversible stays false so
		// the hub never auto-applies at a deadline.
		Reversible: false,
		Urgency:    "high",
		BigPicture: fmt.Sprintf("An agent wants to release a new build of %s to the %s channel for %s, and its readiness review has not been approved yet, so it asks you. %s A release reaches users and cannot be taken back.\n\n"+
			"If you approve, Deployment Manager ships this exact build after checking it is still the same artifact. If you say no or do not answer, nothing is released.", app, in.Channel, targets, readiness),
		TechnicalDetails: tech.String(),
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
	tone, strong := "danger", "Not ready:"
	if in.GateReady {
		tone, strong = "success", "Ready:"
	}
	page := operatordecision.NewPage(fmt.Sprintf("Ship %s to %s?", app, in.Channel)).
		Lead("An agent asks to release this build. Its readiness review is not approved yet, so the release waits for you.").
		Facts(
			operatordecision.Row{Label: "App", Value: app},
			operatordecision.Row{Label: "Build", Value: "commit " + commitShort},
			operatordecision.Row{Label: "Channel", Value: in.Channel},
			operatordecision.Row{Label: "Targets", Value: targets},
			operatordecision.Row{Label: "Asked by", Value: in.RequestingAgent},
		).
		Callout(tone, strong, reason)
	rows := make([][]string, 0, len(in.PerTarget))
	for _, t := range in.PerTarget {
		rows = append(rows, []string{t["target"], plainStatus(t["evidence_disposition"]), plainStatus(t["approval_status"])})
	}
	page.Section("Evidence per target", "").Table([]string{"Target", "Readiness evidence", "Approval"}, rows)
	facts.Explainer = page.Callout("warning", "Cannot be undone:", "a release reaches users. A later fix is a new release.").
		Options(facts, map[string]string{
			"approve": "Deployment Manager checks the build is still this exact artifact, then ships it to the listed targets. This approves only this build.",
			"reject":  "Nothing is released. The agent is told no and can ask again with a new build.",
		}).
		IfNoAnswer("Nothing is released. The request closes when it lapses.").
		HTML()
	return facts
}

// plainStatus turns a status key such as "not_found" into "Not found".
func plainStatus(status string) string {
	status = strings.TrimSpace(strings.ReplaceAll(status, "_", " "))
	if status == "" {
		return "Unknown"
	}
	return strings.ToUpper(status[:1]) + status[1:]
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
