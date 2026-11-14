package database

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runmodel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/eventbus"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

var receiptTestScope = runmodel.TerminalReceiptScope{SourceInstance: "test-installation", SourceEpoch: "epoch-1"}

func receiptRepos(t *testing.T, db *DB) *Repositories {
	t.Helper()
	r, e := NewRepositoriesWithTerminalReceipts(db, logrus.New(), receiptTestScope)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func receiptRun(t *testing.T, r *Repositories) *domain.Run {
	t.Helper()
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning}
	if e := r.Runs.Create(t.Context(), run); e != nil {
		t.Fatal(e)
	}
	return run
}
func receiptCount(t *testing.T, db *DB) int {
	t.Helper()
	var n int
	if e := db.Get(&n, `SELECT COUNT(*) FROM run_terminal_receipts`); e != nil {
		t.Fatal(e)
	}
	return n
}
func readReceipt(t *testing.T, r *Repositories, run *domain.Run) runmodel.TerminalReceipt {
	t.Helper()
	v, e := r.Runs.(*runRepository).TerminalReceipt(t.Context(), run.ID, run.LifecycleVersion)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func finishReceiptRun(t *testing.T, r *Repositories, run *domain.Run, status domain.RunStatus) {
	t.Helper()
	run.Status = status
	at := time.Now().UTC()
	run.EndedAt = &at
	if e := r.Runs.Update(t.Context(), run); e != nil {
		t.Fatal(e)
	}
}
func mustReceiptExec(t *testing.T, db *DB, q string) {
	t.Helper()
	if _, e := db.Exec(q); e != nil {
		t.Fatal(e)
	}
}

func TestTerminalReceiptDisabledDoesNotTouchReceiptStorage(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := NewRepositories(db, logrus.New())
	mustReceiptExec(t, db, `DROP TABLE run_terminal_receipts; DROP TABLE run_terminal_receipt_source`)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	if _, e := r.Runs.(*runRepository).TerminalReceipt(t.Context(), run.ID, run.LifecycleVersion); e == nil {
		t.Fatal("disabled read succeeded")
	}
}
func TestTerminalReceiptStatusesAndMinimization(t *testing.T) {
	for _, status := range []domain.RunStatus{domain.RunStatusComplete, domain.RunStatusFailed, domain.RunStatusCancelled, domain.RunStatusUnknown} {
		t.Run(string(status), func(t *testing.T) {
			db, done := setupTestDB(t)
			defer done()
			r := receiptRepos(t, db)
			run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, Label: "secret title", ErrorMsg: "secret error", WorkReferences: []*eventdomain.WorkReference{{Kind: "flywheel", Id: "invented-work", Verified: true}}}
			if e := r.Runs.Create(t.Context(), run); e != nil {
				t.Fatal(e)
			}
			finishReceiptRun(t, r, run, status)
			v := readReceipt(t, r, run)
			var o runmodel.TerminalObservation
			if e := json.Unmarshal([]byte(v.ObservationJSON), &o); e != nil {
				t.Fatal(e)
			}
			if o.Status != string(status) || o.Coverage != "terminal_state_only" || o.WorkLineage != "unverified" || o.TerminalReceiptScope != receiptTestScope {
				t.Fatalf("bad evidence: %+v", o)
			}
			if v.PublicationState != "not_dispatched" || strings.Contains(v.ObservationJSON, "secret") || strings.Contains(v.ObservationJSON, "invented-work") {
				t.Fatal("overclaim or leak")
			}
			var event eventbus.DomainEvent
			if e := json.Unmarshal([]byte(v.EventIntentJSON), &event); e != nil {
				t.Fatal(e)
			}
			seed, e := json.Marshal(event)
			if e != nil {
				t.Fatal(e)
			}
			if runmodel.SHA256(seed) != v.EventIntentID || event.EventType != runmodel.TerminalEventType || event.Payload["terminal_observation_sha256"] != v.ObservationSHA256 {
				t.Fatal("unstable event intent")
			}
		})
	}
}
func TestTerminalReceiptNoBackfillOrImportedCapture(t *testing.T) {
	for _, kind := range []string{"terminal_create", "terminal_correction", "imported_mode", "import_marker", "needs_review"} {
		t.Run(kind, func(t *testing.T) {
			db, done := setupTestDB(t)
			defer done()
			r := receiptRepos(t, db)
			run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning}
			switch kind {
			case "terminal_create", "terminal_correction":
				run.Status = domain.RunStatusComplete
			case "imported_mode":
				run.ExecutionMode = domain.ExecutionModeImported
			case "import_marker":
				run.ImportSourceHarness = "codex"
			}
			if e := r.Runs.Create(t.Context(), run); e != nil {
				t.Fatal(e)
			}
			status := domain.RunStatusComplete
			if kind == "terminal_correction" {
				status = domain.RunStatusFailed
			}
			if kind == "needs_review" {
				status = domain.RunStatusNeedsReview
			}
			finishReceiptRun(t, r, run, status)
			if receiptCount(t, db) != 0 {
				t.Fatal("unqualified row gained receipt")
			}
			if _, e := r.Runs.(*runRepository).TerminalReceipt(t.Context(), run.ID, run.LifecycleVersion); e == nil {
				t.Fatal("missing receipt read succeeded")
			}
		})
	}
}
func TestTerminalReceiptFailureRollsBackRunTaskAndSource(t *testing.T) {
	for _, target := range []string{"receipt", "task"} {
		t.Run(target, func(t *testing.T) {
			db, done := setupTestDB(t)
			defer done()
			r := receiptRepos(t, db)
			task := createQueuedTask(t, r, "receipt atomicity")
			run := pendingTaskRun(task.ID)
			if e := r.Runs.CreateTaskAttempt(t.Context(), task, task, run, nil); e != nil {
				t.Fatal(e)
			}
			q := `CREATE TRIGGER reject_receipt BEFORE INSERT ON run_terminal_receipts BEGIN SELECT RAISE(ABORT,'injected'); END;`
			if target == "task" {
				q = `CREATE TRIGGER reject_task BEFORE UPDATE ON tasks WHEN NEW.status='completed' BEGIN SELECT RAISE(ABORT,'injected'); END;`
			}
			mustReceiptExec(t, db, q)
			run.Status = domain.RunStatusComplete
			if e := r.Runs.Update(t.Context(), run); e == nil {
				t.Fatal("injected failure committed")
			}
			stored, e := r.Runs.Get(t.Context(), run.ID)
			if e != nil {
				t.Fatal(e)
			}
			current, e := r.Tasks.Get(t.Context(), task.ID)
			if e != nil {
				t.Fatal(e)
			}
			if stored.Status != domain.RunStatusPending || stored.LifecycleVersion != 0 || current.Status != domain.TaskStatusRunning || receiptCount(t, db) != 0 {
				t.Fatalf("partial commit: %s %s", stored.Status, current.Status)
			}
			var n int
			if e := db.Get(&n, `SELECT COUNT(*) FROM run_terminal_receipt_source`); e != nil {
				t.Fatal(e)
			}
			if n != 0 {
				t.Fatal("failed transaction bound source")
			}
		})
	}
}
func TestTerminalReceiptReplayContinuationCorrectionAndDeletion(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	at := time.Now().Add(-time.Hour).UTC()
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeInteractive, InteractiveInvocationStartedAt: &at}
	if e := r.Runs.Create(t.Context(), run); e != nil {
		t.Fatal(e)
	}
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	firstVersion := run.LifecycleVersion
	first := readReceipt(t, r, run)
	run.Label = "mutated"
	if e := r.Runs.Update(t.Context(), run); e != nil {
		t.Fatal(e)
	}
	finishReceiptRun(t, r, run, domain.RunStatusFailed)
	if receiptCount(t, db) != 1 {
		t.Fatal("terminal correction created receipt")
	}
	next := at.Add(time.Minute)
	prior := run.InteractiveInvocationStartedAt
	run.Status = domain.RunStatusRunning
	run.InteractiveInvocationStartedAt = &next
	run.EndedAt = nil
	if e := r.Runs.UpdateInteractiveContinuation(t.Context(), run, domain.RunStatusFailed, prior); e != nil {
		t.Fatal(e)
	}
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	second := readReceipt(t, r, run)
	old, e := r.Runs.(*runRepository).TerminalReceipt(t.Context(), run.ID, firstVersion)
	if e != nil {
		t.Fatal(e)
	}
	if old != first || first.EventIntentID == second.EventIntentID || receiptCount(t, db) != 2 {
		t.Fatal("history changed or conflated")
	}
	for _, q := range []string{`UPDATE run_terminal_receipts SET observation_json='{}'`, `DELETE FROM run_terminal_receipts`, `UPDATE run_terminal_receipt_source SET source_epoch='changed'`, `DELETE FROM run_terminal_receipt_source`} {
		if _, e := db.Exec(q); e == nil {
			t.Fatalf("immutable data changed: %s", q)
		}
	}
	if e := r.Runs.Delete(t.Context(), run.ID, run.Status, run.LifecycleVersion); e != nil {
		t.Fatal(e)
	}
	old, e = r.Runs.(*runRepository).TerminalReceipt(t.Context(), run.ID, firstVersion)
	if e != nil || old != first {
		t.Fatalf("deletion destroyed evidence: %v", e)
	}
}
func TestTerminalReceiptConcurrentAndStaleWriters(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	db.DB.SetMaxOpenConns(1)
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	const n = 8
	results := make(chan error, n)
	var wg sync.WaitGroup
	for range n {
		candidate := *run
		candidate.Status = domain.RunStatusComplete
		wg.Add(1)
		go func() { defer wg.Done(); results <- r.Runs.Update(context.Background(), &candidate) }()
	}
	wg.Wait()
	close(results)
	winners := 0
	for e := range results {
		if e == nil {
			winners++
		}
	}
	if winners != 1 || receiptCount(t, db) != 1 {
		t.Fatalf("%d winners, %d receipts", winners, receiptCount(t, db))
	}
	run.Status = domain.RunStatusFailed
	if e := r.Runs.Update(t.Context(), run); e == nil {
		t.Fatal("stale writer committed")
	}
}
func TestTerminalReceiptCancellation(t *testing.T) {
	for _, kind := range []string{"pending", "active_fence", "capture_failure"} {
		t.Run(kind, func(t *testing.T) {
			db, done := setupTestDB(t)
			defer done()
			r := receiptRepos(t, db)
			task := createQueuedTask(t, r, "cancel receipt")
			run := pendingTaskRun(task.ID)
			if e := r.Runs.CreateTaskAttempt(t.Context(), task, task, run, nil); e != nil {
				t.Fatal(e)
			}
			if kind == "capture_failure" {
				mustReceiptExec(t, db, `CREATE TRIGGER reject_cancel BEFORE INSERT ON run_terminal_receipts BEGIN SELECT RAISE(ABORT,'injected'); END;`)
			}
			_, changed, e := r.Runs.CancelTaskAttempt(t.Context(), task.ID, &run.ID, kind == "active_fence", time.Now())
			if kind == "capture_failure" {
				if e == nil {
					t.Fatal("capture failure committed")
				}
			} else if e != nil || !changed {
				t.Fatalf("cancel failed: %v", e)
			}
			stored, e := r.Runs.Get(t.Context(), run.ID)
			if e != nil {
				t.Fatal(e)
			}
			expected := 0
			if kind == "pending" {
				expected = 1
				if stored.Status != domain.RunStatusCancelled {
					t.Fatal("pending not cancelled")
				}
				readReceipt(t, r, stored)
			}
			if receiptCount(t, db) != expected {
				t.Fatal("wrong cancellation receipt count")
			}
			if kind == "capture_failure" {
				task, e := r.Tasks.Get(t.Context(), task.ID)
				if e != nil {
					t.Fatal(e)
				}
				if stored.Status != domain.RunStatusPending || stored.LifecycleVersion != 0 || task.Status != domain.TaskStatusRunning {
					t.Fatal("partial cancellation")
				}
			}
		})
	}
}
func TestTerminalReceiptSourceCopyAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt.db")
	open := func() (*DB, *sqlx.DB) {
		raw, e := sqlx.Connect("sqlite", path)
		if e != nil {
			t.Fatal(e)
		}
		db := NewDB(raw, logrus.New())
		if e := db.InitializeSchema(); e != nil {
			t.Fatal(e)
		}
		return db, raw
	}
	db, raw := open()
	scope := receiptTestScope
	r, e := NewRepositoriesWithTerminalReceipts(db, logrus.New(), scope)
	if e != nil {
		t.Fatal(e)
	}
	scope.SourceEpoch = "mutated-caller"
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	first := readReceipt(t, r, run)
	if e := raw.Close(); e != nil {
		t.Fatal(e)
	}
	db, raw = open()
	defer raw.Close()
	r = receiptRepos(t, db)
	if readReceipt(t, r, run) != first {
		t.Fatal("reopen changed receipt")
	}
	for _, wrong := range []runmodel.TerminalReceiptScope{{SourceInstance: "other", SourceEpoch: receiptTestScope.SourceEpoch}, {SourceInstance: receiptTestScope.SourceInstance, SourceEpoch: "other"}} {
		other, e := NewRepositoriesWithTerminalReceipts(db, logrus.New(), wrong)
		if e != nil {
			t.Fatal(e)
		}
		if _, e := other.Runs.(*runRepository).TerminalReceipt(t.Context(), run.ID, run.LifecycleVersion); e == nil {
			t.Fatal("wrong source read succeeded")
		}
		candidate := receiptRun(t, other)
		candidate.Status = domain.RunStatusComplete
		if e := other.Runs.Update(t.Context(), candidate); e == nil {
			t.Fatal("source relabeled")
		}
		stored, e := r.Runs.Get(t.Context(), candidate.ID)
		if e != nil {
			t.Fatal(e)
		}
		if stored.Status != domain.RunStatusRunning {
			t.Fatal("source mismatch partially committed")
		}
	}
}
func TestTerminalReceiptInvalidScopeWithoutStorage(t *testing.T) {
	for _, s := range []runmodel.TerminalReceiptScope{{}, {SourceInstance: "valid"}, {SourceInstance: "wrong\n", SourceEpoch: "valid"}, {SourceInstance: "valid", SourceEpoch: strings.Repeat("x", 65)}} {
		if _, e := NewRepositoriesWithTerminalReceipts(nil, nil, s); e == nil {
			t.Fatal("invalid source admitted")
		}
	}
}

func TestTerminalReceiptConflictingRecordRollsBackInsteadOfIgnoring(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	_, e := db.Exec(`INSERT INTO run_terminal_receipts (run_id,lifecycle_version,source_instance,source_epoch,observation_json,observation_sha256,event_intent_json,event_intent_id) VALUES (?,1,'bad','bad','{}','bad','{}','bad')`, run.ID)
	if e != nil {
		t.Fatal(e)
	}
	run.Status = domain.RunStatusComplete
	if e := r.Runs.Update(t.Context(), run); e == nil {
		t.Fatal("conflicting receipt silently reused")
	}
	stored, e := r.Runs.Get(t.Context(), run.ID)
	if e != nil {
		t.Fatal(e)
	}
	if stored.Status != domain.RunStatusRunning || stored.LifecycleVersion != 0 {
		t.Fatal("conflict partially committed")
	}
}

func TestTerminalReceiptDetectsRetainedByteCorruption(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	finishReceiptRun(t, r, run, domain.RunStatusComplete)
	// Simulate storage corruption outside the supported repository, not a write API.
	mustReceiptExec(t, db, `DROP TRIGGER run_terminal_receipts_no_update; UPDATE run_terminal_receipts SET observation_json='{}'`)
	if _, e := r.Runs.(*runRepository).TerminalReceipt(t.Context(), run.ID, run.LifecycleVersion); e == nil {
		t.Fatal("corrupt immutable evidence accepted")
	}
}

func TestTerminalReceiptConcurrentConnectionsHaveOneWinner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent-receipts.db")
	open := func() *DB {
		raw, e := sqlx.Connect("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
		if e != nil {
			t.Fatal(e)
		}
		raw.SetMaxOpenConns(1)
		t.Cleanup(func() { raw.Close() })
		return NewDB(raw, logrus.New())
	}
	firstDB := open()
	if e := firstDB.InitializeSchema(); e != nil {
		t.Fatal(e)
	}
	first := receiptRepos(t, firstDB)
	run := receiptRun(t, first)
	const n = 4
	repos := []*Repositories{first}
	for len(repos) < n {
		repos = append(repos, receiptRepos(t, open()))
	}
	results := make(chan error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, r := range repos {
		candidate := *run
		candidate.Status = domain.RunStatusComplete
		wg.Add(1)
		go func(r *Repositories) {
			defer wg.Done()
			<-start
			results <- r.Runs.Update(context.Background(), &candidate)
		}(r)
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	for e := range results {
		if e == nil {
			winners++
		}
	}
	if winners != 1 || receiptCount(t, firstDB) != 1 {
		t.Fatalf("%d winners, %d receipts", winners, receiptCount(t, firstDB))
	}
}

func TestTerminalReceiptCommitFailureRollsBack(t *testing.T) {
	db, done := setupTestDB(t)
	defer done()
	db.DB.SetMaxOpenConns(1)
	r := receiptRepos(t, db)
	run := receiptRun(t, r)
	mustReceiptExec(t, db, `CREATE TABLE receipt_test_parent (id TEXT PRIMARY KEY);
 CREATE TABLE receipt_test_deferred (id TEXT REFERENCES receipt_test_parent(id) DEFERRABLE INITIALLY DEFERRED);
 CREATE TRIGGER reject_receipt_commit AFTER INSERT ON run_terminal_receipts BEGIN INSERT INTO receipt_test_deferred (id) VALUES ('missing'); END;`)
	run.Status = domain.RunStatusComplete
	if e := r.Runs.Update(t.Context(), run); e == nil {
		t.Fatal("deferred commit failure ignored")
	}
	stored, e := r.Runs.Get(t.Context(), run.ID)
	if e != nil {
		t.Fatal(e)
	}
	if stored.Status != domain.RunStatusRunning || stored.LifecycleVersion != 0 || receiptCount(t, db) != 0 {
		t.Fatal("failed commit exposed partial state")
	}
	var sourceCount int
	if e := db.Get(&sourceCount, `SELECT COUNT(*) FROM run_terminal_receipt_source`); e != nil {
		t.Fatal(e)
	}
	if sourceCount != 0 {
		t.Fatal("failed commit retained source binding")
	}
}

func TestTerminalReceiptProcessExitRecovery(t *testing.T) {
	// Run only this test's native transaction in a finite child. Exit without
	// closing the database either before COMMIT or after COMMIT but before any
	// owner response can return. No service or live owner is involved.
	if phase := os.Getenv("AM_TERMINAL_RECEIPT_TEST_CHILD"); phase != "" {
		raw, e := sqlx.Connect("sqlite", os.Getenv("AM_TERMINAL_RECEIPT_TEST_DB"))
		if e != nil {
			t.Fatal(e)
		}
		db := NewDB(raw, logrus.New())
		r := receiptRepos(t, db)
		id, e := uuid.Parse(os.Getenv("AM_TERMINAL_RECEIPT_TEST_RUN"))
		if e != nil {
			t.Fatal(e)
		}
		tx, e := db.BeginTxx(t.Context(), nil)
		if e != nil {
			t.Fatal(e)
		}
		runs := r.Runs.(*runRepository)
		run, e := runs.getTx(t.Context(), tx, id)
		if e != nil {
			t.Fatal(e)
		}
		run.Status = domain.RunStatusComplete
		var next int64
		if e := runs.updateTx(t.Context(), tx, run, nil, &next); e != nil {
			t.Fatal(e)
		}
		if phase == "before_commit" {
			os.Exit(71)
		}
		if phase != "after_commit" {
			t.Fatal("invalid child phase")
		}
		if e := tx.Commit(); e != nil {
			t.Fatal(e)
		}
		os.Exit(72)
	}
	for _, phase := range []string{"before_commit", "after_commit"} {
		t.Run(phase, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "exit-receipt.db")
			raw, e := sqlx.Connect("sqlite", path)
			if e != nil {
				t.Fatal(e)
			}
			db := NewDB(raw, logrus.New())
			if e := db.InitializeSchema(); e != nil {
				t.Fatal(e)
			}
			r := receiptRepos(t, db)
			run := receiptRun(t, r)
			if e := raw.Close(); e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTerminalReceiptProcessExitRecovery$")
			cmd.Env = append(os.Environ(), "AM_TERMINAL_RECEIPT_TEST_CHILD="+phase, "AM_TERMINAL_RECEIPT_TEST_DB="+path, "AM_TERMINAL_RECEIPT_TEST_RUN="+run.ID.String())
			out, e := cmd.CombinedOutput()
			exit, ok := e.(*exec.ExitError)
			wantCode := 71
			if phase == "after_commit" {
				wantCode = 72
			}
			if !ok || exit.ExitCode() != wantCode {
				t.Fatalf("child exit failed: %v %s", e, out)
			}
			raw, e = sqlx.Connect("sqlite", path)
			if e != nil {
				t.Fatal(e)
			}
			defer raw.Close()
			db = NewDB(raw, logrus.New())
			r = receiptRepos(t, db)
			stored, e := r.Runs.Get(t.Context(), run.ID)
			if e != nil {
				t.Fatal(e)
			}
			if phase == "before_commit" {
				if stored.Status != domain.RunStatusRunning || stored.LifecycleVersion != 0 || receiptCount(t, db) != 0 {
					t.Fatal("uncommitted process exit exposed partial state")
				}
				return
			}
			if stored.Status != domain.RunStatusComplete || receiptCount(t, db) != 1 {
				t.Fatal("committed process exit lost receipt")
			}
			first := readReceipt(t, r, stored)
			run.Status = domain.RunStatusComplete
			if e := r.Runs.Update(t.Context(), run); e == nil {
				t.Fatal("lost-response stale write bypassed fence")
			}
			if second := readReceipt(t, r, stored); second != first || receiptCount(t, db) != 1 {
				t.Fatal("lost-response read changed committed bytes")
			}
		})
	}
}
