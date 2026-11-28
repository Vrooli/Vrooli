package main

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/backlog"
)

func (s *Server) deterministicApplyActions() map[string]struct{} {
	return map[string]struct{}{
		"apply_proposal":     {},
		"dispatch_follow_up": {},
		"mark_goal_achieved": {},
	}
}

// sessionApplyActions documents the existing Agent Session-owned projection.
// Session transitions are catalog-visible only; their behavior intentionally
// remains on the Agent Session path rather than the workflow runner.
func (s *Server) sessionApplyActions() map[string]struct{} {
	if s.agentSessionSvc == nil {
		return map[string]struct{}{}
	}
	return map[string]struct{}{"apply_session_proposal": {}}
}

func mergeApplyActions(groups ...map[string]struct{}) map[string]struct{} {
	merged := map[string]struct{}{}
	for _, group := range groups {
		for action := range group {
			merged[action] = struct{}{}
		}
	}
	return merged
}

// Dispatch executes the registry's deterministic transitions. They mutate a
// domain immediately and intentionally do not create workflow correlations.
func (s *Server) Dispatch(ctx context.Context, transitionKey, subjectRef string) (string, error) {
	switch transitionKey {
	case "proposal.apply":
		parts := strings.SplitN(subjectRef, "/", 2)
		if len(parts) != 2 || s.agentSessionSvc == nil {
			return "", fmt.Errorf("proposal subject must be session_id/proposal_id")
		}
		if _, _, err := s.agentSessionSvc.ApplyProposal(ctx, parts[0], parts[1]); err != nil {
			return "", err
		}
		return "applied", nil
	case "follow_up.dispatch":
		parts := strings.SplitN(subjectRef, "/", 2)
		if len(parts) != 2 || s.backlogHandler == nil {
			return "", fmt.Errorf("follow-up subject must be kind/name")
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			return "", err
		}
		if _, _, err := s.backlogHandler.DispatchPendingFollowUp(ctx, kind, parts[1]); err != nil {
			return "", err
		}
		return "dispatched", nil
	case "goal.close_out":
		if s.goalsHandler == nil {
			return "", fmt.Errorf("goal service is not configured")
		}
		if _, err := s.goalsHandler.CloseOutGoal(subjectRef); err != nil {
			return "", err
		}
		return "achieved", nil
	default:
		return "", fmt.Errorf("transition %q is not deterministic", transitionKey)
	}
}
