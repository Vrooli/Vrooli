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
	// Inputs are the proof keys of the observations a composite over labels read
	// (transitively), so its proof can be derived from theirs at read time.
	Inputs []InputKey `json:"inputs,omitempty"`
}

// Annotation is one immutable observation keyed by definition and model.
type Annotation struct {
	ScopeKind Scope  `json:"scope_kind"`
	ScopeID   string `json:"scope_id"`
	// RunID is the run an observation belongs to. For a run-scope row it equals
	// ScopeID; for a message-scope row it is the parent run.
	RunID          string `json:"run_id,omitempty"`
	LabelID        string `json:"label_id"`
	DefinitionHash string `json:"definition_hash"`
	// EvidenceHash fingerprints the bounded state this answer read. An empty
	// value (rows written before the column existed) never matches, so the
	// answer is recomputed once.
	EvidenceHash string `json:"-"`
	// RequestedRole is the decision role the engine asked. It differs from
	// ModelVersion, which is the resource role that actually answered, so a
	// label whose role changed is re-asked instead of served a stale answer.
	RequestedRole    string   `json:"requested_role,omitempty"`
	Model            string   `json:"model,omitempty"`
	ModelVersion     string   `json:"model_version,omitempty"`
	Value            Value    `json:"value"`
	Probability      *float64 `json:"probability,omitempty"`
	Confidence       *float64 `json:"confidence,omitempty"`
	Method           string   `json:"method"`
	Provider         string   `json:"provider,omitempty"`
	EvidenceEventIDs []string `json:"evidence_event_ids,omitempty"`
	// OccurredAt is when a message-scope subject was written; zero for a run.
	OccurredAt time.Time `json:"occurred_at,omitzero"`
	ObservedAt time.Time `json:"observed_at"`
	State      State     `json:"state"`
	Reason     string    `json:"reason,omitempty"`
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

// Table names. Observations are production answers; trials hold bake-off
// answers produced by a pinned candidate and never mix with them; provenance
// holds one row per annotated run describing whose text it carried.
const (
	TableObservations = "run_annotations"
	TableTrials       = "run_annotation_trials"
)

// Store persists observations in one table. The production store and the
// trials store share every query; only the table differs, which is what keeps a
// pinned candidate's answers from overwriting or mixing with production ones.
type Store struct {
	db    sqlcompat.DB
	now   func() time.Time
	table string
}

// NewStore wraps the shared SQL database seam over the production table.
func NewStore(db sqlcompat.DB) *Store {
	return &Store{db: db, now: time.Now, table: TableObservations}
}

// Trials returns the store of pinned-candidate (bake-off) observations. It has
// the same shape and the same queries as the production store.
func (s *Store) Trials() *Store {
	return &Store{db: s.db, now: s.now, table: TableTrials}
}

// Table names the table this store reads and writes.
func (s *Store) Table() string { return s.table }

// EnsureSchema applies the package schema idempotently. It is safe to call on
// every start and on each dev-routed pool. Columns added after a table first
// shipped are added in place; older rows keep the column defaults.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("annotation store has no database")
	}
	if _, err := s.db.ExecContext(ctx, Schema()); err != nil {
		return fmt.Errorf("apply run_annotations schema: %w", err)
	}
	for _, table := range []string{TableObservations, TableTrials} {
		if err := s.ensureColumns(ctx, table); err != nil {
			return err
		}
	}
	// Provenance rows written before the policy hash existed carry an empty
	// one, which never matches: they read as stale until recomputed.
	return s.addMissingColumns(ctx, tableProvenance, addedProvenanceColumns)
}

// tableProvenance is the per-run authorship table.
const tableProvenance = "run_annotation_provenance"

var addedProvenanceColumns = []struct{ name, ddl string }{
	{"policy_hash", `TEXT NOT NULL DEFAULT ''`},
}

// addedColumns are the columns added after the first release of the table, with
// the DDL that adds them.
var addedColumns = []struct{ name, ddl string }{
	{"evidence_hash", `TEXT NOT NULL DEFAULT ''`},
	{"run_id", `TEXT NOT NULL DEFAULT ''`},
	{"requested_role", `TEXT NOT NULL DEFAULT ''`},
	{"occurred_at", `TEXT NOT NULL DEFAULT ''`},
}

// ensureColumns adds any missing column. Rows written before evidence_hash
// existed carry an empty fingerprint, which never matches a computed one, so
// each run is re-annotated once and cached afterwards. Run-scope rows get their
// run id from their scope id.
func (s *Store) ensureColumns(ctx context.Context, table string) error {
	if err := s.addMissingColumns(ctx, table, addedColumns); err != nil {
		return err
	}
	// The run index is created here, not in the schema, because the column may
	// have only just been added.
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_run ON %s(run_id, label_id, observed_at DESC)`, table, table)); err != nil {
		return fmt.Errorf("index %s run id: %w", table, err)
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_occurred ON %s(label_id, occurred_at)`, table, table)); err != nil {
		return fmt.Errorf("index %s occurred_at: %w", table, err)
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_role ON %s(requested_role, observed_at DESC)`, table, table)); err != nil {
		return fmt.Errorf("index %s requested_role: %w", table, err)
	}
	return nil
}

// addMissingColumns adds each column of added the table lacks.
func (s *Store) addMissingColumns(ctx context.Context, table string, added []struct{ name, ddl string }) error {
	have := map[string]bool{}
	rows, err := s.db.QueryxContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return fmt.Errorf("inspect %s schema: %w", table, err)
	}
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
			rows.Close()
			return err
		}
		have[name] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, column := range added {
		if have[column.name] {
			continue
		}
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column.name, column.ddl)); err != nil {
			return fmt.Errorf("add %s.%s: %w", table, column.name, err)
		}
		if column.name == "run_id" {
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET run_id = scope_id WHERE run_id = '' AND scope_kind = 'run'`, table)); err != nil {
				return fmt.Errorf("backfill %s.run_id: %w", table, err)
			}
		}
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
		runID := annotation.RunID
		if runID == "" && annotation.ScopeKind == ScopeRun {
			runID = annotation.ScopeID
		}
		occurred := ""
		if !annotation.OccurredAt.IsZero() {
			occurred = formatTime(annotation.OccurredAt)
		}
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %[1]s (scope_kind, scope_id, run_id, label_id, definition_hash, evidence_hash, requested_role, model, model_version,
  value_json, probability, confidence, method, provider, evidence_event_ids_json, occurred_at, observed_at, state, reason)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (scope_kind, scope_id, label_id, definition_hash, model, model_version) DO UPDATE SET
  run_id = excluded.run_id,
  evidence_hash = excluded.evidence_hash,
  requested_role = excluded.requested_role,
  value_json = excluded.value_json,
  probability = excluded.probability,
  confidence = excluded.confidence,
  method = excluded.method,
  provider = excluded.provider,
  evidence_event_ids_json = excluded.evidence_event_ids_json,
  occurred_at = excluded.occurred_at,
  observed_at = excluded.observed_at,
  state = excluded.state,
  reason = excluded.reason
WHERE excluded.observed_at >= %[1]s.observed_at`, s.table),
			string(annotation.ScopeKind), annotation.ScopeID, runID, annotation.LabelID, annotation.DefinitionHash,
			annotation.EvidenceHash, annotation.RequestedRole, annotation.Model, annotation.ModelVersion, string(valueJSON), probability, confidence,
			annotation.Method, annotation.Provider, string(evidenceJSON), occurred, formatTime(annotation.ObservedAt),
			string(annotation.State), annotation.Reason); err != nil {
			return fmt.Errorf("save annotation %s/%s: %w", annotation.ScopeID, annotation.LabelID, err)
		}
	}
	return nil
}

const annotationColumns = `scope_kind, scope_id, run_id, label_id, definition_hash, evidence_hash, requested_role, model, model_version,
  value_json, probability, confidence, method, provider, evidence_event_ids_json, occurred_at, observed_at, state, reason`

// Current returns the latest observed or compatible row for an exact definition
// hash, regardless of model. A caller uses it as a compute cache: the
// definition did not change, so the previous answer is still an answer.
func (s *Store) Current(ctx context.Context, scopeKind Scope, scopeID, labelID, hash string) (*Annotation, error) {
	row := s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT %s FROM %s
WHERE scope_kind = ? AND scope_id = ? AND label_id = ? AND definition_hash = ?
  AND state IN ('observed', 'compatible')
ORDER BY observed_at DESC, rowid DESC LIMIT 1`, annotationColumns, s.table),
		string(scopeKind), scopeID, labelID, hash)
	return scanAnnotation(row)
}

// CurrentFor is Current restricted to one resource role: the cache of a pinned
// evaluation, which must reuse only the candidate it measures.
func (s *Store) CurrentFor(ctx context.Context, scopeKind Scope, scopeID, labelID, hash, resourceRole string) (*Annotation, error) {
	row := s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT %s FROM %s
WHERE scope_kind = ? AND scope_id = ? AND label_id = ? AND definition_hash = ? AND model_version = ?
  AND state IN ('observed', 'compatible')
ORDER BY observed_at DESC, rowid DESC LIMIT 1`, annotationColumns, s.table),
		string(scopeKind), scopeID, labelID, hash, resourceRole)
	return scanAnnotation(row)
}

// Compatible returns the newest observed row whose hash is listed as a
// compatible predecessor. The caller supplies only hashes the pack declared.
func (s *Store) Compatible(ctx context.Context, scopeKind Scope, scopeID, labelID string, hashes []string) (*Annotation, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	args := []any{string(scopeKind), scopeID, labelID}
	for _, hash := range hashes {
		args = append(args, hash)
	}
	row := s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT %s FROM %s
WHERE scope_kind = ? AND scope_id = ? AND label_id = ?
  AND definition_hash IN (%s) AND state IN ('observed', 'compatible')
ORDER BY observed_at DESC, rowid DESC LIMIT 1`, annotationColumns, s.table, placeholders(len(hashes))), args...)
	return scanAnnotation(row)
}

// latestOrder ranks the rows of one (scope, label) newest first, except that an
// unavailable row never outranks an answer of the same definition: a transient
// failure (a gateway deadline, an exhausted budget) leaves what is already known
// standing. A row of a newer definition is a different answer and still wins.
// The query it orders must read the table as alias a.
func (s *Store) latestOrder() string {
	return fmt.Sprintf(`CASE WHEN a.state = 'unavailable' AND EXISTS (
    SELECT 1 FROM %s settled
    WHERE settled.scope_kind = a.scope_kind AND settled.scope_id = a.scope_id
      AND settled.label_id = a.label_id AND settled.definition_hash = a.definition_hash
      AND settled.state IN ('observed', 'compatible')
  ) THEN 1 ELSE 0 END, a.observed_at DESC, a.rowid DESC`, s.table)
}

// LatestForScope returns the newest row per label for one scope.
func (s *Store) LatestForScope(ctx context.Context, scopeKind Scope, scopeID string) ([]Annotation, error) {
	return s.queryAnnotations(ctx, fmt.Sprintf(`
SELECT %[1]s FROM (
  SELECT %[1]s, ROW_NUMBER() OVER (PARTITION BY label_id ORDER BY %[3]s) AS rn
  FROM %[2]s AS a WHERE scope_kind = ? AND scope_id = ?
) WHERE rn = 1 ORDER BY label_id`, annotationColumns, s.table, s.latestOrder()), string(scopeKind), scopeID)
}

// LatestMessagesForRun returns the newest row per (message, label) for the
// message-scope rows of one run, oldest message first.
func (s *Store) LatestMessagesForRun(ctx context.Context, runID string) ([]Annotation, error) {
	return s.queryAnnotations(ctx, fmt.Sprintf(`
SELECT %[1]s FROM (
  SELECT %[1]s, ROW_NUMBER() OVER (PARTITION BY scope_id, label_id ORDER BY %[3]s) AS rn
  FROM %[2]s AS a WHERE scope_kind = 'message' AND run_id = ?
) WHERE rn = 1 ORDER BY occurred_at, scope_id, label_id`, annotationColumns, s.table, s.latestOrder()), runID)
}

func (s *Store) queryAnnotations(ctx context.Context, query string, args ...any) ([]Annotation, error) {
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read annotations: %w", err)
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

// windowClause builds the observed_at bounds shared by every windowed read.
func windowClause(since, until time.Time, args []any) (string, []any) {
	clause := ""
	if !since.IsZero() {
		clause += ` AND observed_at >= ?`
		args = append(args, formatTime(since))
	}
	if !until.IsZero() {
		clause += ` AND observed_at <= ?`
		args = append(args, formatTime(until))
	}
	return clause, args
}

// ForEachLatest streams the newest row per (scope, label) observed in
// [since, until]; a zero bound is unbounded. A row's value is decoded only when
// wantValue says its label needs it, so a coverage pass over thousands of rows
// does not decode the vectors of labels that report no rate. Memory is one row.
func (s *Store) ForEachLatest(ctx context.Context, since, until time.Time, wantValue func(labelID string) bool, visit func(Annotation) error) error {
	clause, args := windowClause(since, until, nil)
	rows, err := s.db.QueryxContext(ctx, fmt.Sprintf(`
SELECT %[1]s FROM (
  SELECT %[1]s, ROW_NUMBER() OVER (
    PARTITION BY scope_kind, scope_id, label_id ORDER BY %[4]s
  ) AS rn
  FROM %[2]s AS a WHERE 1 = 1%[3]s
) WHERE rn = 1`, annotationColumns, s.table, clause, s.latestOrder()), args...)
	if err != nil {
		return fmt.Errorf("annotation coverage: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		annotation, err := scanAnnotationFiltered(rows, wantValue)
		if err != nil {
			return err
		}
		if err := visit(*annotation); err != nil {
			return err
		}
	}
	return rows.Err()
}

// RunCoverageFor counts distinct runs with any latest annotation and runs whose
// latest annotation is observed or compatible under the current definition. A
// run whose newest row is from an older revision has no answer for this pack,
// so counting it as observed would overstate coverage after a definition change.
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
	clause, args := windowClause(since, until, args)
	err = s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT COUNT(DISTINCT scope_id),
  COUNT(DISTINCT CASE WHEN state IN ('observed', 'compatible') AND %s THEN scope_id END)
FROM (
  SELECT scope_id, label_id, state, definition_hash, ROW_NUMBER() OVER (
    PARTITION BY scope_id, label_id ORDER BY %s
  ) AS rn
  FROM %s AS a WHERE scope_kind = ?%s
) WHERE rn = 1`, currentMatch, s.latestOrder(), s.table, clause), args...).Scan(&eligible, &observed)
	if err != nil {
		return 0, 0, fmt.Errorf("run annotation coverage: %w", err)
	}
	return eligible, observed, nil
}

// Count returns the total row count, for diagnostics and tests.
func (s *Store) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, s.table)).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// ObservedScope is one subject a label has a current answer for: the scope id
// and the run it belongs to (the same id for a run).
type ObservedScope struct {
	ScopeID string `db:"scope_id"`
	RunID   string `db:"run_id"`
}

// ObservedScopes lists the subjects (runs, or messages for a message-scope
// label) whose latest answer for one label is current for the definition hash
// and answered, newest first.
func (s *Store) ObservedScopes(ctx context.Context, scopeKind Scope, since, until time.Time, labelID, definitionHash string, limit int) ([]ObservedScope, error) {
	if limit <= 0 {
		limit = DefaultGoldDraft
	}
	clause, args := windowClause(since, until, []any{string(scopeKind), labelID})
	args = append(args, definitionHash, limit)
	var scopes []ObservedScope
	err := s.db.SelectContext(ctx, &scopes, fmt.Sprintf(`
SELECT scope_id, CASE WHEN run_id = '' THEN scope_id ELSE run_id END AS run_id FROM (
  SELECT scope_id, run_id, state, definition_hash, observed_at, ROW_NUMBER() OVER (
    PARTITION BY scope_id ORDER BY %s
  ) AS rn
  FROM %s AS a WHERE scope_kind = ? AND label_id = ?%s
) WHERE rn = 1 AND state IN ('observed', 'compatible') AND definition_hash = ?
ORDER BY observed_at DESC, scope_id LIMIT ?`, s.latestOrder(), s.table, clause), args...)
	if err != nil {
		return nil, fmt.Errorf("list observed scopes for %s: %w", labelID, err)
	}
	return scopes, nil
}

// maxScopesPerQuery keeps one IN list far below SQLite's variable limit.
const maxScopesPerQuery = 400

// LatestObservations returns the newest observed or compatible row per scope
// for one label definition. A non-empty model or resource role restricts the
// rows to that candidate; both empty reads whichever model answered last, which
// a benchmark then refuses when the rows span more than one.
func (s *Store) LatestObservations(ctx context.Context, scopeKind Scope, scopeIDs []string, labelID, hash, model, resourceRole string) (map[string]Annotation, error) {
	out := make(map[string]Annotation, len(scopeIDs))
	for start := 0; start < len(scopeIDs); start += maxScopesPerQuery {
		chunk := scopeIDs[start:min(start+maxScopesPerQuery, len(scopeIDs))]
		args := []any{string(scopeKind), labelID, hash}
		filter := ""
		if model != "" {
			filter += ` AND model = ?`
			args = append(args, model)
		}
		if resourceRole != "" {
			filter += ` AND model_version = ?`
			args = append(args, resourceRole)
		}
		for _, id := range chunk {
			args = append(args, id)
		}
		rows, err := s.queryAnnotations(ctx, fmt.Sprintf(`
SELECT %s FROM %s
WHERE scope_kind = ? AND label_id = ? AND definition_hash = ? AND state IN ('observed', 'compatible')%s
  AND scope_id IN (%s)
ORDER BY observed_at DESC, rowid DESC`, annotationColumns, s.table, filter, placeholders(len(chunk))), args...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if _, seen := out[row.ScopeID]; !seen {
				out[row.ScopeID] = row
			}
		}
	}
	return out, nil
}

// MessageObservations returns the newest observed or compatible row per
// (message, label) whose subject was written in [since, until], oldest first.
// It backs the per-message time series.
func (s *Store) MessageObservations(ctx context.Context, labelID, hash string, since, until time.Time) ([]Annotation, error) {
	args := []any{labelID, hash}
	clause := ""
	if !since.IsZero() {
		clause += ` AND occurred_at >= ?`
		args = append(args, formatTime(since))
	}
	if !until.IsZero() {
		clause += ` AND occurred_at <= ?`
		args = append(args, formatTime(until))
	}
	return s.queryAnnotations(ctx, fmt.Sprintf(`
SELECT %[1]s FROM (
  SELECT %[1]s, ROW_NUMBER() OVER (PARTITION BY scope_id ORDER BY observed_at DESC, rowid DESC) AS rn
  FROM %[2]s WHERE scope_kind = 'message' AND label_id = ? AND definition_hash = ?
    AND state IN ('observed', 'compatible')%[3]s
) WHERE rn = 1 ORDER BY occurred_at, scope_id`, annotationColumns, s.table, clause), args...)
}

// LastResolution is the provider, model and resource role that last answered a
// request for a decision role, read from the stored answers. It lets a restarted
// engine size its first call to a role from what the gateway resolved before,
// without the gateway having to be asked how it would walk the role.
func (s *Store) LastResolution(ctx context.Context, requestedRole string) (provider, model, resourceRole string, ok bool, err error) {
	row := s.db.QueryRowContext(ctx, fmt.Sprintf(`
SELECT provider, model, model_version FROM %s
WHERE requested_role = ? AND state = 'observed' AND model <> '' AND model_version <> ''
ORDER BY observed_at DESC LIMIT 1`, s.table), requestedRole)
	switch scanErr := row.Scan(&provider, &model, &resourceRole); {
	case errors.Is(scanErr, sql.ErrNoRows):
		return "", "", "", false, nil
	case scanErr != nil:
		return "", "", "", false, fmt.Errorf("read last resolution of %s: %w", requestedRole, scanErr)
	}
	return provider, model, resourceRole, true, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAnnotation(row rowScanner) (*Annotation, error) {
	annotation, err := scanAnnotationInto(row, nil)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return annotation, err
}

func scanAnnotationRows(rows *sqlx.Rows) (*Annotation, error) {
	return scanAnnotationInto(rows, nil)
}

func scanAnnotationFiltered(rows *sqlx.Rows, wantValue func(string) bool) (*Annotation, error) {
	return scanAnnotationInto(rows, wantValue)
}

func scanAnnotationInto(row rowScanner, wantValue func(string) bool) (*Annotation, error) {
	var (
		scopeKind, scopeID, runID, labelID, hash, evidenceHash, requestedRole, model, modelVersion string
		valueJSON, method, provider, evidenceJSON, occurredAt, observedAt, state, reason           string
		probability, confidence                                                                    sql.NullFloat64
	)
	if err := row.Scan(&scopeKind, &scopeID, &runID, &labelID, &hash, &evidenceHash, &requestedRole, &model, &modelVersion,
		&valueJSON, &probability, &confidence, &method, &provider, &evidenceJSON,
		&occurredAt, &observedAt, &state, &reason); err != nil {
		return nil, err
	}
	var value Value
	if wantValue == nil || wantValue(labelID) {
		if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
			return nil, fmt.Errorf("decode annotation value: %w", err)
		}
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
		ScopeKind: Scope(scopeKind), ScopeID: scopeID, RunID: runID, LabelID: labelID, DefinitionHash: hash,
		EvidenceHash: evidenceHash, RequestedRole: requestedRole,
		Model: model, ModelVersion: modelVersion, Value: value, Method: method, Provider: provider,
		EvidenceEventIDs: evidence, ObservedAt: at, State: State(state), Reason: reason,
	}
	if occurredAt != "" {
		if occurred, err := parseStoreTime(occurredAt); err == nil {
			annotation.OccurredAt = occurred
		}
	}
	if probability.Valid {
		annotation.Probability = &probability.Float64
	}
	if confidence.Valid {
		annotation.Confidence = &confidence.Float64
	}
	return annotation, nil
}

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
