package facets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"source-ledger/internal/policy"

	"github.com/google/uuid"
)

const pinReviewInterval = 90 * 24 * time.Hour

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

var seeds = []Definition{
	{ID: "standing-rule", Label: "Standing rule", ClassificationGuidance: "A durable instruction, policy, invariant, or reusable guidance that should be followed across work.", RetentionPolicy: "pinned-or-review", ResidentBudget: 8},
	{ID: "environment-fact", Label: "Environment fact", ClassificationGuidance: "A stable fact about a host, runtime, tool, configuration, or operating environment.", RetentionPolicy: "retain", ResidentBudget: 4},
	{ID: "gotcha", Label: "Gotcha", ClassificationGuidance: "A recurring failure mode, trap, debugging lesson, or warning about what goes wrong.", RetentionPolicy: "retain", ResidentBudget: 4},
	{ID: "episode", Label: "Episode", ClassificationGuidance: "A completed project, implementation, validation, or historical work event with an outcome. Any work record with Trigger, Approach, Evidence, and Outcome fields is an episode, including partial, failed, or in-progress outcomes, even when it names a system or project.", RetentionPolicy: "compact", CompactionEligible: true, ResidentBudget: 12},
	{ID: "thread", Label: "Thread", ClassificationGuidance: "An active, unresolved line of work, investigation, or follow-up that is not complete.", RetentionPolicy: "expire-on-resolution", ResidentBudget: 4},
	{ID: "entity-record", Label: "Entity record", ClassificationGuidance: "A durable record describing the identity, ownership, or stable reference of a named system, scenario, person, or artifact; it is not a completed implementation work record.", RetentionPolicy: "retain", ResidentBudget: 4},
}

func (r *SQLiteRepository) Seed(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, d := range seeds {
		if _, err = tx.ExecContext(ctx, `INSERT INTO facet_definitions(id,label,classification_guidance,created_at) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET label=excluded.label,classification_guidance=excluded.classification_guidance`, d.ID, d.Label, d.ClassificationGuidance, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO facet_policies(facet_id,retention_policy,compaction_eligible,resident_budget) VALUES(?,?,?,?) ON CONFLICT(facet_id) DO UPDATE SET resident_budget=CASE WHEN facet_policies.resident_budget=0 THEN excluded.resident_budget ELSE facet_policies.resident_budget END`, d.ID, d.RetentionPolicy, d.CompactionEligible, d.ResidentBudget); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SeedExamples records a small, deterministic, corpus-derived example set for
// every facet. The examples are data owned by the facet tables, never prompt
// constants, and are inserted only when a given facet/entry pair is absent.
func (r *SQLiteRepository) SeedExamples(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM facet_examples WHERE scope='agent-memory'`); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
WITH latest AS (
  SELECT entry_id,facet_id,ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC,id DESC) rn
  FROM facet_assignments
), ranked AS (
  SELECT e.id,e.body,l.facet_id,ROW_NUMBER() OVER (PARTITION BY l.facet_id ORDER BY e.id) facet_rank
  FROM entries e JOIN latest l ON l.entry_id=e.id AND l.rn=1
  WHERE e.scope='agent-memory' AND trim(e.body)<>'' AND
    ((l.facet_id='episode' AND e.source_runtime='swarm-manager' AND e.kind='work-record')
      OR (l.facet_id<>'episode' AND e.kind<>'work-record'))
)
INSERT INTO facet_examples(id,scope,facet_id,entry_id,body,created_at)
SELECT 'corpus-example:'||facet_id||':'||id,'agent-memory',facet_id,id,body,?
FROM ranked WHERE facet_rank<=3`, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Definition, error) {
	scope := policy.ScopeFromContext(ctx)
	rows, err := r.db.QueryContext(ctx, `SELECT d.id,d.label,d.classification_guidance,p.retention_policy,p.compaction_eligible,p.resident_budget FROM facet_definitions d JOIN facet_policies p ON p.facet_id=d.id AND p.scope=? WHERE d.scope=? ORDER BY d.id`, scope, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Definition
	for rows.Next() {
		var d Definition
		if err = rows.Scan(&d.ID, &d.Label, &d.ClassificationGuidance, &d.RetentionPolicy, &d.CompactionEligible, &d.ResidentBudget); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		examples, err := r.examples(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].ClassificationExamples = examples
	}
	return out, nil
}

// CountUnassigned reports journal entries that have never received a facet
// assignment. It deliberately checks the immutable journal directly rather
// than the forest, so an assignment blind spot remains visible even when
// compaction cannot see the entry.
func (r *SQLiteRepository) CountUnassigned(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM entries e
WHERE e.scope=?
  AND NOT EXISTS (SELECT 1 FROM facet_assignments a WHERE a.entry_id=e.id)`, policy.ScopeFromContext(ctx)).Scan(&count)
	return count, err
}

// Ensure creates a facet definition and its initial policy when the facet is
// absent. Existing definitions are refreshed for classifier guidance, but
// their mutable policy is left untouched so a later bootstrap cannot reset an
// operator's residency choice.
func (r *SQLiteRepository) Ensure(ctx context.Context, requested Definition) (Definition, error) {
	pol, err := normalizePolicy(FacetPolicy{requested.ID, requested.RetentionPolicy, requested.CompactionEligible, requested.ResidentBudget})
	if err != nil {
		return Definition{}, err
	}
	requested.ID, requested.RetentionPolicy = pol.ID, pol.RetentionPolicy
	scope := policy.ScopeFromContext(ctx)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Definition{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM facet_definitions WHERE id=? AND scope=?`, requested.ID, scope).Scan(&exists); err != nil {
		return Definition{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if exists == 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO facet_definitions(id,scope,label,classification_guidance,created_at) VALUES(?,?,?,?,?)`, requested.ID, scope, requested.Label, requested.ClassificationGuidance, now); err != nil {
			return Definition{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO facet_policies(facet_id,scope,retention_policy,compaction_eligible,resident_budget) VALUES(?,?,?,?,?)`, requested.ID, scope, requested.RetentionPolicy, requested.CompactionEligible, requested.ResidentBudget); err != nil {
			return Definition{}, err
		}
	} else if _, err := tx.ExecContext(ctx, `UPDATE facet_definitions SET label=?,classification_guidance=? WHERE id=? AND scope=?`, requested.Label, requested.ClassificationGuidance, requested.ID, scope); err != nil {
		return Definition{}, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT d.id,d.label,d.classification_guidance,p.retention_policy,p.compaction_eligible,p.resident_budget FROM facet_definitions d JOIN facet_policies p ON p.facet_id=d.id AND p.scope=d.scope WHERE d.id=? AND d.scope=?`, requested.ID, scope).Scan(&requested.ID, &requested.Label, &requested.ClassificationGuidance, &requested.RetentionPolicy, &requested.CompactionEligible, &requested.ResidentBudget); err != nil {
		return Definition{}, err
	}
	if err := tx.Commit(); err != nil {
		return Definition{}, err
	}
	return requested, nil
}

// Delete removes a facet only after no entry currently resolves to it and no
// rule targets it. Historical assignments are copied to the immutable archive
// before their active rows are retired, because the live facet definition has
// a foreign key from facet_assignments.
func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	scope := policy.ScopeFromContext(ctx)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var assignments, rules int
	if err := tx.QueryRowContext(ctx, `WITH latest AS (SELECT entry_id,facet_id,ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY assigned_at DESC,id DESC) rn FROM facet_assignments) SELECT COUNT(*) FROM latest l JOIN entries e ON e.id=l.entry_id WHERE e.scope=? AND l.facet_id=? AND l.rn=1`, scope, id).Scan(&assignments); err != nil {
		return err
	}
	if assignments > 0 {
		return fmt.Errorf("facet %q still has %d assignment(s) in scope %q", id, assignments, scope)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM classification_rules WHERE scope=? AND facet_id=?`, scope, id).Scan(&rules); err != nil {
		return err
	}
	if rules > 0 {
		return fmt.Errorf("facet %q still has %d classification rule(s) in scope %q", id, rules, scope)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO facet_assignment_archive(id,entry_id,facet_id,assigned_at,actor_id,archived_at) SELECT id,entry_id,facet_id,assigned_at,actor_id,? FROM facet_assignments WHERE facet_id=?`, now, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM facet_assignments WHERE facet_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM facet_policies WHERE facet_id=? AND scope=?`, id, scope); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM facet_definitions WHERE id=? AND scope=?`, id, scope); err != nil {
		return err
	}
	return tx.Commit()
}

// normalizePolicy trims a facet policy and checks it: a facet id, a known
// retention policy and a non-negative resident budget.
func normalizePolicy(p FacetPolicy) (FacetPolicy, error) {
	p.ID = strings.TrimSpace(p.ID)
	p.RetentionPolicy = strings.TrimSpace(p.RetentionPolicy)
	switch {
	case p.ID == "":
		return p, errors.New("facet_id is required")
	case !policy.IsValidRetentionPolicy(p.RetentionPolicy):
		return p, errors.New("retention_policy must be one of retain, compact, expire-on-resolution, or pinned-or-review")
	case p.ResidentBudget < 0:
		return p, errors.New("resident_budget cannot be negative")
	}
	return p, nil
}

// SetPolicy changes only the mutable policy attached to an existing facet.
// Facet definitions and journal assignments remain historical data; changing
// a retention or residency rule never rewrites either one.
func (r *SQLiteRepository) SetPolicy(ctx context.Context, requested FacetPolicy) (Definition, error) {
	requested, err := normalizePolicy(requested)
	if err != nil {
		return Definition{}, err
	}
	scope := policy.ScopeFromContext(ctx)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Definition{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM facet_definitions WHERE id=? AND scope=?`, requested.ID, scope).Scan(&exists); err != nil {
		return Definition{}, err
	}
	if exists == 0 {
		return Definition{}, ErrUnknownFacet{ID: requested.ID}
	}
	result, err := tx.ExecContext(ctx, `UPDATE facet_policies SET retention_policy=?,compaction_eligible=?,resident_budget=? WHERE facet_id=? AND scope=?`, requested.RetentionPolicy, requested.CompactionEligible, requested.ResidentBudget, requested.ID, scope)
	if err != nil {
		return Definition{}, err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return Definition{}, err
	} else if affected != 1 {
		return Definition{}, ErrUnknownFacet{ID: requested.ID}
	}
	if err := tx.Commit(); err != nil {
		return Definition{}, err
	}
	return r.definition(ctx, requested.ID)
}

// definition is one facet as List reports it.
func (r *SQLiteRepository) definition(ctx context.Context, id string) (Definition, error) {
	items, err := r.List(ctx)
	if err != nil {
		return Definition{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return Definition{}, ErrUnknownFacet{ID: id}
}

func (r *SQLiteRepository) examples(ctx context.Context, facetID string) ([]string, error) {
	return queryStrings(ctx, r.db, `SELECT body FROM facet_examples WHERE scope=? AND facet_id=? ORDER BY id LIMIT 3`, policy.ScopeFromContext(ctx), facetID)
}

func (r *SQLiteRepository) Validate(ctx context.Context, id string) error {
	if id == UnclassifiedFacet {
		return nil
	}
	var found string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM facet_definitions WHERE scope=? AND id=?`, policy.ScopeFromContext(ctx), id).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUnknownFacet{ID: id}
	}
	return err
}

func (r *SQLiteRepository) CompactionEligible(ctx context.Context, entryID string) (bool, error) {
	var eligible, pinned bool
	err := r.db.QueryRowContext(ctx, `SELECT p.compaction_eligible, EXISTS(SELECT 1 FROM pins WHERE entry_id=? AND (review_at IS NULL OR review_at>?)) FROM facet_assignments a JOIN facet_policies p ON p.scope=? AND p.facet_id=a.facet_id JOIN entries e ON e.id=a.entry_id WHERE e.scope=? AND a.entry_id=? ORDER BY a.assigned_at DESC,a.id DESC LIMIT 1`, entryID, time.Now().UTC().Format(time.RFC3339Nano), policy.ScopeFromContext(ctx), policy.ScopeFromContext(ctx), entryID).Scan(&eligible, &pinned)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return eligible && !pinned, err
}

func (r *SQLiteRepository) Assign(ctx context.Context, a Assignment) (Assignment, error) {
	if err := r.Validate(ctx, a.FacetID); err != nil {
		return Assignment{}, err
	}
	if strings.TrimSpace(a.ActorID) == "" {
		a.ActorID = "system:facets"
	}
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.AssignedAt.IsZero() {
		a.AssignedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO facet_assignments(id,entry_id,facet_id,assigned_at,actor_id) VALUES(?,?,?,?,?)`, a.ID, a.EntryID, a.FacetID, a.AssignedAt.Format(time.RFC3339Nano), a.ActorID)
	return a, err
}

func (r *SQLiteRepository) Assignments(ctx context.Context, entryID string) ([]Assignment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT a.id,a.entry_id,a.facet_id,a.assigned_at,a.actor_id FROM facet_assignments a JOIN entries e ON e.id=a.entry_id WHERE a.entry_id=? AND e.scope=? ORDER BY a.assigned_at,a.id`, entryID, policy.ScopeFromContext(ctx))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Assignment
	for rows.Next() {
		var a Assignment
		var raw string
		if err = rows.Scan(&a.ID, &a.EntryID, &a.FacetID, &raw, &a.ActorID); err != nil {
			return nil, err
		}
		a.AssignedAt, _ = time.Parse(time.RFC3339Nano, raw)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) MarkSuperseded(ctx context.Context, entryID, replacementID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO marks(id,entry_id,kind,replacement_entry_id,created_at) SELECT ?,?,?,?,? WHERE EXISTS(SELECT 1 FROM entries WHERE id=? AND scope=?)`, uuid.NewString(), entryID, "superseded", replacementID, time.Now().UTC().Format(time.RFC3339Nano), entryID, policy.ScopeFromContext(ctx))
	return err
}

func (r *SQLiteRepository) ResolveThread(ctx context.Context, entryID string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO marks(id,entry_id,kind,created_at) SELECT ?,?,?,? WHERE EXISTS(SELECT 1 FROM entries WHERE id=? AND scope=?)`, uuid.NewString(), entryID, "resolved", time.Now().UTC().Format(time.RFC3339Nano), entryID, policy.ScopeFromContext(ctx))
	return err
}
