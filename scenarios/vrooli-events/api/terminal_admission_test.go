package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/testutil"
	"modernc.org/sqlite"
)

func terminalAdmissionFixture(t *testing.T, path string) *Server {
	t.Helper()
	events, err := store.NewSQLiteStore(context.Background(), store.SQLiteConfig{DBPath: path, MaxAge: time.Hour, Now: func() time.Time { return time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = events.Close() })
	subs, err := subscription.NewSQLiteStore(events.DB())
	if err != nil {
		t.Fatal(err)
	}
	// No HTTP listener, broker, dispatcher, service, signer, or network transport.
	return &Server{store: events, db: events.DB(), subStore: subs}
}
func terminalAdmissionIntent(t *testing.T, mutate func(*eventbus.DomainEvent)) (string, string) {
	t.Helper()
	event := eventbus.DomainEvent{Source: "agent-manager", EventType: terminalIntentEventType, Occurred: time.Date(2026, 11, 7, 10, 0, 0, 123456789, time.UTC), Payload: map[string]any{
		"source_instance": "fixture-source", "source_epoch": "epoch-1", "run_id": "11111111-1111-4111-8111-111111111111", "lifecycle_version": "1", "terminal_observation_sha256": strings.Repeat("b", 64), "status": "complete",
	}}
	if mutate != nil {
		mutate(&event)
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(encoded)
	return string(encoded), hex.EncodeToString(sum[:])
}
func terminalAdmissionSubscription(t *testing.T, s *Server, name string) int64 {
	t.Helper()
	id, err := s.subStore.Create(context.Background(), subscription.Subscription{Name: name, OwnerScenario: "fixture", EventPattern: terminalIntentEventType, SourceFilter: "agent-manager", DeliveryType: subscription.DeliveryWebhook, DeliveryTarget: "https://fixture.invalid/never-called", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func terminalAdmissionCounts(t *testing.T, s *Server, events, deliveries, receipts int) {
	t.Helper()
	for _, item := range []struct {
		table string
		want  int
	}{{"events", events}, {"subscription_deliveries", deliveries}, {"terminal_event_admissions", receipts}} {
		var got int
		if err := s.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+item.table).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != item.want {
			t.Fatalf("%s count=%d want %d", item.table, got, item.want)
		}
	}
}
func TestTerminalEventAdmissionCommitReplayRestartAndPruning(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "events.sqlite")
	s := terminalAdmissionFixture(t, path)
	terminalAdmissionSubscription(t, s, "first")
	intent, id := terminalAdmissionIntent(t, nil)
	ack, err := s.acceptTerminalEventIntent(ctx, intent, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := eventbus.ValidateDomainEventAcknowledgment(intent, id, ack); err != nil {
		t.Fatal(err)
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
	_, canonical, err := eventbus.RetainedDomainEventEnvelope(intent, id)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.store.Query(ctx, store.QueryFilters{EventID: id})
	if err != nil || len(rows) != 1 || !bytes.Equal(rows[0].Payload, canonical) {
		t.Fatalf("canonical bytes not preserved: %v %v", rows, err)
	}
	due, err := s.subStore.(subscription.QueueStore).Due(ctx, 10)
	if err != nil || len(due) != 1 {
		t.Fatalf("queue=%+v err=%v", due, err)
	}
	payload := due[0].PayloadJSON
	// Discard the first response and reopen the actual on-disk owner database.
	if err := s.store.Close(); err != nil {
		t.Fatal(err)
	}
	s = terminalAdmissionFixture(t, path)
	replay, err := s.acceptTerminalEventIntent(ctx, intent, id)
	if err != nil || replay != ack {
		t.Fatalf("lost-ack replay=%+v want=%+v err=%v", replay, ack, err)
	}
	due, err = s.subStore.(subscription.QueueStore).Due(ctx, 10)
	if err != nil || len(due) != 1 || due[0].PayloadJSON != payload {
		t.Fatalf("replay changed fanout: %+v %v", due, err)
	}
	result, err := s.store.Prune(ctx)
	if err != nil || result.TimeDeletedCount != 1 {
		t.Fatalf("prune=%+v err=%v", result, err)
	}
	// Independently retired deliveries and newly registered subscriptions must
	// not allow replay to resurrect or reproject a pruned event.
	if _, err := s.db.ExecContext(ctx, "DELETE FROM subscription_deliveries"); err != nil {
		t.Fatal(err)
	}
	terminalAdmissionSubscription(t, s, "later")
	if err := s.store.Close(); err != nil {
		t.Fatal(err)
	}
	s = terminalAdmissionFixture(t, path)
	replay, err = s.acceptTerminalEventIntent(ctx, intent, id)
	if err != nil || replay != ack {
		t.Fatalf("post-prune replay=%+v err=%v", replay, err)
	}
	terminalAdmissionCounts(t, s, 0, 0, 1)
	stats, err := s.store.Stats(ctx)
	if err != nil || stats.TotalPayloadBytes != 0 {
		t.Fatalf("prune accounting=%+v err=%v", stats, err)
	}
}
func TestTerminalEventAdmissionRollsBackEveryWrite(t *testing.T) {
	for _, point := range []string{"second_delivery", "receipt", "commit"} {
		t.Run(point, func(t *testing.T) {
			s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
			terminalAdmissionSubscription(t, s, "first")
			second := terminalAdmissionSubscription(t, s, "second")
			table, condition := "terminal_event_admissions", "1=1"
			if point == "second_delivery" {
				table = "subscription_deliveries"
				condition = fmt.Sprintf("NEW.subscription_id=%d", second)
			}
			trigger := fmt.Sprintf("CREATE TRIGGER fail_admission BEFORE INSERT ON %s WHEN %s BEGIN SELECT RAISE(ABORT,'fixture failure'); END", table, condition)
			if point == "commit" {
				trigger = `PRAGMA foreign_keys=ON; CREATE TABLE terminal_test_parent(id TEXT PRIMARY KEY); CREATE TABLE terminal_test_child(id TEXT REFERENCES terminal_test_parent(id) DEFERRABLE INITIALLY DEFERRED); CREATE TRIGGER fail_admission AFTER INSERT ON terminal_event_admissions BEGIN INSERT INTO terminal_test_child VALUES ('missing'); END;`
			}

			if _, err := s.db.ExecContext(context.Background(), trigger); err != nil {
				t.Fatal(err)
			}
			intent, id := terminalAdmissionIntent(t, nil)
			ack, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
			if err == nil || ack != (eventbus.DomainEventAcknowledgment{}) {
				t.Fatalf("failure returned success ack: %+v %v", ack, err)
			}
			terminalAdmissionCounts(t, s, 0, 0, 0)
			stats, err := s.store.Stats(context.Background())
			if err != nil || stats.TotalPayloadBytes != 0 {
				t.Fatalf("rollback accounting=%+v err=%v", stats, err)
			}
			if _, err := s.db.ExecContext(context.Background(), "DROP TRIGGER fail_admission"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.acceptTerminalEventIntent(context.Background(), intent, id); err != nil {
				t.Fatal(err)
			}
			terminalAdmissionCounts(t, s, 1, 2, 1)
		})
	}
}
func TestTerminalEventAdmissionRejectsConflictingLedgerAndCanonicalBytes(t *testing.T) {
	for _, mutation := range []string{
		"UPDATE terminal_event_admissions SET intent_json='different intent'",
		"UPDATE terminal_event_admissions SET canonical_payload=X'00'",
		"UPDATE terminal_event_admissions SET canonical_sha256='different digest'",
		"UPDATE terminal_event_admissions SET store_id=999",
		"UPDATE events SET payload=X'00'",
		"UPDATE events SET source_scenario='different-source'",
		"UPDATE events SET event_type='different.type'",
		"UPDATE events SET created_at='2000-01-01T00:00:00.000'",
		"UPDATE events SET target_scenario='wrong-target'",
		"UPDATE events SET correlation_id='wrong-correlation'",
	} {
		t.Run(mutation, func(t *testing.T) {
			s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
			terminalAdmissionSubscription(t, s, "first")
			intent, id := terminalAdmissionIntent(t, nil)
			if _, err := s.acceptTerminalEventIntent(context.Background(), intent, id); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(mutation, "UPDATE terminal_event_admissions") {
				if _, err := s.db.ExecContext(context.Background(), "DROP TRIGGER terminal_event_admissions_no_update"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.db.ExecContext(context.Background(), mutation); err != nil {
				t.Fatal(err)
			}
			ack, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
			if !errors.Is(err, store.ErrTerminalAdmissionConflict) || ack != (eventbus.DomainEventAcknowledgment{}) {
				t.Fatalf("conflict ack=%+v err=%v", ack, err)
			}
			terminalAdmissionCounts(t, s, 1, 1, 1)
		})
	}
}
func TestTerminalEventAdmissionGenericDuplicateIsNotDurableAcknowledgment(t *testing.T) {
	s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
	terminalAdmissionSubscription(t, s, "first")
	intent, id := terminalAdmissionIntent(t, nil)
	env, canonical, err := eventbus.RetainedDomainEventEnvelope(intent, id)
	if err != nil {
		t.Fatal(err)
	}
	event, err := convert.EnvelopeToEvent(env)
	if err != nil {
		t.Fatal(err)
	}
	event.Payload = canonical
	if _, err := s.admitEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	ack, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
	if !errors.Is(err, store.ErrDuplicateEvent) || ack != (eventbus.DomainEventAcknowledgment{}) {
		t.Fatalf("generic duplicate ack=%+v err=%v", ack, err)
	}
	terminalAdmissionCounts(t, s, 1, 1, 0)
}
func TestTerminalEventAdmissionRejectsInvalidSourceTypeStatusAndIdentity(t *testing.T) {
	for _, kind := range []string{"source", "type", "status", "identity", "intent"} {
		t.Run(kind, func(t *testing.T) {
			s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
			intent, id := terminalAdmissionIntent(t, func(e *eventbus.DomainEvent) {
				switch kind {
				case "source":
					e.Source = "other"
				case "type":
					e.EventType = "other.v1"
				case "status":
					e.Payload["status"] = "running"
				}
			})
			if kind == "identity" {
				id = strings.Repeat("a", 64)
			}
			if kind == "intent" {
				intent += " "
			}
			ack, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
			if err == nil || ack != (eventbus.DomainEventAcknowledgment{}) {
				t.Fatalf("invalid intent ack=%+v err=%v", ack, err)
			}
			terminalAdmissionCounts(t, s, 0, 0, 0)
		})
	}
}
func TestTerminalEventAdmissionFailsClosedWithoutDurableCapabilities(t *testing.T) {
	intent, id := terminalAdmissionIntent(t, nil)
	mock := &testutil.MockStore{}
	for _, server := range []*Server{nil, {store: mock}} {
		if ack, err := server.acceptTerminalEventIntent(context.Background(), intent, id); err == nil || ack != (eventbus.DomainEventAcknowledgment{}) {
			t.Fatalf("missing DB ack=%+v err=%v", ack, err)
		}
	}
	if len(mock.InsertCalls) != 0 {
		t.Fatal("no-DB fallback wrote event")
	}
	s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
	original := s.store
	s.store = mock
	if _, err := s.acceptTerminalEventIntent(context.Background(), intent, id); err == nil {
		t.Fatal("missing receipt store accepted")
	}
	s.store = original
	s.subStore = legacyQueueStore{QueueStore: s.subStore.(subscription.QueueStore)}
	if _, err := s.acceptTerminalEventIntent(context.Background(), intent, id); err == nil {
		t.Fatal("missing atomic queue accepted")
	}
	terminalAdmissionCounts(t, s, 0, 0, 0)
}
func TestTerminalEventAdmissionConcurrentIdenticalReplay(t *testing.T) {
	s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
	terminalAdmissionSubscription(t, s, "first")
	intent, id := terminalAdmissionIntent(t, nil)
	var wg sync.WaitGroup
	const n = 12
	acks := make(chan eventbus.DomainEventAcknowledgment, n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ack, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
			acks <- ack
			errs <- err
		}()
	}
	wg.Wait()
	close(acks)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first eventbus.DomainEventAcknowledgment
	for ack := range acks {
		if first.StoreID == 0 {
			first = ack
		}
		if ack != first {
			t.Fatalf("concurrent acknowledgments differ: %+v %+v", first, ack)
		}
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
}

type interruptedTerminalAdmissionStore struct {
	*store.SQLiteStore
	cancel context.CancelFunc
}

func (s interruptedTerminalAdmissionStore) InsertTerminalAdmissionTx(ctx context.Context, tx *sql.Tx, receipt store.TerminalAdmission) error {
	if err := s.SQLiteStore.InsertTerminalAdmissionTx(ctx, tx, receipt); err != nil {
		return err
	}
	s.cancel()
	return ctx.Err()
}
func TestTerminalEventAdmissionInterruptedBeforeCommitRetainsNoSuccess(t *testing.T) {
	s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
	terminalAdmissionSubscription(t, s, "first")
	original := s.store.(*store.SQLiteStore)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.store = interruptedTerminalAdmissionStore{SQLiteStore: original, cancel: cancel}
	intent, id := terminalAdmissionIntent(t, nil)
	ack, err := s.acceptTerminalEventIntent(ctx, intent, id)
	if err == nil || ack != (eventbus.DomainEventAcknowledgment{}) {
		t.Fatalf("interrupted admission ack=%+v err=%v", ack, err)
	}
	s.store = original
	terminalAdmissionCounts(t, s, 0, 0, 0)
	if _, err := s.acceptTerminalEventIntent(context.Background(), intent, id); err != nil {
		t.Fatal(err)
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
}

// Explicit finite cross-owner handoff: read Agent Manager's persisted intent,
// admit/replay against real SQLite, and return the exact ack as a local fixture.
func TestTerminalEventAdmissionNativeOutboxFixture(t *testing.T) {
	dir := os.Getenv("VROOLI_TERMINAL_OUTBOX_FIXTURE_DIR")
	if dir == "" {
		t.Skip("explicit native owner fixture directory not supplied")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "intent.json"))
	if err != nil {
		t.Fatal(err)
	}
	var input struct {
		Intent string `json:"event_intent_json"`
		ID     string `json:"event_intent_id"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "events.sqlite")
	s := terminalAdmissionFixture(t, path)
	terminalAdmissionSubscription(t, s, "native-owner-fixture")
	ack, err := s.acceptTerminalEventIntent(context.Background(), input.Intent, input.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := eventbus.ValidateDomainEventAcknowledgment(input.Intent, input.ID, ack); err != nil {
		t.Fatal(err)
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
	if err := s.store.Close(); err != nil {
		t.Fatal(err)
	}
	s = terminalAdmissionFixture(t, path)
	replay, err := s.acceptTerminalEventIntent(context.Background(), input.Intent, input.ID)
	if err != nil || replay != ack {
		t.Fatalf("native persisted replay ack=%+v err=%v", replay, err)
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ack.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

// A real SQLite driver with a one-shot lost COMMIT outcome. Both durable states
// are exercised: committed but the response is lost, and rolled back at COMMIT.
// The seam must return no acknowledgment for either error and replay safely.
type terminalCommitDriver struct {
	fail              atomic.Bool
	commitBeforeError bool
	exitAfterCommit   bool
}

func (d *terminalCommitDriver) Open(name string) (driver.Conn, error) {
	conn, err := (&sqlite.Driver{}).Open(name)
	if err != nil {
		return nil, err
	}
	return &terminalCommitConn{Conn: conn, owner: d}, nil
}

type terminalCommitConn struct {
	driver.Conn
	owner *terminalCommitDriver
}

func (c *terminalCommitConn) Begin() (driver.Tx, error) {
	tx, err := c.Conn.Begin()
	if err != nil {
		return nil, err
	}
	return &terminalCommitTx{Tx: tx, owner: c.owner}, nil
}

type terminalCommitTx struct {
	driver.Tx
	owner *terminalCommitDriver
}

func (tx *terminalCommitTx) Commit() error {
	if tx.owner.exitAfterCommit {
		if err := tx.Tx.Commit(); err != nil {
			return err
		}
		os.Exit(87)
	}

	if !tx.owner.fail.CompareAndSwap(true, false) {
		return tx.Tx.Commit()
	}
	var err error
	if tx.owner.commitBeforeError {
		err = tx.Tx.Commit()
	} else {
		err = tx.Tx.Rollback()
	}
	if err != nil {
		return err
	}
	return errors.New("fixture lost commit outcome")
}

var terminalCommitDriverID atomic.Uint64

func TestTerminalEventAdmissionUncertainCommitReplay(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprintf("committed_%v", committed), func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "events.sqlite")
			d := &terminalCommitDriver{commitBeforeError: committed}
			d.fail.Store(true)
			name := fmt.Sprintf("terminal-commit-%d", terminalCommitDriverID.Add(1))
			sql.Register(name, d)
			db, err := sql.Open(name, path)
			if err != nil {
				t.Fatal(err)
			}
			db.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = db.Close() })
			events, err := store.NewSQLiteStoreWithDB(ctx, db, store.SQLiteConfig{})
			if err != nil {
				t.Fatal(err)
			}
			subs, err := subscription.NewSQLiteStore(db)
			if err != nil {
				t.Fatal(err)
			}
			s := &Server{db: db, store: events, subStore: subs}
			terminalAdmissionSubscription(t, s, "first")
			intent, id := terminalAdmissionIntent(t, nil)
			ack, err := s.acceptTerminalEventIntent(ctx, intent, id)
			if err == nil || ack != (eventbus.DomainEventAcknowledgment{}) {
				t.Fatalf("uncertain commit ack=%+v err=%v", ack, err)
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			s = terminalAdmissionFixture(t, path)
			want := 0
			if committed {
				want = 1
			}
			terminalAdmissionCounts(t, s, want, want, want)
			ack, err = s.acceptTerminalEventIntent(ctx, intent, id)
			if err != nil {
				t.Fatal(err)
			}
			if err := eventbus.ValidateDomainEventAcknowledgment(intent, id, ack); err != nil {
				t.Fatal(err)
			}
			terminalAdmissionCounts(t, s, 1, 1, 1)
		})
	}
}

func TestTerminalEventAdmissionDoesNotSelectNotificationWakeBridge(t *testing.T) {
	s, forwarder, _, _ := nativeBridgeServer(t)
	intent, id := terminalAdmissionIntent(t, nil)
	ack, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
	if err != nil || replay != ack {
		t.Fatalf("terminal replay ack=%+v err=%v", replay, err)
	}
	terminalAdmissionCounts(t, s, 1, 0, 1)
	if len(forwarder.deliveries) != 0 {
		t.Fatal("terminal event activated notification wake delivery")
	}
}

func TestTerminalEventAdmissionLedgerIsAppendOnly(t *testing.T) {
	s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
	intent, id := terminalAdmissionIntent(t, nil)
	ack, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{
		"UPDATE terminal_event_admissions SET canonical_sha256='changed'",
		"DELETE FROM terminal_event_admissions",
		"INSERT OR REPLACE INTO terminal_event_admissions SELECT event_id,'changed',canonical_payload,canonical_sha256,store_id FROM terminal_event_admissions",
	} {
		if _, err := s.db.ExecContext(context.Background(), mutation); err == nil {
			t.Fatalf("immutable receipt mutation succeeded: %s", mutation)
		}
	}
	replay, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
	if err != nil || replay != ack {
		t.Fatalf("immutable receipt changed: %+v %v", replay, err)
	}
	terminalAdmissionCounts(t, s, 1, 0, 1)
}

func TestTerminalEventAdmissionCrashAfterCommitBeforeAcknowledgment(t *testing.T) {
	const crashDB = "VROOLI_TERMINAL_ADMISSION_CRASH_DB"
	if path := os.Getenv(crashDB); path != "" {
		// The fixture driver terminates the process after SQLite COMMIT succeeds,
		// before database/sql or the acceptance method receives that result.
		d := &terminalCommitDriver{exitAfterCommit: true}
		sql.Register("terminal-crash", d)
		db, err := sql.Open("terminal-crash", path)
		if err != nil {
			t.Fatal(err)
		}
		db.SetMaxOpenConns(1)
		events, err := store.NewSQLiteStoreWithDB(context.Background(), db, store.SQLiteConfig{})
		if err != nil {
			t.Fatal(err)
		}
		subs, err := subscription.NewSQLiteStore(db)
		if err != nil {
			t.Fatal(err)
		}
		s := &Server{db: db, store: events, subStore: subs}
		intent, id := terminalAdmissionIntent(t, nil)
		_, err = s.acceptTerminalEventIntent(context.Background(), intent, id)
		t.Fatalf("crash fixture returned instead of exiting at commit: %v", err)
	}
	path := filepath.Join(t.TempDir(), "events.sqlite")
	s := terminalAdmissionFixture(t, path)
	terminalAdmissionSubscription(t, s, "first")
	if err := s.store.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTerminalEventAdmissionCrashAfterCommitBeforeAcknowledgment$")
	cmd.Env = append(os.Environ(), crashDB+"="+path)
	output, err := cmd.CombinedOutput()
	var exited *exec.ExitError
	if !errors.As(err, &exited) || exited.ExitCode() != 87 {
		t.Fatalf("crash exit=%v output=%s", err, output)
	}
	s = terminalAdmissionFixture(t, path)
	terminalAdmissionCounts(t, s, 1, 1, 1)
	intent, id := terminalAdmissionIntent(t, nil)
	ack, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := eventbus.ValidateDomainEventAcknowledgment(intent, id, ack); err != nil {
		t.Fatal(err)
	}
	terminalAdmissionCounts(t, s, 1, 1, 1)
}

func TestTerminalEventAdmissionTerminalStatusesAndFactValidation(t *testing.T) {
	for _, status := range []string{"complete", "failed", "cancelled", "unknown"} {
		t.Run(status, func(t *testing.T) {
			s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
			intent, id := terminalAdmissionIntent(t, func(e *eventbus.DomainEvent) { e.Payload["status"] = status })
			ack, err := s.acceptTerminalEventIntent(context.Background(), intent, id)
			if err != nil {
				t.Fatal(err)
			}
			if err := eventbus.ValidateDomainEventAcknowledgment(intent, id, ack); err != nil {
				t.Fatal(err)
			}
			terminalAdmissionCounts(t, s, 1, 0, 1)
		})
	}
	for key, value := range map[string]any{"source_instance": "", "source_epoch": "invalid scope", "run_id": "00000000-0000-0000-0000-000000000000", "lifecycle_version": "01", "terminal_observation_sha256": "invalid", "extra_field": "unsupported"} {
		t.Run(key, func(t *testing.T) {
			s := terminalAdmissionFixture(t, filepath.Join(t.TempDir(), "events.sqlite"))
			intent, id := terminalAdmissionIntent(t, func(e *eventbus.DomainEvent) { e.Payload[key] = value })
			if _, err := s.acceptTerminalEventIntent(context.Background(), intent, id); err == nil {
				t.Fatal("invalid terminal fact accepted")
			}
			terminalAdmissionCounts(t, s, 0, 0, 0)
		})
	}
}
