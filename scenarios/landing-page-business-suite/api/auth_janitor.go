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
	if s.userAuthService == nil || s.adminAuthService == nil {
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
				purges := []struct {
					name string
					fn   func(context.Context) (int64, error)
				}{
					{"sign_ins", s.userAuthService.PurgeExpiredSignIns},
					{"user_sessions", s.userAuthService.PurgeEndedUserSessions},
					{"refresh_token_history", s.userAuthService.PurgeRefreshHistory},
					{"admin_sessions", s.adminAuthService.PurgeExpiredSessions},
					{"admin_security_events", func(ctx context.Context) (int64, error) {
						result, err := s.db.ExecContext(ctx, `DELETE FROM admin_security_events WHERE created_at < NOW() - INTERVAL '400 days'`)
						if err != nil {
							return 0, err
						}
						return result.RowsAffected()
					}},
					{"webauthn_challenges", func(ctx context.Context) (int64, error) {
						result, err := s.db.ExecContext(ctx, `DELETE FROM webauthn_challenges WHERE expires_at < NOW() - INTERVAL '1 day' OR consumed_at IS NOT NULL AND consumed_at < NOW() - INTERVAL '1 day'`)
						if err != nil {
							return 0, err
						}
						return result.RowsAffected()
					}},
				}
				if s.nativeGrants != nil {
					purges = append(purges, struct {
						name string
						fn   func(context.Context) (int64, error)
					}{"native_auth_grants", s.nativeGrants.PurgeExpired})
				}
				for _, purge := range purges {
					removed, err := purge.fn(runCtx)
					if err != nil {
						logx.Error("auth_janitor_purge_failed", map[string]interface{}{"name": purge.name, "error": err.Error()})
						continue
					}
					if removed > 0 {
						logx.Info("auth_janitor_purged", map[string]interface{}{"name": purge.name, "removed": removed})
					}
				}
				done()
			}
		}
	}()
	return cancel
}
