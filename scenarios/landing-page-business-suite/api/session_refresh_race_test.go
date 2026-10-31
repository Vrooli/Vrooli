package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/administration"
)

func TestCookieRefreshRaceUsesGraceWithoutRevokingFamily(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	service := newUserAuthServiceForTest(db, NewEmailService())
	user, err := service.GetOrCreateUser(context.Background(), "refresh-race@example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupUserTestData(t, db, user.Email)
	pair, err := service.CreateSession(context.Background(), user, "127.0.0.1", "race-test")
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, refreshErr := service.RefreshTokensFromSource(context.Background(), pair.RefreshToken, administration.RefreshSourceCookie)
			results <- refreshErr
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	for refreshErr := range results {
		if refreshErr == nil {
			successes++
		}
	}
	if successes != 2 {
		t.Fatalf("cookie refresh successes = %d, want 2", successes)
	}
	var revoked bool
	if err := db.QueryRow(`SELECT revoked FROM user_sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, user.ID).Scan(&revoked); err != nil {
		t.Fatal(err)
	}
	if revoked {
		t.Fatal("cookie refresh race revoked the session family")
	}
}

func TestRetiredCookieAfterGraceRevokesFamily(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	service := newUserAuthServiceForTest(db, NewEmailService())
	user, err := service.GetOrCreateUser(context.Background(), "refresh-cookie-replay@example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupUserTestData(t, db, user.Email)
	pair, err := service.CreateSession(context.Background(), user, "127.0.0.1", "replay-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.RefreshTokensFromSource(context.Background(), pair.RefreshToken, administration.RefreshSourceCookie); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE refresh_token_history SET retired_at = NOW() - INTERVAL '1 minute' WHERE refresh_token_hash = $1`, administration.HashToken(pair.RefreshToken)); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RefreshTokensFromSource(context.Background(), pair.RefreshToken, administration.RefreshSourceCookie); !errors.Is(err, administration.ErrSessionRevoked) {
		t.Fatalf("retired cookie after grace = %v, want ErrSessionRevoked", err)
	}
	var revoked bool
	var reason string
	if err := db.QueryRow(`SELECT revoked, revoked_reason FROM user_sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, user.ID).Scan(&revoked, &reason); err != nil {
		t.Fatal(err)
	}
	if !revoked || reason != "reuse" {
		t.Fatalf("replayed cookie session state = revoked:%v reason:%q, want revoked/reuse", revoked, reason)
	}
}

func TestRetiredBodyInsideGraceRevokesFamily(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	service := newUserAuthServiceForTest(db, NewEmailService())
	user, err := service.GetOrCreateUser(context.Background(), "refresh-body-replay@example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupUserTestData(t, db, user.Email)
	pair, err := service.CreateSession(context.Background(), user, "127.0.0.1", "replay-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.RefreshTokensFromSource(context.Background(), pair.RefreshToken, administration.RefreshSourceBody); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RefreshTokensFromSource(context.Background(), pair.RefreshToken, administration.RefreshSourceBody); !errors.Is(err, administration.ErrSessionRevoked) {
		t.Fatalf("retired body inside grace = %v, want ErrSessionRevoked", err)
	}
	var revoked bool
	if err := db.QueryRow(`SELECT revoked FROM user_sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, user.ID).Scan(&revoked); err != nil {
		t.Fatal(err)
	}
	if !revoked {
		t.Fatal("body-token replay did not revoke the session family")
	}
}

func TestRefreshAfterAbsoluteExpiryRevokesSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	service := administration.NewUserAuthService(administration.UserAuthServiceOptions{
		Store: db, IdleTTL: 24 * time.Hour, AbsoluteTTL: time.Hour,
	})
	user, err := service.GetOrCreateUser(context.Background(), "refresh-absolute-expired@example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupUserTestData(t, db, user.Email)
	pair, err := service.CreateSessionWithMetadata(context.Background(), user, "127.0.0.1", "absolute-test", "email_code", time.Now().UTC().Add(-2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.RefreshTokensFromSource(context.Background(), pair.RefreshToken, administration.RefreshSourceBody); !errors.Is(err, administration.ErrSessionExpired) {
		t.Fatalf("refresh after absolute expiry = %v, want ErrSessionExpired", err)
	}
	var revoked bool
	if err := db.QueryRow(`SELECT revoked FROM user_sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, user.ID).Scan(&revoked); err != nil {
		t.Fatal(err)
	}
	if !revoked {
		t.Fatal("absolute-expired refresh did not revoke the session")
	}
}
