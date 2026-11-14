// Package preferences owns the notification catalog (observed and declared
// sources/categories) and the per-recipient notification preferences that the
// delivery worker consults before routing. It is engine-independent: the
// storage type hides the database behind a narrow DB interface, so both the
// routed production handle and a raw *sql.DB in tests satisfy it.
package preferences

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// DB is the narrow database seam the storage needs. Both *sql.DB and
// api-core's *database.RoutedDB satisfy it.
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Storage is the preferences data plane. It is safe to construct around any DB
// that already has the preferences schema applied.
type Storage struct {
	db DB
}

// New wraps a DB handle. The schema is applied centrally through
// internal/modules; this constructor does not create tables.
func New(db DB) *Storage { return &Storage{db: db} }

// Source is one notification-producing scenario and its branding.
type Source struct {
	SourceScenario string
	DisplayName    string
	Icon           string
	DeepLinkBase   string
	Declared       bool
	Categories     []Category
}

// Category is one named notification class a source can emit.
type Category struct {
	SourceScenario    string
	Category          string
	DisplayName       string
	Description       string
	Declared          bool
	DefaultEnabled    bool
	DefaultMinUrgency string
}

// Preference is one stored per-recipient preference row. ScopeKey is empty for
// a global (source/category) preference and non-empty for a per-instance
// override that applies only to one entity (e.g. a single web-console session).
type Preference struct {
	RecipientKey    string
	SourceScenario  string
	Category        string
	ScopeKey        string
	Enabled         bool
	MinUrgency      string
	ChannelOverride string
}

// Effective is the resolved decision for one (recipient, source, category).
type Effective struct {
	Enabled    bool
	MinUrgency string
}

// ObserveCatalog records that a source/category was seen in a live event. The
// source and category are written at declared=0 (observed baseline); an
// already-declared source keeps its declared flag and branding. first_seen is
// set once and never overwritten.
func (s *Storage) ObserveCatalog(ctx context.Context, source, category, now string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_sources (source_scenario, display_name, icon, deep_link_base, declared, first_seen, updated_at)
		VALUES (?, '', '', '', 0, ?, ?)
		ON CONFLICT(source_scenario) DO UPDATE SET updated_at=excluded.updated_at`,
		source, now, now); err != nil {
		return err
	}
	category = strings.TrimSpace(category)
	if category == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_categories (source_scenario, category, display_name, description, declared, default_enabled, default_min_urgency, first_seen)
		VALUES (?, ?, '', '', 0, 1, 'low', ?)
		ON CONFLICT(source_scenario, category) DO NOTHING`,
		source, category, now)
	return err
}

// UpsertDeclaredSource records a manifest-declared source and its branding.
// Declared branding always wins over an observed baseline. first_seen is set
// on first insert and preserved thereafter.
func (s *Storage) UpsertDeclaredSource(ctx context.Context, source, displayName, icon, deepLinkBase, now string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return errors.New("source_scenario is required")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_sources (source_scenario, display_name, icon, deep_link_base, declared, first_seen, updated_at)
		VALUES (?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT(source_scenario) DO UPDATE SET
			display_name=excluded.display_name,
			icon=excluded.icon,
			deep_link_base=excluded.deep_link_base,
			declared=1,
			updated_at=excluded.updated_at`,
		source, strings.TrimSpace(displayName), strings.TrimSpace(icon), strings.TrimSpace(deepLinkBase), now, now)
	return err
}

// UpsertDeclaredCategory records a manifest-declared category and its defaults.
func (s *Storage) UpsertDeclaredCategory(ctx context.Context, source, category, displayName, description string, defaultEnabled bool, defaultMinUrgency, now string) error {
	source = strings.TrimSpace(source)
	category = strings.TrimSpace(category)
	if source == "" || category == "" {
		return errors.New("source_scenario and category are required")
	}
	minUrgency := strings.TrimSpace(defaultMinUrgency)
	if minUrgency == "" {
		minUrgency = "low"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_categories (source_scenario, category, display_name, description, declared, default_enabled, default_min_urgency, first_seen)
		VALUES (?, ?, ?, ?, 1, ?, ?, ?)
		ON CONFLICT(source_scenario, category) DO UPDATE SET
			display_name=excluded.display_name,
			description=excluded.description,
			declared=1,
			default_enabled=excluded.default_enabled,
			default_min_urgency=excluded.default_min_urgency`,
		source, category, strings.TrimSpace(displayName), strings.TrimSpace(description), boolInt(defaultEnabled), minUrgency, now)
	return err
}

// UpsertObservedBranding fills in inferred branding for an observed source. It
// sets display_name/icon ONLY when the row is not already declared and its
// display_name is still empty, so it can never clobber manifest-declared
// branding or a name already resolved. A declared or already-branded row is left
// untouched.
func (s *Storage) UpsertObservedBranding(ctx context.Context, source, displayName, icon, now string) error {
	source = strings.TrimSpace(source)
	displayName = strings.TrimSpace(displayName)
	if source == "" || displayName == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE notification_sources
		SET display_name=?, icon=?, updated_at=?
		WHERE source_scenario=? AND declared=0 AND display_name=''`,
		displayName, strings.TrimSpace(icon), now, source)
	return err
}

// ResolveBrand returns a source's display branding. ok is false when the source
// has never been observed or declared.
func (s *Storage) ResolveBrand(ctx context.Context, source string) (displayName, icon, deepLinkBase string, ok bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", "", "", false
	}
	err := s.db.QueryRowContext(ctx, `SELECT display_name, icon, deep_link_base FROM notification_sources WHERE source_scenario=?`, source).Scan(&displayName, &icon, &deepLinkBase)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", false
	}
	if err != nil {
		return "", "", "", false
	}
	return displayName, icon, deepLinkBase, true
}

// GetPreference resolves the effective decision for a (recipient, source,
// category, scopeKey), most-specific first:
//  0. (recipientKey, source, category, scopeKey) — a per-instance override
//     0b. (recipientKey, source, "",       scopeKey) — a source-wide instance override
//  1. (recipientKey, source, category)
//  2. (recipientKey, source, "")  — a source-wide preference
//  3. category defaults from notification_categories
//  4. source default: enabled true, min urgency ""
//
// The scope tiers (0/0b) are consulted only when scopeKey is non-empty, so an
// unscoped lookup resolves exactly as before.
func (s *Storage) GetPreference(ctx context.Context, recipientKey, source, category, scopeKey string) (Effective, error) {
	recipientKey = strings.TrimSpace(recipientKey)
	source = strings.TrimSpace(source)
	category = strings.TrimSpace(category)
	scopeKey = strings.TrimSpace(scopeKey)

	// 0. per-instance override, most-specific first. Only when a scope is given.
	if scopeKey != "" {
		if category != "" {
			eff, found, err := s.lookupScopePreference(ctx, recipientKey, source, category, scopeKey)
			if err != nil {
				return Effective{}, err
			}
			if found {
				return eff, nil
			}
		}
		eff, found, err := s.lookupScopePreference(ctx, recipientKey, source, "", scopeKey)
		if err != nil {
			return Effective{}, err
		}
		if found {
			return eff, nil
		}
	}
	// 1. exact category-level preference.
	if category != "" {
		eff, found, err := s.lookupPreference(ctx, recipientKey, source, category)
		if err != nil {
			return Effective{}, err
		}
		if found {
			return eff, nil
		}
	}
	// 2. source-wide preference.
	eff, found, err := s.lookupPreference(ctx, recipientKey, source, "")
	if err != nil {
		return Effective{}, err
	}
	if found {
		return eff, nil
	}
	// 3. category defaults.
	if category != "" {
		var defaultEnabled int
		var defaultMin string
		err := s.db.QueryRowContext(ctx, `SELECT default_enabled, default_min_urgency FROM notification_categories WHERE source_scenario=? AND category=?`, source, category).Scan(&defaultEnabled, &defaultMin)
		if err == nil {
			return Effective{Enabled: defaultEnabled != 0, MinUrgency: strings.TrimSpace(defaultMin)}, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return Effective{}, err
		}
	}
	// 4. source default.
	return Effective{Enabled: true, MinUrgency: ""}, nil
}

func (s *Storage) lookupPreference(ctx context.Context, recipientKey, source, category string) (Effective, bool, error) {
	var enabled int
	var minUrgency string
	err := s.db.QueryRowContext(ctx, `SELECT enabled, min_urgency FROM notification_preferences WHERE recipient_key=? AND source_scenario=? AND category=?`, recipientKey, source, category).Scan(&enabled, &minUrgency)
	if errors.Is(err, sql.ErrNoRows) {
		return Effective{}, false, nil
	}
	if err != nil {
		return Effective{}, false, err
	}
	return Effective{Enabled: enabled != 0, MinUrgency: strings.TrimSpace(minUrgency)}, true, nil
}

func (s *Storage) lookupScopePreference(ctx context.Context, recipientKey, source, category, scopeKey string) (Effective, bool, error) {
	var enabled int
	var minUrgency string
	err := s.db.QueryRowContext(ctx, `SELECT enabled, min_urgency FROM notification_scope_preferences WHERE recipient_key=? AND source_scenario=? AND category=? AND scope_key=?`, recipientKey, source, category, scopeKey).Scan(&enabled, &minUrgency)
	if errors.Is(err, sql.ErrNoRows) {
		return Effective{}, false, nil
	}
	if err != nil {
		return Effective{}, false, err
	}
	return Effective{Enabled: enabled != 0, MinUrgency: strings.TrimSpace(minUrgency)}, true, nil
}

// SetPreference writes (or replaces) one preference row. A non-empty scopeKey
// writes a per-instance override into notification_scope_preferences; an empty
// scopeKey writes the global (source/category) preference as before.
func (s *Storage) SetPreference(ctx context.Context, recipientKey, source, category string, enabled bool, minUrgency, channelOverride, scopeKey, now string) error {
	recipientKey = strings.TrimSpace(recipientKey)
	source = strings.TrimSpace(source)
	if recipientKey == "" || source == "" {
		return errors.New("recipient_key and source_scenario are required")
	}
	if scopeKey = strings.TrimSpace(scopeKey); scopeKey != "" {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO notification_scope_preferences (recipient_key, source_scenario, category, scope_key, enabled, min_urgency, channel_override, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(recipient_key, source_scenario, category, scope_key) DO UPDATE SET
				enabled=excluded.enabled,
				min_urgency=excluded.min_urgency,
				channel_override=excluded.channel_override,
				updated_at=excluded.updated_at`,
			recipientKey, source, strings.TrimSpace(category), scopeKey, boolInt(enabled), strings.TrimSpace(minUrgency), strings.TrimSpace(channelOverride), now)
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_preferences (recipient_key, source_scenario, category, enabled, min_urgency, channel_override, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(recipient_key, source_scenario, category) DO UPDATE SET
			enabled=excluded.enabled,
			min_urgency=excluded.min_urgency,
			channel_override=excluded.channel_override,
			updated_at=excluded.updated_at`,
		recipientKey, source, strings.TrimSpace(category), boolInt(enabled), strings.TrimSpace(minUrgency), strings.TrimSpace(channelOverride), now)
	return err
}

// ListCatalog returns every known source with its categories, for the RPC/UI
// layer. A source that is neither declared nor resolvable to a real installed
// scenario is excluded, which cleanly hides stray or test sources (for example
// a one-off smoke-test probe) without database surgery. Declared sources are
// always kept. If the installed-scenario set cannot be resolved, no filtering is
// applied so a contract outage never hides real sources.
func (s *Storage) ListCatalog(ctx context.Context) ([]Source, error) {
	installed, filter := s.installedSet()
	return s.listCatalog(ctx, installed, filter)
}

// ListCatalogForRepo is ListCatalog with an explicit repo root, used where the
// active root is already resolved (and by tests that must be deterministic). A
// root that cannot be read as a repo contract disables filtering.
func (s *Storage) ListCatalogForRepo(ctx context.Context, repoRoot string) ([]Source, error) {
	installed, filter := installedScenarioSet(repoRoot)
	return s.listCatalog(ctx, installed, filter)
}

// installedSet resolves the installed-scenario set for catalog filtering. filter
// is false when the repo root or contract cannot be read, in which case the
// caller must not filter.
func (s *Storage) installedSet() (set map[string]struct{}, filter bool) {
	repoRoot, err := ResolveRepoRoot()
	if err != nil {
		return nil, false
	}
	return installedScenarioSet(repoRoot)
}

// listCatalog is the filterable core of ListCatalog. When filter is false, every
// source is returned; when true, only declared sources and sources present in
// installed are kept.
func (s *Storage) listCatalog(ctx context.Context, installed map[string]struct{}, filter bool) ([]Source, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT source_scenario, display_name, icon, deep_link_base, declared FROM notification_sources ORDER BY source_scenario`)
	if err != nil {
		return nil, err
	}
	var sources []Source
	index := map[string]int{}
	for rows.Next() {
		var src Source
		var declared int
		if err := rows.Scan(&src.SourceScenario, &src.DisplayName, &src.Icon, &src.DeepLinkBase, &declared); err != nil {
			rows.Close()
			return nil, err
		}
		src.Declared = declared != 0
		if filter && !src.Declared {
			if _, ok := installed[src.SourceScenario]; !ok {
				continue
			}
		}
		index[src.SourceScenario] = len(sources)
		sources = append(sources, src)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	catRows, err := s.db.QueryContext(ctx, `SELECT source_scenario, category, display_name, description, declared, default_enabled, default_min_urgency FROM notification_categories ORDER BY source_scenario, category`)
	if err != nil {
		return nil, err
	}
	defer catRows.Close()
	for catRows.Next() {
		var cat Category
		var declared, defaultEnabled int
		if err := catRows.Scan(&cat.SourceScenario, &cat.Category, &cat.DisplayName, &cat.Description, &declared, &defaultEnabled, &cat.DefaultMinUrgency); err != nil {
			return nil, err
		}
		cat.Declared = declared != 0
		cat.DefaultEnabled = defaultEnabled != 0
		if i, ok := index[cat.SourceScenario]; ok {
			sources[i].Categories = append(sources[i].Categories, cat)
		}
	}
	return sources, catRows.Err()
}

// ListPreferences returns every stored preference row for one recipient: the
// global rows (ScopeKey "") followed by the per-instance scope overrides.
func (s *Storage) ListPreferences(ctx context.Context, recipientKey string) ([]Preference, error) {
	recipientKey = strings.TrimSpace(recipientKey)
	rows, err := s.db.QueryContext(ctx, `SELECT recipient_key, source_scenario, category, enabled, min_urgency, channel_override FROM notification_preferences WHERE recipient_key=? ORDER BY source_scenario, category`, recipientKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Preference
	for rows.Next() {
		var p Preference
		var enabled int
		if err := rows.Scan(&p.RecipientKey, &p.SourceScenario, &p.Category, &enabled, &p.MinUrgency, &p.ChannelOverride); err != nil {
			return nil, err
		}
		p.Enabled = enabled != 0
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	scopeRows, err := s.db.QueryContext(ctx, `SELECT recipient_key, source_scenario, category, scope_key, enabled, min_urgency, channel_override FROM notification_scope_preferences WHERE recipient_key=? ORDER BY source_scenario, category, scope_key`, recipientKey)
	if err != nil {
		return nil, err
	}
	defer scopeRows.Close()
	for scopeRows.Next() {
		var p Preference
		var enabled int
		if err := scopeRows.Scan(&p.RecipientKey, &p.SourceScenario, &p.Category, &p.ScopeKey, &enabled, &p.MinUrgency, &p.ChannelOverride); err != nil {
			return nil, err
		}
		p.Enabled = enabled != 0
		out = append(out, p)
	}
	return out, scopeRows.Err()
}

// urgencyRank orders the urgency ladder. Unknown values rank with "normal" so a
// producer typo never silently suppresses everything.
func urgencyRank(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return 0
	case "normal", "":
		return 1
	case "high":
		return 2
	case "critical":
		return 3
	default:
		return 1
	}
}

// UrgencyAtLeast reports whether have meets or exceeds min on the ladder
// low<normal<high<critical. An empty min always passes.
func UrgencyAtLeast(have, min string) bool {
	if strings.TrimSpace(min) == "" {
		return true
	}
	return urgencyRank(have) >= urgencyRank(min)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
