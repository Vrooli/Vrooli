package operatingmode

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
)

// payloadPendingAutoStart marks the predecessor round as having a pending
// auto-start dispatch that the round refresher should retry on the next
// tick. The marker is set when StartPhase returns ErrLaneSaturated; the
// refresher clears it once the dispatch finally succeeds (or the round is
// canceled out of the active set).
const payloadPendingAutoStart = "pending_auto_start"

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
		// A completed round with a pending_auto_start marker means a previous
		// auto-dispatch was deferred (typically by lane saturation). Retry
		// the dispatch so eventually-consistent capacity recovers without
		// operator intervention.
		if round.Status == RoundStatusCompleted && RoundPayload(round.Payload).HasPendingAutoStart() {
			s.maybeAutoStartNext(ctx, round)
			refreshed, loadErr := s.store.LoadRound(initiativeName, mode, number)
			if loadErr != nil {
				return RoundEnvelope{}, loadErr
			}
			return refreshed, nil
		}
		return round, nil
	}
	state, err := s.agent.GetRunState(ctx, round.RunID)
	if err != nil {
		return RoundEnvelope{}, err
	}
	completedNormally := false
	switch strings.ToLower(strings.TrimSpace(state.Status)) {
	case "complete":
		payload := MutableRoundPayload(&round)
		payload.SetAgentSummary(state.Summary)
		payload.SetFinishedAt(finishTime(state, s.clock))
		resolved, err := s.applyPhaseResult(ctx, &round, s.resolutionCandidates(ctx, round, state))
		switch {
		case err != nil:
			// Infrastructure or post-apply contract failure (e.g. a required
			// artifact was not produced) — a real, non-recoverable failure.
			round.Status = RoundStatusFailed
			round.Error = err.Error()
			slog.Warn("operating mode: phase result contract failed", "err", err, "initiative", round.InitiativeName, "round", round.Round)
			s.emitPhaseFailed(round, err.Error())
		case resolved.Resolved():
			round.Status = RoundStatusCompleted
			s.emitPhaseCompleted(round)
			s.emitParsedPhaseSignals(round)
			completedNormally = true
		default:
			// Honest abstain: the ladder could not resolve or reconstruct the
			// phase's required structured output. This is not the old
			// fail-on-malformed path (recoverable output resolves above) — it is a
			// safe stop with structured diagnostics so nothing auto-progresses on
			// absent data and the operator can see exactly what was unresolved.
			round.Status = RoundStatusFailed
			round.Error = resolved.AbstainReason()
			slog.Warn("operating mode: phase output abstained",
				"initiative", round.InitiativeName, "round", round.Round,
				"missing", resolved.Missing, "violations", resolved.Violations)
			s.emitPhaseFailed(round, round.Error)
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
	// Auto-dispatch fires *after* lock release and *after* the predecessor
	// round is persisted, so the next phase sees a clean lock and a stable
	// upstream record. Failure / cancel paths skip auto-dispatch — the
	// reconcile contract is "the predecessor completed cleanly."
	if completedNormally {
		s.maybeAutoStartNext(ctx, round)
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

// resolutionCandidates assembles the ordered candidate messages (oldest→newest)
// the resolution ladder scans. When the agent exposes the run's assistant
// messages (RunMessageReader), they feed L0 true-final-message detection so a
// trailing subagent message doesn't mask the real answer. The run summary — the
// agent-manager's official final description — is appended as the newest
// candidate unless it already matches the last message, so a summary that
// carries the result is tried first while older messages remain a fallback. When
// no message stream is available, the summary is the sole candidate.
func (s *Service) resolutionCandidates(ctx context.Context, round RoundEnvelope, state agentmanager.RunState) []string {
	var messages []string
	if reader, ok := s.agent.(RunMessageReader); ok && strings.TrimSpace(round.RunID) != "" {
		if msgs, err := reader.GetRunMessages(ctx, round.RunID); err != nil {
			slog.Debug("operating mode: run messages unavailable; falling back to summary",
				"err", err, "initiative", round.InitiativeName, "run_id", round.RunID)
		} else {
			messages = msgs
		}
	}
	summary := strings.TrimSpace(state.Summary)
	if summary != "" && (len(messages) == 0 || strings.TrimSpace(messages[len(messages)-1]) != summary) {
		messages = append(messages, summary)
	}
	return messages
}

// maybeAutoStartNext implements the AutoStartAfter contract: when the
// completed round's phase appears in any phase's AutoStartAfter list (≤1
// per validator), start that next phase. Lane saturation is treated as a
// soft failure — the predecessor round is marked with pending_auto_start
// and the periodic RefreshRound retries on its next tick. Other failures
// (no initiative, missing definition, locked, prompt-render error) are
// logged but never blow up the refresher: a missed auto-dispatch is
// recoverable; a panicked refresher is not.
func (s *Service) maybeAutoStartNext(ctx context.Context, round RoundEnvelope) {
	def, err := DefinitionFor(Mode(round.Mode))
	if err != nil {
		slog.Warn("operating mode: auto-start lookup failed",
			"err", err, "initiative", round.InitiativeName, "mode", round.Mode, "phase", round.Phase)
		return
	}
	next, ok := nextAutoStartPhase(def, Phase(round.Phase))
	if !ok {
		// Idempotency: if the predecessor was previously marked pending and
		// no longer has a downstream auto-start (e.g. reconcile is the
		// terminal phase), clear the marker so it doesn't linger forever.
		s.clearPendingAutoStart(round)
		return
	}
	_, err = s.StartPhase(ctx, StartPhaseRequest{
		InitiativeName: round.InitiativeName,
		Phase:          string(next),
		RequestedBy:    s.requestedBy,
		Note:           "auto-started after " + round.Phase,
	})
	switch {
	case err == nil:
		s.clearPendingAutoStart(round)
	case errors.Is(err, agentactivity.ErrLaneSaturated):
		// Lane is full — defer. The periodic RefreshRound walks active
		// rounds; we'll retry the dispatch the next time this completed
		// round is observed. The marker doubles as an audit trail so
		// operators can see why the next phase didn't start immediately.
		s.markPendingAutoStart(round, next, err.Error())
		slog.Info("operating mode: auto-start deferred by lane saturation",
			"initiative", round.InitiativeName, "from_phase", round.Phase, "next_phase", next, "err", err)
	default:
		slog.Warn("operating mode: auto-start failed",
			"err", err, "initiative", round.InitiativeName, "from_phase", round.Phase, "next_phase", next)
	}
}

// nextAutoStartPhase scans the phase graph for the (at most one, per
// validator) phase whose AutoStartAfter declares the predecessor. Returns
// false when no auto-start successor exists for this predecessor.
func nextAutoStartPhase(def Definition, predecessor Phase) (Phase, bool) {
	for _, phase := range orderedPhases(def) {
		phaseDef := def.PhaseGraph.Phases[phase]
		for _, after := range phaseDef.AutoStartAfter {
			if after == predecessor {
				return phase, true
			}
		}
	}
	return "", false
}

// markPendingAutoStart records on the predecessor round that an auto-start
// dispatch is deferred and should be retried on the next refresh tick.
// The marker is a small JSON sidecar on the round payload so it's
// inspectable from CLI / logs without a separate file format.
func (s *Service) markPendingAutoStart(round RoundEnvelope, next Phase, reason string) {
	payload := MutableRoundPayload(&round)
	payload.set(payloadPendingAutoStart, map[string]any{
		"next_phase":  string(next),
		"reason":      strings.TrimSpace(reason),
		"observed_at": s.clock().UTC().Format(time.RFC3339),
	})
	if err := s.store.SaveRound(round); err != nil {
		slog.Warn("operating mode: persist pending_auto_start marker failed",
			"err", err, "initiative", round.InitiativeName, "round", round.Round)
	}
}

// clearPendingAutoStart removes the deferred-dispatch marker. Called once
// the auto-start either succeeds or is no longer applicable (predecessor
// has no downstream auto-start phase).
func (s *Service) clearPendingAutoStart(round RoundEnvelope) {
	payload := RoundPayload(round.Payload)
	if _, ok := payload.get(payloadPendingAutoStart); !ok {
		return
	}
	MutableRoundPayload(&round).clear(payloadPendingAutoStart)
	if err := s.store.SaveRound(round); err != nil {
		slog.Warn("operating mode: clear pending_auto_start marker failed",
			"err", err, "initiative", round.InitiativeName, "round", round.Round)
	}
}

// HasPendingAutoStart reports whether the round is currently waiting on a
// retried auto-start dispatch (e.g. lane saturation deferred the next
// phase). Exposed for tests and CLI inspection; production code drives the
// retry through periodic RefreshRound, not by reading this field.
func (p RoundPayloadView) HasPendingAutoStart() bool {
	_, ok := p.get(payloadPendingAutoStart)
	return ok
}

func finishTime(state agentmanager.RunState, clock func() time.Time) string {
	if strings.TrimSpace(state.FinishedAt) != "" {
		return strings.TrimSpace(state.FinishedAt)
	}
	return clock().UTC().Format(time.RFC3339)
}
