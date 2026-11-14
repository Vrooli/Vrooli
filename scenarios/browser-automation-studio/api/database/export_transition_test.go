package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type transitionDriver struct{ result driver.Result }

func (d transitionDriver) Open(string) (driver.Conn, error) {
	return transitionConn{result: d.result}, nil
}

type transitionConn struct{ result driver.Result }

func (c transitionConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c transitionConn) Close() error              { return nil }
func (c transitionConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }
func (c transitionConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return c.result, nil
}

type transitionResult struct {
	count int64
	err   error
}

func (r transitionResult) LastInsertId() (int64, error) { return 0, errors.New("not applicable") }
func (r transitionResult) RowsAffected() (int64, error) { return r.count, r.err }

func TestExportTerminalTransitionsRequireConfirmedOneRow(t *testing.T) {
	unknown := errors.New("driver cannot confirm affected rows")
	for _, test := range []struct {
		name   string
		result transitionResult
		want   error
	}{
		{"unknown", transitionResult{err: unknown}, unknown},
		{"absent", transitionResult{}, ErrNotFound},
		{"multiple", transitionResult{count: 2}, errors.New("expected refusal")},
		{"one", transitionResult{count: 1}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			name := "export-transition-" + uuid.NewString()
			sql.Register(name, transitionDriver{result: test.result})
			db, err := sql.Open(name, "")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			repo := &repository{db: &DB{DB: sqlx.NewDb(db, "sqlite")}}
			for operation, err := range map[string]error{"complete": repo.UpdateExportComplete(t.Context(), uuid.New(), "film.mp4", 4, JSONMap{}), "failed": repo.UpdateExportStatus(t.Context(), uuid.New(), "failed", "render error")} {
				if test.want == nil {
					if err != nil {
						t.Fatalf("%s: %v", operation, err)
					}
					continue
				}
				if err == nil {
					t.Fatalf("%s accepted an unverified transition", operation)
				}
				if test.name != "multiple" && !errors.Is(err, test.want) {
					t.Fatalf("%s lost driver/missing error: %v", operation, err)
				}
			}
		})
	}
}
