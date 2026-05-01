package mocks

import (
	"context"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestFakeEventStoreCapturesLogMessages(t *testing.T) {
	store := NewFakeEventStore()
	runID := uuid.New()

	err := store.Append(context.Background(), runID, &domain.RunEvent{
		Data: &domain.LogEventData{Level: "warn", Message: "sandbox cleanup failed"},
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, ok := store.FindLogMessage("cleanup")
	if !ok {
		t.Fatal("expected log message to be captured")
	}
	if got.Level != "warn" {
		t.Fatalf("expected level warn, got %q", got.Level)
	}
}
