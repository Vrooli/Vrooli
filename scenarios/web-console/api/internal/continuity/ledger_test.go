package continuity

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSQLLedgerPutIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(context.Background(), `CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	l := NewSQLLedger(db)
	r := Receipt{OperationID: "op-1", SessionID: "s-1", ActorKind: "operator", Command: "archive", FromState: StateLive, ToState: StateArchived, ReasonCode: "close", Status: "succeeded", CreatedAt: time.Unix(10, 0)}
	first, err := l.Put(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	second, err := l.Put(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if first.OperationID != second.OperationID || first.CreatedAt != second.CreatedAt {
		t.Fatalf("retry changed receipt: first=%+v second=%+v", first, second)
	}
}

func TestSQLLedgerCompleteIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	l := NewSQLLedger(db)
	_, err = l.Put(context.Background(), Receipt{OperationID: "op-2", SessionID: "catalog", ActorKind: "operator", Command: "reconcile", FromState: StateArchived, ToState: StateRecoverable, Status: "pending", CreatedAt: time.Unix(10, 0)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := l.Complete(context.Background(), "op-2", "succeeded", "", time.Unix(20, 0))
	if err != nil || got.Status != "succeeded" || got.CompletedAt.Unix() != 20 {
		t.Fatalf("completed receipt=%+v err=%v", got, err)
	}
	replay, err := l.Complete(context.Background(), "op-2", "failed", "late", time.Unix(30, 0))
	if err != nil || replay.Status != "succeeded" || replay.CompletedAt.Unix() != 20 {
		t.Fatalf("completion replay changed receipt=%+v err=%v", replay, err)
	}
}

func TestSQLLedgerConcurrentPutConverges(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "ledger.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE session_lifecycle_receipts (
		operation_id TEXT PRIMARY KEY, session_id TEXT NOT NULL, actor_kind TEXT NOT NULL,
		actor_id TEXT NOT NULL, command TEXT NOT NULL, from_state TEXT NOT NULL,
		to_state TEXT NOT NULL, reason_code TEXT NOT NULL, status TEXT NOT NULL,
		error_code TEXT NOT NULL, created_at TEXT NOT NULL, completed_at TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	l := NewSQLLedger(db)
	receipt := Receipt{OperationID: "concurrent-op", SessionID: "session-1", ActorKind: "operator", Command: "archive", FromState: StateLive, ToState: StateArchived, Status: "pending", CreatedAt: time.Unix(10, 0)}
	const callers = 16
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, putErr := l.Put(context.Background(), receipt)
			if putErr != nil {
				errs <- putErr
				return
			}
			if got.OperationID != receipt.OperationID || !got.CreatedAt.Equal(receipt.CreatedAt) {
				errs <- fmt.Errorf("concurrent receipt diverged: %+v", got)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for putErr := range errs {
		if putErr != nil {
			t.Fatal(putErr)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM session_lifecycle_receipts WHERE operation_id = ?`, receipt.OperationID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("receipt rows = %d, want 1", count)
	}
	completionErrors := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			status, code := "succeeded", ""
			if i%2 == 1 {
				status, code = "failed", "lifecycle_operation_failed"
			}
			got, completeErr := l.Complete(context.Background(), receipt.OperationID, status, code, time.Unix(int64(20+i), 0))
			if completeErr != nil {
				completionErrors <- completeErr
				return
			}
			if got.Status != "succeeded" && got.Status != "failed" {
				completionErrors <- fmt.Errorf("unexpected concurrent completion: %+v", got)
			}
		}(i)
	}
	wg.Wait()
	close(completionErrors)
	for completionErr := range completionErrors {
		if completionErr != nil {
			t.Fatal(completionErr)
		}
	}
	completed, err := l.Get(context.Background(), receipt.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "succeeded" && completed.Status != "failed" {
		t.Fatalf("final concurrent completion = %+v", completed)
	}
}
