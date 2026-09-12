package events

import (
	"context"
	"testing"

	intevents "web-console/internal/events"
)

func TestAdapterProjectsRecentEventsAndCount(t *testing.T) {
	logger := intevents.NewLogger(2)
	logger.Emit("session.created", "s1", map[string]string{"kind": "test"})
	logger.Emit("session.connected", "s2", nil)
	logger.Emit("session.deleted", "s3", nil)
	adapter := &Adapter{Logger: logger}
	rows := adapter.Recent(context.Background(), 2)
	if len(rows) != 2 || rows[0].SessionID != "s2" || rows[1].Type != "session.deleted" || rows[0].Timestamp == "" {
		t.Fatalf("recent=%+v", rows)
	}
	if adapter.Count(context.Background()) != 2 {
		t.Fatalf("count=%d", adapter.Count(context.Background()))
	}
}
