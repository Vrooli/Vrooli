package agentsessions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/attempt"
)

const AttemptSubjectProposal = "agent-session-proposal"

// AttemptDecider adapts durable mutation proposals to the shared operator
// decision envelope. The session service remains the sole mutation authority.
type AttemptDecider struct{ service *Service }

func NewAttemptDecider(service *Service) *AttemptDecider { return &AttemptDecider{service: service} }

func (d *AttemptDecider) DecideAttempt(ctx context.Context, request attempt.DecisionRequest) (attempt.DecisionResult, error) {
	if d == nil || d.service == nil {
		return attempt.DecisionResult{}, fmt.Errorf("agent-session proposal decisions are unavailable")
	}
	parts := strings.Split(strings.TrimSpace(request.SubjectRef), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return attempt.DecisionResult{}, fmt.Errorf("agent-session-proposal subject_ref must be session_id/proposal_id")
	}
	if request.RoundNum != 1 {
		return attempt.DecisionResult{}, fmt.Errorf("agent-session proposals have one attempt round")
	}
	accepted := request.AcceptedProposalIDs
	switch request.Decision {
	case "accept":
	case "fail", "drop":
		accepted = nil
	default:
		return attempt.DecisionResult{}, fmt.Errorf("proposal decision %q is unsupported", request.Decision)
	}
	note := strings.TrimSpace(request.Rationale)
	if note == "" {
		note = "operator decision"
	}
	note = "operator " + strings.TrimSpace(request.Actor) + ": " + note
	session, err := d.service.DecideMutationListProposal(ctx, parts[0], parts[1], accepted, note)
	if err != nil {
		return attempt.DecisionResult{}, err
	}
	for _, proposal := range session.Proposals {
		if proposal.ID != parts[1] {
			continue
		}
		decidedAt := time.Now().UTC().Format(time.RFC3339Nano)
		if len(proposal.Decisions) > 0 {
			decidedAt = proposal.Decisions[len(proposal.Decisions)-1].DecidedAt
		}
		return attempt.DecisionResult{Decision: request.Decision, Status: string(proposal.Status), Rationale: request.Rationale, DecidedAt: decidedAt}, nil
	}
	return attempt.DecisionResult{}, fmt.Errorf("proposal %q disappeared after decision", parts[1])
}
