// Package policy resolves domain policy values independently from the memory
// engine. The first deployment has one scope; keeping the scope explicit
// makes a future ledger a configuration/data addition instead of an engine
// rewrite.
package policy

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

type Scope string

const (
	AgentMemory       Scope = "agent-memory"
	FrontierTargetEnv       = "VROOLI_MEMORY_FRONTIER_TARGET"
	WakeBudgetEnv           = "VROOLI_MEMORY_WAKE_BUDGET"
)

type Config struct {
	FrontierTarget int
	WakeBudget     int
}

const (
	DefaultFrontierTarget = 16
	DefaultWakeBudget     = 40
)

// Resolve loads the policy values for the single active scope. Environment
// variables are inputs, not policy logic: every caller receives one validated
// result from this package.
func Resolve(lookupEnv func(string) (string, bool)) (Config, error) {
	c := Config{FrontierTarget: DefaultFrontierTarget, WakeBudget: DefaultWakeBudget}
	for _, setting := range []struct {
		name string
		set  func(int)
	}{
		{name: FrontierTargetEnv, set: func(v int) { c.FrontierTarget = v }},
		{name: WakeBudgetEnv, set: func(v int) { c.WakeBudget = v }},
	} {
		raw, ok := lookupEnv(setting.name)
		if !ok || strings.TrimSpace(raw) == "" {
			continue
		}
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return Config{}, fmt.Errorf("%s must be a positive integer, got %q", setting.name, raw)
		}
		setting.set(v)
	}
	return c, nil
}

type Facet struct {
	ID, Label, Guidance string
	Examples            []string
}

type Resolver struct {
	scope      Scope
	config     Config
	vocabulary func(context.Context) ([]Facet, error)
}

func NewResolver(scope Scope, config Config, vocabulary func(context.Context) ([]Facet, error)) *Resolver {
	if scope == "" {
		scope = AgentMemory
	}
	return &Resolver{scope: scope, config: config, vocabulary: vocabulary}
}

func (r *Resolver) Scope() Scope   { return r.scope }
func (r *Resolver) Config() Config { return r.config }
func (r *Resolver) Vocabulary(ctx context.Context) ([]Facet, error) {
	if r.vocabulary == nil {
		return nil, nil
	}
	return r.vocabulary(ctx)
}

// EnsureMigrations adds the scope seam to brownfield databases. The
// operations are additive and replay-safe; the journal remains append-only.
func EnsureMigrations(ctx context.Context, db *sql.DB) error {
	for _, table := range []string{"entries", "facet_definitions", "facet_policies", "summaries"} {
		exists, err := tableExists(ctx, db, table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if err := addScopeColumn(ctx, db, table); err != nil {
			return err
		}
	}
	if exists, err := tableExists(ctx, db, "entries"); err != nil {
		return err
	} else if exists {
		if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_entries_import_key`); err != nil {
			return fmt.Errorf("drop legacy import-key index: %w", err)
		}
		if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_entries_scope_import_key ON entries(scope, import_key) WHERE import_key IS NOT NULL`); err != nil {
			return fmt.Errorf("create scoped import-key index: %w", err)
		}
	}
	return nil
}

func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil {
		return false, fmt.Errorf("inspect %s table: %w", table, err)
	}
	return n != 0, nil
}

func addScopeColumn(ctx context.Context, db *sql.DB, table string) error {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name='scope'`, table).Scan(&n); err != nil {
		return fmt.Errorf("inspect %s.scope: %w", table, err)
	}
	if n == 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN scope TEXT NOT NULL DEFAULT 'agent-memory'`); err != nil {
			return fmt.Errorf("add %s.scope: %w", table, err)
		}
	}
	return nil
}
