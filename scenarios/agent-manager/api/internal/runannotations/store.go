package runannotations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/sqlcompat"

	"github.com/jmoiron/sqlx"
)

// State says what an annotation row means. Unknown is never zero.
type State string

const (
	// StateObserved means the evaluator answered for this definition hash.
	StateObserved State = "observed"
	// StateCompatible means a declared compatible predecessor's value was reused.
	StateCompatible State = "compatible"
	// StateNotObserved means the evaluator ran but could not determine a value.
	StateNotObserved State = "not_observed"
	// StateUnavailable means the evaluator could not run at all.
	StateUnavailable State = "unavailable"
)

// Value is the stored answer. Probabilities always carry the full distribution
// the method reported, so thresholds can be re-tuned without re-running a model.
type Value struct {
	Kind          string             `json:"kind"`
	Choice        string             `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Noul          *float64           `json:"noul,omitempty"`
	Number        *float64           `json:"number,omitempty"`
	Text          string             `json:"text,omitempty"`
	Bool          *bool              `json:"bool,omitempty"`
	Options       []string           `json:"options,omitempty"`
	Levels        []string           `json:"levels,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
}

// Annotation is one immutable observation keyed by definition and model.
type Annotation struct {
	ScopeKind        Scope     `json:"scope_kind"`
	ScopeID          string    `json:"scope_id"`
	LabelID          string    `json:"label_id"`
	DefinitionHash   string    `json:"definition_hash"`
	// EvidenceHash fingerprints the bounded state this answer read. An empty
	// value (rows written before the column existed) never matches, so the
	// answer is recomputed once.
	EvidenceHash     string    `json:"-"`
	Model            string    `json:"model,omitempty"`
	ModelVersion     string    `json:"model_version,omitempty"`
	Value            Value     `json:"value"`
	Probability      *float64  `json:"probability,omitempty"`
	Confidence       *float64  `json:"confidence,omitempty"`
	Method           string    `json:"method"`
	Provider         string    `json:"provider,omitempty"`
	EvidenceEventIDs []string  `json:"evidence_event_ids,omitempty"`
	ObservedAt       time.Time `json:"observed_at"`
	State            State     `json:"state"`
	Reason           string    `json:"reason,omitempty"`
}

// StateCounts is one label's coverage in a window. Every field is a count of
// latest observations, never a rate.
type StateCounts struct {
	Observed    int `json:"observed"`
	Compatible  int `json:"compatible"`
	NotObserved int `json:"not_observed"`
	Unavailable int `json:"unavailable"`
	// Stale counts latest rows written under another definition revision.
	// They are evidence of earlier work, never an observation of the current
	// label, so they are excluded from Total.
	Stale int `json:"stale,omitempty"`
}

// Total is the number of scopes answered for the current definition.
func (c StateCounts) Total() int {
	return c.Observed + c.Compatible + c.NotObserved + c.Unavailable
}

// Store persists run_annotations.
type Store struct {
	db  sqlcompat.DB
	now func() time.Time
}

// NewStore wraps the shared SQL database seam.
func NewStore(db sqlcompat.DB) *Store {
	return &Store{db: db, now: time.Now}
}

// EnsureSchema applies the package schema idempotently. It is safe to call on
// every start and on each dev-routed pool.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("annotation store has no database")
	}
	if _, err := s.db.ExecContext(ctx, Schema()); err != nil {
		return fmt.Errorf("apply run_annotations schema: %w", err)
	}
	return s.ensureEvidenceHashColumn(ctx)
}

// ensureEvidenceHashColumn adds the evidence fingerprint to a database created
// before it existed. Older rows keep an empty fingerprint, which never matches
// a computed one, so each run is re-annotated once and cached afterwards.
func (s *Store) ensureEvidenceHashColumn(ctx context.Context) error {
	rows, err := s.db.QueryxContext(ctx, `PRAGMA table_info(run_annotations)`)
	if err != nil {
		return fmt.Errorf("inspect run_annotations schema: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			position int
			name     string
			kind     string
			notNull  int
			fallback any
			primary  int
		)
		if err := rows.Scan(&position, &name, &kind, &notNull, &fallback, &primary); err != nil {
			return err
		}
		if name == "evidence_hash" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE run_annotations ADD COLUMN evidence_hash TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add evidence_hash column: %w", err)
	}
	return nil
}

// Save upserts observations. A newer observation replaces an older one for the
// same key; an older replayed observation does not move the row backwards.
func (s *Store) Save(ctx context.Context, annotations []Annotation) error {
	if len(annotations) == 0 {
		return nil
	}
	for _, annotation := range annotations {
		if annotation.ObservedAt.IsZero() {
			annotation.ObservedAt = s.now()
		}
		annotation.ObservedAt = annotation.ObservedAt.UTC()
		valueJSON, err := json.Marshal(annotation.Value)
		if err != nil {
			return fmt.Errorf("encode annotation value for %s: %w", annotation.LabelID, err)
		}
		evidence := annotation.EvidenceEventIDs
		if evidence == nil {
			evidence = []string{}
		}
		evidenceJSON, err := json.Marshal(evidence)
		if err != nil {
			return fmt.Errorf("encode evidence for %s: %w", annotation.LabelID, err)
		}
		var probability any
		if annotation.Probability != nil {
			probability = *annotation.Probability
		}
		var confidence any
		if annotation.Confidence != nil {
			confidence = *annotation.Confidence
		}
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO run_annotations (scope_kind, scope_id, label_id, definition_hash, evidence_hash, model, model_version,
  value_json, probability, confidence, method, provider, evidence_event_ids_json, observed_at, state, reason)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (scope_kind, scope_id, label_id, definition_hash, model, model_version) DO UPDATE SET
  evidence_hash = excluded.evidence_hash,
  value_json = excluded.value_json,
  probability = excluded.probability,
  confidence = excluded.confidence,
  method = excluded.method,
  provider = excluded.provider,
  evidence_event_ids_json = excluded.evidence_event_ids_json,
  observed_at = excluded.observed_at,
  state = excluded.state,
  reason = excluded.reason
WHERE excluded.observed_at >= run_annotations.observed_at`,
			string(annotation.ScopeKind), annotation.ScopeID, annotation.LabelID, annotation.DefinitionHash,
			annotation.EvidenceHash, annotation.Model, annotation.ModelVersion, string(valueJSON), probability, confidence,
			annotation.Method, annotation.Provider, string(evidenceJSON), formatTime(annotation.ObservedAt),
			string(annotation.State), annotation.Reason); err != nil {
			return fmt.Errorf("save annotation %s/%s: %w", annotation.ScopeID, annotation.LabelID, err)
		}
	}
	return nil
}

const annotationColumns = `scope_kind, scope_id, label_id, definition_hash, evidence_hash, model, model_version,
  value_json, probability, confidence, method, provider, evidence_event_ids_json, observed_at, state, reason`

// Current returns the latest observation for an exact definition hash,
// regardless of model. A caller uses it as a compute cache: the definition did
// not change, so the previous answer is still an answer.
func (s *Store) Current(ctx context.Context, scopeKind Scope, scopeID, labelID, hash string) (*Annotation, error) {
	row := s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT %s FROM run_annotations
WHERE scope_kind = ? AND scope_id = ? AND label_id = ? AND definition_hash = ?
  AND state IN ('observed', 'compatible')
ORDER BY observed_at DESC, rowid DESC LIMIT 1`, annotationColumns),
		string(scopeKind), scopeID, labelID, hash)
	return scanAnnotation(row)
}

// LatestForLabel returns the newest row for one label under any hash.
func (s *Store) LatestForLabel(ctx context.Context, scopeKind Scope, scopeID, labelID string) (*Annotation, error) {
	row := s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT %s FROM run_annotations
WHERE scope_kind = ? AND scope_id = ? AND label_id = ?
ORDER BY observed_at DESC, rowid DESC LIMIT 1`, annotationColumns),
		string(scopeKind), scopeID, labelID)
	return scanAnnotation(row)
}

// Compatible returns the newest observed row whose hash is listed as a
// compatible predecessor. The caller supplies only hashes the pack declared.
func (s *Store) Compatible(ctx context.Context, scopeKind Scope, scopeID, labelID string, hashes []string) (*Annotation, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(hashes)), ",")
	args := []any{string(scopeKind), scopeID, labelID}
	for _, hash := range hashes {
		args = append(args, hash)
	}
	row := s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT %s FROM run_annotations
WHERE scope_kind = ? AND scope_id = ? AND label_id = ?
  AND definition_hash IN (%s) AND state IN ('observed', 'compatible')
ORDER BY observed_at DESC, rowid DESC LIMIT 1`, annotationColumns, placeholders), args...)
	return scanAnnotation(row)
}

// LatestForScope returns the newest row per label for one scope.
func (s *Store) LatestForScope(ctx context.Context, scopeKind Scope, scopeID string) ([]Annotation, error) {
	rows, err := s.db.QueryxContext(ctx, fmt.Sprintf(`
SELECT %s FROM (
  SELECT %s, ROW_NUMBER() OVER (PARTITION BY label_id ORDER BY observed_at DESC, rowid DESC) AS rn
  FROM run_annotations WHERE scope_kind = ? AND scope_id = ?
) WHERE rn = 1 ORDER BY label_id`, annotationColumns, annotationColumns),
		string(scopeKind), scopeID)
	if err != nil {
		return nil, fmt.Errorf("read annotations for %s: %w", scopeID, err)
	}
	defer rows.Close()
	var out []Annotation
	for rows.Next() {
		annotation, err := scanAnnotationRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *annotation)
	}
	return out, rows.Err()
}

// Coverage counts latest state per label in the window. Since zero means all
// time. The scope kind is run in this slice.
func (s *Store) Coverage(ctx context.Context, since time.Time) (map[string]StateCounts, error) {
	return s.CoverageWindow(ctx, since, time.Time{})
}

// CoverageWindow counts latest state per label in [since, until). Zero bounds
// mean unbounded on that side.
func (s *Store) CoverageWindow(ctx context.Context, since, until time.Time) (map[string]StateCounts, error) {
	return s.CoverageWindowFor(ctx, since, until, nil)
}

// CoverageWindowFor counts latest state per label, separating rows that belong
// to another definition revision. Without current hashes every row counts as
// its stored state; callers with a compiled pack pass them so a definition
// change cannot keep an old answer looking like a current observation.
func (s *Store) CoverageWindowFor(ctx context.Context, since, until time.Time, current map[string]string) (map[string]StateCounts, error) {
	args := []any{string(ScopeRun)}
	where := `scope_kind = ?`
	if !since.IsZero() {
		where += ` AND observed_at >= ?`
		args = append(args, formatTime(since))
	}
	if !until.IsZero() {
		where += ` AND observed_at <= ?`
		args = append(args, formatTime(until))
	}
	rows, err := s.db.QueryxContext(ctx, fmt.Sprintf(`
SELECT label_id, state, definition_hash, COUNT(*) FROM (
  SELECT label_id, state, definition_hash, ROW_NUMBER() OVER (
    PARTITION BY scope_id, label_id ORDER BY observed_at DESC, rowid DESC
  ) AS rn
  FROM run_annotations WHERE %s
) WHERE rn = 1 GROUP BY label_id, state, definition_hash`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("annotation coverage: %w", err)
	}
	defer rows.Close()
	out := map[string]StateCounts{}
	for rows.Next() {
		var (
			labelID string
			state   string
			hash    string
			count   int
		)
		if err := rows.Scan(&labelID, &state, &hash, &count); err != nil {
			return nil, err
		}
		counts := out[labelID]
		if current != nil && current[labelID] != hash {
			counts.Stale += count
			out[labelID] = counts
			continue
		}
		// Several definition revisions can share a state, so accumulate.
		switch State(state) {
		case StateObserved:
			counts.Observed += count
		case StateCompatible:
			counts.Compatible += count
		case StateNotObserved:
			counts.NotObserved += count
		case StateUnavailable:
			counts.Unavailable += count
		}
		out[labelID] = counts
	}
	return out, rows.Err()
}

// RunCoverage counts distinct runs with any latest annotation and runs with at
// least one observed/compatible latest annotation in the window. Eligible is
// the annotated population in this slice: WS-E may supply a wider candidate
// census later without changing this shape.
func (s *Store) RunCoverage(ctx context.Context, since, until time.Time) (eligible, observed int, err error) {
	return s.RunCoverageFor(ctx, since, until, nil)
}

// RunCoverageFor counts runs whose latest annotation belongs to the current
// definition. A run whose newest row is from an older revision has no answer
// for this pack, so counting it as observed would overstate coverage after a
// definition change.
func (s *Store) RunCoverageFor(ctx context.Context, since, until time.Time, current map[string]string) (eligible, observed int, err error) {
	// The current-definition predicate is printed before the window predicate,
	// so its arguments must be collected first.
	var currentArgs []any
	currentMatch := "1 = 1"
	if len(current) > 0 {
		// Sort the label ids before building both the SQL and its arguments so
		// the placeholders stay aligned regardless of map iteration order.
		labelIDs := make([]string, 0, len(current))
		for labelID := range current {
			labelIDs = append(labelIDs, labelID)
		}
		sort.Strings(labelIDs)
		parts := make([]string, 0, len(labelIDs))
		for _, labelID := range labelIDs {
			parts = append(parts, "(label_id = ? AND definition_hash = ?)")
			currentArgs = append(currentArgs, labelID, current[labelID])
		}
		currentMatch = "(" + strings.Join(parts, " OR ") + ")"
	}
	args := append([]any{}, currentArgs...)
	args = append(args, string(ScopeRun))
	where := `scope_kind = ?`
	if !since.IsZero() {
		where += ` AND observed_at >= ?`
		args = append(args, formatTime(since))
	}
	if !until.IsZero() {
		where += ` AND observed_at <= ?`
		args = append(args, formatTime(until))
	}
	err = s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT COUNT(DISTINCT scope_id),
  COUNT(DISTINCT CASE WHEN state IN ('observed', 'compatible') AND %s THEN scope_id END)
FROM (
  SELECT scope_id, label_id, state, definition_hash, ROW_NUMBER() OVER (
    PARTITION BY scope_id, label_id ORDER BY observed_at DESC, rowid DESC
  ) AS rn
  FROM run_annotations WHERE %s
) WHERE rn = 1`, currentMatch, where), args...).Scan(&eligible, &observed)
	if err != nil {
		return 0, 0, fmt.Errorf("run annotation coverage: %w", err)
	}
	return eligible, observed, nil
}

// FrictionSelector is the friction rule a label declares, expressed in the
// stored shape so one SQL statement can count positives. LevelBound is set for
// score labels: it is the inclusive level index where the friction region
// starts (LevelAtLeast) or ends.
type FrictionSelector struct {
	Threshold    float64
	LevelBound   *int
	LevelAtLeast bool
	// ScoreBound is the raw score limit for observations that predate stored
	// level probabilities. A calibrated answer is judged by its region mass;
	// a legacy one falls back to the score plus its top probability.
	ScoreBound *float64
}

// Positives counts the latest observation per scope whose value clears the
// label's friction selector. A score label is judged by the probability mass
// in its friction region, because the winning level's probability on a
// multi-level scale is structurally low; a choice or noul label keeps the top
// probability.
func (s *Store) Positives(ctx context.Context, since, until time.Time, labelID string, selector FrictionSelector) (int, error) {
	return s.PositivesFor(ctx, since, until, labelID, "", selector)
}

// PositivesFor counts friction positives for one definition revision. An
// empty hash keeps the legacy behaviour for callers that have no compiled
// pack; with a hash, a stale row can never contribute to a rate.
func (s *Store) PositivesFor(ctx context.Context, since, until time.Time, labelID, definitionHash string, selector FrictionSelector) (int, error) {
	args := []any{string(ScopeRun), labelID}
	where := `scope_kind = ? AND label_id = ?`
	if definitionHash != "" {
		where += ` AND definition_hash = ?`
		args = append(args, definitionHash)
	}
	if !since.IsZero() {
		where += ` AND observed_at >= ?`
		args = append(args, formatTime(since))
	}
	if !until.IsZero() {
		where += ` AND observed_at <= ?`
		args = append(args, formatTime(until))
	}
	condition := `probability IS NOT NULL AND probability >= ?`
	if selector.LevelBound == nil {
		args = append(args, selector.Threshold)
	}
	if selector.LevelBound != nil {
		comparison := "<="
		if selector.LevelAtLeast {
			comparison = ">="
		}
		condition = `value_json IS NOT NULL
  AND json_extract(value_json, '$.score') IS NOT NULL
  AND json_extract(value_json, '$.score') ` + comparison + ` ?
  AND ((json_type(value_json, '$.probabilities') IS NOT NULL
    AND (SELECT COALESCE(SUM(CAST(level.value AS REAL)), 0)
         FROM json_each(json_extract(value_json, '$.probabilities')) AS level
         WHERE CAST(level.key AS INTEGER) ` + comparison + ` ?) >= ?)
    OR (json_type(value_json, '$.probabilities') IS NULL AND probability >= ?))`
		scoreBound := 0.0
		if selector.ScoreBound != nil {
			scoreBound = *selector.ScoreBound
		}
		args = append(args, scoreBound, *selector.LevelBound, selector.Threshold, selector.Threshold)
	}
	var count int
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT COUNT(*) FROM (
  SELECT probability, value_json, ROW_NUMBER() OVER (
    PARTITION BY scope_id, label_id ORDER BY observed_at DESC, rowid DESC
  ) AS rn
  FROM run_annotations WHERE %s
) WHERE rn = 1 AND %s`, where, condition), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count positives for %s: %w", labelID, err)
	}
	return count, nil
}

// Count returns the total row count, for diagnostics and tests.
func (s *Store) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM run_annotations`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAnnotation(row rowScanner) (*Annotation, error) {
	annotation, err := scanAnnotationInto(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return annotation, err
}

func scanAnnotationRows(rows *sqlx.Rows) (*Annotation, error) {
	return scanAnnotationInto(rows)
}

func scanAnnotationInto(row rowScanner) (*Annotation, error) {
	var (
		scopeKind, scopeID, labelID, hash, evidenceHash, model, modelVersion string
		valueJSON, method, provider, evidenceJSON, observedAt, state, reason string
		probability, confidence                                              sql.NullFloat64
	)
	if err := row.Scan(&scopeKind, &scopeID, &labelID, &hash, &evidenceHash, &model, &modelVersion,
		&valueJSON, &probability, &confidence, &method, &provider, &evidenceJSON,
		&observedAt, &state, &reason); err != nil {
		return nil, err
	}
	var value Value
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return nil, fmt.Errorf("decode annotation value: %w", err)
	}
	var evidence []string
	if err := json.Unmarshal([]byte(evidenceJSON), &evidence); err != nil {
		return nil, fmt.Errorf("decode annotation evidence: %w", err)
	}
	at, err := parseStoreTime(observedAt)
	if err != nil {
		return nil, err
	}
	annotation := &Annotation{
		ScopeKind: Scope(scopeKind), ScopeID: scopeID, LabelID: labelID, DefinitionHash: hash,
		EvidenceHash: evidenceHash,
		Model: model, ModelVersion: modelVersion, Value: value, Method: method, Provider: provider,
		EvidenceEventIDs: evidence, ObservedAt: at, State: State(state), Reason: reason,
	}
	if probability.Valid {
		annotation.Probability = &probability.Float64
	}
	if confidence.Valid {
		annotation.Confidence = &confidence.Float64
	}
	return annotation, nil
}

// sqlxRows is the concrete iterator shape sqlcompat exposes.
type sqlxRows = sqlx.Rows

// annotationTimeLayout is fixed-width so SQLite TEXT comparisons sort
// chronologically. RFC3339Nano drops trailing zeros, which makes a fractional
// timestamp sort before the same second without a fraction.
const annotationTimeLayout = "2006-01-02T15:04:05.000000000Z07:00"

func formatTime(at time.Time) string { return at.UTC().Format(annotationTimeLayout) }

func parseStoreTime(text string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
		if at, err := time.Parse(layout, text); err == nil {
			return at.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable annotation time %q", text)
}

// ObservedScopes lists the runs whose latest answer for one label is current
// and observed (or compatibly reused) in the window, newest first. It backs
// the blind gold draft: a run can be labelled only when the benchmark will
// find a stored answer to compare against.
func (s *Store) ObservedScopes(ctx context.Context, since, until time.Time, labelID, definitionHash string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = DefaultGoldDraft
	}
	args := []any{string(ScopeRun), labelID}
	where := `scope_kind = ? AND label_id = ?`
	if !since.IsZero() {
		where += ` AND observed_at >= ?`
		args = append(args, formatTime(since))
	}
	if !until.IsZero() {
		where += ` AND observed_at <= ?`
		args = append(args, formatTime(until))
	}
	args = append(args, definitionHash, limit)
	var scopes []string
	err := s.db.SelectContext(ctx, &scopes, fmt.Sprintf(`
SELECT scope_id FROM (
  SELECT scope_id, state, definition_hash, observed_at, ROW_NUMBER() OVER (
    PARTITION BY scope_id ORDER BY observed_at DESC, rowid DESC
  ) AS rn
  FROM run_annotations WHERE %s
) WHERE rn = 1 AND state IN ('observed', 'compatible') AND definition_hash = ?
ORDER BY observed_at DESC, scope_id LIMIT ?`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("list observed scopes for %s: %w", labelID, err)
	}
	return scopes, nil
}
