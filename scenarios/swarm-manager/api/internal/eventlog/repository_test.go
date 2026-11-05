package eventlog_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"swarm-manager/internal/eventlog"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *database.RoutedDB {
	t.Helper()
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { sqldb.Close() })

	db := database.NewFromPrimary(sqldb)
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

func TestInitSchemaMigratesLegacyEvidenceTables(t *testing.T) {
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	legacy := []string{
		`CREATE TABLE evidence_observations (id INTEGER PRIMARY KEY AUTOINCREMENT, source_system TEXT NOT NULL, source_event_id TEXT NOT NULL, run_id TEXT NOT NULL, subject_kind TEXT NOT NULL, subject_id TEXT NOT NULL, action TEXT NOT NULL, confidence TEXT NOT NULL, verification TEXT NOT NULL, content_digest TEXT NOT NULL DEFAULT '', metadata_json TEXT NOT NULL DEFAULT '{}', observed_at TEXT NOT NULL, ownership_status TEXT NOT NULL, actor TEXT NOT NULL DEFAULT '', reason TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE evidence_links (observation_id INTEGER NOT NULL, owner_kind TEXT NOT NULL, owner_id TEXT NOT NULL, owner_round INTEGER NOT NULL DEFAULT 0, linked_at TEXT NOT NULL, PRIMARY KEY(observation_id, owner_kind, owner_id, owner_round))`,
		`CREATE TABLE evidence_migration_audits (migration_key TEXT PRIMARY KEY, source_count INTEGER NOT NULL, projected_count INTEGER NOT NULL, source_digest TEXT NOT NULL, projected_digest TEXT NOT NULL, completed_at TEXT NOT NULL)`,
		`CREATE TABLE evidence_checkpoints (producer_id TEXT NOT NULL, run_id TEXT NOT NULL, fact_kind TEXT NOT NULL, cursor TEXT NOT NULL, updated_at TEXT NOT NULL, PRIMARY KEY(producer_id, run_id, fact_kind))`,
		`CREATE TABLE evidence_watermarks (producer_id TEXT NOT NULL, run_id TEXT NOT NULL, fact_kind TEXT NOT NULL, coverage TEXT NOT NULL, completed_at TEXT NOT NULL, PRIMARY KEY(producer_id, run_id, fact_kind))`,
		`INSERT INTO evidence_observations (source_system, source_event_id, run_id, subject_kind, subject_id, action, confidence, verification, metadata_json, observed_at, ownership_status) VALUES ('old-review', 'event-1', 'run-1', 'criterion', 'execute/item/criterion', 'settled', 'reported', 'none', '{"legacy":true}', '2026-07-30T00:00:00Z', 'owned')`,
		`INSERT INTO evidence_links (observation_id, owner_kind, owner_id, owner_round, linked_at) VALUES (1, 'backlog', 'execute/item', 2, '2026-07-30T00:00:00Z')`,
		`INSERT INTO evidence_migration_audits (migration_key, source_count, projected_count, source_digest, projected_digest, completed_at) VALUES ('old/v1', 1, 1, 'source', 'projection', '2026-07-30T00:00:00Z')`,
		`INSERT INTO evidence_checkpoints (producer_id, run_id, fact_kind, cursor, updated_at) VALUES ('old-review', 'run-1', 'review_evidence', '0', '2026-07-30T00:00:00Z')`,
		`INSERT INTO evidence_watermarks (producer_id, run_id, fact_kind, coverage, completed_at) VALUES ('old-review', 'run-1', 'review_evidence', 'complete', '2026-07-30T00:00:00Z')`,
	}
	for _, statement := range legacy {
		if _, err := sqldb.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := eventlog.NewSQLiteRepository(database.NewFromPrimary(sqldb))
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}
	var producer, attemptRef string
	if err := sqldb.QueryRow(`SELECT producer FROM evidence_observations WHERE id = 'legacy/1'`).Scan(&producer); err != nil {
		t.Fatal(err)
	}
	if err := sqldb.QueryRow(`SELECT attempt_ref FROM evidence_links WHERE observation_id = 'legacy/1'`).Scan(&attemptRef); err != nil {
		t.Fatal(err)
	}
	if producer != "legacy:old-review" || attemptRef != "legacy/backlog/execute/item/2" {
		t.Fatalf("migrated evidence = producer %q, attempt %q", producer, attemptRef)
	}
	var parity bool
	if err := sqldb.QueryRow(`SELECT parity_proven FROM evidence_migration_audits WHERE migration_key = 'old/v1'`).Scan(&parity); err != nil {
		t.Fatal(err)
	}
	if !parity {
		t.Fatal("expected migrated parity audit")
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

func TestAppendAttributedRequiresActorID(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)
	if _, err := repo.AppendAttributed(context.Background(), eventlog.Event{EntityType: eventlog.EntitySystem, EntityID: "test", EventType: eventlog.EventBacklogCreated}); err == nil {
		t.Fatal("AppendAttributed() error = nil, want actor_id validation")
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

func TestQueryByEntityUsesCursorAndEntityScope(t *testing.T) {
	db := setupTestDB(t)
	repo := eventlog.NewSQLiteRepository(db)
	ctx := context.Background()
	first, err := repo.Append(ctx, eventlog.Event{Timestamp: time.Now().UTC(), EntityType: eventlog.EntityBacklogItem, EntityID: "execute/item-a", EventType: eventlog.EventBacklogCreated})
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Append(ctx, eventlog.Event{Timestamp: time.Now().UTC(), EntityType: eventlog.EntityBacklogItem, EntityID: "execute/item-a", EventType: eventlog.EventBacklogStatusChanged})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Append(ctx, eventlog.Event{Timestamp: time.Now().UTC(), EntityType: eventlog.EntityBacklogItem, EntityID: "execute/item-b", EventType: eventlog.EventBacklogCreated}); err != nil {
		t.Fatal(err)
	}
	events, err := repo.QueryByEntity(ctx, eventlog.EntityBacklogItem, "execute/item-a", first, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != second {
		t.Fatalf("QueryByEntity() = %#v, want only event %d", events, second)
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
