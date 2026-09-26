package mocks

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/adapters/event"
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

func TestFakeEventStoreLifecycleTypedFilteringAndErrorKnobs(t *testing.T) {
	store := NewFakeEventStore()
	runID := uuid.New()
	typed := &domain.RunEvent{EventType: domain.EventTypeRetryAttempt, Data: &domain.TypedEventData{}}
	if err := store.Append(context.Background(), runID, nil, typed); err != nil {
		t.Fatal(err)
	}
	if count, err := store.Count(context.Background(), runID); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	got, err := store.Get(context.Background(), runID, event.GetOptions{})
	if err != nil || len(got) != 1 || got[0] != typed || len(store.TypedEvents(runID, domain.EventTypeRetryAttempt)) != 1 {
		t.Fatalf("events=%+v err=%v", got, err)
	}
	stream, err := store.Stream(context.Background(), runID, event.StreamOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, open := <-stream; open {
		t.Fatal("default stream should close")
	}
	if err := store.Delete(context.Background(), runID); err != nil {
		t.Fatal(err)
	}

	injected := errors.New("injected")
	store.AppendErr, store.GetErr, store.StreamErr, store.CountErr, store.DeleteErr = injected, injected, injected, injected, injected
	if err := store.Append(context.Background(), runID, typed); !errors.Is(err, injected) {
		t.Fatalf("append=%v", err)
	}
	if _, err := store.Get(context.Background(), runID, event.GetOptions{}); !errors.Is(err, injected) {
		t.Fatalf("get=%v", err)
	}
	if _, err := store.Stream(context.Background(), runID, event.StreamOptions{}); !errors.Is(err, injected) {
		t.Fatalf("stream=%v", err)
	}
	if _, err := store.Count(context.Background(), runID); !errors.Is(err, injected) {
		t.Fatalf("count=%v", err)
	}
	if err := store.Delete(context.Background(), runID); !errors.Is(err, injected) {
		t.Fatalf("delete=%v", err)
	}
}
