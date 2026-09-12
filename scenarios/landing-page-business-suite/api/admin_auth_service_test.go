package main

import (
	"context"
	"errors"
	"testing"

	"landing-page-business-suite-api/internal/administration"
)

func TestAdminAuthServicePasswordHashHonorsCanceledContext(t *testing.T) {
	db := setupTestDB(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := administration.NewAdminAuthService(db).PasswordHash(ctx, "admin@example.com")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PasswordHash error = %v, want context.Canceled", err)
	}
}
