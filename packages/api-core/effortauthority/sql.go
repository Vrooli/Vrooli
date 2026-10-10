package effortauthority

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"github.com/vrooli/api-core/database"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

// SQLStore currently supports SQLite only. Setup must select this qualified
// engine; this is not a claim of PostgreSQL or HA support.
type SQLStore struct{ DB *sql.DB }

func (s *SQLStore) Ensure(ctx context.Context) error {
	return database.EnsureSchemas(ctx, s.DB, database.SchemaProviderFunc(Schema))
}
func (s *SQLStore) Get(ctx context.Context, id string) (Record, error) {
	var data string
	var r Record
	e := s.DB.QueryRowContext(ctx, "SELECT record FROM finite_effort_authority WHERE id = ?", id).Scan(&data)
	if e != nil {
		return r, e
	}
	e = json.Unmarshal([]byte(data), &r)
	return r, e
}
func (s *SQLStore) Create(ctx context.Context, r Record) error {
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	_, e = s.DB.ExecContext(ctx, "INSERT INTO finite_effort_authority (id,record,version) VALUES (?, ?, 1)", r.Policy.ID, string(b))
	return e
}

// Optimistic version CAS is safe across processes/restarts. Contention refuses;
// it never retries a callback with effects or overwrites a concurrent revocation.
func (s *SQLStore) Update(ctx context.Context, id string, fn func(*Record) error) error {
	var data string
	var version int64
	e := s.DB.QueryRowContext(ctx, "SELECT record,version FROM finite_effort_authority WHERE id = ?", id).Scan(&data, &version)
	if e != nil {
		return e
	}
	var r Record
	if e = json.Unmarshal([]byte(data), &r); e != nil {
		return e
	}
	if e = fn(&r); e != nil {
		return e
	}
	b, e := json.Marshal(r)
	if e != nil {
		return e
	}
	result, e := s.DB.ExecContext(ctx, "UPDATE finite_effort_authority SET record = ?, version = version + 1 WHERE id = ? AND version = ?", string(b), id, version)
	if e != nil {
		return e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return ErrRefused
	}
	return nil
}
