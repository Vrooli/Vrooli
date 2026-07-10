// Round-trip and dispatch-table coverage for the typed-operational
// event log. These tests are the executable proof that:
//
//   - Every event type Phase 1 introduces is registered in the dispatch
//     table at schema_version 1.
//   - BuildEvent → SQLiteRepository.SinceForRun preserves payload values
//     across JSON marshal/unmarshal.
//   - Unknown (event_type, schema_version) pairs are rejected, not
//     silently decoded as zero-valued payloads.
//
// Phase 3's TestAllEmittedEventsAreProcessed extends the dispatch-table
// invariant to cover the stats engine; the registration check here is
// the Phase 1 prerequisite.

package eventlog

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func newTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "events.db")
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
	CREATE TABLE IF NOT EXISTS run_events (
		id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		sequence INTEGER NOT NULL,
		event_type TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		schema_version INTEGER NOT NULL DEFAULT 1,
		data TEXT NOT NULL,
		UNIQUE(run_id, sequence)
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return NewSQLiteRepository(db)
}

// insertRaw is the test-only counterpart to event.SQLiteStore.Append:
// stores a typed event row using the same column shape, without
// depending on the orchestration package.
func insertRaw(t *testing.T, repo *SQLiteRepository, runID uuid.UUID, seq int64, evt *domain.RunEvent) {
	t.Helper()
	body := evt.Data.(*domain.TypedEventData).Body
	_, err := repo.db.Exec(
		`INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, schema_version, data)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		evt.ID, runID, seq, string(evt.EventType),
		evt.Timestamp.UTC().Format(time.RFC3339Nano),
		evt.SchemaVersion, []byte(body),
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
}

// TestAllPhase1EventTypesAreRegistered asserts that every typed event
// the eventlog package's payload structs name is registered in the
// dispatch table. Adding a new payload struct without registering it is
// a programmer error and CI catches it here.
func TestAllPhase1EventTypesAreRegistered(t *testing.T) {
	expected := []domain.RunEventType{
		domain.EventTypeRunnerFallbackAttempted,
		domain.EventTypeRunnerFallbackExhausted,
		domain.EventTypeModelFallbackAttempted,
		domain.EventTypeModelFallbackExhausted,
		domain.EventTypePolicyCandidateAttempt,
		domain.EventTypeModelHealthTransition,
		domain.EventTypeRunnerHealthTransition,
		domain.EventTypeSandboxOperation,
		domain.EventTypeHeartbeatMiss,
		domain.EventTypeCheckpointFailure,
		domain.EventTypeRetryAttempt,
	}
	for _, et := range expected {
		if v := LatestSchemaVersion(et); v == 0 {
			t.Errorf("event type %s has no registered schema_version", et)
		}
	}
}

// TestBuildEvent_PopulatesEventTypeAndSchemaVersion confirms the
// emitter-side construction pulls metadata from the registry.
func TestBuildEvent_PopulatesEventTypeAndSchemaVersion(t *testing.T) {
	runID := uuid.New()
	payload := RunnerFallbackAttemptedPayload{
		From:      "claude-code",
		To:        "codex",
		Reason:    "binary missing",
		AttemptNo: 1,
	}
	evt, err := BuildEvent(runID, payload)
	if err != nil {
		t.Fatalf("BuildEvent: %v", err)
	}
	if evt.EventType != domain.EventTypeRunnerFallbackAttempted {
		t.Errorf("EventType = %s, want %s", evt.EventType, domain.EventTypeRunnerFallbackAttempted)
	}
	if evt.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d, want 1", evt.SchemaVersion)
	}
	if evt.RunID != runID {
		t.Errorf("RunID = %s, want %s", evt.RunID, runID)
	}
	if evt.Data == nil {
		t.Fatal("Data is nil")
	}
	body := evt.Data.(*domain.TypedEventData).Body
	if len(body) == 0 {
		t.Fatal("payload body is empty")
	}
}

// TestRoundTrip_PerEventType writes one fixture event per registered type
// and reads it back through SinceForRun, asserting the decoded Go value
// equals the original.
func TestRoundTrip_PerEventType(t *testing.T) {
	repo := newTestRepo(t)
	runID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	cases := []struct {
		name    string
		payload Payload
	}{
		{"runner.fallback.attempted", RunnerFallbackAttemptedPayload{From: "claude-code", To: "codex", Reason: "binary missing", AttemptNo: 1}},
		{"runner.fallback.exhausted", RunnerFallbackExhaustedPayload{Primary: "claude-code", CandidatesTried: []string{"codex", "opencode"}, LastReason: "all unavailable"}},
		{"model.fallback.attempted", ModelFallbackAttemptedPayload{From: "sonnet-4", To: "haiku", Reason: "rate limited", AttemptNo: 2, ChainPosition: 2, ChainLength: 3}},
		{"model.fallback.exhausted", ModelFallbackExhaustedPayload{Preset: "CHEAP", Chain: []string{"a", "b", "c"}, LastReason: "all unavailable"}},
		{"policy.candidate.attempt", PolicyCandidateAttemptPayload{CatalogDigest: "sha256:test", SnapshotIndex: 1, Runner: "codex", SelectionType: "runner_default", Outcome: "selected"}},
		{"model.health.transition", ModelHealthTransitionPayload{Runner: "claude-code", Model: "sonnet-4", FromStatus: "ok", ToStatus: "failed", Reason: "rate_limit", Message: "429"}},
		{"runner.health.transition", RunnerHealthTransitionPayload{Runner: "codex", FromStatus: "unknown", ToStatus: "ok", Reason: "probe_pass"}},
		{"sandbox.operation", SandboxOperationPayload{Operation: SandboxOpDelete, Success: true, DurationMS: 42, Reason: "finalize"}},
		{"heartbeat.miss", HeartbeatMissPayload{Target: HeartbeatTargetRun, AttemptNo: 1, Message: "db locked"}},
		{"checkpoint.failure", CheckpointFailurePayload{Phase: "running", Step: CheckpointFailureSavePhase, Message: "io error"}},
		{"retry.attempt", RetryAttemptPayload{Operation: "execute", AttemptNo: 1, MaxAttempts: 3, Reason: "network_transient"}},
	}

	for i, tc := range cases {
		evt, err := BuildEvent(runID, tc.payload)
		if err != nil {
			t.Fatalf("%s: BuildEvent: %v", tc.name, err)
		}
		evt.Timestamp = now.Add(time.Duration(i) * time.Millisecond)
		insertRaw(t, repo, runID, int64(i), evt)
	}

	records, err := repo.SinceForRun(context.Background(), runID, -1, 0)
	if err != nil {
		t.Fatalf("SinceForRun: %v", err)
	}
	if len(records) != len(cases) {
		t.Fatalf("got %d records, want %d", len(records), len(cases))
	}

	for i, tc := range cases {
		got := records[i]
		if got.SchemaVersion != 1 {
			t.Errorf("%s: SchemaVersion = %d, want 1", tc.name, got.SchemaVersion)
		}
		// Compare via reflection so each case can use its concrete struct
		// value (passed by value into BuildEvent) and the decoded *Payload
		// pointer. payloadValue strips the pointer for direct equality.
		decoded := payloadValue(got.Payload)
		if !reflect.DeepEqual(decoded, tc.payload) {
			t.Errorf("%s: decoded payload mismatch\n got: %#v\nwant: %#v", tc.name, decoded, tc.payload)
		}
	}
}

// TestDecode_RejectsUnregisteredVersion confirms that asking for a
// schema_version that hasn't been registered fails loudly. This is the
// guard rail behind the schema-version-is-forever contract: a typo in a
// version number becomes a hard error, not a silent zero-value decode.
func TestDecode_RejectsUnregisteredVersion(t *testing.T) {
	_, err := Decode(domain.EventTypeRunnerFallbackAttempted, 99, nil)
	if err == nil {
		t.Fatal("expected error for unregistered schema_version")
	}
}

// TestDecode_RejectsUnknownEventType confirms unknown event types are
// rejected even at schema_version 1.
func TestDecode_RejectsUnknownEventType(t *testing.T) {
	_, err := Decode(domain.RunEventType("totally.fake.event"), 1, nil)
	if err == nil {
		t.Fatal("expected error for unregistered event type")
	}
}

// TestEmitter_NilSinkIsNoOp makes sure constructing an Emitter with a
// nil sink doesn't panic — the contract for tests that pass a partial
// Deps in.
func TestEmitter_NilSinkIsNoOp(t *testing.T) {
	em := NewEmitter(nil, uuid.New())
	em.EmitRunnerFallbackAttempted(RunnerFallbackAttemptedPayload{})
	// no panic = pass
}

// payloadValue dereferences a Payload pointer to the underlying struct
// for reflect.DeepEqual comparisons.
func payloadValue(p Payload) any {
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Ptr {
		return v.Elem().Interface()
	}
	return p
}
