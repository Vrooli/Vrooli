//go:build rehearsal

package storagehealth

// The rehearsal drives the real compaction against a COPY of the production
// database. It never runs in the normal suite:
//
//	AM_STORAGE_REHEARSAL_DB=/path/to/copy.db \
//	  go test -tags rehearsal -run TestRehearsal -v -timeout 2h ./internal/storagehealth/
//
// The copy is converted in place and its conversation-search staging rows are
// deleted to measure online reclamation. Never point it at the live file.

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	corestorage "github.com/vrooli/api-core/storage"
)

func peakRSS() string {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmHWM:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "VmHWM:"))
		}
	}
	return "unknown"
}

func logJSON(t *testing.T, label string, value any) {
	body, _ := json.Marshal(value)
	t.Logf("%s %s", label, body)
}

// TestRehearsalDeclaredBudget times the owner-inventory read against the real
// repository; the reconciler repeats it at most hourly.
func TestRehearsalDeclaredBudget(t *testing.T) {
	root := os.Getenv("AM_STORAGE_REHEARSAL_REPO")
	if root == "" {
		t.Skip("set AM_STORAGE_REHEARSAL_REPO to the repository root")
	}
	start := time.Now()
	budget, err := LoadDeclaredBudget(root, "agent-manager", "data")
	t.Logf("declared budget %+v err=%v in %s", budget, err, time.Since(start))
	if err != nil {
		t.Fatal(err)
	}
}

// TestRehearsalMaintainTickIsBounded frees pages on an already-converted copy
// and times reconciler passes: each must stay near TickBudget because the
// adaptive batch keeps every writer hold near BatchHoldTarget.
func TestRehearsalMaintainTickIsBounded(t *testing.T) {
	path := os.Getenv("AM_STORAGE_REHEARSAL_DB")
	if path == "" || strings.Contains(path, "/scenarios/agent-manager/data/") {
		t.Skip("set AM_STORAGE_REHEARSAL_DB to a converted copy; the live file is refused")
	}
	dsn, err := corestorage.SQLiteDSNAt(path, corestorage.SQLiteTuning{PageSizeBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn+"&_pragma=journal_size_limit(67108864)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	svc, err := New(Options{DB: func() *sql.DB { return db }, Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM invocation_read_model_facts"); err != nil {
		t.Fatal(err)
	}
	freed, _ := svc.Stats(ctx)
	logJSON(t, "freed", freed)
	for pass := 1; pass <= 3; pass++ {
		before, _ := svc.Stats(ctx)
		start := time.Now()
		if err := svc.Maintain(ctx); err != nil {
			t.Fatal(err)
		}
		elapsed := time.Since(start)
		after, _ := svc.Stats(ctx)
		t.Logf("pass %d returned %d bytes in %s", pass, before.FreeBytes-after.FreeBytes, elapsed)
		if elapsed > svc.opts.Policy.TickBudget+2*time.Second {
			t.Fatalf("pass %d took %s against a %s budget", pass, elapsed, svc.opts.Policy.TickBudget)
		}
	}
}

func TestRehearsal(t *testing.T) {
	path := os.Getenv("AM_STORAGE_REHEARSAL_DB")
	if path == "" || strings.Contains(path, "/scenarios/agent-manager/data/") {
		t.Skip("set AM_STORAGE_REHEARSAL_DB to a copy of the database; the live file is refused")
	}
	dsn, err := corestorage.SQLiteDSNAt(path, corestorage.SQLiteTuning{PageSizeBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn+"&_pragma=journal_size_limit(67108864)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(5)
	ctx := context.Background()
	svc, err := New(Options{
		DB: func() *sql.DB { return db }, Path: path,
		Budget: func() Budget { return Budget{Bytes: 12 << 30, Source: "rehearsal"} },
		Fence: func(context.Context) (FenceState, error) {
			return FenceState{Closed: true, Drained: true, Revision: 1}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := svc.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logJSON(t, "before", before)

	receipt, err := svc.Compact(ctx, "rehearsal", "rehearsal on a copy")
	logJSON(t, "compaction", receipt)
	t.Logf("peak RSS after compaction: %s", peakRSS())
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	result, err := db.ExecContext(ctx, "DELETE FROM conversation_search_generation_documents")
	if err != nil {
		t.Fatal(err)
	}
	deleted, _ := result.RowsAffected()
	t.Logf("deleted %d staging rows in %s", deleted, time.Since(start))
	freed, err := svc.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logJSON(t, "after delete", freed)

	start = time.Now()
	svc.mu.Lock()
	svc.lastMaintain = time.Time{}
	svc.mu.Unlock()
	if err := svc.Maintain(ctx); err != nil {
		t.Fatal(err)
	}
	tick, _ := svc.Stats(ctx)
	t.Logf("one reconciler tick (budget %s) returned %d bytes in %s", svc.opts.Policy.TickBudget, freed.FreeBytes-tick.FreeBytes, time.Since(start))

	svc.opts.Policy.ReclaimBudget = time.Hour
	start = time.Now()
	reclaim, err := svc.Reclaim(ctx, false)
	logJSON(t, "reclaim", reclaim)
	t.Logf("explicit reclaim took %s", time.Since(start))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		t.Fatal(err)
	}
	final, _ := svc.Stats(ctx)
	logJSON(t, "final", final)
	if final.AutoVacuum != "incremental" || final.FreelistCount != 0 {
		t.Fatalf("rehearsal did not end converted and reclaimed: %+v", final)
	}
}
