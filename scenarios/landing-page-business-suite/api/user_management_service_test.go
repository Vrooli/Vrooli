package main

import (
	"context"
	"errors"
	"testing"

	"landing-page-business-suite-api/internal/administration"
)

func TestUserManagementListPreservesRequestCancellation(t *testing.T) {
	db := setupTestDB(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := administration.NewUserManagementService(db).List(ctx, "", 1, 20)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("List() error = %v, want context cancellation", err)
	}
}
