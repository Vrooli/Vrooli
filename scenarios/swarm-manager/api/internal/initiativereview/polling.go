package initiativereview

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/review"
)

// handleTerminalRound flips the initiative from in_review to review_pending
// when the review round reaches a terminal status and releases the per-
// initiative lock so feedback submissions can proceed once the user is
// looking at the review verdict. Called from both the ListRounds inline-
// poll and the background worker.
func (s *Service) handleTerminalRound(ctx context.Context, initiativeName string, round review.Round) {
	if round.Status != review.RoundStatusComplete && round.Status != review.RoundStatusFailed {
		return
	}

	// Release the lock first — even if the status flip below fails, we'd
	// rather leak the status machine (which the user can fix with decide)
	// than leak a lock that blocks every subsequent feedback submission.
	if s.lock != nil && round.RunID != "" {
		if err := s.lock.Release(initiativeName, round.RunID); err != nil {
			slog.Warn("initiative review: release lock", "initiative", initiativeName, "round", round.RoundNum, "err", err)
		}
	}

	init, err := s.initStore.Load(initiativeName)
	if err != nil {
		slog.Warn("initiative review: load after terminal round", "initiative", initiativeName, "err", err)
		return
	}
	// Only transition from in_review; if the user has already decided or
	// another flow moved us, don't overwrite.
	if init.Status != initiatives.InitiativeStatusInReview {
		return
	}
	if err := s.setInitiativeStatus(init, initiatives.InitiativeStatusReviewPending, s.clock().UTC().Format(time.RFC3339)); err != nil {
		slog.Warn("initiative review: flip to review_pending", "initiative", initiativeName, "err", err)
	}
	if s.roundTerminalObserver != nil {
		s.roundTerminalObserver(ctx, "initiative", initiativeName, round)
	}
}

// --- Background polling ---------------------------------------------------

func (s *Service) trackActiveRound(initiativeName string, roundNum int, runID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeRounds[runID] = activeRound{
		InitiativeName: initiativeName,
		RoundNum:       roundNum,
		RunID:          runID,
	}
}

// RefreshGatheringRounds polls each tracked gathering round for terminal
// status and, on transition, flips the initiative to review_pending.
// Safe to call concurrently with Start/List; the internal map is guarded.
func (s *Service) RefreshGatheringRounds(ctx context.Context) {
	if s.inspector == nil {
		return
	}
	s.mu.Lock()
	snapshot := make(map[string]activeRound, len(s.activeRounds))
	for k, v := range s.activeRounds {
		snapshot[k] = v
	}
	s.mu.Unlock()

	for runID, ar := range snapshot {
		state, err := s.inspector.GetRunState(ctx, runID)
		if err != nil {
			continue
		}
		itemDir := s.initStore.InitDir(ar.InitiativeName)
		round, loadErr := review.LoadRound(itemDir, ar.RoundNum)
		if loadErr != nil || round == nil {
			s.mu.Lock()
			delete(s.activeRounds, runID)
			s.mu.Unlock()
			continue
		}
		finalized, changed := finalizeRoundIfTerminal(*round, state)
		if !changed {
			continue
		}
		if err := review.SaveRound(itemDir, finalized); err != nil {
			slog.Warn("initiative review: save round", "initiative", ar.InitiativeName, "round", ar.RoundNum, "err", err)
			continue
		}
		s.handleTerminalRound(ctx, ar.InitiativeName, finalized)
		s.mu.Lock()
		delete(s.activeRounds, runID)
		s.mu.Unlock()
	}
}

// StartBackgroundWorker polls gathering rounds on a 5-second tick until
// the stop channel is closed. Runs once per process, started from main.go.
func (s *Service) StartBackgroundWorker(stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.RefreshGatheringRounds(context.Background())
		}
	}
}

// RecoverActiveRounds scans every initiative for rounds in gathering state
// and re-populates the in-memory tracking map. Call this at startup so
// rounds spawned before a restart resume polling.
//
// Discovers initiatives itself via the injected InitiativeStore — callers
// must not pre-filter the list, otherwise initiatives created immediately
// before a crash (and thus absent from a cached name list) leak their
// gathering rounds until the next manual trigger.
func (s *Service) RecoverActiveRounds() {
	inits, err := s.initStore.LoadAll()
	if err != nil {
		slog.Warn("initiative review: list initiatives for recovery failed", "err", err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	recovered := 0
	for _, init := range inits {
		rounds, err := review.LoadRounds(s.initStore.InitDir(init.Name))
		if err != nil {
			continue
		}
		for _, r := range rounds {
			// Runner-owned rounds are recovered + driven by the operation runner's
			// refresh driver (from durable workflow state), not this poller.
			if r.RunnerOwned() || r.WorkflowOwned() {
				continue
			}
			if r.Status == review.RoundStatusGathering && r.RunID != "" {
				s.activeRounds[r.RunID] = activeRound{
					InitiativeName: init.Name,
					RoundNum:       r.RoundNum,
					RunID:          r.RunID,
				}
				recovered++
			}
		}
	}
	if recovered > 0 {
		slog.Info("recovered active initiative review rounds", "count", recovered)
	}
}

// --- Helpers --------------------------------------------------------------

// finalizeRoundIfTerminal returns (newRound, true) when the agent state
// indicates the run has reached a terminal status; otherwise (round, false).
// Failure reasons are recorded on the round for UI rendering.
func finalizeRoundIfTerminal(round review.Round, state agentmanager.RunState) (review.Round, bool) {
	switch strings.ToLower(strings.TrimSpace(state.Status)) {
	case "complete":
		if err := validateCompletedRound(round); err != nil {
			round.Status = review.RoundStatusFailed
			round.FailureReason = err.Error()
			return round, true
		}
		round.Status = review.RoundStatusComplete
		round.FailureReason = ""
		return round, true
	case "failed":
		round.Status = review.RoundStatusFailed
		round.FailureReason = firstNonEmpty(state.ErrorMsg, "initiative review agent run failed")
		return round, true
	case "cancelled":
		round.Status = review.RoundStatusFailed
		round.FailureReason = firstNonEmpty(state.ErrorMsg, "initiative review agent run was cancelled")
		return round, true
	}
	return round, false
}

func validateCompletedRound(round review.Round) error {
	if strings.TrimSpace(round.AgentAssessment) == "" {
		return fmt.Errorf("review run completed without agent_assessment")
	}
	classification := strings.TrimSpace(round.Classification)
	if classification == "" {
		return fmt.Errorf("review run completed without classification")
	}
	switch classification {
	case "delivered", "partial", "failed":
		return nil
	}
	return fmt.Errorf("review run completed with invalid classification %q (want delivered|partial|failed)", classification)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
