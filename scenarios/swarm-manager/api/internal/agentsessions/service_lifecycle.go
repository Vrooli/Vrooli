package agentsessions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
)

const defaultRunningSessionTTL = 24 * time.Hour

// Complete refreshes a session from its attributed run and succeeds only when
// the run has reached a terminal completed state. It is intentionally not a
// forceful status setter: callers must prove completion through Agent Manager.
func (s *Service) Complete(ctx context.Context, sessionID string) (Session, error) {
	session, err := s.Refresh(ctx, sessionID)
	if err != nil {
		return Session{}, err
	}
	if session.Status != StatusComplete {
		return Session{}, apierr.Conflict("session is not complete (current status: %s)", session.Status)
	}
	return session, nil
}

// Reap marks running sessions older than the configured TTL as failed and
// emits the normal failed-session event with an explicit expiry reason. The
// operation is idempotent because only running sessions are eligible.
func (s *Service) Reap(ctx context.Context, now time.Time) ([]Session, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return nil, err
	}
	ttl := s.runningSessionTTL
	if ttl <= 0 {
		ttl = defaultRunningSessionTTL
	}
	cutoff := now.UTC().Add(-ttl)
	sessions, err := store.ListSessions(ListFilters{Status: StatusRunning})
	if err != nil {
		return nil, err
	}
	var reaped []Session
	for _, session := range sessions {
		updated, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(session.UpdatedAt))
		if parseErr != nil || !updated.Before(cutoff) {
			continue
		}
		if strings.TrimSpace(session.RunID) != "" && s.spawner != nil {
			if stopErr := s.spawner.StopRun(ctx, session.RunID); stopErr != nil && !isTerminalRunStopConflict(stopErr) {
				return nil, fmt.Errorf("stop expired session %s: %w", session.ID, stopErr)
			}
		}
		session.Status = StatusFailed
		session.FailureReason = fmt.Sprintf("running session TTL expired after %s", ttl)
		session.UpdatedAt = now.UTC().Format(time.RFC3339Nano)
		if err := store.SaveSession(session); err != nil {
			return nil, err
		}
		s.emitFailed(session)
		reaped = append(reaped, session)
	}
	return reaped, nil
}
