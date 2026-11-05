package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/securevalue"
)

func TestAdminMFALifecycle(t *testing.T) {
	db := setupTestDB(t)
	cleanupAdminUsers(t, db)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password-1234"), bcrypt.MinCost)
	if _, err := db.Exec(`INSERT INTO admin_users (email, password_hash) VALUES ('mfa@test.com', $1)`, string(hash)); err != nil {
		t.Fatal(err)
	}
	ring, err := securevalue.NewRing()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_900_000_000, 0)
	mfa := administration.NewAdminMFA(db, func() (securevalue.Ring, error) { return ring, nil }, func() string { return "LPBS Test" })
	mfa.UseClock(func() time.Time { return now })
	ctx := context.Background()

	enrollment, err := mfa.BeginEnrollment(ctx, "mfa@test.com")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(enrollment.OTPAuthURI, "issuer=LPBS+Test") {
		t.Fatalf("authenticator apps would not show the site name: %s", enrollment.OTPAuthURI)
	}
	if enabled, _ := mfa.Enabled(ctx, "mfa@test.com"); enabled {
		t.Fatal("pending enrollment must not change login policy")
	}
	var stored string
	_ = db.QueryRow(`SELECT totp_pending_secret_encrypted FROM admin_users WHERE email = 'mfa@test.com'`).Scan(&stored)
	if stored == "" || stored == enrollment.Secret {
		t.Fatal("secret is not sealed at rest")
	}

	code, _ := administration.TOTPCode(enrollment.Secret, now)
	recovery, err := mfa.ConfirmEnrollment(ctx, "mfa@test.com", code)
	if err != nil || len(recovery) != 10 {
		t.Fatalf("confirm: codes=%d err=%v", len(recovery), err)
	}
	if err := mfa.Verify(ctx, "mfa@test.com", code); !errors.Is(err, administration.ErrMFAInvalid) {
		t.Fatalf("enrollment code replayed at login: %v", err)
	}
	now = now.Add(30 * time.Second)
	next, _ := administration.TOTPCode(enrollment.Secret, now)
	if err := mfa.Verify(ctx, "mfa@test.com", next); err != nil {
		t.Fatalf("current code rejected: %v", err)
	}
	if err := mfa.Verify(ctx, "mfa@test.com", recovery[0]); err != nil {
		t.Fatalf("recovery code rejected: %v", err)
	}
	if err := mfa.Verify(ctx, "mfa@test.com", recovery[0]); !errors.Is(err, administration.ErrMFAInvalid) {
		t.Fatalf("recovery code reused: %v", err)
	}
	status, _ := mfa.Status(ctx, "mfa@test.com")
	if !status.Enabled || status.RecoveryCodesLeft != 9 {
		t.Fatalf("status = %+v", status)
	}
	if err := mfa.Reset(ctx, "mfa@test.com"); err != nil {
		t.Fatal(err)
	}
	if enabled, _ := mfa.Enabled(ctx, "mfa@test.com"); enabled {
		t.Fatal("operator reset left two-factor enabled")
	}
}
