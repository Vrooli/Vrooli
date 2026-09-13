package maintenance

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInventoryCountsFalseFailedPhysicalExecutorAndPreservesRows(t *testing.T) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('usi','failed',315576,315576,'none'),('pending','pending',0,0,'none'),('parked','parked',0,0,'none'),('done','complete',0,0,'none');
INSERT INTO workflow_executions VALUES('workflow','running'),('history','succeeded');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		return ExecutorEvidence{ExecutorRef: ref, Alive: true, HasChildren: true}, nil
	})
	before, err := observer.Observe(t.Context())
	if err != nil || before.Remaining == nil || *before.Remaining != 4 || len(before.Executors) != 1 || before.Executors[0].Status != "failed" || !before.Executors[0].HasChildren || before.ControlPlaneGap == "" {
		t.Fatalf("false terminal work disappeared: %+v %v", before, err)
	}
	_, _ = gate.Enter(t.Context(), "owner", "rollout")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	standing, err := gate.Wait(ctx, observer.Remaining, 0)
	if err == nil || standing.Drained {
		t.Fatalf("cancelled drain=%+v %v", standing, err)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM runs WHERE id='usi'`).Scan(&status); err != nil || status != "failed" {
		t.Fatalf("inventory changed accounting: %q %v", status, err)
	}
}

func TestInventoryUnknownPhysicalEvidenceCannotDrain(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('usi','failed',315576,315576,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	observer := NewInventory(db, func(context.Context, ExecutorRef) (ExecutorEvidence, error) {
		return ExecutorEvidence{}, errors.New("executor scope unavailable")
	})
	state, err := observer.Observe(t.Context())
	if err == nil || state.Remaining != nil || len(state.Unknown) == 0 {
		t.Fatalf("unknown became empty: %+v %v", state, err)
	}
}

func TestInventoryCompletedAndRestingReviewHistoryAllowsDrain(t *testing.T) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<6000)
INSERT INTO runs SELECT 'run-'||x,CASE WHEN x%2=0 THEN 'complete' ELSE 'needs_review' END,0,0,'none' FROM n;
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	observer := NewInventory(db, func(context.Context, ExecutorRef) (ExecutorEvidence, error) {
		calls++
		return ExecutorEvidence{}, errors.New("no executor should need observation")
	})
	state, err := observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 0 || calls != 0 {
		t.Fatalf("resting history blocks drain: %+v calls=%d err=%v", state, calls, err)
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	standing, err := gate.Wait(ctx, observer.Remaining, time.Second)
	if err != nil || !standing.Drained {
		t.Fatalf("resting history did not drain: %+v %v", standing, err)
	}
	if _, err := gate.Admit(t.Context()); !errors.Is(err, ErrClosed) {
		t.Fatalf("future admission opened after drain: %v", err)
	}
	var retained int
	if err := db.QueryRow(`SELECT COUNT(*) FROM runs`).Scan(&retained); err != nil || retained != 6000 {
		t.Fatalf("history altered: count=%d err=%v", retained, err)
	}
}

func TestInventoryRetainedPIDsRequireOwnerProofAndDoNotTruncateLiveTail(t *testing.T) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<2500)
INSERT INTO runs SELECT 'old-'||x,'complete',x,x,'none' FROM n;
INSERT INTO runs VALUES('zz-review','needs_review',9001,9001,'none'),('zz-usi','failed',315576,315576,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	liveUSI := true
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		calls++
		// This is an injected complete control-plane exclusion proof, not a
		// missing-root heuristic. USI stays alive beyond the first result page.
		return ExecutorEvidence{ExecutorRef: ref, Alive: liveUSI && ref.ID == "zz-usi", HasChildren: liveUSI && ref.ID == "zz-usi"}, nil
	})
	state, err := observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 1 || calls != 2502 || len(state.Executors) != 1 || state.Executors[0].PID != 315576 {
		t.Fatalf("physical tail lost or history blocked: %+v calls=%d err=%v", state, calls, err)
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	standing, err := gate.Wait(ctx, func(ctx context.Context) (int, error) {
		count, err := observer.Remaining(ctx)
		cancel()
		return count, err
	}, time.Second)
	if err == nil || standing.Drained {
		t.Fatalf("live false-failed executor allowed drain: %+v %v", standing, err)
	}
	// Only a subsequent complete owner proof permits the retained PIDs to drain.
	liveUSI = false
	ctx, done := context.WithTimeout(t.Context(), 5*time.Second)
	defer done()
	standing, err = gate.Wait(ctx, observer.Remaining, time.Second)
	if err != nil || !standing.Drained {
		t.Fatalf("excluded historic executor scopes block drain: %+v %v", standing, err)
	}
}

func TestInventoryRestingReviewStillCountsExecutorOrFinalization(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('live-review','needs_review',123,123,'none'),('pending-finalization','needs_review',0,0,'pending'),('running-finalization','needs_review',0,0,'running');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		return ExecutorEvidence{ExecutorRef: ref, Alive: true}, nil
	})
	state, err := observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 3 {
		t.Fatalf("review executor/finalization excluded: %+v %v", state, err)
	}
}

func TestInventoryRetainedReviewExecutorRequiresCompleteProof(t *testing.T) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('review','needs_review',123,123,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		return ExecutorEvidence{ExecutorRef: ref, Alive: false}, errors.New("root absent; descendant scope not excluded")
	})
	state, err := observer.Observe(t.Context())
	if err == nil || state.Remaining != nil || len(state.Unknown) != 1 {
		t.Fatalf("resting review relaxed physical exclusion: %+v %v", state, err)
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	standing, err := gate.Wait(t.Context(), observer.Remaining, time.Second)
	if err == nil || standing.Drained {
		t.Fatalf("unknown review executor scope drained: %+v %v", standing, err)
	}
}

func TestInventoryInterruptedProofDoesNotReportEmpty(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
INSERT INTO runs VALUES('history','complete',123,123,'none');
ALTER TABLE runs ADD COLUMN execution_mode TEXT DEFAULT 'codec_pipe';`)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	observer := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		cancel()
		return ExecutorEvidence{ExecutorRef: ref}, nil
	})
	state, err := observer.Observe(ctx)
	if !errors.Is(err, context.Canceled) || state.Remaining != nil {
		t.Fatalf("interrupted exclusion became empty: %+v %v", state, err)
	}
}

func TestInventoryImportedHistoryIsNotOwnedExecution(t *testing.T) {
	for _, batched := range []bool{false, true} {
		t.Run(fmt.Sprintf("batched=%v", batched), func(t *testing.T) {
			testImportedHistoryIsNotOwnedExecution(t, batched)
		})
	}
}

func testImportedHistoryIsNotOwnedExecution(t *testing.T, batched bool) {
	gate, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, execution_mode TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT, status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<1000)
INSERT INTO runs SELECT 'imported-'||x,CASE WHEN x%2=0 THEN 'unknown' ELSE 'running' END,'imported',900000+x,900000+x,'none' FROM n;`)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	physical := func(ctx context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		calls++
		if ref.ID != "managed-usi" {
			return ExecutorEvidence{}, fmt.Errorf("foreign imported executor entered owner scope: %s", ref.ID)
		}
		return ExecutorEvidence{ExecutorRef: ref, Alive: true, HasChildren: true, PIDs: []int{315576, 567605}}, nil
	}
	observer := NewInventory(db, physical)
	if batched {
		observer = NewBatchedInventory(db, func(ctx context.Context, refs []ExecutorRef) ([]ExecutorEvidence, error) {
			if len(refs) != 1 {
				return nil, fmt.Errorf("imported history consumed root batch capacity: %d references", len(refs))
			}
			evidence, err := physical(ctx, refs[0])
			return []ExecutorEvidence{evidence}, err
		})
	}
	if _, err := gate.Enter(t.Context(), "owner", "rollout"); err != nil {
		t.Fatal(err)
	}
	state, err := observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 0 || len(state.Work) != 0 || calls != 0 {
		t.Fatalf("read-only history blocks drain: %+v %v", state, err)
	}
	standing, err := gate.Wait(t.Context(), observer.Remaining, time.Second)
	if err != nil || !standing.Drained {
		t.Fatalf("history blocked drain: %+v %v", standing, err)
	}
	// The mode exclusion is not a general unknown-status exemption. Legacy and
	// managed unknown runs and managed terminal executor residue stay visible.
	_, err = db.Exec(`INSERT INTO runs VALUES('managed','unknown','codec_pipe',0,0,'none'),('legacy','unknown',NULL,0,0,'none'),('managed-usi','failed','codec_pipe',315576,315576,'skipped');`)
	if err != nil {
		t.Fatal(err)
	}
	state, err = observer.Observe(t.Context())
	if err != nil || state.Remaining == nil || *state.Remaining != 3 || len(state.Executors) != 1 || calls != 1 || state.Executors[0].PID != 315576 {
		t.Fatalf("owned unknown or managed terminal residue erased: %+v calls=%d err=%v", state, calls, err)
	}
	// Imported active finalization contradicts the read-only ownership contract.
	// Keep that contradiction explicit without inspecting a foreign historical PID.
	_, err = db.Exec(`INSERT INTO runs VALUES('finalization-import-pending','unknown','imported',999998,999998,'pending'),('finalization-import-running','unknown','imported',999999,999999,'running');`)
	if err != nil {
		t.Fatal(err)
	}
	state, err = observer.Observe(t.Context())
	if err == nil || state.Remaining != nil || len(state.Unknown) != 2 || len(state.Executors) != 1 || calls != 2 {
		t.Fatalf("contradictory finalization became exclusion or foreign proof: %+v calls=%d err=%v", state, calls, err)
	}
	for _, status := range []string{"pending", "running"} {
		if !strings.Contains(strings.Join(state.Unknown, "\n"), "finalization-import-"+status+": imported read-only run claims active finalization") {
			t.Fatalf("missing explicit ownership contradiction: %+v", state)
		}
	}
	var retained int
	if err := db.QueryRow(`SELECT COUNT(*) FROM runs WHERE runner_pid > 0`).Scan(&retained); err != nil || retained != 1003 {
		t.Fatalf("history modified: count=%d err=%v", retained, err)
	}
}

func TestInventoryOnePhysicalBatchAcrossHistory(t *testing.T) {
	for _, references := range []int{2, 129, 10811, 16385} {
		t.Run(fmt.Sprintf("references=%d", references), func(t *testing.T) {
			_, db := openGate(t, filepath.Join(t.TempDir(), "inventory.db"))
			db.SetMaxOpenConns(1)
			_, err := db.Exec(`CREATE TABLE runs(id TEXT, status TEXT, execution_mode TEXT, runner_pid INTEGER, runner_pgid INTEGER, finalization_status TEXT);
CREATE TABLE workflow_executions(id TEXT,status TEXT);
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<300)
INSERT INTO runs SELECT 'active-'||x,'unknown','codec_pipe',0,0,'none' FROM n;`)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<?)
INSERT INTO runs SELECT 'executor-'||x,'failed','codec_pipe',x,x,'none' FROM n`, references); err != nil {
				t.Fatal(err)
			}
			calls := 0
			reader := NewBatchedInventory(db, func(ctx context.Context, refs []ExecutorRef) ([]ExecutorEvidence, error) {
				calls++
				var retained int
				if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runs`).Scan(&retained); err != nil || retained != references+300 {
					t.Fatalf("SQL cursor was not closed before observer hydration: count=%d err=%v", retained, err)
				}
				if len(refs) != references {
					t.Fatalf("SQL pages fragmented root inventory: %d refs", len(refs))
				}
				var out []ExecutorEvidence
				for _, ref := range refs {
					out = append(out, ExecutorEvidence{ExecutorRef: ref})
				}
				return out, nil
			})
			state, err := reader.Observe(t.Context())
			if references <= 16384 {
				if err != nil || calls != 1 || state.Remaining == nil || *state.Remaining != 300 {
					t.Fatalf("single read lost logical work: calls=%d state=%+v err=%v", calls, state, err)
				}
			} else if err == nil || calls != 0 || state.Remaining != nil || !strings.Contains(err.Error(), "16385 physical references") {
				t.Fatalf("capacity became repeated scans or empty: calls=%d state=%+v err=%v", calls, state, err)
			}
		})
	}
}
