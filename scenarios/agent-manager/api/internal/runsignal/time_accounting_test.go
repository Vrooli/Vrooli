package runsignal

import (
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

func TestDeriveTimeAccountingConservesDurationAndAttributesTokens(t *testing.T) {
	start := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)
	event := func(offset time.Duration, data domain.EventPayload) *domain.RunEvent {
		return &domain.RunEvent{ID: uuid.New(), Timestamp: start.Add(offset), Data: data}
	}
	accounting := DeriveTimeAccounting([]*domain.RunEvent{
		event(time.Minute, &domain.MessageEventData{Role: "user"}),
		event(3*time.Minute, &domain.ToolCallEventData{}),
		event(5*time.Minute, &domain.UsageEventData{InputTokens: 4, OutputTokens: 6}),
		event(6*time.Minute, &domain.ToolResultEventData{}),
		event(8*time.Minute, &domain.MessageEventData{Role: "assistant"}),
	}, &start, &end)
	if got, want := accounting.DurationMS(), end.Sub(start).Milliseconds(); got != want {
		t.Fatalf("duration=%d want %d", got, want)
	}
	if accounting.UnattributableMS != time.Minute.Milliseconds() || accounting.ModelGeneratingMS != 4*time.Minute.Milliseconds() || accounting.ToolExecutingMS != 3*time.Minute.Milliseconds() || accounting.AwaitingHumanMS != 2*time.Minute.Milliseconds() {
		t.Fatalf("unexpected accounting: %+v", accounting)
	}
	if accounting.ToolTokens != 10 || accounting.Tokens() != 10 {
		t.Fatalf("unexpected token accounting: %+v", accounting)
	}
}

func TestDeriveTimeAccountingLeavesMissingRunBoundsUnknown(t *testing.T) {
	if got := DeriveTimeAccounting(nil, nil, nil); got != (TimeAccounting{}) {
		t.Fatalf("got %+v", got)
	}
}
