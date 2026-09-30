package main

import (
	"context"
	"testing"

	"landing-page-business-suite-api/internal/administration"
)

func TestAuthJanitorPurgesEndedRowsAndRetainsActiveRows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	var userID, sessionID string
	if err := db.QueryRow(`INSERT INTO users (email) VALUES ('janitor@example.com') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO user_sessions (user_id, refresh_token_hash, expires_at, last_used_at, revoked) VALUES ($1, 'janitor-old-refresh', NOW() - INTERVAL '31 days', NOW() - INTERVAL '31 days', TRUE) RETURNING id`, userID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	var activeSessionID string
	if err := db.QueryRow(`INSERT INTO user_sessions (user_id, refresh_token_hash, expires_at, last_used_at, revoked) VALUES ($1, 'janitor-active-refresh', NOW() + INTERVAL '1 day', NOW(), FALSE) RETURNING id`, userID).Scan(&activeSessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO refresh_token_history (refresh_token_hash, session_id, family_id, retired_at) VALUES ('janitor-old-history', $1, gen_random_uuid(), NOW() - INTERVAL '101 days')`, activeSessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO refresh_token_history (refresh_token_hash, session_id, family_id, retired_at) VALUES ('janitor-recent-history', $1, gen_random_uuid(), NOW())`, activeSessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO admin_sessions (id, admin_email, expires_at) VALUES ('janitor-old-admin', 'janitor@example.com', NOW() - INTERVAL '2 days'), ('janitor-active-admin', 'janitor@example.com', NOW() + INTERVAL '1 day')`); err != nil {
		t.Fatal(err)
	}
	service := administration.NewUserAuthService(administration.UserAuthServiceOptions{Store: db, JWTIssuer: "test", BaseURL: "http://localhost"})
	if removed, err := service.PurgeEndedUserSessions(ctx); err != nil || removed != 1 {
		t.Fatalf("user session purge removed=%d err=%v", removed, err)
	}
	if removed, err := service.PurgeRefreshHistory(ctx); err != nil || removed != 1 {
		t.Fatalf("refresh history purge removed=%d err=%v", removed, err)
	}
	admin := administration.NewAdminAuthService(db)
	if removed, err := admin.PurgeExpiredSessions(ctx); err != nil || removed != 1 {
		t.Fatalf("admin session purge removed=%d err=%v", removed, err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_sessions WHERE refresh_token_hash = 'janitor-active-refresh'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("active user session count=%d err=%v", count, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM refresh_token_history WHERE refresh_token_hash = 'janitor-recent-history'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("recent history count=%d err=%v", count, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_sessions WHERE id = 'janitor-active-admin'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("active admin session count=%d err=%v", count, err)
	}
}
