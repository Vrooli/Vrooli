package main

import (
	"context"
	"errors"
	"testing"
)

func TestUserManagementListPreservesRequestCancellation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewUserManagementService(db).List(ctx, "", 1, 20)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("List() error = %v, want context cancellation", err)
	}
}
