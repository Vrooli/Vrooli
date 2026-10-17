package store

import (
	"context"
	"testing"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func receiptEvent(t *testing.T, id, target, operation string, at time.Time, caller string, verified bool) Event {
	t.Helper()
	envelope := &domain.EventEnvelope{
		EventId: id, EventType: receiptEventType, OccurredAt: timestamppb.New(at),
		Source:      &domain.EventSource{Scenario: "source"},
		Target:      &domain.EventTarget{Scenario: target, Operation: operation, Protocol: "connect"},
		Attribution: &domain.EventAttribution{SubjectKind: "agent", SubjectId: caller, Verified: verified},
	}
	envelope.Data, _ = anypb.New(&domain.ReceiptData{Outcome: "success", StatusCode: 200})
	payload, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return Event{EventID: id, EventType: receiptEventType, TargetScenario: target, Payload: payload, CreatedAt: at}
}

func TestAggregateReceiptsGroupsWindowAndAttribution(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 18, 0, 0, 0, time.UTC)
	s, err := NewSQLiteStore(ctx, SQLiteConfig{DBPath: ":memory:", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, event := range []Event{
		receiptEvent(t, "one", "target-a", "POST /read", now.Add(-time.Hour), "agent-a", true),
		receiptEvent(t, "two", "target-a", "POST /read", now.Add(-30*time.Minute), "agent-a", true),
		receiptEvent(t, "three", "target-a", "POST /read", now.Add(-20*time.Minute), "", false),
		receiptEvent(t, "four", "target-a", "POST /write", now.Add(-10*time.Minute), "agent-b", true),
		receiptEvent(t, "old", "target-a", "POST /read", now.Add(-48*time.Hour), "agent-c", true),
	} {
		_, err := s.Insert(ctx, event)
		if err != nil {
			t.Fatal(err)
		}
	}
	rows, err := s.AggregateReceipts(ctx, ReceiptAggregateFilter{Since: now.Add(-2 * time.Hour), Until: now})
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if rows[0].GetInvocationCount() != 3 || rows[0].GetDistinctVerifiedCallers() != 1 || rows[0].GetUnattributedRemainder() != 1 || rows[0].GetLastInvokedAt() != "2026-08-18T17:40:00Z" || rows[1].GetInvocationCount() != 1 {
		t.Fatalf("aggregates=%+v", rows)
	}
}

func TestAggregateReceiptsReturnsEmptyWindow(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 18, 0, 0, 0, time.UTC)
	s, err := NewSQLiteStore(ctx, SQLiteConfig{DBPath: ":memory:", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rows, err := s.AggregateReceipts(ctx, ReceiptAggregateFilter{Since: now.Add(-time.Hour), Until: now})
	if err != nil || len(rows) != 0 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}
