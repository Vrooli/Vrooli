package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"prompt-manager/internal/testsqlite"
)

func newHeartbeatAttemptPublisherDB(t *testing.T) *database.RoutedDB {
	t.Helper()
	db := testsqlite.Open(t)
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(HeartbeatAttemptPublicationSchema)); err != nil {
		t.Fatalf("initialize heartbeat publication schema: %v", err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(ctx, pool, database.SchemaProviderFunc(HeartbeatAttemptPublicationSchema))
	})
	return db
}

func TestHeartbeatAttemptPublisherFreezesFactAndRejectsChangedTerms(t *testing.T) {
	db := newHeartbeatAttemptPublisherDB(t)
	publisher := NewHeartbeatAttemptPublisher(db, nil)
	entry := HeartbeatAttempt{ID: "logical-attempt", TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusFailed, Phase: "run_terminal", EndedAt: "2026-10-17T01:00:00Z", Error: "run failed"}
	if err := publisher.Capture(t.Context(), &entry); err != nil {
		t.Fatalf("capture initial fact: %v", err)
	}
	var first []byte
	if err := db.QueryRowContext(t.Context(), `SELECT envelope FROM heartbeat_attempt_publications WHERE event_id=?`, heartbeatAttemptFactEventID(entry.ID, entry.Phase)).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Capture(t.Context(), &entry); err != nil {
		t.Fatalf("recapture identical fact: %v", err)
	}
	var second []byte
	if err := db.QueryRowContext(t.Context(), `SELECT envelope FROM heartbeat_attempt_publications WHERE event_id=?`, heartbeatAttemptFactEventID(entry.ID, entry.Phase)).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("identical recapture changed the frozen envelope")
	}
	entry.Error = "changed immutable terms"
	if err := publisher.Capture(t.Context(), &entry); err == nil {
		t.Fatal("changed terms under an existing fact identity must fail closed")
	}
}

func TestHeartbeatAttemptPublisherReplaysFrozenEnvelopeAfterReconstruction(t *testing.T) {
	dsn, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "heartbeat.db"), storage.SQLiteTuning{})
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(HeartbeatAttemptPublicationSchema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var mu sync.Mutex
	var posts [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		mu.Lock()
		posts = append(posts, append([]byte(nil), body...))
		count := len(posts)
		mu.Unlock()
		if count == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	resolve := func(context.Context) (string, error) { return server.URL, nil }
	firstPublisher := NewHeartbeatAttemptPublisher(db, resolve)
	entry := HeartbeatAttempt{ID: "logical-retry", TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusFailed, Phase: "pre_run_failure", StartedAt: "2026-10-17T01:00:00Z", Error: "owner unavailable"}
	if err := firstPublisher.Capture(t.Context(), &entry); err != nil {
		t.Fatalf("capture pending fact: %v", err)
	}
	firstPublisher.drain(t.Context(), 1)
	if _, err := db.ExecContext(t.Context(), `UPDATE heartbeat_attempt_publications SET next_attempt_at=? WHERE event_id=?`, heartbeatPublicationTimestamp(time.Now().Add(-time.Second)), heartbeatAttemptFactEventID(entry.ID, entry.Phase)); err != nil {
		t.Fatal(err)
	}
	// Reopen the same SQLite owner file to exercise retained pending state across
	// a Prompt Manager process lifetime boundary.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(HeartbeatAttemptPublicationSchema)); err != nil {
		t.Fatal(err)
	}
	restartedPublisher := NewHeartbeatAttemptPublisher(db, resolve)
	restartedPublisher.drain(t.Context(), 1)
	mu.Lock()
	defer mu.Unlock()
	if len(posts) != 2 || !bytes.Equal(posts[0], posts[1]) {
		t.Fatalf("retry did not resend the frozen envelope exactly: %d posts", len(posts))
	}
	var status string
	if err := db.QueryRowContext(t.Context(), `SELECT status FROM heartbeat_attempt_publications WHERE event_id=?`, heartbeatAttemptFactEventID(entry.ID, entry.Phase)).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "accepted" {
		t.Fatalf("status=%q, want accepted", status)
	}
}

func TestHeartbeatAttemptPublisherMissingEndpointRetainsPendingError(t *testing.T) {
	db := newHeartbeatAttemptPublisherDB(t)
	publisher := NewHeartbeatAttemptPublisher(db, func(context.Context) (string, error) {
		return "", errors.New("events discovery unavailable")
	})
	entry := HeartbeatAttempt{ID: "logical-no-endpoint", TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusFailed, Phase: "run_terminal", StartedAt: "2026-10-17T01:00:00Z", Error: "run failed"}
	if err := publisher.Capture(t.Context(), &entry); err != nil {
		t.Fatal(err)
	}
	publisher.drain(t.Context(), 1)
	var status, lastError string
	var attempts int
	if err := db.QueryRowContext(t.Context(), `SELECT status, attempts, last_error FROM heartbeat_attempt_publications WHERE event_id=?`, heartbeatAttemptFactEventID(entry.ID, entry.Phase)).Scan(&status, &attempts, &lastError); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || attempts != 1 || lastError == "" {
		t.Fatalf("missing endpoint state = %q, %d, %q; want retained pending error", status, attempts, lastError)
	}
}

func TestHeartbeatAttemptPublisherUsesRoutedTestPool(t *testing.T) {
	primaryDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "primary.db"), storage.SQLiteTuning{})
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: primaryDSN, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(HeartbeatAttemptPublicationSchema)); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(ctx, pool, database.SchemaProviderFunc(HeartbeatAttemptPublicationSchema))
	})
	testDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "leased.db"), storage.SQLiteTuning{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InstallTestPool(context.Background(), testDSN, "heartbeat-attempt-test", time.Minute); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.ClearTestPool("heartbeat-attempt-test") })
	entry := HeartbeatAttempt{ID: "test-only-attempt", TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusFailed, Phase: "pre_run_failure", StartedAt: "2026-10-17T01:00:00Z"}
	if err := NewHeartbeatAttemptPublisher(db, nil).Capture(database.WithTestMode(context.Background()), &entry); err != nil {
		t.Fatal(err)
	}
	var primaryCount, testCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM heartbeat_attempt_publications`).Scan(&primaryCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(database.WithTestMode(context.Background()), `SELECT COUNT(*) FROM heartbeat_attempt_publications`).Scan(&testCount); err != nil {
		t.Fatal(err)
	}
	if primaryCount != 0 || testCount != 1 {
		t.Fatalf("publication rows escaped routed pool: primary=%d test=%d", primaryCount, testCount)
	}
}

func TestHeartbeatAttemptPublisher409NeedsExactEventsSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mutate     bool
		wantStatus string
	}{
		{name: "matching snapshot is receipt", wantStatus: "accepted"},
		{name: "different snapshot remains pending", mutate: true, wantStatus: "pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newHeartbeatAttemptPublisherDB(t)
			var existing []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodPost:
					body, _ := io.ReadAll(r.Body)
					_ = r.Body.Close()
					existing = body
					w.WriteHeader(http.StatusConflict)
				case http.MethodGet:
					var env domain.EventEnvelope
					if err := protojson.Unmarshal(existing, &env); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if tc.mutate {
						env.EventType = "different.event.v1"
					}
					body, err := protojson.Marshal(&env)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte("["))
					_, _ = w.Write(body)
					_, _ = w.Write([]byte("]"))
				default:
					http.Error(w, "method", http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()
			publisher := NewHeartbeatAttemptPublisher(db, func(context.Context) (string, error) { return server.URL, nil })
			entry := HeartbeatAttempt{ID: "logical-conflict", TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusFailed, Phase: "run_terminal", StartedAt: "2026-10-17T01:00:00Z", Error: "failed"}
			if err := publisher.Capture(t.Context(), &entry); err != nil {
				t.Fatal(err)
			}
			publisher.drain(t.Context(), 1)
			var status string
			if err := db.QueryRowContext(t.Context(), `SELECT status FROM heartbeat_attempt_publications WHERE event_id=?`, heartbeatAttemptFactEventID(entry.ID, entry.Phase)).Scan(&status); err != nil {
				t.Fatal(err)
			}
			if status != tc.wantStatus {
				t.Fatalf("status=%q, want %q", status, tc.wantStatus)
			}
		})
	}
}

func TestHeartbeatAttemptEnvelopeIsAnEventsEnvelope(t *testing.T) {
	entry := &HeartbeatAttempt{ID: "attempt", TeamID: "team", AgentID: "member", Status: HeartbeatStatusRunning, Phase: "run_created", StartedAt: "2026-10-17T01:00:00Z"}
	body, eventID, err := heartbeatAttemptEnvelope(entry)
	if err != nil {
		t.Fatal(err)
	}
	var env domain.EventEnvelope
	if err := protojson.Unmarshal(body, &env); err != nil {
		t.Fatal(err)
	}
	if eventID != heartbeatAttemptFactEventID(entry.ID, entry.Phase) || env.GetEventId() != eventID || env.GetEventType() != heartbeatAttemptEventType {
		t.Fatalf("unexpected envelope identity: event=%q type=%q", env.GetEventId(), env.GetEventType())
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
}

func TestRetainedOwnerStateEnvelopeUsesSeparateEventsTypeAndObservationTime(t *testing.T) {
	entry := &HeartbeatAttempt{
		ID: "owner-obligation", TeamID: "team-a", AgentID: "member-a",
		OwnerObligationID: "owner-obligation", RetainedOwnerState: HeartbeatOwnerStateDispatchUncertain,
		Status: HeartbeatOwnerStateDispatchUncertain, Phase: "retained_owner_state:" + HeartbeatOwnerStateDispatchUncertain,
		ObservedAt: "2026-10-17T01:02:03.123456789Z", StartedAt: "",
	}
	body, eventID, err := heartbeatAttemptEnvelope(entry)
	if err != nil {
		t.Fatal(err)
	}
	var env domain.EventEnvelope
	if err := protojson.Unmarshal(body, &env); err != nil {
		t.Fatal(err)
	}
	if env.GetEventType() != heartbeatOwnerStateEventType {
		t.Fatalf("event type=%q, want %q", env.GetEventType(), heartbeatOwnerStateEventType)
	}
	if env.GetTarget().GetOperation() != "heartbeat-owner-state" {
		t.Fatalf("operation=%q, want heartbeat-owner-state", env.GetTarget().GetOperation())
	}
	if eventID != heartbeatAttemptFactEventID(entry.OwnerObligationID, entry.Phase) || env.GetEventId() != eventID {
		t.Fatalf("event identity=%q, want stable owner-state identity", env.GetEventId())
	}
	if got := env.GetOccurredAt().AsTime().UTC().Format(time.RFC3339Nano); got != entry.ObservedAt {
		t.Fatalf("occurredAt=%q, want persisted observation %q", got, entry.ObservedAt)
	}
}

func TestRetainedPreparationInterruptedOwnerStateIsSupported(t *testing.T) {
	entry := &HeartbeatAttempt{
		ID: "owner-obligation", TeamID: "team-a", AgentID: "member-a",
		OwnerObligationID: "owner-obligation", RetainedOwnerState: HeartbeatOwnerStatePreparationInterrupted,
		Status:     HeartbeatOwnerStatePreparationInterrupted,
		Phase:      "retained_owner_state:" + HeartbeatOwnerStatePreparationInterrupted,
		ObservedAt: "2026-10-17T01:02:03.123456789Z",
	}
	body, _, err := heartbeatAttemptEnvelope(entry)
	if err != nil {
		t.Fatalf("preparation-interrupted owner fact was rejected: %v", err)
	}
	var env domain.EventEnvelope
	if err := protojson.Unmarshal(body, &env); err != nil {
		t.Fatal(err)
	}
	if env.GetEventType() != heartbeatOwnerStateEventType || env.GetCorrelation() != nil {
		t.Fatalf("preparation interruption must be an owner-state fact without task/run correlation: %#v", env)
	}
	entry.RunID = "run-that-would-contradict-preparation"
	if _, _, err := heartbeatAttemptEnvelope(entry); err == nil {
		t.Fatal("preparation interruption accepted a run identity despite proving no owner POST")
	}
	entry.RunID = ""
	entry.TaskID = "task-that-would-contradict-preparation"
	if _, _, err := heartbeatAttemptEnvelope(entry); err == nil {
		t.Fatal("preparation interruption accepted a task identity despite proving no owner POST")
	}
}

func TestHeartbeatAttemptPublisherConcurrentIdenticalCaptureKeepsOneFrozenFact(t *testing.T) {
	db := newHeartbeatAttemptPublisherDB(t)
	publisher := NewHeartbeatAttemptPublisher(db, nil)
	entry := HeartbeatAttempt{ID: "logical-concurrent", TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusFailed, Phase: "run_terminal", StartedAt: "2026-10-17T01:00:00Z", Error: "same immutable terms"}
	const writers = 8
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			copy := entry
			errs <- publisher.Capture(t.Context(), &copy)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("identical concurrent capture failed: %v", err)
		}
	}
	var count int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM heartbeat_attempt_publications WHERE event_id=?`, heartbeatAttemptFactEventID(entry.ID, entry.Phase)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("captured rows=%d, want exactly one", count)
	}
}

func TestHeartbeatAttemptPublisherExpiredClaimCannotAcknowledgeReclaimedRow(t *testing.T) {
	db := newHeartbeatAttemptPublisherDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	publisher := NewHeartbeatAttemptPublisher(db, func(context.Context) (string, error) { return server.URL, nil })
	entry := HeartbeatAttempt{ID: "logical-reclaim", TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusFailed, Phase: "run_terminal", StartedAt: "2026-10-17T01:00:00Z"}
	if err := publisher.Capture(t.Context(), &entry); err != nil {
		t.Fatal(err)
	}
	oldClaim, err := publisher.claim(t.Context())
	if err != nil || oldClaim == nil {
		t.Fatalf("first claim=%+v err=%v", oldClaim, err)
	}
	if _, err := db.ExecContext(t.Context(), `UPDATE heartbeat_attempt_publications SET lease_until=? WHERE event_id=?`, heartbeatPublicationTimestamp(time.Now().Add(-time.Second)), oldClaim.eventID); err != nil {
		t.Fatal(err)
	}
	newClaim, err := publisher.claim(t.Context())
	if err != nil || newClaim == nil || newClaim.leaseToken == oldClaim.leaseToken {
		t.Fatalf("reclaimed claim=%+v err=%v", newClaim, err)
	}
	publisher.publish(t.Context(), oldClaim)
	var status, lease string
	if err := db.QueryRowContext(t.Context(), `SELECT status, lease_until FROM heartbeat_attempt_publications WHERE event_id=?`, oldClaim.eventID).Scan(&status, &lease); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || lease != newClaim.leaseToken {
		t.Fatalf("expired worker changed current claim: status=%q lease=%q want pending/%q", status, lease, newClaim.leaseToken)
	}
	publisher.publish(t.Context(), newClaim)
	if err := db.QueryRowContext(t.Context(), `SELECT status FROM heartbeat_attempt_publications WHERE event_id=?`, oldClaim.eventID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "accepted" {
		t.Fatalf("current lease completion status=%q, want accepted", status)
	}
}

func TestHeartbeatAttemptPublisherDoesNotSendCorruptFrozenEnvelope(t *testing.T) {
	db := newHeartbeatAttemptPublisherDB(t)
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	publisher := NewHeartbeatAttemptPublisher(db, func(context.Context) (string, error) { return server.URL, nil })
	entry := HeartbeatAttempt{ID: "logical-corrupt", TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusFailed, Phase: "run_terminal", StartedAt: "2026-10-17T01:00:00Z"}
	if err := publisher.Capture(t.Context(), &entry); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(t.Context(), `UPDATE heartbeat_attempt_publications SET envelope=? WHERE event_id=?`, []byte(`{}`), heartbeatAttemptFactEventID(entry.ID, entry.Phase)); err != nil {
		t.Fatal(err)
	}
	publisher.drain(t.Context(), 1)
	var status, lastError string
	if err := db.QueryRowContext(t.Context(), `SELECT status, last_error FROM heartbeat_attempt_publications WHERE event_id=?`, heartbeatAttemptFactEventID(entry.ID, entry.Phase)).Scan(&status, &lastError); err != nil {
		t.Fatal(err)
	}
	if requests != 0 || status != "pending" || !strings.Contains(lastError, "digest mismatch") {
		t.Fatalf("corrupt row result: requests=%d status=%q error=%q", requests, status, lastError)
	}
}

func TestHeartbeatAttemptPublisherPruneThenRecaptureVerifiesSameEventsFact(t *testing.T) {
	db := newHeartbeatAttemptPublisherDB(t)
	var mu sync.Mutex
	var stored []byte
	var postBodies [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			mu.Lock()
			postBodies = append(postBodies, append([]byte(nil), body...))
			first := stored == nil
			if first {
				stored = append([]byte(nil), body...)
			}
			mu.Unlock()
			if first {
				w.WriteHeader(http.StatusAccepted)
			} else {
				w.WriteHeader(http.StatusConflict)
			}
		case http.MethodGet:
			mu.Lock()
			body := append([]byte(nil), stored...)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("["))
			_, _ = w.Write(body)
			_, _ = w.Write([]byte("]"))
		default:
			http.Error(w, "method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	publisher := NewHeartbeatAttemptPublisher(db, func(context.Context) (string, error) { return server.URL, nil })
	entry := HeartbeatAttempt{ID: "logical-prune-replay", TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusFailed, Phase: "run_terminal", StartedAt: "2026-10-17T01:00:00Z", EndedAt: "2026-10-17T01:01:00Z", Error: "terminal error"}
	if err := publisher.Capture(t.Context(), &entry); err != nil {
		t.Fatal(err)
	}
	publisher.drain(t.Context(), 1)
	if _, err := db.ExecContext(t.Context(), `UPDATE heartbeat_attempt_publications SET accepted_at=? WHERE event_id=?`, heartbeatPublicationTimestamp(time.Now().Add(-31*24*time.Hour)), heartbeatAttemptFactEventID(entry.ID, entry.Phase)); err != nil {
		t.Fatal(err)
	}
	publisher.mu.Lock()
	publisher.lastPrune = time.Time{}
	publisher.mu.Unlock()
	publisher.pruneAcceptedReceipts(t.Context())
	var count int
	if err := db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM heartbeat_attempt_publications WHERE event_id=?`, heartbeatAttemptFactEventID(entry.ID, entry.Phase)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expired receipt rows=%d, want pruned", count)
	}
	if err := publisher.Capture(t.Context(), &entry); err != nil {
		t.Fatalf("recapture after receipt pruning: %v", err)
	}
	publisher.drain(t.Context(), 1)
	mu.Lock()
	defer mu.Unlock()
	if len(postBodies) != 2 || !bytes.Equal(postBodies[0], postBodies[1]) {
		t.Fatalf("pruned-fact replay changed envelope: got %d posts", len(postBodies))
	}
	if !bytes.Equal(postBodies[0], stored) {
		t.Fatal("reconstructed source fact differs from the original Events fact")
	}
	var status string
	if err := db.QueryRowContext(t.Context(), `SELECT status FROM heartbeat_attempt_publications WHERE event_id=?`, heartbeatAttemptFactEventID(entry.ID, entry.Phase)).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "accepted" {
		t.Fatalf("verified duplicate status=%q, want accepted", status)
	}
}
