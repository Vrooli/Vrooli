package stats

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/eventlog"

	"github.com/google/uuid"
)

func TestFoldRunIsolatesTypedOperationalEvents(t *testing.T) {
	_, _, db := newTestEngine(t)
	runID, otherRunID := uuid.New(), uuid.New()
	payloads := []eventlog.Payload{
		eventlog.ModelFallbackAttemptedPayload{From: "primary", To: "fallback", Reason: "rate_limit", AttemptNo: 1},
		eventlog.HeartbeatMissPayload{Target: "run"},
	}
	for index, payload := range payloads {
		event, err := eventlog.BuildEvent(runID, payload)
		if err != nil {
			t.Fatal(err)
		}
		event.Timestamp = time.Now().UTC()
		insertEvent(t, db, runID, int64(index+1), event)
	}
	other, err := eventlog.BuildEvent(otherRunID, eventlog.HeartbeatMissPayload{Target: "other"})
	if err != nil {
		t.Fatal(err)
	}
	other.Timestamp = time.Now().UTC()
	insertEvent(t, db, otherRunID, 1, other)

	snapshot, err := FoldRun(context.Background(), eventlog.NewSQLiteRepository(db), runID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.EventCount != 2 || snapshot.Fallback.ModelAttempts != 1 || snapshot.Heartbeat.TotalMisses != 1 {
		t.Fatalf("unexpected isolated run snapshot: %#v", snapshot)
	}
}
