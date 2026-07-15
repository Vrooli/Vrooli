package review

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// activeRound tracks a gathering round so the poller knows which runs to check.
type activeRound struct {
	Kind     string
	Name     string
	ItemDir  string
	RoundNum int
	RunID    string
}

func (s *Service) trackActiveRound(runID, kind, name, itemDir string, roundNum int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeRounds[runID] = activeRound{
		Kind:     kind,
		Name:     name,
		ItemDir:  itemDir,
		RoundNum: roundNum,
		RunID:    runID,
	}
}

// RefreshGatheringRounds polls agent-manager for the status of all tracked
// gathering rounds and updates their on-disk status when the run completes.
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
		round, loadErr := LoadRound(ar.ItemDir, ar.RoundNum)
		if loadErr != nil || round == nil {
			s.untrackRound(runID)
			continue
		}

		if round.Status == RoundStatusComplete || round.Status == RoundStatusFailed {
			// Already terminal (e.g. agent wrote the round file itself).
			s.untrackRound(runID)
			continue
		}

		state, err := s.inspector.GetRunState(ctx, runID)
		switch {
		case err == nil && mapRunStatusToRoundStatus(state.Status) != "":
			// Run reached a terminal state; adopt its outcome.
			*round = finalizeRoundFromRunState(*round, state)
		case s.roundExceededMaxAge(*round):
			// Run is unreachable or wedged past the max age — treat the round
			// as abandoned so the item can leave in_review. This is the
			// backstop for a review run that died without agent-manager ever
			// reporting a terminal status (a top cause of orphaned in_review).
			round.Status = RoundStatusFailed
			if strings.TrimSpace(round.FailureReason) == "" {
				round.FailureReason = fmt.Sprintf("review run did not finish within %s and was treated as abandoned", s.maxRoundAge())
			}
			slog.Warn("review round exceeded max age, finalizing as failed",
				"round", ar.RoundNum, "run_id", runID, "max_age", s.maxRoundAge())
		default:
			// Transient inspector error or run still in flight within budget;
			// retry next tick.
			continue
		}

		if saveErr := SaveRound(ar.ItemDir, *round); saveErr != nil {
			slog.Error("update review round status", "round", ar.RoundNum, "run_id", runID, "error", saveErr)
			continue
		}

		slog.Info("review round status updated", "round", ar.RoundNum, "run_id", runID, "status", round.Status)

		if s.eventLogger != nil && round.ExecutionID != "" {
			if round.Status == RoundStatusComplete {
				started, _ := time.Parse(time.RFC3339, round.GeneratedAt)
				duration := time.Since(started).Seconds()
				s.eventLogger.EmitReviewRoundCompleted(round.ExecutionID, round.RoundNum, len(round.Evidence), round.Classification, duration)
			} else if round.Status == RoundStatusFailed {
				s.eventLogger.EmitReviewFailed(round.ExecutionID, round.FailureReason, 0)
			}
		}

		// Notify the backlog layer so the item can transition from
		// in_review to review_pending. The handler decides what to do for
		// complete vs failed outcomes; both paths surface the review to the
		// user for a terminal decision.
		if s.onRoundTerminal != nil && ar.Kind != "" && ar.Name != "" {
			s.onRoundTerminal(ctx, ar.Kind, ar.Name, *round)
		}

		s.untrackRound(runID)
	}
}

// untrackRound removes a run from the active-round tracking map.
func (s *Service) untrackRound(runID string) {
	s.mu.Lock()
	delete(s.activeRounds, runID)
	s.mu.Unlock()
}

// maxRoundAge returns the configured abandoned-round threshold, defaulting when
// unset.
func (s *Service) maxRoundAge() time.Duration {
	if s.roundMaxAge > 0 {
		return s.roundMaxAge
	}
	return DefaultRoundMaxAge
}

// roundExceededMaxAge reports whether a gathering round was generated longer
// ago than the max-age threshold. An unparseable timestamp is treated as not
// exceeded so a malformed round isn't force-failed on a clock guess.
func (s *Service) roundExceededMaxAge(round Round) bool {
	generated, err := time.Parse(time.RFC3339, round.GeneratedAt)
	if err != nil {
		return false
	}
	return s.now().Sub(generated) > s.maxRoundAge()
}

func (s *Service) now() time.Time {
	if s.clock != nil {
		return s.clock().UTC()
	}
	return time.Now().UTC()
}

// StartBackgroundWorker polls gathering rounds on a 5-second interval until
// the stop channel is closed.
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

// RecoverActiveRounds scans all backlog items for rounds in gathering state
// and re-populates the in-memory tracking map. Call this at startup so that
// rounds spawned before a restart are still polled.
func (s *Service) RecoverActiveRounds() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Map each on-disk kind directory back to its backlog kind so recovered
	// rounds carry Kind/Name. Without these, RefreshGatheringRounds skips the
	// onRoundTerminal flip (guarded on ar.Kind/ar.Name), which would leave an
	// item stuck in in_review even after its review run completes — a silent
	// contributor to the orphaned-in_review dead-end after a restart.
	kindByDir := map[string]string{
		"ideas":    "idea",
		"research": "research",
		"fix":      "fix",
		"execute":  "execute",
		"chore":    "chore",
	}
	for kindDir, kind := range kindByDir {
		baseDir := s.dataRoot + "/" + kindDir
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			itemDir := baseDir + "/" + entry.Name()
			rounds, loadErr := LoadRounds(itemDir)
			if loadErr != nil {
				continue
			}
			for _, round := range rounds {
				// Runner-owned rounds are recovered + driven by the operation
				// runner's refresh driver (from durable workflow state), not this
				// poller — skip them so the two never race.
				if round.RunnerOwned() {
					continue
				}
				if round.Status == RoundStatusGathering && round.RunID != "" {
					s.activeRounds[round.RunID] = activeRound{
						Kind:     kind,
						Name:     entry.Name(),
						ItemDir:  itemDir,
						RoundNum: round.RoundNum,
						RunID:    round.RunID,
					}
				}
			}
		}
	}

	if len(s.activeRounds) > 0 {
		slog.Info("recovered active review rounds", "count", len(s.activeRounds))
	}
}
