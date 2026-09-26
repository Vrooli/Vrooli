package validation

import (
	"context"
	"testing"
)

func TestHygieneSQLitePoolDeadlock(t *testing.T) {
	a := hygieneSQLitePoolDeadlock{}

	t.Run("positive_pool1_nested_query", func(t *testing.T) {
		// Pool of 1 AND a nested db.Query inside an open `for rows.Next()` loop:
		// the exact deadlock shape the project memory records.
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"internal/orders/repo.go": `package orders

import (
	"context"
	"database/sql"
)

type Repo struct{ db *sql.DB }

func (r *Repo) Setup() *sql.DB {
	db := r.db
	cfg := struct{ MaxOpenConns int }{MaxOpenConns: 1}
	_ = cfg
	return db
}

func (r *Repo) Walk(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "SELECT id FROM orders")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		// nested query while the outer rows cursor still holds the only conn
		_, _ = r.db.ExecContext(ctx, "UPDATE orders SET seen = 1 WHERE id = 2")
	}
	return nil
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) == 0 {
			t.Fatalf("expected SQLITE_POOL_DEADLOCK finding, got none")
		}
		if got[0].Code != "SQLITE_POOL_DEADLOCK" || got[0].Severity != SeverityError {
			t.Fatalf("unexpected finding %+v", got[0])
		}
	})

	t.Run("positive_setmaxopenconns_method", func(t *testing.T) {
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"internal/orders/repo.go": `package orders

import (
	"context"
	"database/sql"
)

type Repo struct{ db *sql.DB }

func (r *Repo) Init() { r.db.SetMaxOpenConns(1) }

func (r *Repo) Walk(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "SELECT id FROM orders")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		_, _ = r.db.QueryContext(ctx, "SELECT 1")
	}
	return nil
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) == 0 {
			t.Fatalf("expected finding for SetMaxOpenConns(1) + nested query, got none")
		}
	})

	t.Run("negative_pool1_no_nested_query", func(t *testing.T) {
		// MaxOpenConns: 1 ALONE is the correct SQLite config — NOT a finding.
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"internal/orders/repo.go": `package orders

import (
	"context"
	"database/sql"
)

type Repo struct{ db *sql.DB }

func (r *Repo) Setup() {
	cfg := struct{ MaxOpenConns int }{MaxOpenConns: 1}
	_ = cfg
}

func (r *Repo) Walk(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "SELECT id FROM orders")
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		ids = append(ids, id) // collect first, no nested query
	}
	return nil
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("MaxOpenConns:1 alone must NOT be flagged, got %+v", got)
		}
	})

	t.Run("negative_nested_query_no_pool1", func(t *testing.T) {
		// A nested query without the pool-of-1 signal is at most a perf smell,
		// not a deadlock — not this analyzer's finding.
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"internal/orders/repo.go": `package orders

import (
	"context"
	"database/sql"
)

type Repo struct{ db *sql.DB }

func (r *Repo) Walk(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, "SELECT id FROM orders")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		_ = rows.Scan(&id)
		_, _ = r.db.ExecContext(ctx, "UPDATE orders SET seen = 1")
	}
	return nil
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("nested query without MaxOpenConns:1 must NOT be flagged, got %+v", got)
		}
	})

	t.Run("exempt_test_file", func(t *testing.T) {
		ac := hygieneACForTempScenario(t, "demo", map[string]string{
			"internal/orders/repo_test.go": `package orders

import (
	"context"
	"database/sql"
	"testing"
)

func TestWalk(t *testing.T) {
	var db *sql.DB
	cfg := struct{ MaxOpenConns int }{MaxOpenConns: 1}
	_ = cfg
	rows, _ := db.QueryContext(context.Background(), "SELECT id FROM orders")
	for rows.Next() {
		_, _ = db.ExecContext(context.Background(), "UPDATE orders SET seen = 1")
	}
}
`,
		})
		got, err := a.Analyze(context.Background(), ac)
		if err != nil {
			t.Fatalf("analyze: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no finding (test file exempt), got %+v", got)
		}
	})
}
