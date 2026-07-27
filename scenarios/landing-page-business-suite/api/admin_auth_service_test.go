package main

import (
	"context"
	"errors"
	"testing"
)

func TestAdminAuthServicePasswordHashHonorsCanceledContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewAdminAuthService(db).PasswordHash(ctx, "admin@example.com")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PasswordHash error = %v, want context.Canceled", err)
	}
}
