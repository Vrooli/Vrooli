package feedback

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// EnsurePolledTurn checks the agent run state for a round in
// RoundStatusAgentThinking and, if the run has reached a terminal state,
// records the agent's output as a turn. Mirrors the clarification pull
// pattern (clarification_state.go:GetClarification) so the UI can advance
// rounds by polling rather than depending on an inbound webhook from the
// agent-manager.
//
// Returns the (possibly updated) round. Callers — typically Handler.Get —
// should invoke this whenever they hand a round to the user. Idempotent
// and safe under poll storms: rounds not in agent_thinking, runs without a
// RunID, missing pollers, and non-terminal run states are no-ops.
func (s *Service) EnsurePolledTurn(ctx context.Context, round Round) (Round, error) {
	if round.Status != RoundStatusAgentThinking {
		return round, nil
	}
	if s.poller == nil || !s.poller.IsEnabled() || strings.TrimSpace(round.RunID) == "" {
		return round, nil
	}
	now := s.clock().UTC().Format(time.RFC3339)
	state, err := s.poller.GetRunState(ctx, round.RunID)
	if err != nil {
		// Polling failures used to be silently logged-and-forgotten,
		// which is exactly how rounds wedged in agent_thinking forever
		// when the agent-manager run died. Now: record the error on the
		// round, increment the failure counter, and synthesize a
		// terminal failure once we've seen enough consecutive failures
		// that the run is clearly gone.
		slog.Warn("feedback: poll run state failed",
			"err", err, "initiative", round.InitiativeName, "round", round.Number, "run_id", round.RunID)
		round.LastPolledAt = now
		round.LastPollError = err.Error()
		round.PollFailureCount++
		if saveErr := s.store.SaveRound(round); saveErr != nil {
			slog.Warn("feedback: persist poll failure failed",
				"err", saveErr, "initiative", round.InitiativeName, "round", round.Number)
		}
		if round.PollFailureCount >= s.pollFailureThreshold() {
			body := fmt.Sprintf("agent run failed: run no longer reachable after %d consecutive poll attempts (last error: %s)",
				round.PollFailureCount, err.Error())
			return s.RecordAgentTurn(round.InitiativeName, round.Number, body)
		}
		return round, nil
	}
	round.LastPolledAt = now
	if !isTerminalRunStatus(state.Status) {
		// Non-terminal poll: clear any prior error so a transient hiccup
		// followed by a recovered run doesn't trip the failure threshold.
		if round.LastPollError != "" || round.PollFailureCount != 0 {
			round.LastPollError = ""
			round.PollFailureCount = 0
			if saveErr := s.store.SaveRound(round); saveErr != nil {
				slog.Warn("feedback: persist poll recovery failed",
					"err", saveErr, "initiative", round.InitiativeName, "round", round.Number)
			}
		}
		return round, nil
	}

	body := strings.TrimSpace(state.Summary)
	if isFailureRunStatus(state.Status) {
		msg := strings.TrimSpace(state.ErrorMsg)
		if msg == "" {
			msg = "agent run failed without an error message"
		}
		body = "agent run failed: " + msg
	}
	if body == "" {
		body = "agent run completed without producing output"
	}
	return s.RecordAgentTurn(round.InitiativeName, round.Number, body)
}

// pollFailureThreshold reports how many consecutive poll failures must be
// observed before EnsurePolledTurn synthesizes a terminal failure turn.
// Reads SWARM_MANAGER_FEEDBACK_POLL_FAILURE_THRESHOLD from the env at call
// time so operators can tune it without restarting; defaults to 3.
func (s *Service) pollFailureThreshold() int {
	const defaultThreshold = 3
	raw := strings.TrimSpace(os.Getenv("SWARM_MANAGER_FEEDBACK_POLL_FAILURE_THRESHOLD"))
	if raw == "" {
		return defaultThreshold
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultThreshold
	}
	return n
}

// isTerminalRunStatus matches the clarification_service.isTerminalStatus
// list. Kept private to feedback so the package doesn't import backlog.
// "not_found" / "missing" are treated as terminal — if agent-manager
// reports the run as gone, there is nothing left to wait for.
func isTerminalRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "complete", "completed", "success",
		"failed", "error", "cancelled", "canceled",
		"not_found", "missing":
		return true
	}
	return false
}

func isFailureRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "cancelled", "canceled", "not_found", "missing":
		return true
	}
	return false
}
