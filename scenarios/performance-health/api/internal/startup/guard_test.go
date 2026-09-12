package startup

import (
	"context"
	"database/sql"
	"testing"
)

func openSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/startup.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return db
}

// [REQ:PH-STARTUP-002] Benchmarking performance-health itself is rejected
// (restarting the process answering the request would deadlock).
func TestSelfBenchmarkRejected(t *testing.T) {
	svc := NewService(fakeRunner{}, newTestStore(t), "performance-health")
	if _, err := svc.Benchmark(context.Background(), "performance-health", 0); err == nil {
		t.Fatal("expected self-benchmark to be rejected")
	}
}

func TestBenchmarkRequiresScenario(t *testing.T) {
	svc := NewService(fakeRunner{}, newTestStore(t), "performance-health")
	if _, err := svc.Benchmark(context.Background(), "", 0); err == nil {
		t.Fatal("expected error for empty scenario")
	}
}
