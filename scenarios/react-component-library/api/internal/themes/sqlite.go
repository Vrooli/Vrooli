package themes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"react-component-library/internal/sqliterows"
)

type sqliteRepository struct {
	db Database
}

// Database is the narrow SQL surface needed by the theme repository. Both
// *sql.DB and api-core's RoutedDB implement it; request contexts select the
// pool when the routed implementation is used.
type Database interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func NewSQLiteRepository(db Database) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (s *sqliteRepository) UpsertBuiltin(ctx context.Context, t Theme) error {
	id := strings.TrimSpace(t.ID)
	if id == "" {
		return fmt.Errorf("theme id required")
	}
	tokensJSON, err := encodeTokens(t.Tokens)
	if err != nil {
		return fmt.Errorf("encode tokens: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO builtin_themes (id, name, tokens_json) VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, tokens_json=excluded.tokens_json`, id, t.Name, tokensJSON)
	if err != nil {
		return fmt.Errorf("upsert builtin theme %q: %w", id, err)
	}
	return nil
}

// ReplaceBuiltins reconciles the seed registry in place. It deliberately keeps
// the table and its history-bearing timestamps intact; only superseded seed
// rows are removed and canonical rows are upserted in one transaction.
func (s *sqliteRepository) ReplaceBuiltins(ctx context.Context, themes []Theme) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace builtins: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ids := make([]string, 0, len(themes))
	for _, theme := range themes {
		id := strings.TrimSpace(theme.ID)
		if id == "" {
			return fmt.Errorf("theme id required")
		}
		ids = append(ids, id)
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	if len(ids) > 0 {
		args := make([]any, len(ids))
		for i := range ids {
			args[i] = ids[i]
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM builtin_themes WHERE id NOT IN (`+placeholders+`)`, args...); err != nil {
			return fmt.Errorf("remove superseded builtins: %w", err)
		}
	}
	for _, theme := range themes {
		tokensJSON, err := encodeTokens(theme.Tokens)
		if err != nil {
			return fmt.Errorf("encode tokens: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO builtin_themes (id, name, tokens_json) VALUES (?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET name=excluded.name, tokens_json=excluded.tokens_json`, theme.ID, theme.Name, tokensJSON); err != nil {
			return fmt.Errorf("upsert builtin theme %q: %w", theme.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace builtins: %w", err)
	}
	return nil
}

func (s *sqliteRepository) GetBuiltin(ctx context.Context, id string) (Theme, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Theme{}, fmt.Errorf("theme id required")
	}
	row := s.db.QueryRowContext(ctx, `SELECT id, name, tokens_json FROM builtin_themes WHERE id = ?`, id)
	var gotID, name, tokensJSON string
	if err := row.Scan(&gotID, &name, &tokensJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Theme{}, ErrThemeNotFound{ID: id}
		}
		return Theme{}, fmt.Errorf("scan theme: %w", err)
	}
	tokens, err := decodeTokens(tokensJSON)
	if err != nil {
		return Theme{}, fmt.Errorf("decode tokens: %w", err)
	}
	return Theme{ID: gotID, Name: name, Tokens: tokens, Source: "builtin"}, nil
}

func (s *sqliteRepository) ListBuiltins(ctx context.Context) ([]Theme, error) {
	out, err := sqliterows.Collect(ctx, s.db, `SELECT id, name, tokens_json FROM builtin_themes ORDER BY id`, nil, false, func(row sqliterows.Scanner) (Theme, error) {
		var id, name, tokensJSON string
		if err := row.Scan(&id, &name, &tokensJSON); err != nil {
			return Theme{}, fmt.Errorf("scan: %w", err)
		}
		tokens, err := decodeTokens(tokensJSON)
		if err != nil {
			return Theme{}, fmt.Errorf("decode tokens %s: %w", id, err)
		}
		return Theme{ID: id, Name: name, Tokens: tokens, Source: "builtin"}, nil
	}, sqliterows.ErrorContext{Query: "query builtins"})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *sqliteRepository) CountBuiltins(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM builtin_themes`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count builtins: %w", err)
	}
	return n, nil
}

func encodeTokens(in map[string]string) (string, error) {
	if in == nil {
		in = map[string]string{}
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(in))
	for _, k := range keys {
		ordered[k] = in[k]
	}
	b, err := json.Marshal(ordered)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeTokens(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}, nil
	}
	out := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}
