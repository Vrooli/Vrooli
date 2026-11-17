package facets

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"source-ledger/internal/policy"

	"github.com/google/uuid"
)

type Rule struct {
	ID, Scope, FacetID, SourceRuntime, Kind, KindGlob, SourcePathGlob, BodyPattern string
	Priority                                                                       int
	Enabled                                                                        bool
	CreatedAt, UpdatedAt                                                           time.Time
}

type RuleInput struct{ Body, SourceRuntime, Kind, SourcePath string }

// CorpusEntry is the immutable journal provenance needed to re-facet an
// existing corpus. The operation appends assignments; it never edits this
// entry projection or the journal body.
type CorpusEntry struct {
	ID string
	RuleInput
}

type DryRun struct {
	ID, RuleID, Scope, CorpusFingerprint string
	MatchCount                           int
	Samples                              []string
	CreatedAt                            time.Time
}

// DistributionMeasurement separates deterministic provenance coverage from
// the distribution produced by the classifier for the remaining corpus. A
// deterministic rule is reviewed by its own dry-run gate, so rule-matched
// entries are intentionally excluded from the classifier ceiling.
type DistributionMeasurement struct {
	Scope                 string
	Total                 int
	RuleMatched           int
	ClassifierTail        int
	RuleCoverage          map[string]int
	ClassifierTailByFacet map[string]int
	CeilingPercent        float64
	MaxTailFacet          string
	MaxTailPercent        float64
	WithinCeiling         bool
}

const ClassifierTailCeiling = 0.45

func (r *SQLiteRepository) CreateRule(ctx context.Context, rule Rule) (Rule, error) {
	if rule.ID == "" {
		rule.ID = uuid.NewString()
	}
	if rule.Scope == "" {
		rule.Scope = string(policy.ScopeFromContext(ctx))
	}
	if err := r.validateRule(ctx, rule); err != nil {
		return Rule{}, err
	}
	now := time.Now().UTC()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	if rule.UpdatedAt.IsZero() {
		rule.UpdatedAt = now
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO classification_rules(id,scope,priority,facet_id,source_runtime,kind,kind_glob,source_path_glob,body_pattern,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, rule.ID, rule.Scope, rule.Priority, rule.FacetID, rule.SourceRuntime, rule.Kind, rule.KindGlob, rule.SourcePathGlob, rule.BodyPattern, rule.Enabled, rule.CreatedAt.Format(time.RFC3339Nano), rule.UpdatedAt.Format(time.RFC3339Nano))
	return rule, err
}

// validateRule checks a rule before it is stored: a known facet, kind or
// kind_glob (not both), a valid glob and a compilable body pattern.
func (r *SQLiteRepository) validateRule(ctx context.Context, rule Rule) error {
	if err := r.Validate(ctx, rule.FacetID); err != nil {
		return err
	}
	if _, err := globMatch(rule.KindGlob, "", "kind glob", rule.ID); err != nil {
		return err
	}
	_, err := compileRule(rule)
	return err
}

func (r *SQLiteRepository) DeleteRule(ctx context.Context, id string) error {
	scope := policy.ScopeFromContext(ctx)
	var enabled bool
	if err := r.db.QueryRowContext(ctx, `SELECT enabled FROM classification_rules WHERE id=? AND scope=?`, id, scope).Scan(&enabled); err != nil {
		return err
	}
	if enabled {
		return fmt.Errorf("rule %s must be disabled before deletion", id)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM classification_rule_dry_runs WHERE rule_id=? AND scope=?`, id, scope); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM classification_rules WHERE id=? AND scope=?`, id, scope); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) ListRules(ctx context.Context, scope string) ([]Rule, error) {
	if scope == "" {
		scope = "agent-memory"
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,scope,priority,facet_id,source_runtime,kind,kind_glob,source_path_glob,body_pattern,enabled,created_at,updated_at FROM classification_rules WHERE scope=? ORDER BY priority,id`, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Rule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) ListCorpus(ctx context.Context, scope string) ([]CorpusEntry, error) {
	if scope == "" {
		scope = "agent-memory"
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,body,source_runtime,kind,source_path FROM entries WHERE scope=? ORDER BY created_at,id`, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CorpusEntry
	for rows.Next() {
		var entry CorpusEntry
		if err := rows.Scan(&entry.ID, &entry.Body, &entry.SourceRuntime, &entry.Kind, &entry.SourcePath); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) MatchRule(ctx context.Context, scope string, input RuleInput) (Rule, bool, error) {
	rules, err := r.ListRules(ctx, scope)
	if err != nil {
		return Rule{}, false, err
	}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		matched, err := matches(rule, input)
		if err != nil {
			return Rule{}, false, err
		}
		if matched {
			return rule, true, nil
		}
	}
	return Rule{}, false, nil
}

func (r *SQLiteRepository) DryRunRule(ctx context.Context, ruleID string) (DryRun, error) {
	rule, err := r.rule(ctx, ruleID)
	if err != nil {
		return DryRun{}, err
	}
	fingerprint, err := corpusFingerprint(ctx, r.db, rule.Scope)
	if err != nil {
		return DryRun{}, err
	}
	corpus, err := r.ListCorpus(ctx, rule.Scope)
	if err != nil {
		return DryRun{}, err
	}
	dry := DryRun{ID: uuid.NewString(), RuleID: rule.ID, Scope: rule.Scope, CorpusFingerprint: fingerprint, CreatedAt: time.Now().UTC()}
	for _, entry := range corpus {
		matched, err := matches(rule, entry.RuleInput)
		if err != nil {
			return DryRun{}, err
		}
		if matched {
			dry.MatchCount++
			if len(dry.Samples) < 20 {
				dry.Samples = append(dry.Samples, entry.Body)
			}
		}
	}
	samples, err := json.Marshal(dry.Samples)
	if err != nil {
		return DryRun{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO classification_rule_dry_runs(id,rule_id,scope,corpus_fingerprint,match_count,samples_json,created_at) VALUES(?,?,?,?,?,?,?)`, dry.ID, dry.RuleID, dry.Scope, dry.CorpusFingerprint, dry.MatchCount, string(samples), dry.CreatedAt.Format(time.RFC3339Nano))
	return dry, err
}

// MeasureDistribution measures the current corpus using the same enabled
// deterministic rules that refacet uses. The latest assignment is used for
// the classifier tail, preserving assignment history while reporting the
// current operator-visible facet.
func (r *SQLiteRepository) MeasureDistribution(ctx context.Context, scope string) (DistributionMeasurement, error) {
	if scope == "" {
		scope = "agent-memory"
	}
	rules, err := r.ListRules(ctx, scope)
	if err != nil {
		return DistributionMeasurement{}, err
	}
	measurement := DistributionMeasurement{
		Scope:                 scope,
		RuleCoverage:          make(map[string]int),
		ClassifierTailByFacet: make(map[string]int),
		CeilingPercent:        ClassifierTailCeiling,
		WithinCeiling:         true,
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT e.body,e.source_runtime,e.kind,e.source_path,
       COALESCE(a.facet_id,?)
FROM entries e
LEFT JOIN (
  SELECT entry_id,facet_id,
         ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC,id DESC) rn
  FROM facet_assignments
) a ON a.entry_id=e.id AND a.rn=1
WHERE e.scope=?
ORDER BY e.created_at,e.id`, UnclassifiedFacet, scope)
	if err != nil {
		return DistributionMeasurement{}, err
	}
	defer rows.Close()
	for rows.Next() {
		measurement.Total++
		var input RuleInput
		var facetID string
		if err := rows.Scan(&input.Body, &input.SourceRuntime, &input.Kind, &input.SourcePath, &facetID); err != nil {
			return DistributionMeasurement{}, err
		}
		matched := false
		for _, rule := range rules {
			if !rule.Enabled {
				continue
			}
			ok, err := matches(rule, input)
			if err != nil {
				return DistributionMeasurement{}, err
			}
			if !ok {
				continue
			}
			measurement.RuleMatched++
			measurement.RuleCoverage[rule.ID]++
			matched = true
			break
		}
		if !matched {
			measurement.ClassifierTail++
			measurement.ClassifierTailByFacet[facetID]++
		}
	}
	if err := rows.Err(); err != nil {
		return DistributionMeasurement{}, err
	}
	for facetID, count := range measurement.ClassifierTailByFacet {
		if measurement.ClassifierTail == 0 {
			continue
		}
		fraction := float64(count) / float64(measurement.ClassifierTail)
		if fraction > measurement.MaxTailPercent || (fraction == measurement.MaxTailPercent && facetID < measurement.MaxTailFacet) {
			measurement.MaxTailFacet = facetID
			measurement.MaxTailPercent = fraction
		}
	}
	measurement.WithinCeiling = measurement.MaxTailPercent <= measurement.CeilingPercent
	return measurement, nil
}

func (r *SQLiteRepository) EnableRule(ctx context.Context, ruleID string) error {
	rule, err := r.rule(ctx, ruleID)
	if err != nil {
		return err
	}
	fingerprint, err := corpusFingerprint(ctx, r.db, rule.Scope)
	if err != nil {
		return err
	}
	var found int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM classification_rule_dry_runs WHERE rule_id=? AND scope=? AND corpus_fingerprint=?`, rule.ID, rule.Scope, fingerprint).Scan(&found); err != nil {
		return err
	}
	if found == 0 {
		return fmt.Errorf("rule %s cannot be enabled before a dry-run against the current corpus", rule.ID)
	}
	_, err = r.db.ExecContext(ctx, `UPDATE classification_rules SET enabled=1,updated_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339Nano), rule.ID)
	return err
}

// RevertRule appends the latest prior assignment for every entry the rule
// decided. It never deletes assignments or journal entries.
func (r *SQLiteRepository) RevertRule(ctx context.Context, ruleID string) (int, error) {
	actor := "rule:" + ruleID
	ids, err := queryStrings(ctx, r.db, `SELECT DISTINCT entry_id FROM facet_assignments WHERE actor_id=?`, actor)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entryID := range ids {
		assignments, err := r.Assignments(ctx, entryID)
		if err != nil {
			return count, err
		}
		prior := priorFacet(assignments, actor)
		if prior == "" {
			continue
		}
		if _, err := r.Assign(ctx, Assignment{EntryID: entryID, FacetID: prior, ActorID: "operator:rule-revert"}); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// priorFacet is the newest facet in assignments not decided by actor.
func priorFacet(assignments []Assignment, actor string) string {
	for i := len(assignments) - 1; i >= 0; i-- {
		if assignments[i].ActorID != actor {
			return assignments[i].FacetID
		}
	}
	return ""
}

// queryStrings returns the single text column of every row.
func queryStrings(ctx context.Context, db *sql.DB, query string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) rule(ctx context.Context, id string) (Rule, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,scope,priority,facet_id,source_runtime,kind,kind_glob,source_path_glob,body_pattern,enabled,created_at,updated_at FROM classification_rules WHERE id=? AND scope=?`, id, policy.ScopeFromContext(ctx))
	return scanRule(row)
}

type rowScanner interface{ Scan(...any) error }

func scanRule(row rowScanner) (Rule, error) {
	var rule Rule
	var enabled int
	var created, updated string
	if err := row.Scan(&rule.ID, &rule.Scope, &rule.Priority, &rule.FacetID, &rule.SourceRuntime, &rule.Kind, &rule.KindGlob, &rule.SourcePathGlob, &rule.BodyPattern, &enabled, &created, &updated); err != nil {
		return Rule{}, err
	}
	rule.Enabled = enabled != 0
	rule.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	rule.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return rule, nil
}

// compileRule checks a rule's shape (kind or kind_glob, not both) and
// compiles its body pattern (nil when it has none).
func compileRule(rule Rule) (*regexp.Regexp, error) {
	if rule.Kind != "" && rule.KindGlob != "" {
		return nil, fmt.Errorf("rule %s cannot set both kind and kind_glob", rule.ID)
	}
	if strings.TrimSpace(rule.BodyPattern) == "" {
		return nil, nil
	}
	re, err := regexp.Compile(rule.BodyPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid body pattern for rule %s: %w", rule.ID, err)
	}
	return re, nil
}

func matches(rule Rule, input RuleInput) (bool, error) {
	pattern, err := compileRule(rule)
	if err != nil {
		return false, err
	}
	for _, f := range [][2]string{{rule.SourceRuntime, input.SourceRuntime}, {rule.Kind, input.Kind}} {
		if f[0] != "" && f[0] != f[1] {
			return false, nil
		}
	}
	for _, g := range [][3]string{{rule.KindGlob, input.Kind, "kind glob"}, {rule.SourcePathGlob, input.SourcePath, "source path glob"}} {
		if ok, err := globMatch(g[0], g[1], g[2], rule.ID); !ok || err != nil {
			return false, err
		}
	}
	return pattern == nil || pattern.MatchString(input.Body), nil
}

// globMatch matches value against glob; an empty glob matches everything.
func globMatch(glob, value, what, ruleID string) (bool, error) {
	if glob == "" {
		return true, nil
	}
	ok, err := path.Match(glob, value)
	if err != nil {
		return false, fmt.Errorf("invalid %s for rule %s: %w", what, ruleID, err)
	}
	return ok, nil
}

func corpusFingerprint(ctx context.Context, db *sql.DB, scope string) (string, error) {
	var count int
	var latest sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*),MAX(created_at) FROM entries WHERE scope=?`, scope).Scan(&count, &latest); err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s", scope, count, latest.String)))
	return hex.EncodeToString(h[:]), nil
}
