package operatingmode

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
)

func (s *Service) RefreshRound(ctx context.Context, initiativeName string, mode Mode, number int) (RoundEnvelope, error) {
	round, err := s.store.LoadRound(initiativeName, mode, number)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if !isRoundActive(round) || strings.TrimSpace(round.RunID) == "" || s.agent == nil {
		return round, nil
	}
	state, err := s.agent.GetRunState(ctx, round.RunID)
	if err != nil {
		return RoundEnvelope{}, err
	}
	switch strings.ToLower(strings.TrimSpace(state.Status)) {
	case "complete":
		round.Status = RoundStatusCompleted
		round.Payload = ensurePayload(round.Payload)
		round.Payload["agent_summary"] = strings.TrimSpace(state.Summary)
		round.Payload["finished_at"] = finishTime(state, s.clock)
		if err := s.applyPhaseResult(&round, state.Summary); err != nil {
			round.Error = err.Error()
			slog.Warn("operating mode: parse/apply phase result failed", "err", err, "initiative", round.InitiativeName, "round", round.Round)
		}
		s.emitPhaseCompleted(round)
		s.emitParsedPhaseSignals(round)
		_ = s.lock.Release(round.InitiativeName, round.RunID)
	case "failed":
		round.Status = RoundStatusFailed
		round.Error = strings.TrimSpace(state.ErrorMsg)
		if round.Error == "" {
			round.Error = "agent run failed"
		}
		round.Payload = ensurePayload(round.Payload)
		round.Payload["finished_at"] = finishTime(state, s.clock)
		s.emitPhaseFailed(round, round.Error)
		_ = s.lock.Release(round.InitiativeName, round.RunID)
	case "cancelled":
		round.Status = RoundStatusCanceled
		round.Payload = ensurePayload(round.Payload)
		round.Payload["finished_at"] = finishTime(state, s.clock)
		s.emitPhaseCanceled(round)
		_ = s.lock.Release(round.InitiativeName, round.RunID)
	default:
		return round, nil
	}
	if err := s.store.SaveRound(round); err != nil {
		return RoundEnvelope{}, err
	}
	return round, nil
}

func (s *Service) CancelRound(ctx context.Context, initiativeName string, mode Mode, number int) (RoundEnvelope, error) {
	round, err := s.store.LoadRound(initiativeName, mode, number)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if strings.TrimSpace(round.RunID) != "" && s.agent != nil && isRoundActive(round) {
		if err := s.agent.StopRun(ctx, round.RunID); err != nil {
			return RoundEnvelope{}, err
		}
	}
	round.Status = RoundStatusCanceled
	round.Payload = ensurePayload(round.Payload)
	round.Payload["canceled_at"] = s.clock().UTC().Format(time.RFC3339)
	if err := s.store.SaveRound(round); err != nil {
		return RoundEnvelope{}, err
	}
	if round.RunID != "" {
		_ = s.lock.Release(initiativeName, round.RunID)
	}
	s.emitPhaseCanceled(round)
	return round, nil
}

func isRoundActive(round RoundEnvelope) bool {
	return round.Status == RoundStatusReserved || round.Status == RoundStatusAgentRunning
}

func finishTime(state agentmanager.RunState, clock func() time.Time) string {
	if strings.TrimSpace(state.FinishedAt) != "" {
		return strings.TrimSpace(state.FinishedAt)
	}
	return clock().UTC().Format(time.RFC3339)
}
