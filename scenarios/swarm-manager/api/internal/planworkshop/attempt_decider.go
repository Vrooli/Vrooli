package planworkshop

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/attempt"
)

const AttemptSubjectCandidate = "plan-workshop-candidate"

// AttemptDecider adapts the final Plan Manager candidate authorization to the
// common operator decision envelope. Service.ApplyCandidate and
// Service.DiscardCandidate remain the only mutation authorities.
type AttemptDecider struct{ service *Service }

func NewAttemptDecider(service *Service) *AttemptDecider { return &AttemptDecider{service: service} }

func (d *AttemptDecider) DecideAttempt(ctx context.Context, request attempt.DecisionRequest) (attempt.DecisionResult, error) {
	if d == nil || d.service == nil {
		return attempt.DecisionResult{}, fmt.Errorf("plan workshop candidate decisions are unavailable")
	}
	parts := strings.Split(strings.TrimSpace(request.SubjectRef), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return attempt.DecisionResult{}, fmt.Errorf("plan-workshop-candidate subject_ref must be workshop_id/response_id")
	}
	if request.RoundNum != 1 {
		return attempt.DecisionResult{}, fmt.Errorf("plan workshop candidates have one attempt round")
	}
	var (
		resolution Resolution
		err        error
	)
	switch request.Decision {
	case "accept":
		_, resolution, err = d.service.ApplyCandidate(ctx, parts[0], parts[1], true)
	case "drop", "fail":
		reason := strings.TrimSpace(request.Rationale)
		if reason == "" {
			reason = "operator " + strings.TrimSpace(request.Actor) + " declined candidate"
		} else {
			reason = "operator " + strings.TrimSpace(request.Actor) + ": " + reason
		}
		_, resolution, err = d.service.DiscardCandidate(ctx, parts[0], parts[1], reason)
	default:
		return attempt.DecisionResult{}, fmt.Errorf("plan workshop candidate decision %q is unsupported", request.Decision)
	}
	if err != nil {
		return attempt.DecisionResult{}, err
	}
	return attempt.DecisionResult{Decision: request.Decision, Status: string(resolution.State), Rationale: request.Rationale, DecidedAt: resolution.AppliedAt}, nil
}
