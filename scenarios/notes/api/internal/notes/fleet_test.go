package notes

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fleetTestDriver struct {
	exec  func(string, []driver.NamedValue) (driver.Result, error)
	query func(string, []driver.NamedValue) (driver.Rows, error)
}
type (
	fleetTestConn struct{ driver *fleetTestDriver }
	fleetTestRows struct {
		columns []string
		data    [][]driver.Value
		index   int
	}
)
type fleetTestResult int64

var fakeFleetDriverSequence atomic.Uint64

func (d fleetTestDriver) Open(string) (driver.Conn, error)  { return fleetTestConn{driver: &d}, nil }
func (c fleetTestConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (c fleetTestConn) Close() error                        { return nil }
func (c fleetTestConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }
func (c fleetTestConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.driver.exec(query, args)
}

func (c fleetTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.driver.query(query, args)
}
func (r *fleetTestRows) Columns() []string { return r.columns }
func (r *fleetTestRows) Close() error      { return nil }
func (r *fleetTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.index])
	r.index++
	return nil
}
func (fleetTestResult) LastInsertId() (int64, error)   { return 0, nil }
func (r fleetTestResult) RowsAffected() (int64, error) { return int64(r), nil }

// openFakeFleetDB is a SQL-shape fake; these tests do not prove PostgreSQL trigger,
// transaction, row-lock, or migration behavior.
func openFakeFleetDB(t *testing.T, d fleetTestDriver) *sql.DB {
	t.Helper()
	name := "notes-fleet-test-" + t.Name() + "-" + strconv.FormatUint(fakeFleetDriverSequence.Add(1), 10)
	sql.Register(name, d)
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type recordingEmbedder struct {
	recipe     string
	vector     []float32
	dimensions int
}

type fixedPolicy struct{ policy EmbeddingRolePolicy }

func (p fixedPolicy) ResolveEmbeddingPolicy(_ context.Context, _ string) (EmbeddingRolePolicy, error) {
	return p.policy, nil
}

type sequencePolicy struct {
	policies []EmbeddingRolePolicy
	reads    int
}

func (p *sequencePolicy) ResolveEmbeddingPolicy(_ context.Context, _ string) (EmbeddingRolePolicy, error) {
	index := p.reads
	p.reads++
	if index >= len(p.policies) {
		index = len(p.policies) - 1
	}
	return p.policies[index], nil
}

type recordingGateway struct {
	role   string
	vector []float32
	calls  int
}

func (g *recordingGateway) Embed(_ context.Context, role, _ string) ([]float32, error) {
	g.calls++
	g.role = role
	return g.vector, nil
}

type fixedFleetGenerationConfig struct {
	config FleetGenerationConfig
	err    error
}

func (r fixedFleetGenerationConfig) ResolveFleetGenerationConfig(context.Context, string) (FleetGenerationConfig, error) {
	return r.config, r.err
}

func TestConfiguredFleetSourceRequiresExplicitOwnerAndRecipeIdentity(t *testing.T) {
	db := openFakeFleetDB(t, fleetTestDriver{
		exec:  func(string, []driver.NamedValue) (driver.Result, error) { return fleetTestResult(0), nil },
		query: func(string, []driver.NamedValue) (driver.Rows, error) { return &fleetTestRows{}, nil },
	})
	config := FleetGenerationConfig{
		GenerationID: "arm-42",
		OwnerID:      "notes.api", Alias: "notes_live", Collection: "notes_candidate_42",
		RecipeID: "recipe-sha256-123", EmbeddingRole: "embedding.notes.v2", EmbeddingDimensions: 2,
	}
	policy := fixedPolicy{policy: EmbeddingRolePolicy{RecipeID: config.RecipeID, Role: config.EmbeddingRole, Dimensions: config.EmbeddingDimensions}}
	gateway := &recordingGateway{vector: []float32{.1, .2}}
	configured, err := NewConfiguredFleetSource(context.Background(), db, "arm-42", fixedFleetGenerationConfig{config: config}, policy, gateway)
	if err != nil {
		t.Fatalf("NewConfiguredFleetSource() error = %v", err)
	}
	if configured.Config != config || configured.Source.DB != db {
		t.Fatalf("configured source = %+v, want exact owner-supplied generation config", configured)
	}
	if _, _, err := configured.ReadFleetChanges(context.Background(), "environment-default-recipe", "0", "0", 1); err == nil {
		t.Fatal("configured source accepted a recipe other than the generation's immutable recipe")
	}
	if _, _, err := configured.Source.Embedder.EmbedDocument(context.Background(), config.RecipeID, "note"); err != nil || gateway.role != config.EmbeddingRole {
		t.Fatalf("source embedding did not use configured role %q: role=%q err=%v", config.EmbeddingRole, gateway.role, err)
	}
}

func TestConfiguredFleetSourceRejectsMissingCollectionAndDimensionDrift(t *testing.T) {
	db := openFakeFleetDB(t, fleetTestDriver{
		exec:  func(string, []driver.NamedValue) (driver.Result, error) { return fleetTestResult(0), nil },
		query: func(string, []driver.NamedValue) (driver.Rows, error) { return &fleetTestRows{}, nil },
	})
	valid := FleetGenerationConfig{
		GenerationID: "arm-42",
		OwnerID:      "notes.api", Alias: "notes_live", Collection: "notes_candidate_42",
		RecipeID: "recipe-1", EmbeddingRole: "embedding.notes.v1", EmbeddingDimensions: 2,
	}
	policy := fixedPolicy{policy: EmbeddingRolePolicy{RecipeID: "recipe-1", Role: "embedding.notes.v1", Dimensions: 3}}
	for name, config := range map[string]FleetGenerationConfig{
		"missing collection":        func() FleetGenerationConfig { value := valid; value.Collection = ""; return value }(),
		"embedding dimension drift": valid,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewConfiguredFleetSource(context.Background(), db, "arm-42", fixedFleetGenerationConfig{config: config}, policy, &recordingGateway{vector: []float32{.1, .2}})
			if err == nil {
				t.Fatal("NewConfiguredFleetSource() accepted incomplete or mismatched config")
			}
		})
	}
}

func TestRoleBoundEmbeddingUsesRecipePolicyAndRejectsMismatches(t *testing.T) {
	policy := fixedPolicy{policy: EmbeddingRolePolicy{RecipeID: "recipe-1", Role: "embedding.notes.v2", Dimensions: 2}}
	gateway := &recordingGateway{vector: []float32{.1, .2}}
	resolver := RoleBoundEmbeddingResolver{Policies: policy, Gateway: gateway}
	vector, dimensions, err := resolver.EmbedDocument(context.Background(), "recipe-1", "note")
	if err != nil || dimensions != 2 || len(vector) != 2 || gateway.role != "embedding.notes.v2" {
		t.Fatalf("vector=%v dimensions=%d role=%q err=%v", vector, dimensions, gateway.role, err)
	}
	if _, _, err := resolver.EmbedDocument(context.Background(), "recipe-other", "note"); err == nil {
		t.Fatal("resolver accepted policy for another immutable recipe")
	}
	gateway.vector = []float32{.1}
	if _, _, err := resolver.EmbedDocument(context.Background(), "recipe-1", "note"); err == nil {
		t.Fatal("resolver accepted vector with recipe dimension mismatch")
	}
}

func (e *recordingEmbedder) EmbedDocument(_ context.Context, recipe, _ string) ([]float32, int, error) {
	e.recipe = recipe
	return e.vector, e.dimensions, nil
}

func TestFleetSnapshotPagesResumeAtStableNoteCursor(t *testing.T) {
	// The fake returns the requested ordered page. A second read uses the first
	// page's cursor and resumes after it without changing the pinned watermark.
	rowsByPage := map[string][][]driver.Value{
		"":                                     {{"00000000-0000-0000-0000-000000000001", fleetPayload(t, "00000000-0000-0000-0000-000000000001", "first")}, {"00000000-0000-0000-0000-000000000002", fleetPayload(t, "00000000-0000-0000-0000-000000000002", "second")}},
		"00000000-0000-0000-0000-000000000001": {{"00000000-0000-0000-0000-000000000002", fleetPayload(t, "00000000-0000-0000-0000-000000000002", "second")}},
	}
	db := openFakeFleetDB(t, fleetTestDriver{query: func(query string, args []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(query, "FROM notes_fleet_snapshots AS snapshot") {
			return &fleetTestRows{columns: []string{"watermark", "snapshot_version", "baseline_version", "baseline_watermark"}, data: [][]driver.Value{{int64(19), int64(1), int64(1), int64(19)}}}, nil
		}
		if !strings.Contains(query, "DISTINCT ON (note_id)") || !strings.Contains(query, "sequence <= $1") {
			return nil, errors.New("unexpected query")
		}
		return &fleetTestRows{columns: []string{"note_id", "payload"}, data: rowsByPage[args[1].Value.(string)]}, nil
	}})
	embedder := &recordingEmbedder{vector: []float32{1, 2}, dimensions: 2}
	source := FleetSource{DB: db, Embedder: embedder}
	snapshot := FleetSnapshot{ID: "pinned", Watermark: "19"}
	first, cursor, done, err := source.ReadFleetSnapshotPage(context.Background(), "recipe-sha256-a", snapshot, "", 1)
	if err != nil || done || len(first) != 1 || cursor != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("first page = %#v cursor=%q done=%v err=%v", first, cursor, done, err)
	}
	second, next, done, err := source.ReadFleetSnapshotPage(context.Background(), "recipe-sha256-a", snapshot, cursor, 1)
	if err != nil || !done || len(second) != 1 || next != "00000000-0000-0000-0000-000000000002" {
		t.Fatalf("resumed page = %#v cursor=%q done=%v err=%v", second, next, done, err)
	}
	if first[0].ID != fleetSourceID("user-1", "00000000-0000-0000-0000-000000000001") {
		t.Fatalf("snapshot point ID does not use canonical tenant source identity: %q", first[0].ID)
	}
	if embedder.recipe != "recipe-sha256-a" {
		t.Fatalf("embedder received recipe %q", embedder.recipe)
	}
}

func TestFleetSnapshotRejectsPreBackfillGeneration(t *testing.T) {
	queries := 0
	db := openFakeFleetDB(t, fleetTestDriver{query: func(query string, _ []driver.NamedValue) (driver.Rows, error) {
		queries++
		if !strings.Contains(query, "FROM notes_fleet_snapshots AS snapshot") {
			return nil, errors.New("page read must not proceed before snapshot baseline validation")
		}
		return &fleetTestRows{columns: []string{"watermark", "snapshot_version", "baseline_version", "baseline_watermark"}, data: [][]driver.Value{{int64(10), int64(0), int64(1), int64(10)}}}, nil
	}})
	_, _, _, err := (FleetSource{DB: db, Embedder: &recordingEmbedder{}}).ReadFleetSnapshotPage(context.Background(), "recipe-1", FleetSnapshot{ID: "old-snapshot", Watermark: "10"}, "", 10)
	if err == nil || !strings.Contains(err.Error(), "predates the completed source backfill") || queries != 1 {
		t.Fatalf("old snapshot error=%v queries=%d", err, queries)
	}
}

func fleetPayload(t *testing.T, id, title string) []byte {
	return fleetPayloadForUser(t, id, "user-1", title)
}

func fleetPayloadForUser(t *testing.T, id, userID, title string) []byte {
	t.Helper()
	raw, err := json.Marshal(FleetDocument{NoteID: id, UserID: userID, Title: title, Content: "body"})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestFleetChangeFeedRetainsUpdatesAndDeletesInSequence(t *testing.T) {
	db := openFakeFleetDB(t, fleetTestDriver{query: func(query string, args []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(query, "sequence > GREATEST($1::BIGINT, $2::BIGINT)") || !strings.Contains(query, "ORDER BY sequence") || !strings.Contains(query, "user_id::text") {
			return nil, errors.New("unexpected query")
		}
		return &fleetTestRows{columns: []string{"sequence", "note_id", "user_id", "event_type", "payload", "move_group"}, data: [][]driver.Value{
			{int64(20), "note-a", "user-a", "upsert", fleetPayloadForUser(t, "note-a", "user-a", "updated"), nil},
			{int64(21), "note-a", "user-a", "delete", nil, nil},
		}}, nil
	}})
	embedder := &recordingEmbedder{vector: []float32{0, 1}, dimensions: 2}
	changes, next, err := (FleetSource{DB: db, Embedder: embedder}).ReadFleetChanges(context.Background(), "recipe-v2", "19", "19", 10)
	if err != nil || len(changes) != 2 || next != "21" {
		t.Fatalf("changes=%#v next=%q err=%v", changes, next, err)
	}
	if changes[0].Sequence != "20" || len(changes[0].Upserts) != 1 || changes[0].Upserts[0].Payload["embedding_recipe_id"] != "recipe-v2" || changes[0].Upserts[0].Payload["user_id"] != "user-a" || changes[0].Upserts[0].Payload["note_id"] != "note-a" {
		t.Fatalf("upsert not bound to recipe: %#v", changes[0])
	}
	if changes[0].Upserts[0].ID != changes[0].Upserts[0].Payload["source_id"] {
		t.Fatalf("upsert point ID %q differs from its source_id %q", changes[0].Upserts[0].ID, changes[0].Upserts[0].Payload["source_id"])
	}
	if changes[1].Sequence != "21" || len(changes[1].Deletes) != 1 || changes[1].Deletes[0] != fleetSourceID("user-a", "note-a") || changes[1].DeleteUserID != "user-a" {
		t.Fatalf("delete tombstone lost: %#v", changes[1])
	}
	if changes[0].Upserts[0].ID != changes[1].Deletes[0] {
		t.Fatalf("delete key %q differs from prior upsert point ID %q", changes[1].Deletes[0], changes[0].Upserts[0].ID)
	}
}

func TestFleetSnapshotUpsertAndDeleteShareCanonicalTenantIdentity(t *testing.T) {
	// These SQL-shape fakes prove identity agreement in Notes' source contract;
	// they do not prove the PostgreSQL trigger or Qdrant applied the tombstone.
	noteID, userID := "note-a", "user-a"
	snapshotDB := openFakeFleetDB(t, fleetTestDriver{query: func(query string, _ []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(query, "FROM notes_fleet_snapshots AS snapshot") {
			return &fleetTestRows{columns: []string{"watermark", "snapshot_version", "baseline_version", "baseline_watermark"}, data: [][]driver.Value{{int64(10), int64(1), int64(1), int64(10)}}}, nil
		}
		if !strings.Contains(query, "DISTINCT ON (note_id)") {
			return nil, errors.New("unexpected snapshot query")
		}
		return &fleetTestRows{columns: []string{"note_id", "payload"}, data: [][]driver.Value{{noteID, fleetPayloadForUser(t, noteID, userID, "title")}}}, nil
	}})
	snapshotPoints, _, _, err := (FleetSource{DB: snapshotDB, Embedder: &recordingEmbedder{vector: []float32{1, 2}, dimensions: 2}}).ReadFleetSnapshotPage(context.Background(), "recipe-1", FleetSnapshot{ID: "snapshot", Watermark: "10"}, "", 1)
	if err != nil || len(snapshotPoints) != 1 {
		t.Fatalf("snapshot points=%#v err=%v", snapshotPoints, err)
	}

	changesDB := openFakeFleetDB(t, fleetTestDriver{query: func(query string, _ []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(query, "ORDER BY sequence") || !strings.Contains(query, "user_id::text") {
			return nil, errors.New("unexpected change query")
		}
		return &fleetTestRows{columns: []string{"sequence", "note_id", "user_id", "event_type", "payload", "move_group"}, data: [][]driver.Value{
			{int64(11), noteID, userID, "upsert", fleetPayloadForUser(t, noteID, userID, "title"), nil},
			{int64(12), noteID, userID, "delete", nil, nil},
		}}, nil
	}})
	changes, _, err := (FleetSource{DB: changesDB, Embedder: &recordingEmbedder{vector: []float32{1, 2}, dimensions: 2}}).ReadFleetChanges(context.Background(), "recipe-1", "10", "10", 10)
	if err != nil || len(changes) != 2 || len(changes[0].Upserts) != 1 || len(changes[1].Deletes) != 1 || changes[1].DeleteUserID != userID {
		t.Fatalf("changes=%#v err=%v", changes, err)
	}
	canonicalID := fleetSourceID(userID, noteID)
	if snapshotPoints[0].ID != canonicalID || changes[0].Upserts[0].ID != canonicalID || changes[1].Deletes[0] != canonicalID {
		t.Fatalf("snapshot/upsert/delete IDs differ: snapshot=%q upsert=%q delete=%q canonical=%q", snapshotPoints[0].ID, changes[0].Upserts[0].ID, changes[1].Deletes[0], canonicalID)
	}
	if snapshotPoints[0].Payload["user_id"] != userID || snapshotPoints[0].Payload["note_id"] != noteID || changes[0].Upserts[0].Payload["user_id"] != userID || changes[0].Upserts[0].Payload["note_id"] != noteID {
		t.Fatalf("tenant ownership is not auditable in point payloads: snapshot=%#v upsert=%#v", snapshotPoints[0].Payload, changes[0].Upserts[0].Payload)
	}
}

func TestFleetTenantReassignmentIsAtomicAcrossPageBoundaryAndResume(t *testing.T) {
	moveGroup := "move-group-1"
	db := openFakeFleetDB(t, fleetTestDriver{query: func(query string, args []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(query, "ORDER BY sequence") || !strings.Contains(query, "user_id::text") || !strings.Contains(query, "move_group::text") {
			return nil, errors.New("unexpected tenant-change feed query")
		}
		if len(args) < 3 {
			return nil, errors.New("change query omitted snapshot, cursor, or limit")
		}
		rows := [][]driver.Value{
			{int64(18), "note-before-move", "tenant-a", "upsert", fleetPayloadForUser(t, "note-before-move", "tenant-a", "before"), nil},
			{int64(19), "note-a", "old-tenant", "delete", nil, moveGroup},
			{int64(20), "note-a", "new-tenant", "upsert", fleetPayloadForUser(t, "note-a", "new-tenant", "moved"), moveGroup},
			{int64(21), "note-after-move", "tenant-a", "delete", nil, nil},
		}
		cursor, ok := args[1].Value.(int64)
		if !ok {
			return nil, fmt.Errorf("cursor argument has type %T", args[1].Value)
		}
		page := make([][]driver.Value, 0, 2)
		for _, row := range rows {
			if row[0].(int64) > cursor && len(page) < 2 { // limit=1 plus one lookahead row
				page = append(page, row)
			}
		}
		return &fleetTestRows{columns: []string{"sequence", "note_id", "user_id", "event_type", "payload", "move_group"}, data: page}, nil
	}})
	source := FleetSource{DB: db, Embedder: &recordingEmbedder{vector: []float32{1, 2}, dimensions: 2}}
	firstPage, cursor, err := source.ReadFleetChanges(context.Background(), "recipe-1", "17", "17", 1)
	if err != nil || len(firstPage) != 1 || firstPage[0].Sequence != "18" || cursor != "18" {
		t.Fatalf("preceding page=%#v cursor=%q err=%v", firstPage, cursor, err)
	}
	// Reopening the source at the durable cursor must return the whole move pair.
	changes, next, err := source.ReadFleetChanges(context.Background(), "recipe-1", "17", cursor, 1)
	if err != nil || len(changes) != 1 || next != "20" {
		t.Fatalf("resumed tenant changes=%#v next=%q err=%v", changes, next, err)
	}
	oldKey := fleetSourceID("old-tenant", "note-a")
	newKey := fleetSourceID("new-tenant", "note-a")
	if changes[0].Sequence != "20" || changes[0].MoveGroup != moveGroup || changes[0].DeleteUserID != "old-tenant" || !changes[0].DeleteTenantKnown || len(changes[0].Deletes) != 1 || changes[0].Deletes[0] != oldKey {
		t.Fatalf("old-tenant tombstone group=%#v", changes[0])
	}
	if len(changes[0].Upserts) != 1 || changes[0].Upserts[0].ID != newKey || changes[0].Upserts[0].Payload["user_id"] != "new-tenant" {
		t.Fatalf("new-tenant upsert group=%#v", changes[0])
	}
	if oldKey == newKey {
		t.Fatal("tenant reassignment retained the old point identity")
	}
	lastPage, finalCursor, err := source.ReadFleetChanges(context.Background(), "recipe-1", "17", next, 1)
	if err != nil || len(lastPage) != 1 || lastPage[0].Sequence != "21" || finalCursor != "21" || len(lastPage[0].Deletes) != 1 {
		t.Fatalf("page after tenant move=%#v cursor=%q err=%v", lastPage, finalCursor, err)
	}
}

func TestFleetChangeFeedRejectsSequenceGap(t *testing.T) {
	db := openFakeFleetDB(t, fleetTestDriver{query: func(query string, _ []driver.NamedValue) (driver.Rows, error) {
		return &fleetTestRows{columns: []string{"sequence", "note_id", "user_id", "event_type", "payload", "move_group"}, data: [][]driver.Value{{int64(21), "note-b", "user-b", "delete", nil, nil}}}, nil
	}})
	_, _, err := (FleetSource{DB: db, Embedder: &recordingEmbedder{}}).ReadFleetChanges(context.Background(), "recipe", "19", "19", 10)
	if err == nil || !strings.Contains(err.Error(), "stream gap") {
		t.Fatalf("sequence gap error = %v", err)
	}
}

func TestFleetChangeFeedRejectsCursorBehindSnapshot(t *testing.T) {
	db := openFakeFleetDB(t, fleetTestDriver{query: func(string, []driver.NamedValue) (driver.Rows, error) {
		t.Fatal("query should not run for a cursor behind its snapshot")
		return nil, nil
	}})
	_, _, err := (FleetSource{DB: db, Embedder: &recordingEmbedder{}}).ReadFleetChanges(context.Background(), "recipe", "19", "18", 10)
	if err == nil || !strings.Contains(err.Error(), "precedes its pinned snapshot") {
		t.Fatalf("stale cursor error = %v", err)
	}
}

func TestFleetWatermarkUsesMonotonicSequenceLedger(t *testing.T) {
	db := openFakeFleetDB(t, fleetTestDriver{query: func(query string, _ []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(query, "SELECT current_sequence FROM notes_fleet_sequence") {
			return nil, errors.New("watermark must come from the serialized sequence ledger")
		}
		return &fleetTestRows{columns: []string{"current_sequence"}, data: [][]driver.Value{{int64(14)}}}, nil
	}})
	watermark, err := (FleetSource{DB: db}).FleetWatermark(context.Background())
	if err != nil || watermark != "14" {
		t.Fatalf("watermark=%q err=%v", watermark, err)
	}
}

func TestExpectedFleetPointCountBindsPinnedSnapshotAndWatermark(t *testing.T) {
	queries := 0
	db := openFakeFleetDB(t, fleetTestDriver{query: func(query string, args []driver.NamedValue) (driver.Rows, error) {
		queries++
		switch {
		case strings.Contains(query, "FROM notes_fleet_snapshots AS snapshot"):
			if args[0].Value != "snapshot-1" {
				return nil, errors.New("wrong snapshot identity")
			}
			return &fleetTestRows{columns: []string{"watermark", "snapshot_version", "baseline_version", "baseline_watermark"}, data: [][]driver.Value{{int64(10), int64(1), int64(1), int64(10)}}}, nil
		case strings.Contains(query, "SELECT current_sequence FROM notes_fleet_sequence"):
			return &fleetTestRows{columns: []string{"current_sequence"}, data: [][]driver.Value{{int64(12)}}}, nil
		case strings.Contains(query, "SELECT COUNT(*)") && strings.Contains(query, "DISTINCT ON (note_id)"):
			if args[0].Value != int64(12) {
				return nil, errors.New("wrong count watermark")
			}
			return &fleetTestRows{columns: []string{"count"}, data: [][]driver.Value{{int64(2)}}}, nil
		default:
			return nil, errors.New("unexpected point count query")
		}
	}})
	count, err := (FleetSource{DB: db}).ExpectedFleetPointCount(context.Background(), "recipe-1", "snapshot-1", "12")
	if err != nil || count != 2 || queries != 3 {
		t.Fatalf("point count=%d queries=%d err=%v", count, queries, err)
	}
}

func TestExpectedFleetPointCountRejectsWatermarkBeforeSnapshot(t *testing.T) {
	db := openFakeFleetDB(t, fleetTestDriver{query: func(query string, _ []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(query, "FROM notes_fleet_snapshots AS snapshot") {
			return nil, errors.New("expected pinned baseline snapshot query")
		}
		return &fleetTestRows{columns: []string{"watermark", "snapshot_version", "baseline_version", "baseline_watermark"}, data: [][]driver.Value{{int64(10), int64(1), int64(1), int64(10)}}}, nil
	}})
	_, err := (FleetSource{DB: db}).ExpectedFleetPointCount(context.Background(), "recipe-1", "snapshot-1", "9")
	if err == nil || !strings.Contains(err.Error(), "precedes its pinned snapshot") {
		t.Fatalf("old watermark error = %v", err)
	}
}

func TestFleetEmbeddingRejectsRecipeDimensionMismatch(t *testing.T) {
	embedder := &recordingEmbedder{vector: []float32{1, 2}, dimensions: 3}
	source := FleetSource{Embedder: embedder}
	_, err := source.embedDocument(context.Background(), "recipe-v3", "note-1", FleetDocument{NoteID: "note-1", UserID: "user-1"})
	if err == nil || !strings.Contains(err.Error(), "expected 3 dimensions") {
		t.Fatalf("dimension mismatch error = %v", err)
	}
	if embedder.recipe != "recipe-v3" {
		t.Fatalf("recipe identity not resolved: %q", embedder.recipe)
	}
}

func TestUnownedNotesKeepExplicitUnknownTenantPayload(t *testing.T) {
	point, err := (FleetSource{Embedder: &recordingEmbedder{vector: []float32{1, 2}, dimensions: 2}}).embedDocument(
		context.Background(), "recipe-1", "note-a", FleetDocument{NoteID: "note-a", Title: "legacy", Content: "body"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if point.ID != fleetSourceID("", "note-a") || point.Payload["source_id"] != point.ID || point.Payload["user_id"] != nil || point.Payload["user_id_known"] != false {
		t.Fatalf("unowned note identity/payload = id=%q payload=%#v", point.ID, point.Payload)
	}
	db := openFakeFleetDB(t, fleetTestDriver{query: func(query string, _ []driver.NamedValue) (driver.Rows, error) {
		if !strings.Contains(query, "ORDER BY sequence") {
			return nil, errors.New("unexpected unowned delete query")
		}
		return &fleetTestRows{columns: []string{"sequence", "note_id", "user_id", "event_type", "payload", "move_group"}, data: [][]driver.Value{{int64(1), "note-a", nil, "delete", nil, nil}}}, nil
	}})
	changes, _, err := (FleetSource{DB: db, Embedder: &recordingEmbedder{}}).ReadFleetChanges(context.Background(), "recipe-1", "0", "0", 1)
	if err != nil || len(changes) != 1 || len(changes[0].Deletes) != 1 || changes[0].Deletes[0] != point.ID || changes[0].DeleteTenantKnown {
		t.Fatalf("unowned delete tenant/source identity = %#v err=%v", changes, err)
	}
}

func TestFleetCursorRejectsInvalidAndNegativeValues(t *testing.T) {
	for _, value := range []string{"bad", "-1", "1.1"} {
		if _, err := parseFleetCursor(value); err == nil {
			t.Errorf("parseFleetCursor(%q) succeeded", value)
		}
	}
	if cursor, err := parseFleetCursor("42"); err != nil || cursor != 42 {
		t.Fatalf("cursor=%d err=%v", cursor, err)
	}
	if order, err := (FleetSource{}).CompareFleetSequence(context.Background(), "9", "10"); err != nil || order != -1 {
		t.Fatalf("sequence order=%d err=%v", order, err)
	}
	if order, err := (FleetSource{}).CompareFleetSequence(context.Background(), "10", "10"); err != nil || order != 0 {
		t.Fatalf("equal sequence order=%d err=%v", order, err)
	}
}

func TestFleetCatalogRejectsCursorRegressionAndInvalidTransitions(t *testing.T) {
	old := CatalogIdentity{ArmID: "arm", OwnerID: "notes", Alias: "notes", RecipeID: "recipe", SnapshotID: "snapshot", SnapshotWatermark: "10", CatchupWatermark: "10", Dimensions: 2, Collection: "notes-arm", State: "candidate", Cursor: "note-b", Version: 2}
	next := old
	next.Version++
	next.Cursor = "note-a"
	if validFleetCatalogAdvance(old, next) {
		t.Fatal("catalog accepted a regressed source cursor")
	}
	next = old
	next.Version++
	next.State = "building"
	if validFleetCatalogAdvance(old, next) {
		t.Fatal("catalog accepted an invalid candidate-to-building transition")
	}
	next = old
	next.Version++
	next.Protected = true
	next.State = "protected"
	if !validFleetCatalogAdvance(old, next) {
		t.Fatal("catalog rejected a valid protection transition")
	}
	old.Protected = true
	next.Protected = false
	next.State = "retired"
	if validFleetCatalogAdvance(old, next) {
		t.Fatal("catalog allowed a protected generation to become unprotected")
	}
}

func TestFleetCatalogCASRejectsStaleAndIdentityChangedWrites(t *testing.T) {
	var seenQuery string
	db := openFakeFleetDB(t, fleetTestDriver{
		query: func(query string, _ []driver.NamedValue) (driver.Rows, error) {
			if strings.Contains(query, "SELECT current_sequence FROM notes_fleet_sequence") {
				return &fleetTestRows{columns: []string{"current_sequence"}, data: [][]driver.Value{{int64(21)}}}, nil
			}
			if strings.Contains(query, "FROM notes_fleet_snapshots AS snapshot") {
				return &fleetTestRows{columns: []string{"watermark", "snapshot_version", "baseline_version", "baseline_watermark"}, data: [][]driver.Value{{int64(10), int64(1), int64(1), int64(10)}}}, nil
			}
			if !strings.Contains(query, "FROM notes_fleet_generations WHERE arm_id = $1") {
				return nil, errors.New("unexpected catalog load")
			}
			return &fleetTestRows{columns: []string{"arm_id", "owner_id", "alias", "recipe_id", "snapshot_id", "snapshot_watermark", "catchup_watermark", "dimensions", "collection", "rollback_to", "state", "cursor", "version", "protected", "updated_at"}, data: [][]driver.Value{{"arm-1", "notes", "notes", "recipe-a", "snap", "10", "10", int64(2), "notes-arm-1", "", "candidate", "note-b", int64(1), false, time.Now()}}}, nil
		},
		exec: func(query string, args []driver.NamedValue) (driver.Result, error) {
			if !strings.Contains(query, "UPDATE notes_fleet_generations") {
				return nil, errors.New("unexpected query")
			}
			seenQuery = query
			// A zero-row update models a concurrent version change or a mismatched
			// immutable identity predicate; the caller must retry from a fresh read.
			return fleetTestResult(0), nil
		},
	})
	item := CatalogIdentity{ArmID: "arm-1", OwnerID: "notes", Alias: "notes", RecipeID: "recipe-a", SnapshotID: "snap", SnapshotWatermark: "10", CatchupWatermark: "10", Dimensions: 2, Collection: "notes-arm-1", State: "candidate", Cursor: "note-b", Version: 2}
	if err := (FleetCatalog{DB: db}).SaveCAS(context.Background(), item, 1); err == nil || !strings.Contains(err.Error(), "compare-and-swap conflict") {
		t.Fatalf("CAS conflict = %v", err)
	}
	for _, immutable := range []string{"version=$16", "owner_id=$2", "recipe_id=$4", "snapshot_id=$5", "snapshot_watermark=$6", "collection=$9", "dimensions=$8"} {
		if !strings.Contains(seenQuery, immutable) {
			t.Errorf("CAS query lacks immutable identity predicate %q", immutable)
		}
	}
	item.Version = 4
	if err := (FleetCatalog{DB: db}).SaveCAS(context.Background(), item, 1); err == nil {
		t.Fatal("non-contiguous cursor/version write succeeded")
	}
}

func TestFleetSchemaSerializesChangeSequenceWithTransaction(t *testing.T) {
	for _, invariant := range []string{
		"CREATE TABLE IF NOT EXISTS notes_fleet_sequence",
		"UPDATE notes_fleet_sequence SET current_sequence = current_sequence + 1",
		"PERFORM set_config('notes.fleet.sequence', next_sequence::TEXT, TRUE)",
		"sequence BIGINT PRIMARY KEY",
		"CREATE TRIGGER notes_fleet_change_trigger",
		"IS NOT DISTINCT FROM",
		"NEW.user_id::text",
		"CREATE TABLE IF NOT EXISTS notes_fleet_backfill",
		"LOCK TABLE notes IN SHARE ROW EXCLUSIVE MODE",
		"row_number() OVER (ORDER BY id)",
		"INSERT INTO notes_fleet_backfill",
		"OLD.user_id IS DISTINCT FROM NEW.user_id",
	} {
		if !strings.Contains(schemaSQL, invariant) {
			t.Errorf("fleet schema missing invariant %q", invariant)
		}
	}
	if got := strings.Count(schemaSQL, "PERFORM set_config('notes.fleet.sequence', next_sequence::TEXT, TRUE)"); got != 3 {
		t.Fatalf("trigger transaction-local sequence writes = %d, want delete, tenant-move tombstone, and upsert branches", got)
	}
	moveBranch := strings.Index(schemaSQL, "IF TG_OP = 'UPDATE' AND OLD.user_id IS DISTINCT FROM NEW.user_id THEN")
	if moveBranch < 0 {
		t.Fatal("trigger has no tenant reassignment branch")
	}
	oldTenantDelete := strings.Index(schemaSQL[moveBranch:], "VALUES (next_sequence, OLD.id, OLD.user_id, 'delete', NULL, move_group)")
	newTenantUpsert := strings.Index(schemaSQL[moveBranch:], "NEW.id::text")
	if oldTenantDelete < 0 || newTenantUpsert < 0 || oldTenantDelete > newTenantUpsert || !strings.Contains(schemaSQL[moveBranch:], "move_group := uuid_generate_v4()") {
		t.Fatal("tenant reassignment trigger does not append the old-scope tombstone before the new-scope upsert")
	}
}

func TestFleetBaselineBackfillSQLDeclaresIdempotentWriteFence(t *testing.T) {
	baselineStart := strings.Index(schemaSQL, "DO $$\nDECLARE\n    seeded BIGINT")
	if baselineStart < 0 {
		t.Fatal("Notes fleet backfill block is missing")
	}
	block := schemaSQL[baselineStart:]
	lock := strings.Index(block, "LOCK TABLE notes IN SHARE ROW EXCLUSIVE MODE")
	markerCheck := strings.Index(block, "IF NOT EXISTS (SELECT 1 FROM notes_fleet_backfill WHERE singleton = TRUE)")
	seed := strings.Index(block, "row_number() OVER (ORDER BY id)")
	advance := strings.Index(block, "SET current_sequence = sequence.current_sequence + row_count.total")
	markerWrite := strings.Index(block, "INSERT INTO notes_fleet_backfill(singleton, version, baseline_watermark, seeded_notes)")
	if lock < 0 || markerCheck < 0 || seed < 0 || advance < 0 || markerWrite < 0 || lock > markerCheck || markerCheck > seed || seed > advance || advance > markerWrite {
		t.Fatal("backfill must lock writers, skip a committed marker, seed in deterministic order, advance the ledger, then commit its marker")
	}
}

func TestConfiguredFleetSourceRefusesEmbeddingIdentityDriftWithinGeneration(t *testing.T) {
	tests := []struct {
		name  string
		drift EmbeddingRolePolicy
	}{
		{name: "role", drift: EmbeddingRolePolicy{RecipeID: "recipe-1", Role: "embedding.notes.changed", Dimensions: 2}},
		{name: "dimensions", drift: EmbeddingRolePolicy{RecipeID: "recipe-1", Role: "embedding.notes.v1", Dimensions: 3}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := openFakeFleetDB(t, fleetTestDriver{
				exec:  func(string, []driver.NamedValue) (driver.Result, error) { return fleetTestResult(0), nil },
				query: func(string, []driver.NamedValue) (driver.Rows, error) { return &fleetTestRows{}, nil },
			})
			config := FleetGenerationConfig{
				GenerationID: "arm-42",
				OwnerID:      "notes.api", Alias: "notes_live", Collection: "notes_candidate_42",
				RecipeID: "recipe-1", EmbeddingRole: "embedding.notes.v1", EmbeddingDimensions: 2,
			}
			policies := &sequencePolicy{policies: []EmbeddingRolePolicy{
				{RecipeID: config.RecipeID, Role: config.EmbeddingRole, Dimensions: config.EmbeddingDimensions},
				test.drift,
			}}
			gateway := &recordingGateway{vector: []float32{.1, .2}}
			configured, err := NewConfiguredFleetSource(context.Background(), db, "arm-42", fixedFleetGenerationConfig{config: config}, policies, gateway)
			if err != nil {
				t.Fatalf("NewConfiguredFleetSource() error = %v", err)
			}
			if _, _, err := configured.Source.Embedder.EmbedDocument(context.Background(), config.RecipeID, "note"); err == nil || !strings.Contains(err.Error(), "changed from the configured generation") {
				t.Fatalf("EmbedDocument() error = %v, want %s drift refusal", err, test.name)
			}
			if gateway.calls != 0 {
				t.Fatalf("gateway calls = %d, want zero after recipe %s drift", gateway.calls, test.name)
			}
		})
	}
}

func TestFleetSourceIDPreservesTenantInTombstoneIdentity(t *testing.T) {
	got := fleetSourceID(" user-a ", " note-1 ")
	if _, err := uuid.Parse(got); err != nil {
		t.Fatalf("tenant-scoped source ID is not a Qdrant-compatible UUID: %q (%v)", got, err)
	}
	if got == fleetSourceID("user-b", "note-1") {
		t.Fatal("different tenants shared a source identity")
	}
	if got != fleetSourceID("user-a", "note-1") {
		t.Fatal("tenant source ID is not deterministic")
	}
	if fleetSourceID("", "note-1") == "" || fleetSourceID("user-a", "") != "" {
		t.Fatal("unowned note identity is missing or empty note identity was accepted")
	}
}
