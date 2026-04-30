package operatingmode

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
)

func (s *Service) RefreshRound(ctx context.Context, initiativeName string, mode Mode, number int) (RoundEnvelope, error) {
	mode, err := requireRoundActionMode(mode)
	if err != nil {
		return RoundEnvelope{}, err
	}
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
		payload := MutableRoundPayload(&round)
		payload.SetAgentSummary(state.Summary)
		payload.SetFinishedAt(finishTime(state, s.clock))
		if err := s.applyPhaseResult(&round, state.Summary); err != nil {
			round.Status = RoundStatusFailed
			round.Error = err.Error()
			slog.Warn("operating mode: phase result contract failed", "err", err, "initiative", round.InitiativeName, "round", round.Round)
			s.emitPhaseFailed(round, err.Error())
		} else {
			round.Status = RoundStatusCompleted
			s.emitPhaseCompleted(round)
			s.emitParsedPhaseSignals(round)
		}
		_ = s.lock.Release(round.InitiativeName, round.RunID)
	case "failed":
		round.Status = RoundStatusFailed
		round.Error = strings.TrimSpace(state.ErrorMsg)
		if round.Error == "" {
			round.Error = "agent run failed"
		}
		MutableRoundPayload(&round).SetFinishedAt(finishTime(state, s.clock))
		s.emitPhaseFailed(round, round.Error)
		_ = s.lock.Release(round.InitiativeName, round.RunID)
	case "cancelled":
		round.Status = RoundStatusCanceled
		MutableRoundPayload(&round).SetFinishedAt(finishTime(state, s.clock))
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
	mode, err := requireRoundActionMode(mode)
	if err != nil {
		return RoundEnvelope{}, err
	}
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
	MutableRoundPayload(&round).SetCanceledAt(s.clock().UTC().Format(time.RFC3339))
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
