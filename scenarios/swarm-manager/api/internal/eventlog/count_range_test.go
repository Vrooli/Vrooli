package eventlog_test

import (
	"context"
	"testing"
	"time"

	"swarm-manager/internal/eventlog"
)

// appendAt is a tiny helper that appends one event at ts with the given type and
// optional raw metadata.
func appendAt(t *testing.T, repo *eventlog.SQLiteRepository, ts time.Time, et eventlog.EventType, meta string) {
	t.Helper()
	e := eventlog.Event{
		Timestamp:  ts,
		EntityType: eventlog.EntityBacklogItem,
		EntityID:   "execute/x",
		EventType:  et,
		ActorType:  "user",
	}
	if meta != "" {
		e.Metadata = []byte(meta)
	}
	if _, err := repo.Append(context.Background(), e); err != nil {
		t.Fatalf("append: %v", err)
	}
}

func TestCountEventsInRange(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)
	ctx := context.Background()

	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	// Three creations: at base, base+1h (with a fractional-second offset to
	// exercise the parsed-time comparison, not lexical), base+48h.
	appendAt(t, repo, base, eventlog.EventBacklogCreated, "")
	appendAt(t, repo, base.Add(time.Hour).Add(500*time.Millisecond), eventlog.EventBacklogCreated, "")
	appendAt(t, repo, base.Add(48*time.Hour), eventlog.EventBacklogCreated, "")
	// A different event type that must never be counted.
	appendAt(t, repo, base, eventlog.EventBacklogArchived, "")

	from := base
	to := base.Add(24 * time.Hour)
	n, err := repo.CountEventsInRange(ctx, eventlog.EventBacklogCreated, from, to)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 creations in [base, base+24h) (incl. the +1h.5s one), got %d", n)
	}

	// Boundary: `to` is exclusive — the base+48h event is outside a window
	// ending exactly at base+48h's start.
	n, err = repo.CountEventsInRange(ctx, eventlog.EventBacklogCreated, base.Add(48*time.Hour), base.Add(72*time.Hour))
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 creation at the inclusive `from` boundary, got %d", n)
	}

	// Empty window.
	n, err = repo.CountEventsInRange(ctx, eventlog.EventBacklogCreated, base.Add(100*time.Hour), base.Add(200*time.Hour))
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("want 0 in an empty window, got %d", n)
	}
}

func TestCountStatusTransitionsInRange(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)
	ctx := context.Background()

	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// Two completions in window, one non-completion transition, one completion
	// out of window, one malformed payload.
	appendAt(t, repo, base.Add(time.Hour), eventlog.EventBacklogStatusChanged, `{"from":"in_progress","to":"completed"}`)
	appendAt(t, repo, base.Add(2*time.Hour), eventlog.EventBacklogStatusChanged, `{"from":"queued","to":"completed"}`)
	appendAt(t, repo, base.Add(3*time.Hour), eventlog.EventBacklogStatusChanged, `{"from":"todo","to":"in_progress"}`)
	appendAt(t, repo, base.Add(72*time.Hour), eventlog.EventBacklogStatusChanged, `{"from":"in_progress","to":"completed"}`)
	appendAt(t, repo, base.Add(4*time.Hour), eventlog.EventBacklogStatusChanged, `not-json`)

	n, err := repo.CountStatusTransitionsInRange(ctx, eventlog.EventBacklogStatusChanged, "completed", base, base.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("count transitions: %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 completions in window (excluding non-completion, out-of-window, and malformed), got %d", n)
	}
}
