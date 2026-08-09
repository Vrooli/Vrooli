// Package policy resolves domain policy values independently from the memory
// engine. The first deployment has one scope; keeping the scope explicit
// makes a future ledger a configuration/data addition instead of an engine
// rewrite.
package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Scope string

const AgentMemory Scope = "agent-memory"

const (
	OriginScopeOverride = "scope-override"
	OriginFileDefault   = "file-default"
	OriginBuiltIn       = "built-in"
	PolicyFileName      = ".vrooli/ledger-policy.json"
)

type Config struct {
	Scope           Scope
	FrontierTarget  int
	WakeBudget      int
	WakeBudgetChars int
	MaxEntryLines   int
	MaxEntryChars   int
	FacetBudgets    map[string]int
	Origins         Origins
}

type Origins struct {
	FrontierTarget  string
	WakeBudget      string
	WakeBudgetChars string
	MaxEntryLines   string
	MaxEntryChars   string
}

// Override contains only values an operator explicitly changed. Nil values
// preserve file-default inheritance; this is why the database never needs to
// copy defaults into every scope.
type Override struct {
	FrontierTarget  *int
	WakeBudget      *int
	WakeBudgetChars *int
	MaxEntryLines   *int
	MaxEntryChars   *int
}

type fileDefaults struct {
	Schema          string `json:"$schema"`
	FrontierTarget  int    `json:"frontierTarget"`
	WakeBudgetLines int    `json:"wakeBudgetLines"`
	WakeBudgetChars int    `json:"wakeBudgetChars"`
	MaxEntryLines   int    `json:"maxEntryLines"`
	MaxEntryChars   int    `json:"maxEntryChars"`
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
	// DefaultWakeBudgetChars is the whole-view character backstop. Characters
	// bind for the single-line JSON documents written by the journal.
	DefaultWakeBudgetChars = 12000
	// DefaultMaxEntryChars keeps one verbose journal document from dominating
	// an agent's ambient context.
	DefaultMaxEntryChars = 200
)

func BuiltInDefaults() Config {
	return Config{
		Scope:           AgentMemory,
		FrontierTarget:  DefaultFrontierTarget,
		WakeBudget:      DefaultWakeBudget,
		WakeBudgetChars: DefaultWakeBudgetChars,
		MaxEntryLines:   DefaultMaxEntryLines,
		MaxEntryChars:   DefaultMaxEntryChars,
		Origins: Origins{
			FrontierTarget: OriginBuiltIn, WakeBudget: OriginBuiltIn,
			WakeBudgetChars: OriginBuiltIn, MaxEntryLines: OriginBuiltIn,
			MaxEntryChars: OriginBuiltIn,
		},
	}
}

// DefaultPolicyPath finds the checked-in policy relative to the scenario root.
// Lifecycle starts normally from the scenario directory, while unit tests and
// developer tools may start from api/ or a repository subdirectory.
func DefaultPolicyPath() string {
	for _, start := range candidateRoots() {
		for dir := start; ; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, PolicyFileName)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return filepath.Join(".", PolicyFileName)
}

func candidateRoots() []string {
	roots := []string{}
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}
	if executable, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(executable))
	}
	return roots
}

// LoadDefaults validates the operator-controlled file before it enters the
// request path. A malformed or unsafe policy is a startup error, never a
// silently widened context budget.
func LoadDefaults(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read policy defaults %q: %w", path, err)
	}
	var file fileDefaults
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return Config{}, fmt.Errorf("parse policy defaults %q: %w", path, err)
	}
	if err := file.validate(); err != nil {
		return Config{}, fmt.Errorf("validate policy defaults %q: %w", path, err)
	}
	return Config{
		Scope:          fileScope,
		FrontierTarget: file.FrontierTarget, WakeBudget: file.WakeBudgetLines,
		WakeBudgetChars: file.WakeBudgetChars, MaxEntryLines: file.MaxEntryLines,
		MaxEntryChars: file.MaxEntryChars,
		Origins: Origins{
			FrontierTarget: OriginFileDefault, WakeBudget: OriginFileDefault,
			WakeBudgetChars: OriginFileDefault, MaxEntryLines: OriginFileDefault,
			MaxEntryChars: OriginFileDefault,
		},
	}, nil
}

var fileScope = Scope("defaults")

func (f fileDefaults) validate() error {
	values := map[string]int{
		"frontierTarget": f.FrontierTarget, "wakeBudgetLines": f.WakeBudgetLines,
		"wakeBudgetChars": f.WakeBudgetChars, "maxEntryLines": f.MaxEntryLines,
		"maxEntryChars": f.MaxEntryChars,
	}
	for name, value := range values {
		if value <= 0 {
			return fmt.Errorf("%s must be a positive integer", name)
		}
	}
	if f.MaxEntryLines > f.WakeBudgetLines {
		return fmt.Errorf("maxEntryLines cannot exceed wakeBudgetLines")
	}
	if f.MaxEntryChars > f.WakeBudgetChars {
		return fmt.Errorf("maxEntryChars cannot exceed wakeBudgetChars")
	}
	return nil
}

func normalizeDefaults(config Config) Config {
	if config.FrontierTarget <= 0 {
		config.FrontierTarget = DefaultFrontierTarget
	}
	if config.WakeBudget <= 0 {
		config.WakeBudget = DefaultWakeBudget
	}
	if config.WakeBudgetChars <= 0 {
		config.WakeBudgetChars = DefaultWakeBudgetChars
	}
	if config.MaxEntryLines <= 0 {
		config.MaxEntryLines = DefaultMaxEntryLines
	}
	if config.MaxEntryChars <= 0 {
		config.MaxEntryChars = DefaultMaxEntryChars
	}
	if config.Origins.FrontierTarget == "" {
		config.Origins.FrontierTarget = OriginBuiltIn
	}
	if config.Origins.WakeBudget == "" {
		config.Origins.WakeBudget = OriginBuiltIn
	}
	if config.Origins.WakeBudgetChars == "" {
		config.Origins.WakeBudgetChars = OriginBuiltIn
	}
	if config.Origins.MaxEntryLines == "" {
		config.Origins.MaxEntryLines = OriginBuiltIn
	}
	if config.Origins.MaxEntryChars == "" {
		config.Origins.MaxEntryChars = OriginBuiltIn
	}
	return config
}

// DefaultSummaryInstruction is the agent-memory scope's opening line for the
// summarization prompt. It lives here rather than in the compaction engine
// because it names what this scope stores; a second scope replaces it without
// touching the engine.
const DefaultSummaryInstruction = "Summarize these episode memories for future agent context."

// ExpireOnResolution is the retention policy whose members leave the ambient
// frontier once marked resolved. The engine knows that some facets expire on
// resolution; only this package knows what that policy is called.
const ExpireOnResolution = "expire-on-resolution"

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
	db       *sql.DB
	defaults Config
	mu       sync.RWMutex
	cache    map[Scope]Config
}

func NewRegistry(db *sql.DB, defaults ...Config) *Registry {
	configured := BuiltInDefaults()
	if len(defaults) > 0 {
		configured = normalizeDefaults(defaults[0])
	}
	return &Registry{db: db, defaults: configured, cache: make(map[Scope]Config)}
}

// Ensure creates the registry and reconciles the first deployment's
// file-backed agent-memory policy. Other scopes inherit the same file defaults
// until an operator explicitly creates a nullable database override.
func (r *Registry) Ensure(ctx context.Context, defaults Config) error {
	defaults = normalizeDefaults(defaults)
	r.defaults = defaults
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
	if _, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS scope_policies (
		scope TEXT PRIMARY KEY,
		frontier_target INTEGER,
		wake_budget_lines INTEGER,
		wake_budget_chars INTEGER,
		max_entry_lines INTEGER,
		max_entry_chars INTEGER,
		updated_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create scope_policies table: %w", err)
	}
	now := "now"
	if _, err := r.db.ExecContext(ctx, `INSERT INTO scopes(id,label,frontier_target,wake_budget,max_entry_lines,created_at,updated_at) VALUES(?,?,?,?,?,?,?)
      ON CONFLICT(id) DO NOTHING`,
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
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scopes WHERE id=?`, string(scope)).Scan(&exists)
	if err != nil {
		return Config{}, fmt.Errorf("resolve scope %q: %w", scope, err)
	}
	if exists == 0 {
		return Config{}, fmt.Errorf("scope %q is not registered", scope)
	}
	c := normalizeDefaults(r.defaults)
	c.Scope = scope
	if err := r.applyOverride(ctx, scope, &c); err != nil {
		return Config{}, err
	}
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

func (r *Registry) applyOverride(ctx context.Context, scope Scope, c *Config) error {
	var frontier, lines, chars, entryLines, entryChars sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT frontier_target,wake_budget_lines,wake_budget_chars,max_entry_lines,max_entry_chars FROM scope_policies WHERE scope=?`, string(scope)).Scan(&frontier, &lines, &chars, &entryLines, &entryChars)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve scope %q override: %w", scope, err)
	}
	if frontier.Valid {
		c.FrontierTarget, c.Origins.FrontierTarget = int(frontier.Int64), OriginScopeOverride
	}
	if lines.Valid {
		c.WakeBudget, c.Origins.WakeBudget = int(lines.Int64), OriginScopeOverride
	}
	if chars.Valid {
		c.WakeBudgetChars, c.Origins.WakeBudgetChars = int(chars.Int64), OriginScopeOverride
	}
	if entryLines.Valid {
		c.MaxEntryLines, c.Origins.MaxEntryLines = int(entryLines.Int64), OriginScopeOverride
	}
	if entryChars.Valid {
		c.MaxEntryChars, c.Origins.MaxEntryChars = int(entryChars.Int64), OriginScopeOverride
	}
	return validateConfig(*c)
}

func validateConfig(c Config) error {
	for name, value := range map[string]int{"frontierTarget": c.FrontierTarget, "wakeBudgetLines": c.WakeBudget, "wakeBudgetChars": c.WakeBudgetChars, "maxEntryLines": c.MaxEntryLines, "maxEntryChars": c.MaxEntryChars} {
		if value <= 0 {
			return fmt.Errorf("%s must be a positive integer", name)
		}
	}
	if c.MaxEntryLines > c.WakeBudget {
		return fmt.Errorf("maxEntryLines cannot exceed wakeBudgetLines")
	}
	if c.MaxEntryChars > c.WakeBudgetChars {
		return fmt.Errorf("maxEntryChars cannot exceed wakeBudgetChars")
	}
	return nil
}

func (r *Registry) SetOverride(ctx context.Context, raw string, override Override) (Config, error) {
	scope := NormalizeScope(raw)
	if err := validateOverride(override); err != nil {
		return Config{}, err
	}
	current, err := r.Resolve(ctx, string(scope))
	if err != nil {
		return Config{}, err
	}
	prospective := current
	if override.FrontierTarget != nil {
		prospective.FrontierTarget = *override.FrontierTarget
	}
	if override.WakeBudget != nil {
		prospective.WakeBudget = *override.WakeBudget
	}
	if override.WakeBudgetChars != nil {
		prospective.WakeBudgetChars = *override.WakeBudgetChars
	}
	if override.MaxEntryLines != nil {
		prospective.MaxEntryLines = *override.MaxEntryLines
	}
	if override.MaxEntryChars != nil {
		prospective.MaxEntryChars = *override.MaxEntryChars
	}
	if err := validateConfig(prospective); err != nil {
		return Config{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Config{}, err
	}
	defer func() { _ = tx.Rollback() }()
	columns := []struct {
		name  string
		value *int
	}{
		{"frontier_target", override.FrontierTarget},
		{"wake_budget_lines", override.WakeBudget},
		{"wake_budget_chars", override.WakeBudgetChars},
		{"max_entry_lines", override.MaxEntryLines},
		{"max_entry_chars", override.MaxEntryChars},
	}
	for _, column := range columns {
		if column.value == nil {
			continue
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO scope_policies(scope,`+column.name+`,updated_at) VALUES(?,?,?) ON CONFLICT(scope) DO UPDATE SET `+column.name+`=excluded.`+column.name+`,updated_at=excluded.updated_at`, string(scope), *column.value, "now")
		if err != nil {
			return Config{}, fmt.Errorf("set scope %q policy: %w", scope, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Config{}, fmt.Errorf("commit scope %q policy: %w", scope, err)
	}
	r.Invalidate(scope)
	return r.Resolve(ctx, string(scope))
}

func validateOverride(o Override) error {
	for name, value := range map[string]*int{"frontierTarget": o.FrontierTarget, "wakeBudgetLines": o.WakeBudget, "wakeBudgetChars": o.WakeBudgetChars, "maxEntryLines": o.MaxEntryLines, "maxEntryChars": o.MaxEntryChars} {
		if value != nil && *value <= 0 {
			return fmt.Errorf("%s must be a positive integer", name)
		}
	}
	return nil
}

func (r *Registry) ResetOverride(ctx context.Context, raw string) (Config, error) {
	scope := NormalizeScope(raw)
	if _, err := r.Resolve(ctx, string(scope)); err != nil {
		return Config{}, err
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM scope_policies WHERE scope=?`, string(scope)); err != nil {
		return Config{}, fmt.Errorf("reset scope %q policy: %w", scope, err)
	}
	r.Invalidate(scope)
	return r.Resolve(ctx, string(scope))
}

func (r *Registry) Defaults() Config { return normalizeDefaults(r.defaults) }

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
	type scopeRow struct {
		id, label                 string
		frontier, wake, entryLine int
	}
	var scopes []scopeRow
	for rows.Next() {
		var scope scopeRow
		if err := rows.Scan(&scope.id, &scope.label, &scope.frontier, &scope.wake, &scope.entryLine); err != nil {
			_ = rows.Close()
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]ScopeDefinition, 0, len(scopes))
	for _, scope := range scopes {
		d := ScopeDefinition{ID: scope.id, Label: scope.label}
		resolved, err := r.Resolve(ctx, scope.id)
		if err != nil {
			return nil, err
		}
		d.Config = resolved
		out = append(out, d)
	}
	return out, nil
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
	definition.Config = normalizeDefaults(definition.Config)
	if err := validateConfig(definition.Config); err != nil {
		return err
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO scope_policies(scope,frontier_target,wake_budget_lines,wake_budget_chars,max_entry_lines,max_entry_chars,updated_at) VALUES(?,?,?,?,?,?,?)`, definition.ID, definition.Config.FrontierTarget, definition.Config.WakeBudget, definition.Config.WakeBudgetChars, definition.Config.MaxEntryLines, definition.Config.MaxEntryChars, now); err != nil {
		return fmt.Errorf("create scope %q policy: %w", definition.ID, err)
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
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS scope_policies (
		scope TEXT PRIMARY KEY,
		frontier_target INTEGER,
		wake_budget_lines INTEGER,
		wake_budget_chars INTEGER,
		max_entry_lines INTEGER,
		max_entry_chars INTEGER,
		updated_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create scope_policies migration: %w", err)
	}
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
