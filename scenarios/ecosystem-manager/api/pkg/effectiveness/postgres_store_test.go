package effectiveness

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/dimensions"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupPostgres spins up a throwaway postgres with the effectiveness schema. It
// skips the test (rather than failing) when Docker is unavailable.
func setupPostgres(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	ctx := context.Background()

	pg, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Skipf("Could not start postgres container: %v (Docker may not be available)", err)
		return nil, nil
	}

	connStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pg.Terminate(ctx)
		t.Fatalf("connection string: %v", err)
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		_ = pg.Terminate(ctx)
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		_ = pg.Terminate(ctx)
		t.Fatalf("ping db: %v", err)
	}
	if err := CreateSchema(db); err != nil {
		db.Close()
		_ = pg.Terminate(ctx)
		t.Fatalf("create schema: %v", err)
	}
	return db, func() {
		db.Close()
		_ = pg.Terminate(ctx)
	}
}

func TestPostgresStoreRoundTrip(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	store := NewPostgresStore(db)

	if err := store.Record(CreditEvent{
		SkillID:               "lint-fix",
		TargetDimension:       standards,
		ClosedByDimension:     map[dimensions.Dimension]int{standards: 3},
		IntroducedByDimension: map[dimensions.Dimension]int{"tests": 1},
		Tokens:                1500,
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	got, ok, err := store.Get("lint-fix", standards)
	if err != nil || !ok {
		t.Fatalf("get standards: ok=%v err=%v", ok, err)
	}
	if got.ClosedCount != 3 || got.TotalRuns != 1 || got.TotalTokens != 1500 {
		t.Fatalf("unexpected target stat: %+v", got)
	}

	tdebt, ok, _ := store.Get("lint-fix", "tests")
	if !ok || tdebt.IntroducedCount != 1 || tdebt.TotalRuns != 0 {
		t.Fatalf("expected collateral tests debt with zero runs, got %+v (ok=%v)", tdebt, ok)
	}

	// No row for an unobserved pair (cold start).
	if _, ok, _ := store.Get("lint-fix", "security"); ok {
		t.Fatal("expected no row for unobserved (skill, dimension)")
	}
}

func TestPostgresStoreConcurrentUpsert(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()
	store := NewPostgresStore(db)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = store.Record(CreditEvent{
				SkillID:           "refactor",
				TargetDimension:   standards,
				ClosedByDimension: map[dimensions.Dimension]int{standards: 1},
				Tokens:            10,
			})
		}()
	}
	wg.Wait()

	got, ok, err := store.Get("refactor", standards)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.TotalRuns != n || got.ClosedCount != n || got.TotalTokens != n*10 {
		t.Fatalf("concurrent upserts lost increments: %+v", got)
	}
}
