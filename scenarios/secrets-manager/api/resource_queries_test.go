package main

import (
	"context"
	"testing"
)

func TestFetchResourceDetail_FallbacksToConfigWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	detail, err := fetchResourceDetail(context.Background(), nil, "home-assistant")
	if err != nil {
		t.Fatalf("fetchResourceDetail returned error: %v", err)
	}
	if len(detail.Secrets) == 0 {
		t.Fatalf("expected secrets from config fallback, got none")
	}
	if detail.TotalSecrets != len(detail.Secrets) {
		t.Fatalf("TotalSecrets should equal len(Secrets); got %d vs %d", detail.TotalSecrets, len(detail.Secrets))
	}
}

func TestFetchResourceDetail_EmptyResource(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if _, err := fetchResourceDetail(context.Background(), nil, ""); err == nil {
		t.Fatal("expected error for empty resource name, got nil")
	}
}
