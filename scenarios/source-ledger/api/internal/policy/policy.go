// Package policy resolves domain policy values independently from the memory
// engine. The first deployment has one scope; keeping the scope explicit
// makes a future ledger a configuration/data addition instead of an engine
// rewrite.
package policy

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Scope string

const AgentMemory Scope = "agent-memory"

const (
	FrontierTargetEnv = "VROOLI_MEMORY_FRONTIER_TARGET"
	WakeBudgetEnv     = "VROOLI_MEMORY_WAKE_BUDGET"
	MaxEntryLinesEnv  = "VROOLI_MEMORY_MAX_ENTRY_LINES"
)

type Config struct {
	Scope          Scope
	FrontierTarget int
	WakeBudget     int
	MaxEntryLines  int
	FacetBudgets   map[string]int
}

const (
	DefaultFrontierTarget = 16
	// DefaultWakeBudget bounds the whole ambient view in lines. It must be able
	// to hold every facet's resident budget at DefaultMaxEntryLines apiece, or
	// the declared residencies are unsatisfiable and the last facets in order
	// silently never appear.
	DefaultWakeBudget = 96
	// DefaultMaxEntryLines bounds one memory's contribution to the ambient view.
	// Wake is an index into memory, not a copy of it: the journal keeps every
	// line and recall returns them.
	DefaultMaxEntryLines = 2
)

// DefaultSummaryInstruction is the agent-memory scope's opening line for the
// summarization prompt. It lives here rather than in the compaction engine
// because it names what this scope stores; a second scope replaces it without
// touching the engine.
const DefaultSummaryInstruction = "Summarize these episode memories for future agent context."

// ExpireOnResolution is the retention policy whose members leave the ambient
// frontier once marked resolved. The engine knows that some facets expire on
// resolution; only this package knows what that policy is called.
const ExpireOnResolution = "expire-on-resolution"

// Resolve loads the policy values for the single active scope. Environment
// variables are inputs, not policy logic: every caller receives one validated
// result from this package.
func Resolve(lookupEnv func(string) (string, bool)) (Config, error) {
	c := Config{Scope: AgentMemory, FrontierTarget: DefaultFrontierTarget, WakeBudget: DefaultWakeBudget, MaxEntryLines: DefaultMaxEntryLines}
	for _, setting := range []struct {
		name string
		set  func(int)
	}{
		{name: FrontierTargetEnv, set: func(v int) { c.FrontierTarget = v }},
		{name: WakeBudgetEnv, set: func(v int) { c.WakeBudget = v }},
		{name: MaxEntryLinesEnv, set: func(v int) { c.MaxEntryLines = v }},
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

// ScopeDefinition is the durable policy contract for one ledger. Facets are
// included so creation is one atomic operation from the operator's point of
// view: the scope cannot exist without the vocabulary its classifier needs.
type ScopeDefinition struct {
	ID, Label string
	Config    Config
	Facets    []FacetDefinition
}

type FacetDefinition struct {
	ID, Label, Guidance, RetentionPolicy string
	CompactionEligible                   bool
	ResidentBudget                       int
}

type Registry struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[Scope]Config
}

func NewRegistry(db *sql.DB) *Registry {
	return &Registry{db: db, cache: make(map[Scope]Config)}
}

// Ensure creates the registry and reconciles the first deployment's
// environment-backed agent-memory policy. Other scopes are always database
// records and are never affected by those environment variables.
func (r *Registry) Ensure(ctx context.Context, defaults Config) error {
	if defaults.Scope == "" {
		defaults.Scope = AgentMemory
	}
	if _, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS scopes (
        id TEXT PRIMARY KEY,
        label TEXT NOT NULL,
        frontier_target INTEGER NOT NULL,
        wake_budget INTEGER NOT NULL,
        max_entry_lines INTEGER NOT NULL,
        created_at TEXT NOT NULL,
        updated_at TEXT NOT NULL
      )`); err != nil {
		return fmt.Errorf("create scopes table: %w", err)
	}
	now := "now"
	if _, err := r.db.ExecContext(ctx, `INSERT INTO scopes(id,label,frontier_target,wake_budget,max_entry_lines,created_at,updated_at) VALUES(?,?,?,?,?,?,?)
      ON CONFLICT(id) DO UPDATE SET label=excluded.label,frontier_target=excluded.frontier_target,wake_budget=excluded.wake_budget,max_entry_lines=excluded.max_entry_lines,updated_at=excluded.updated_at`,
		string(AgentMemory), "Agent memory", defaults.FrontierTarget, defaults.WakeBudget, defaults.MaxEntryLines, now, now); err != nil {
		return fmt.Errorf("ensure agent-memory scope: %w", err)
	}
	r.Invalidate(AgentMemory)
	return nil
}

func (r *Registry) Resolve(ctx context.Context, raw string) (Config, error) {
	scope := NormalizeScope(raw)
	r.mu.RLock()
	config, ok := r.cache[scope]
	r.mu.RUnlock()
	if ok {
		return config, nil
	}
	var c Config
	err := r.db.QueryRowContext(ctx, `SELECT frontier_target,wake_budget,max_entry_lines FROM scopes WHERE id=?`, string(scope)).Scan(&c.FrontierTarget, &c.WakeBudget, &c.MaxEntryLines)
	if err != nil {
		if err == sql.ErrNoRows {
			return Config{}, fmt.Errorf("scope %q is not registered", scope)
		}
		return Config{}, fmt.Errorf("resolve scope %q: %w", scope, err)
	}
	c.Scope = scope
	c.FacetBudgets = make(map[string]int)
	if exists, err := tableExists(ctx, r.db, "facet_policies"); err != nil {
		return Config{}, err
	} else if exists {
		rows, err := r.db.QueryContext(ctx, `SELECT facet_id,resident_budget FROM facet_policies WHERE scope=?`, string(scope))
		if err != nil {
			return Config{}, fmt.Errorf("resolve scope %q facet budgets: %w", scope, err)
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			var budget int
			if err := rows.Scan(&id, &budget); err != nil {
				_ = rows.Close()
				return Config{}, err
			}
			c.FacetBudgets[id] = budget
		}
		if err := rows.Close(); err != nil {
			return Config{}, err
		}
	}
	r.mu.Lock()
	r.cache[scope] = c
	r.mu.Unlock()
	return c, nil
}

func (r *Registry) Invalidate(scope Scope) {
	r.mu.Lock()
	delete(r.cache, NormalizeScope(string(scope)))
	r.mu.Unlock()
}

func (r *Registry) List(ctx context.Context) ([]ScopeDefinition, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,label,frontier_target,wake_budget,max_entry_lines FROM scopes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScopeDefinition
	for rows.Next() {
		var d ScopeDefinition
		if err := rows.Scan(&d.ID, &d.Label, &d.Config.FrontierTarget, &d.Config.WakeBudget, &d.Config.MaxEntryLines); err != nil {
			return nil, err
		}
		d.Config.Scope = Scope(d.ID)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Registry) Create(ctx context.Context, definition ScopeDefinition) error {
	definition.ID = strings.TrimSpace(definition.ID)
	definition.Label = strings.TrimSpace(definition.Label)
	if definition.ID == "" || definition.ID == string(AgentMemory) {
		return fmt.Errorf("scope id must be non-empty and may not be %q", AgentMemory)
	}
	if definition.Label == "" {
		return fmt.Errorf("scope %q label is required", definition.ID)
	}
	if definition.Config.FrontierTarget <= 0 || definition.Config.WakeBudget <= 0 || definition.Config.MaxEntryLines <= 0 {
		return fmt.Errorf("scope %q frontier target, wake budget, and max entry lines must be positive", definition.ID)
	}
	var resident int
	for _, facet := range definition.Facets {
		if facet.ID == "" || facet.ResidentBudget < 0 {
			return fmt.Errorf("scope %q has invalid facet %q resident budget", definition.ID, facet.ID)
		}
		resident += facet.ResidentBudget
	}
	capacity := definition.Config.WakeBudget / definition.Config.MaxEntryLines
	if resident > capacity {
		return fmt.Errorf("scope %q residency budgets require %d entries (%d lines at %d max lines), exceeding wake budget %d", definition.ID, resident, resident*definition.Config.MaxEntryLines, definition.Config.MaxEntryLines, definition.Config.WakeBudget)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := "now"
	if _, err := tx.ExecContext(ctx, `INSERT INTO scopes(id,label,frontier_target,wake_budget,max_entry_lines,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, definition.ID, definition.Label, definition.Config.FrontierTarget, definition.Config.WakeBudget, definition.Config.MaxEntryLines, now, now); err != nil {
		return fmt.Errorf("create scope %q: %w", definition.ID, err)
	}
	// Facet ids are globally keyed by the existing schema. Reject a collision
	// instead of silently attaching another scope to the wrong vocabulary.
	for _, facet := range definition.Facets {
		if _, err := tx.ExecContext(ctx, `INSERT INTO facet_definitions(id,scope,label,classification_guidance,created_at) VALUES(?,?,?,?,?)`, facet.ID, definition.ID, facet.Label, facet.Guidance, now); err != nil {
			return fmt.Errorf("create scope %q facet %q: %w", definition.ID, facet.ID, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO facet_policies(facet_id,scope,retention_policy,compaction_eligible,resident_budget) VALUES(?,?,?,?,?)`, facet.ID, definition.ID, facet.RetentionPolicy, facet.CompactionEligible, facet.ResidentBudget); err != nil {
			return fmt.Errorf("create scope %q facet %q policy: %w", definition.ID, facet.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit scope %q: %w", definition.ID, err)
	}
	r.Invalidate(Scope(definition.ID))
	return nil
}

// ResolveAll is useful to startup reconciliation and keeps its result stable
// for callers that need deterministic federation registration order.
func (r *Registry) ResolveAll(ctx context.Context) ([]Config, error) {
	items, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Config, 0, len(items))
	for _, item := range items {
		config, err := r.Resolve(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, config)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scope < out[j].Scope })
	return out, nil
}

type Resolver struct {
	scope      Scope
	config     Config
	registry   *Registry
	vocabulary func(context.Context) ([]Facet, error)
}

func NewResolver(scope Scope, config Config, vocabulary func(context.Context) ([]Facet, error)) *Resolver {
	if scope == "" {
		scope = AgentMemory
	}
	return &Resolver{scope: scope, config: config, vocabulary: vocabulary}
}

// NewRequestResolver keeps policy lookup keyed by the request scope while the
// registry supplies a small in-memory cache for repeated reads.
func NewRequestResolver(registry *Registry, vocabulary func(context.Context) ([]Facet, error)) *Resolver {
	return &Resolver{scope: AgentMemory, registry: registry, vocabulary: vocabulary}
}

func (r *Resolver) Scope() Scope   { return r.scope }
func (r *Resolver) Config() Config { return r.config }
func (r *Resolver) Resolve(ctx context.Context) (Config, error) {
	if r.registry != nil {
		return r.registry.Resolve(ctx, string(ScopeFromContext(ctx)))
	}
	return r.config, nil
}

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
