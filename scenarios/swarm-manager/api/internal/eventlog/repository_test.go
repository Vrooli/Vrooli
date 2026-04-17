package eventlog_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"swarm-manager/internal/eventlog"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	repo := eventlog.NewSQLiteRepository(db)
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return db
}

func TestInitSchemaIdempotent(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)

	// Calling InitSchema a second time must not fail.
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("second InitSchema call failed: %v", err)
	}
}

func TestAppendAndSince(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)

	// Append 3 events.
	ids := make([]int64, 3)
	for i := range 3 {
		id, err := repo.Append(ctx, eventlog.Event{
			Timestamp:  now.Add(time.Duration(i) * time.Second),
			EntityType: eventlog.EntityBacklogItem,
			EntityID:   "execute/item-" + string(rune('a'+i)),
			EventType:  eventlog.EventBacklogCreated,
			ActorType:  "user",
			Metadata:   json.RawMessage(`{"kind":"execute"}`),
		})
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		ids[i] = id
	}

	// IDs should be sequential.
	if ids[0] >= ids[1] || ids[1] >= ids[2] {
		t.Errorf("IDs not sequential: %v", ids)
	}

	// Since(0) returns all.
	all, err := repo.Since(ctx, 0, 100)
	if err != nil {
		t.Fatalf("since(0): %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("since(0) expected 3, got %d", len(all))
	}

	// Since(ids[1]) returns only the last event.
	after, err := repo.Since(ctx, ids[1], 100)
	if err != nil {
		t.Fatalf("since(%d): %v", ids[1], err)
	}
	if len(after) != 1 {
		t.Fatalf("since(%d) expected 1, got %d", ids[1], len(after))
	}
	if after[0].ID != ids[2] {
		t.Errorf("expected ID %d, got %d", ids[2], after[0].ID)
	}

	// Since with limit.
	limited, err := repo.Since(ctx, 0, 2)
	if err != nil {
		t.Fatalf("since with limit: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("since with limit expected 2, got %d", len(limited))
	}
}

func TestAll(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)
	ctx := context.Background()

	// Empty table.
	events, err := repo.All(ctx)
	if err != nil {
		t.Fatalf("all on empty: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}

	// Add events.
	for i := range 5 {
		_, err := repo.Append(ctx, eventlog.Event{
			Timestamp:  time.Now().UTC(),
			EntityType: eventlog.EntityInitiative,
			EntityID:   "init-" + string(rune('a'+i)),
			EventType:  eventlog.EventInitiativeCreated,
		})
		if err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	events, err = repo.All(ctx)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}

	// Verify ordering.
	for i := 1; i < len(events); i++ {
		if events[i].ID <= events[i-1].ID {
			t.Errorf("events not ordered: ID %d <= %d", events[i].ID, events[i-1].ID)
		}
	}
}

func TestMaxID(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)
	ctx := context.Background()

	// Empty table returns 0.
	maxID, err := repo.MaxID(ctx)
	if err != nil {
		t.Fatalf("maxid empty: %v", err)
	}
	if maxID != 0 {
		t.Errorf("expected 0, got %d", maxID)
	}

	// After appending, returns latest ID.
	var lastID int64
	for range 3 {
		id, err := repo.Append(ctx, eventlog.Event{
			Timestamp:  time.Now().UTC(),
			EntityType: eventlog.EntityExecution,
			EntityID:   "exec-1",
			EventType:  eventlog.EventExecutionCreated,
		})
		if err != nil {
			t.Fatalf("append: %v", err)
		}
		lastID = id
	}

	maxID, err = repo.MaxID(ctx)
	if err != nil {
		t.Fatalf("maxid: %v", err)
	}
	if maxID != lastID {
		t.Errorf("expected %d, got %d", lastID, maxID)
	}
}

func TestMetadataRoundTrip(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)
	ctx := context.Background()

	payload := eventlog.StatusChangePayload{From: "backlog", To: "in_progress"}
	data, _ := json.Marshal(payload)

	_, err := repo.Append(ctx, eventlog.Event{
		Timestamp:  time.Now().UTC(),
		EntityType: eventlog.EntityBacklogItem,
		EntityID:   "execute/my-item",
		EventType:  eventlog.EventBacklogStatusChanged,
		Metadata:   data,
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	events, err := repo.All(ctx)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1, got %d", len(events))
	}

	var got eventlog.StatusChangePayload
	if err := json.Unmarshal(events[0].Metadata, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.From != "backlog" || got.To != "in_progress" {
		t.Errorf("metadata mismatch: %+v", got)
	}
}

func TestNilMetadata(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)
	ctx := context.Background()

	_, err := repo.Append(ctx, eventlog.Event{
		Timestamp:  time.Now().UTC(),
		EntityType: eventlog.EntityInitiative,
		EntityID:   "init-a",
		EventType:  eventlog.EventInitiativeArchived,
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	events, err := repo.All(ctx)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1, got %d", len(events))
	}
	if events[0].Metadata != nil {
		t.Errorf("expected nil metadata, got %s", events[0].Metadata)
	}
}
