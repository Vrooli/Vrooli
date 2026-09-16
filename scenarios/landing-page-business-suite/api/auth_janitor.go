package main

import (
	"context"
	"time"

	"landing-page-business-suite-api/internal/logx"
)

const authJanitorInterval = time.Hour

// startAuthJanitor removes dead sign-in requests hourly so pending codes and
// links do not accumulate. The returned function stops it.
func (s *Server) startAuthJanitor() func() {
	if s.userAuthService == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(authJanitorInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCtx, done := context.WithTimeout(ctx, 30*time.Second)
				removed, err := s.userAuthService.PurgeExpiredSignIns(runCtx)
				done()
				if err != nil {
					logx.Error("auth_janitor_purge_failed", map[string]interface{}{"error": err.Error()})
				} else if removed > 0 {
					logx.Info("auth_janitor_purged", map[string]interface{}{"removed": removed})
				}
			}
		}
	}()
	return cancel
}
