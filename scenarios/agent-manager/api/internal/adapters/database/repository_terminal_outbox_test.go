package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runmodel"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/eventbus"
)

type terminalAcceptorFunc func(context.Context, string, string) (eventbus.DomainEventAcknowledgment, error)

func (f terminalAcceptorFunc) AcceptTerminalEventIntent(ctx context.Context, seed, id string) (eventbus.DomainEventAcknowledgment, error) {
	return f(ctx, seed, id)
}

// codecAck is a unit-test response only. Durable Events acceptance is qualified
// separately by the real two-owner SQLite fixture, never by this fake.
func codecAck(_ context.Context, seed, id string) (eventbus.DomainEventAcknowledgment, error) {
	_, wire, err := eventbus.RetainedDomainEventEnvelope(seed, id)
	return eventbus.DomainEventAcknowledgment{Schema: eventbus.DomainEventAcknowledgmentSchema, EventID: id, CanonicalSHA256: runmodel.SHA256(wire), StoreID: 1}, err
}

func terminalOutbox(t *testing.T, r *Repositories, f terminalAcceptorFunc) *TerminalOutbox {
	t.Helper()
	o, err := NewTerminalOutbox(r, f)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func outboxOpen(t *testing.T, path string) *DB {
	t.Helper()
	raw, err := sqlx.Connect("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(1)
	db := NewDB(raw, logrus.New())
	if err := db.InitializeSchema(); err != nil {
		t.Fatal(err)
	}
	return db
}

func ackCount(t *testing.T, db *DB) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(t.Context(), `SELECT count(*) FROM run_terminal_event_acknowledgments`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestTerminalOutboxDisabledAndEmpty(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	if _, err := NewTerminalOutbox(NewRepositories(db, logrus.New()), terminalAcceptorFunc(codecAck)); err == nil {
		t.Fatal("enabled default constructor")
	}
	if _, err := NewTerminalOutbox(receiptRepos(t, db), nil); err == nil {
		t.Fatal("allowed no acceptor")
	}
	calls := 0
	o := terminalOutbox(t, receiptRepos(t, db), func(ctx context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
		calls++
		return codecAck(ctx, s, id)
	})
	if n, err := o.Dispatch(t.Context(), 1); n != 0 || err != nil || calls != 0 {
		t.Fatalf("empty dispatch %d %v calls=%d", n, err, calls)
	}
	for _, limit := range []int{0, -1, 101} {
		if _, err := o.Dispatch(t.Context(), limit); err == nil {
			t.Fatal("unbounded dispatch")
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := o.Dispatch(ctx, 1); err == nil {
		t.Fatal("ignored cancellation")
	}
}

func TestTerminalOutboxRetainsExactAcknowledgmentAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.db")
	db := outboxOpen(t, path)
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	original := readReceipt(t, r, run)
	calls := 0
	o := terminalOutbox(t, r, func(ctx context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
		calls++
		if s != original.EventIntentJSON || id != original.EventIntentID {
			t.Fatal("changed retained seed")
		}
		return codecAck(ctx, s, id)
	})
	if n, err := o.Dispatch(t.Context(), 10); n != 1 || err != nil {
		t.Fatalf("dispatch %d %v", n, err)
	}
	ack, err := r.Runs.(*runRepository).TerminalEventAcknowledgment(t.Context(), run.ID, run.LifecycleVersion)
	if err != nil {
		t.Fatal(err)
	}
	if readReceipt(t, r, run) != original {
		t.Fatal("publication mutated immutable capture")
	}
	for _, q := range []string{`DELETE FROM run_terminal_event_acknowledgments`, `UPDATE run_terminal_event_acknowledgments SET acknowledgment_json='{}'`} {
		if _, err := db.Exec(q); err == nil {
			t.Fatal("ack was mutable")
		}
	}
	if err := r.Runs.Delete(t.Context(), run.ID, run.Status, run.LifecycleVersion); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db = outboxOpen(t, path)
	defer db.Close()
	r = receiptRepos(t, db)
	o = terminalOutbox(t, r, func(context.Context, string, string) (eventbus.DomainEventAcknowledgment, error) {
		t.Fatal("replayed acknowledged intent")
		return eventbus.DomainEventAcknowledgment{}, nil
	})
	if n, err := o.Dispatch(t.Context(), 10); n != 0 || err != nil {
		t.Fatalf("reopen %d %v", n, err)
	}
	got, err := r.Runs.(*runRepository).TerminalEventAcknowledgment(t.Context(), run.ID, run.LifecycleVersion)
	if err != nil || got != ack || calls != 1 {
		t.Fatalf("lost ack %#v %v", got, err)
	}
}

func TestTerminalOutboxUncertainAndMismatchedAcknowledgmentsStayPending(t *testing.T) {
	for _, mode := range []string{"lost_response", "zero", "wrong_id", "wrong_digest", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			db, done := setupTestDB(t)
			defer done()
			r := receiptRepos(t, db)
			run := receiptRun(t, r)
			finishReceiptRun(t, r, run, domain.RunStatusFailed)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			o := terminalOutbox(t, r, func(c context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
				ack, err := codecAck(c, s, id)
				switch mode {
				case "lost_response":
					return ack, errors.New("response lost after possible commit")
				case "zero":
					return eventbus.DomainEventAcknowledgment{}, nil
				case "wrong_id":
					ack.EventID = "wrong"
				case "wrong_digest":
					ack.CanonicalSHA256 = "wrong"
				case "cancelled":
					cancel()
				}
				return ack, err
			})
			if n, err := o.Dispatch(ctx, 1); n != 0 || err == nil || ackCount(t, db) != 0 {
				t.Fatalf("invalid acknowledgment persisted %d %v", n, err)
			}
			if _, err := r.Runs.(*runRepository).TerminalEventAcknowledgment(t.Context(), run.ID, run.LifecycleVersion); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("unexpected ack read: %v", err)
			}
			o = terminalOutbox(t, r, codecAck)
			if n, err := o.Dispatch(t.Context(), 1); n != 1 || err != nil {
				t.Fatalf("retry %d %v", n, err)
			}
		})
	}
}

func TestTerminalOutboxAcknowledgmentStatementAndCommitFailure(t *testing.T) {
	for _, failure := range []string{"statement", "commit"} {
		t.Run(failure, func(t *testing.T) {
			db := outboxOpen(t, filepath.Join(t.TempDir(), "failure.db"))
			defer db.Close()
			r := receiptRepos(t, db)
			run := receiptRun(t, r)
			finishReceiptRun(t, r, run, domain.RunStatusComplete)
			trigger := `CREATE TRIGGER reject_ack BEFORE INSERT ON run_terminal_event_acknowledgments BEGIN SELECT RAISE(ABORT,'fixture failure'); END;`
			if failure == "commit" {
				trigger = `CREATE TABLE ack_test_parent(id TEXT PRIMARY KEY); CREATE TABLE ack_test_child(id TEXT REFERENCES ack_test_parent(id) DEFERRABLE INITIALLY DEFERRED); CREATE TRIGGER reject_ack AFTER INSERT ON run_terminal_event_acknowledgments BEGIN INSERT INTO ack_test_child VALUES ('missing'); END;`
			}
			mustReceiptExec(t, db, trigger)
			o := terminalOutbox(t, r, codecAck)
			if n, err := o.Dispatch(t.Context(), 1); n != 0 || err == nil || ackCount(t, db) != 0 {
				t.Fatalf("failure lost pending intent %d %v", n, err)
			}
			mustReceiptExec(t, db, `DROP TRIGGER reject_ack`)
			if n, err := o.Dispatch(t.Context(), 1); n != 1 || err != nil {
				t.Fatalf("retry %d %v", n, err)
			}
		})
	}
}

func TestTerminalOutboxFiniteBatchAndSourceFence(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	for i := 0; i < 3; i++ {
		run := receiptRun(t, r)
		finishReceiptRun(t, r, run, domain.RunStatusCancelled)
	}
	o := terminalOutbox(t, r, codecAck)
	for _, want := range []int{2, 1, 0} {
		if n, err := o.Dispatch(t.Context(), 2); n != want || err != nil {
			t.Fatalf("batch %d want %d %v", n, want, err)
		}
	}
	other, err := NewRepositoriesWithTerminalReceipts(db, logrus.New(), runmodel.TerminalReceiptScope{SourceInstance: "wrong-source", SourceEpoch: receiptTestScope.SourceEpoch})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := terminalOutbox(t, other, codecAck).Dispatch(t.Context(), 1); err == nil {
		t.Fatal("wrong source allowed")
	}
}

func TestTerminalOutboxConcurrentIndependentConnections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent.db")
	a := outboxOpen(t, path)
	defer a.Close()
	b := outboxOpen(t, path)
	defer b.Close()
	ra, rb := receiptRepos(t, a), receiptRepos(t, b)
	run := receiptRun(t, ra)
	finishReceiptRun(t, ra, run, domain.RunStatusComplete)
	var arrived sync.WaitGroup
	arrived.Add(2)
	f := func(ctx context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
		arrived.Done()
		arrived.Wait()
		return codecAck(ctx, s, id)
	}
	var workers sync.WaitGroup
	workers.Add(2)
	for _, r := range []*Repositories{ra, rb} {
		go func(r *Repositories) {
			defer workers.Done()
			if n, err := terminalOutbox(t, r, f).Dispatch(t.Context(), 1); n != 1 || err != nil {
				t.Errorf("concurrent dispatch %d %v", n, err)
			}
		}(r)
	}
	workers.Wait()
	if ackCount(t, a) != 1 {
		t.Fatal("duplicate acknowledgment")
	}
	original := readReceipt(t, ra, run)
	bad, _ := codecAck(t.Context(), original.EventIntentJSON, original.EventIntentID)
	bad.StoreID = 999
	if err := terminalOutbox(t, ra, codecAck).retainAcknowledgment(t.Context(), original, bad); err == nil {
		t.Fatal("conflicting durable ack ignored")
	}
}

func TestTerminalOutboxCorruptionStopsBeforeAcceptor(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	mustReceiptExec(t, db, `DROP TRIGGER run_terminal_receipts_no_update; UPDATE run_terminal_receipts SET event_intent_json='{}'`)
	o := terminalOutbox(t, r, func(context.Context, string, string) (eventbus.DomainEventAcknowledgment, error) {
		t.Fatal("called acceptor with corrupt seed")
		return eventbus.DomainEventAcknowledgment{}, nil
	})
	if _, err := o.Dispatch(t.Context(), 1); err == nil {
		t.Fatal("corruption ignored")
	}
}

func TestTerminalOutboxProcessExitRecovery(t *testing.T) {
	if path := os.Getenv("AM_TERMINAL_OUTBOX_EXIT_DB"); path != "" {
		db := outboxOpen(t, path)
		r := receiptRepos(t, db)
		phase := os.Getenv("AM_TERMINAL_OUTBOX_EXIT_PHASE")
		o := terminalOutbox(t, r, func(ctx context.Context, seed, id string) (eventbus.DomainEventAcknowledgment, error) {
			ack, err := codecAck(ctx, seed, id)
			if err != nil {
				t.Fatal(err)
			}
			if phase == "before_ack_commit" {
				tx, err := db.BeginTxx(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				encoded, _ := json.Marshal(ack)
				if _, err := tx.ExecContext(ctx, `INSERT INTO run_terminal_event_acknowledgments(event_intent_id,acknowledgment_json) VALUES (?,?)`, id, string(encoded)); err != nil {
					t.Fatal(err)
				}
				os.Exit(73)
			}
			if phase == "before_acceptance_return" {
				os.Exit(73)
			}
			return ack, nil
		})
		if _, err := o.Dispatch(t.Context(), 1); err != nil {
			t.Fatal(err)
		}
		os.Exit(74)
		return
	}
	for _, phase := range []string{"before_acceptance_return", "before_ack_commit", "after_ack_commit"} {
		t.Run(phase, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "exit.db")
			db := outboxOpen(t, path)
			r := receiptRepos(t, db)
			run := receiptRun(t, r)
			finishReceiptRun(t, r, run, domain.RunStatusComplete)
			original := readReceipt(t, r, run)
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTerminalOutboxProcessExitRecovery$")
			cmd.Env = append(os.Environ(), "AM_TERMINAL_OUTBOX_EXIT_DB="+path, "AM_TERMINAL_OUTBOX_EXIT_PHASE="+phase)
			out, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			code := 73
			if phase == "after_ack_commit" {
				code = 74
			}
			if !ok || exit.ExitCode() != code {
				t.Fatalf("exit: %v %s", err, out)
			}
			db = outboxOpen(t, path)
			defer db.Close()
			r = receiptRepos(t, db)
			if readReceipt(t, r, run) != original {
				t.Fatal("process exit altered immutable receipt")
			}
			want := 1
			if phase == "after_ack_commit" {
				want = 0
			}
			if ackCount(t, db) != 1-want {
				t.Fatal("process exit exposed incorrect acknowledgment state")
			}
			if n, err := terminalOutbox(t, r, codecAck).Dispatch(t.Context(), 1); n != want || err != nil {
				t.Fatalf("recovery %d %v", n, err)
			}
		})
	}
}

type terminalOwnerFixture struct {
	EventIntentJSON  string    `json:"event_intent_json"`
	EventIntentID    string    `json:"event_intent_id"`
	RunID            uuid.UUID `json:"run_id"`
	LifecycleVersion int64     `json:"lifecycle_version"`
}

// Explicit file handoff spans real owner SQLite databases in separate processes.
// It is fixture-only and never a runtime environment toggle or transport.
func TestTerminalOutboxCrossOwnerFixture(t *testing.T) {
	dir := os.Getenv("VROOLI_TERMINAL_OUTBOX_FIXTURE_DIR")
	if dir == "" {
		t.Skip("explicit cross-owner fixture path not requested")
	}
	stage := os.Getenv("VROOLI_TERMINAL_OUTBOX_FIXTURE_STAGE")
	path := filepath.Join(dir, "agent-manager.db")
	db := outboxOpen(t, path)
	defer db.Close()
	r := receiptRepos(t, db)
	if stage == "capture" {
		if receiptCount(t, db) != 0 {
			t.Fatal("fixture capture requires fresh database")
		}
		run := receiptRun(t, r)
		finishReceiptRun(t, r, run, domain.RunStatusComplete)
		receipt := readReceipt(t, r, run)
		value := terminalOwnerFixture{receipt.EventIntentJSON, receipt.EventIntentID, run.ID, run.LifecycleVersion}
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "intent.json"), encoded, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	if stage != "acknowledge" {
		t.Fatal("explicit capture or acknowledge stage required")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "intent.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture terminalOwnerFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filepath.Join(dir, "ack.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ack eventbus.DomainEventAcknowledgment
	if err := json.Unmarshal(raw, &ack); err != nil {
		t.Fatal(err)
	}
	called := 0
	o := terminalOutbox(t, r, func(_ context.Context, s, id string) (eventbus.DomainEventAcknowledgment, error) {
		called++
		if s != fixture.EventIntentJSON || id != fixture.EventIntentID {
			t.Fatal("cross-owner identity changed")
		}
		return ack, nil
	})
	if n, err := o.Dispatch(t.Context(), 1); n != 1 || err != nil {
		t.Fatalf("cross-owner acceptance %d %v", n, err)
	}
	if n, err := o.Dispatch(t.Context(), 1); n != 0 || err != nil || called != 1 {
		t.Fatalf("cross-owner replay %d %v calls=%d", n, err, called)
	}
	got, err := r.Runs.(*runRepository).TerminalEventAcknowledgment(t.Context(), fixture.RunID, fixture.LifecycleVersion)
	if err != nil || got != ack {
		t.Fatalf("cross-owner retained ack %#v %v", got, err)
	}
}

func TestTerminalOutboxImmutabilityIncludesSQLiteReplace(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	original := readReceipt(t, r, run)
	o := terminalOutbox(t, r, codecAck)
	if _, err := o.Dispatch(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	ack, err := r.Runs.(*runRepository).TerminalEventAcknowledgment(t.Context(), run.ID, run.LifecycleVersion)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT OR REPLACE INTO run_terminal_event_acknowledgments SELECT event_intent_id,'{}' FROM run_terminal_event_acknowledgments`,
		`INSERT OR REPLACE INTO run_terminal_receipt_source VALUES(1,'changed-source','changed-epoch')`,
		`INSERT OR REPLACE INTO run_terminal_receipts SELECT run_id,lifecycle_version,source_instance,source_epoch,observation_json,observation_sha256,'{}',event_intent_id FROM run_terminal_receipts`,
	} {
		if _, err := db.Exec(q); err == nil {
			t.Fatalf("SQLite REPLACE bypassed immutable ledger: %s", q)
		}
	}
	if readReceipt(t, r, run) != original {
		t.Fatal("receipt changed")
	}
	got, err := r.Runs.(*runRepository).TerminalEventAcknowledgment(t.Context(), run.ID, run.LifecycleVersion)
	if err != nil || got != ack {
		t.Fatal("ack changed")
	}
	if err := o.retainAcknowledgment(t.Context(), original, ack); err != nil {
		t.Fatalf("exact idempotent ack blocked: %v", err)
	}
}

func TestTerminalOutboxInvalidRepositoryAndCorruptAcknowledgmentFailClosed(t *testing.T) {
	missing, err := NewRepositoriesWithTerminalReceipts(nil, logrus.New(), receiptTestScope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewTerminalOutbox(missing, terminalAcceptorFunc(codecAck)); err == nil {
		t.Fatal("accepted missing database")
	}
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	if _, err := terminalOutbox(t, r, codecAck).Dispatch(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	mustReceiptExec(t, db, `DROP TRIGGER run_terminal_event_acknowledgments_no_update; UPDATE run_terminal_event_acknowledgments SET acknowledgment_json='{"schema":"invalid","store_id":1}'`)
	if ack, err := r.Runs.(*runRepository).TerminalEventAcknowledgment(t.Context(), run.ID, run.LifecycleVersion); err == nil || ack != (eventbus.DomainEventAcknowledgment{}) {
		t.Fatalf("corrupt proof returned acknowledgment %#v %v", ack, err)
	}
}
