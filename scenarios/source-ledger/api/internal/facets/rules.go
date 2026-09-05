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
	ID, Scope, FacetID, SourceRuntime, Kind, SourcePathGlob, BodyPattern string
	Priority                                                             int
	Enabled                                                              bool
	CreatedAt, UpdatedAt                                                 time.Time
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
	if err := r.Validate(ctx, rule.FacetID); err != nil {
		return Rule{}, err
	}
	if _, err := compileRule(rule); err != nil {
		return Rule{}, err
	}
	now := time.Now().UTC()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	if rule.UpdatedAt.IsZero() {
		rule.UpdatedAt = now
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO classification_rules(id,scope,priority,facet_id,source_runtime,kind,source_path_glob,body_pattern,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, rule.ID, rule.Scope, rule.Priority, rule.FacetID, rule.SourceRuntime, rule.Kind, rule.SourcePathGlob, rule.BodyPattern, rule.Enabled, rule.CreatedAt.Format(time.RFC3339Nano), rule.UpdatedAt.Format(time.RFC3339Nano))
	return rule, err
}

func (r *SQLiteRepository) ListRules(ctx context.Context, scope string) ([]Rule, error) {
	if scope == "" {
		scope = "agent-memory"
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,scope,priority,facet_id,source_runtime,kind,source_path_glob,body_pattern,enabled,created_at,updated_at FROM classification_rules WHERE scope=? ORDER BY priority,id`, scope)
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
	rows, err := r.db.QueryContext(ctx, `SELECT body,source_runtime,kind,source_path FROM entries WHERE scope=? ORDER BY created_at,id`, rule.Scope)
	if err != nil {
		return DryRun{}, err
	}
	defer rows.Close()
	dry := DryRun{ID: uuid.NewString(), RuleID: rule.ID, Scope: rule.Scope, CorpusFingerprint: fingerprint, CreatedAt: time.Now().UTC()}
	for rows.Next() {
		var input RuleInput
		if err := rows.Scan(&input.Body, &input.SourceRuntime, &input.Kind, &input.SourcePath); err != nil {
			return DryRun{}, err
		}
		matched, err := matches(rule, input)
		if err != nil {
			return DryRun{}, err
		}
		if matched {
			dry.MatchCount++
			if len(dry.Samples) < 20 {
				dry.Samples = append(dry.Samples, input.Body)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return DryRun{}, err
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
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT entry_id FROM facet_assignments WHERE actor_id=?`, "rule:"+ruleID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	count := 0
	for _, entryID := range ids {
		assignments, err := r.Assignments(ctx, entryID)
		if err != nil {
			return count, err
		}
		var prior string
		for i := len(assignments) - 1; i >= 0; i-- {
			if assignments[i].ActorID != "rule:"+ruleID {
				prior = assignments[i].FacetID
				break
			}
		}
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

func (r *SQLiteRepository) rule(ctx context.Context, id string) (Rule, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,scope,priority,facet_id,source_runtime,kind,source_path_glob,body_pattern,enabled,created_at,updated_at FROM classification_rules WHERE id=? AND scope=?`, id, policy.ScopeFromContext(ctx))
	return scanRule(row)
}

type rowScanner interface{ Scan(...any) error }

func scanRule(row rowScanner) (Rule, error) {
	var rule Rule
	var enabled int
	var created, updated string
	if err := row.Scan(&rule.ID, &rule.Scope, &rule.Priority, &rule.FacetID, &rule.SourceRuntime, &rule.Kind, &rule.SourcePathGlob, &rule.BodyPattern, &enabled, &created, &updated); err != nil {
		return Rule{}, err
	}
	rule.Enabled = enabled != 0
	rule.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	rule.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return rule, nil
}

func compileRule(rule Rule) (*regexp.Regexp, error) {
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
	if rule.SourceRuntime != "" && rule.SourceRuntime != input.SourceRuntime || rule.Kind != "" && rule.Kind != input.Kind {
		return false, nil
	}
	if rule.SourcePathGlob != "" {
		ok, err := path.Match(rule.SourcePathGlob, input.SourcePath)
		if err != nil {
			return false, fmt.Errorf("invalid source path glob for rule %s: %w", rule.ID, err)
		}
		if !ok {
			return false, nil
		}
	}
	return pattern == nil || pattern.MatchString(input.Body), nil
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
