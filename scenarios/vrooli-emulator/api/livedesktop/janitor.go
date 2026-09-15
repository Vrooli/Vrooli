package livedesktop

import (
	"context"
	"log/slog"
	"time"
)

// StartJanitor runs a background goroutine that reaps idle sessions.
func StartJanitor(ctx context.Context, svc *Service, checkInterval, idleTimeout time.Duration) {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reapIdleSessions(svc, idleTimeout)
			}
		}
	}()
}

func reapIdleSessions(svc *Service, idleTimeout time.Duration) {
	now := time.Now()
	for _, session := range svc.store.ActiveSessions() {
		session.mu.Lock()
		idle := now.Sub(session.LastHeartbeat) > idleTimeout
		sessionID := session.ID
		session.mu.Unlock()

		if idle {
			slog.Info("reaping idle desktop session", "session_id", sessionID, "idle_timeout", idleTimeout)
			if err := svc.StopSession(sessionID); err != nil {
				slog.Warn("failed to reap session", "session_id", sessionID, "error", err)
			}
		}
	}
}
