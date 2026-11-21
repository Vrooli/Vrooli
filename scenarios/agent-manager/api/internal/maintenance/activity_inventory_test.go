package maintenance

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestActivityUsesPhysicalScopesRatherThanLogicalStates(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "activity.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT,status TEXT,runner_pid INTEGER,runner_pgid INTEGER,finalization_status TEXT,execution_mode TEXT);
 CREATE TABLE workflow_executions(id TEXT,status TEXT);
 INSERT INTO runs VALUES('logical-running','running',0,0,'none','codec_pipe'),('terminal-live','failed',77,77,'none','codec_pipe'),('queued','pending',0,0,'none','codec_pipe'),('imported','running',99,99,'none','imported');`)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	terminalPresent := true
	r := NewInventory(db, func(_ context.Context, ref ExecutorRef) (ExecutorEvidence, error) {
		seen[ref.ID]++
		return ExecutorEvidence{ExecutorRef: ref, Alive: terminalPresent && ref.ID == "terminal-live"}, nil
	})
	at := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	r.now = func() time.Time { return at }
	activity, err := r.ObserveActivity(t.Context())
	if err == nil || activity.PhysicalComplete || activity.ObservedExecutors != 1 || activity.LogicalQueuedRuns != 1 || !activity.ObservedAt.Equal(at) || len(activity.Unknown) == 0 {
		t.Fatalf("partial positive activity %+v err=%v", activity, err)
	}
	if len(seen) != 1 || seen["terminal-live"] != 1 {
		t.Fatalf("ordinary activity expanded canonical physical references: %+v", seen)
	}
	if activity.Remaining != nil {
		t.Fatalf("missing physical coverage manufactured a total: %+v", activity)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM runs WHERE id='logical-running'`).Scan(&status); err != nil || status != "running" {
		t.Fatalf("activity mutated roster: %s %v", status, err)
	}
	// Once the existing owner has recorded references, its negative physical
	// proof can establish zero despite a logical running/queued roster.
	if _, err := db.Exec(`UPDATE runs SET runner_pid=31,runner_pgid=31 WHERE id='logical-running';UPDATE runs SET runner_pid=32,runner_pgid=32 WHERE id='queued'`); err != nil {
		t.Fatal(err)
	}
	terminalPresent = false
	activity, err = r.ObserveActivity(t.Context())
	if err != nil || !activity.PhysicalComplete || activity.ObservedExecutors != 0 {
		t.Fatalf("complete physical coverage: %+v %v", activity, err)
	}
	if _, err := db.Exec(`UPDATE runs SET runner_pid=0,runner_pgid=0 WHERE id IN ('logical-running','queued')`); err != nil {
		t.Fatal(err)
	}
	terminalPresent = true
	clear(seen)
	legacy, err := r.Observe(t.Context())
	if err != nil || legacy.Remaining == nil || *legacy.Remaining != 3 || len(seen) != 1 || seen["terminal-live"] != 1 {
		t.Fatalf("ordinary activity changed existing drain observation: %+v seen=%v err=%v", legacy, seen, err)
	}
}

func TestActivityTotalIsNotTheDisplaySampleAndPartialZeroIsUnknown(t *testing.T) {
	_, db := openGate(t, filepath.Join(t.TempDir(), "activity.db"))
	_, err := db.Exec(`CREATE TABLE runs(id TEXT,status TEXT,runner_pid INTEGER,runner_pgid INTEGER,finalization_status TEXT,execution_mode TEXT);
 CREATE TABLE workflow_executions(id TEXT,status TEXT);`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 100; i++ {
		if _, err := db.Exec(`INSERT INTO runs VALUES(?,'complete',?,?,'none','codec_pipe')`, fmt.Sprint(i), i, i); err != nil {
			t.Fatal(err)
		}
	}
	partial := false
	r := NewBatchedInventory(db, func(_ context.Context, refs []ExecutorRef) ([]ExecutorEvidence, error) {
		evidence := make([]ExecutorEvidence, 0, len(refs))
		for _, ref := range refs {
			evidence = append(evidence, ExecutorEvidence{ExecutorRef: ref, Alive: !partial})
		}
		if partial {
			return evidence, errors.New("attached scope exclusion unknown")
		}
		return evidence, nil
	})
	complete, err := r.ObserveActivity(t.Context())
	if err != nil || !complete.PhysicalComplete || complete.ObservedExecutors != 100 || len(complete.Executors) != 64 {
		t.Fatalf("display sample became activity total: %+v %v", complete, err)
	}
	partial = true
	unknown, err := r.ObserveActivity(t.Context())
	if err == nil || unknown.PhysicalComplete || unknown.ObservedExecutors != 0 || len(unknown.Unknown) == 0 {
		t.Fatalf("zero observed was accepted as zero activity: %+v %v", unknown, err)
	}
}
