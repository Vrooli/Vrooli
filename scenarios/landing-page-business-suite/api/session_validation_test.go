package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/administration"
)

func TestValidateSessionRejectsRevokedAndAbsoluteExpiredRows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	var userID, revokedID, expiredID string
	if err := db.QueryRow(`INSERT INTO users (email) VALUES ('session-validation@example.com') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	defer cleanupUserTestData(t, db, "session-validation@example.com")
	if err := db.QueryRow(`INSERT INTO user_sessions (user_id, refresh_token_hash, expires_at, absolute_expires_at, revoked) VALUES ($1, 'validation-revoked', NOW() + INTERVAL '1 day', NOW() + INTERVAL '90 days', TRUE) RETURNING id`, userID).Scan(&revokedID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO user_sessions (user_id, refresh_token_hash, expires_at, absolute_expires_at, revoked) VALUES ($1, 'validation-expired', NOW() + INTERVAL '1 day', NOW() - INTERVAL '1 second', FALSE) RETURNING id`, userID).Scan(&expiredID); err != nil {
		t.Fatal(err)
	}
	service := administration.NewUserAuthService(administration.UserAuthServiceOptions{Store: db})
	if err := service.ValidateSession(context.Background(), revokedID); !errors.Is(err, administration.ErrSessionRevoked) {
		t.Fatalf("revoked validation = %v", err)
	}
	if err := service.ValidateSession(context.Background(), expiredID); !errors.Is(err, administration.ErrSessionExpired) {
		t.Fatalf("expired validation = %v", err)
	}
}

func TestRefreshCapsIdleExpiryAtAbsoluteExpiry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	service := administration.NewUserAuthService(administration.UserAuthServiceOptions{Store: db, IdleTTL: 24 * time.Hour, AbsoluteTTL: 2 * time.Hour})
	user, err := service.GetOrCreateUser(context.Background(), "session-cap@example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupUserTestData(t, db, user.Email)
	pair, err := service.CreateSessionWithMetadata(context.Background(), user, "127.0.0.1", "test", "email_code", time.Now().UTC().Add(-90*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if pair.SessionExpiresAt.After(time.Now().UTC().Add(-90 * time.Minute).Add(2 * time.Hour).Add(time.Second)) {
		t.Fatalf("session expiry exceeded absolute lifetime: %s", pair.SessionExpiresAt)
	}
}
