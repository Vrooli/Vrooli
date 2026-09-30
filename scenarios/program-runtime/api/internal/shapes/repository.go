package shapes

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

var ErrNotFound = errors.New("program shape not found")

type Outcome string

const (
	OutcomeInserted    Outcome = "inserted"
	OutcomeIncremented Outcome = "incremented"
	OutcomeExisting    Outcome = "already_observed"
	OutcomeSkipped     Outcome = "skipped_insufficient_bindings"
)

type (
	Repository struct {
		db       SQLExecutor
		resolver CoverageResolver
		events   EventSink
	}
	Filter struct {
		MinOccurrences, MinSessions int64
		UncoveredOnly               bool
		State                       string
	}
	Shape struct {
		ShapeKey                                                                           string
		BindingIDs                                                                         []string
		BindingCount, Occurrences, AgentRuns, OperatorRuns, TestRuns, ReplayRuns, Sessions int64
		FirstSeen, LastSeen, ExemplarProgramID                                             string
		ExemplarBytes                                                                      int64
		CoveredBy, CoveredReason, State                                                    string
		DominantScenario                                                                   string
	}
)

type ShapeEvent struct {
	Kind                                 telemetryv1.EventKind
	ShapeKey                             string
	BindingIDs                           []string
	Occurrences, Sessions                int64
	DominantScenario, CoveringContractID string
	OccurredAt                           time.Time
}
type EventSink interface{ AppendShapeEvent(ShapeEvent) }

const (
	MinOccurrences = 3
	MinSessions    = 2
	ShapeWindow    = 30 * 24 * time.Hour
)

type CoverageResolver interface {
	CoveredBy(bindingIDs []string) string
}

type GateResult struct {
	Nominated int64
	Observed  int64
	Covered   int64
}

func NewRepository(db SQLExecutor, resolver ...CoverageResolver) *Repository {
	var coverage CoverageResolver
	if len(resolver) > 0 {
		coverage = resolver[0]
	}
	return &Repository{db: db, resolver: coverage}
}

func (r *Repository) SetEventSink(events EventSink) { r.events = events }

func (r *Repository) Observe(ctx context.Context, programID, sessionID string, provenance programsv1.Provenance, now time.Time) (Outcome, error) {
	ids, err := Derive(ctx, r.db, programID)
	if err != nil {
		return "", fmt.Errorf("derive shape: %w", err)
	}
	if len(ids) < 2 {
		return OutcomeSkipped, nil
	}
	key := Key(ids)
	marker, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO program_shape_observations(program_id,shape_key,observed_at) VALUES(?,?,?)`, programID, key, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return "", fmt.Errorf("record shape observation: %w", err)
	}
	if affected, _ := marker.RowsAffected(); affected == 0 {
		return OutcomeExisting, nil
	}
	encoded, _ := json.Marshal(ids)
	var source string
	if err := r.db.QueryRowContext(ctx, `SELECT source FROM programs WHERE id=?`, programID).Scan(&source); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	nowText := now.UTC().Format(time.RFC3339Nano)
	_, err = r.db.ExecContext(ctx, `INSERT INTO program_shapes (shape_key,binding_ids,binding_count,occurrences,agent_runs,operator_runs,test_runs,replay_runs,sessions,first_seen,last_seen,exemplar_program_id,exemplar_bytes) VALUES (?,?,?,1,?,?,?,?,0,?,?,?,?) ON CONFLICT(shape_key) DO UPDATE SET occurrences=program_shapes.occurrences+1, agent_runs=program_shapes.agent_runs+excluded.agent_runs, operator_runs=program_shapes.operator_runs+excluded.operator_runs, test_runs=program_shapes.test_runs+excluded.test_runs, replay_runs=program_shapes.replay_runs+excluded.replay_runs, last_seen=excluded.last_seen, exemplar_program_id=CASE WHEN program_shapes.exemplar_bytes=0 OR excluded.exemplar_bytes < program_shapes.exemplar_bytes THEN excluded.exemplar_program_id ELSE program_shapes.exemplar_program_id END, exemplar_bytes=CASE WHEN program_shapes.exemplar_bytes=0 OR excluded.exemplar_bytes < program_shapes.exemplar_bytes THEN excluded.exemplar_bytes ELSE program_shapes.exemplar_bytes END`, key, string(encoded), len(ids), provenanceCount(provenance, programsv1.Provenance_PROVENANCE_AGENT), provenanceCount(provenance, programsv1.Provenance_PROVENANCE_OPERATOR), provenanceCount(provenance, programsv1.Provenance_PROVENANCE_TEST), provenanceCount(provenance, programsv1.Provenance_PROVENANCE_REPLAY), nowText, nowText, programID, len(source))
	if err != nil {
		return "", fmt.Errorf("upsert shape: %w", err)
	}
	if provenance != programsv1.Provenance_PROVENANCE_TEST && strings.TrimSpace(sessionID) != "" {
		if _, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO program_shape_sessions(shape_key,session_id) VALUES(?,?)`, key, sessionID); err != nil {
			return "", err
		}
		if _, err := r.db.ExecContext(ctx, `UPDATE program_shapes SET sessions=(SELECT COUNT(*) FROM program_shape_sessions WHERE shape_key=?) WHERE shape_key=?`, key, key); err != nil {
			return "", err
		}
	}
	var occurrences int64
	if err := r.db.QueryRowContext(ctx, `SELECT occurrences FROM program_shapes WHERE shape_key=?`, key).Scan(&occurrences); err != nil {
		return "", err
	}
	if occurrences == 1 {
		if r.resolver != nil {
			_, _ = r.ResolveCoverage(ctx, r.resolver)
		}
		_, _ = r.ApplyGate(ctx, now)
		return OutcomeInserted, nil
	}
	if r.resolver != nil {
		_, _ = r.ResolveCoverage(ctx, r.resolver)
	}
	_, _ = r.ApplyGate(ctx, now)
	return OutcomeIncremented, nil
}

func (r *Repository) ResolveCoverage(ctx context.Context, resolver CoverageResolver) (int64, error) {
	if resolver == nil {
		return 0, errors.New("coverage resolver is required")
	}
	rows, err := r.List(ctx, Filter{})
	if err != nil {
		return 0, err
	}
	var changed int64
	for _, shape := range rows {
		coveredBy := resolver.CoveredBy(shape.BindingIDs)
		state := shape.State
		if coveredBy != "" {
			state = "covered"
		} else if state == "covered" {
			state = "observed"
		}
		if coveredBy == shape.CoveredBy && state == shape.State {
			continue
		}
		if _, err := r.db.ExecContext(ctx, `UPDATE program_shapes SET covered_by=?, state=? WHERE shape_key=?`, coveredBy, state, shape.ShapeKey); err != nil {
			return changed, err
		}
		if coveredBy != "" && coveredBy != shape.CoveredBy && shape.Occurrences >= MinOccurrences && shape.Sessions >= MinSessions {
			r.emit(ctx, telemetryv1.EventKind_COVERAGE_MISS, shape, coveredBy, time.Now().UTC())
		}
		changed++
	}
	return changed, nil
}

func (r *Repository) ApplyGate(ctx context.Context, now time.Time) (GateResult, error) {
	var result GateResult
	rows, err := r.db.QueryContext(ctx, `SELECT shape_key,binding_ids,occurrences,sessions,state FROM program_shapes WHERE covered_by='' AND occurrences >= ? AND sessions >= ?`, MinOccurrences, MinSessions)
	if err != nil {
		return result, err
	}
	var nominate []Shape
	for rows.Next() {
		var shape Shape
		var encoded string
		if err := rows.Scan(&shape.ShapeKey, &encoded, &shape.Occurrences, &shape.Sessions, &shape.State); err != nil {
			rows.Close()
			return result, err
		}
		if shape.State == "observed" {
			_ = json.Unmarshal([]byte(encoded), &shape.BindingIDs)
			nominate = append(nominate, shape)
		}
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	res, err := r.db.ExecContext(ctx, `UPDATE program_shapes SET state='nominated' WHERE state <> 'covered' AND covered_by='' AND occurrences >= ? AND sessions >= ?`, MinOccurrences, MinSessions)
	if err != nil {
		return result, err
	}
	result.Nominated, _ = res.RowsAffected()
	for _, shape := range nominate {
		r.emit(ctx, telemetryv1.EventKind_NOMINATION, shape, "", now)
	}
	res, err = r.db.ExecContext(ctx, `UPDATE program_shapes SET state='observed' WHERE state='nominated' AND (covered_by<>'' OR occurrences < ? OR sessions < ?)`, MinOccurrences, MinSessions)
	if err != nil {
		return result, err
	}
	result.Observed, _ = res.RowsAffected()
	var covered int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM program_shapes WHERE state='covered'`).Scan(&covered); err != nil {
		return result, err
	}
	result.Covered = covered
	return result, nil
}

func (r *Repository) emit(_ context.Context, kind telemetryv1.EventKind, shape Shape, coveredBy string, now time.Time) {
	if r.events == nil {
		return
	}
	r.events.AppendShapeEvent(ShapeEvent{Kind: kind, ShapeKey: shape.ShapeKey, BindingIDs: append([]string(nil), shape.BindingIDs...), Occurrences: shape.Occurrences, Sessions: shape.Sessions, DominantScenario: DominantScenario(shape.BindingIDs), CoveringContractID: coveredBy, OccurredAt: now.UTC()})
}

func (r *Repository) Expire(ctx context.Context, now time.Time, window time.Duration) (int64, error) {
	cutoff := now.UTC().Add(-window).Format(time.RFC3339Nano)
	type transactioner interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	}
	if db, ok := r.db.(transactioner); ok {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return 0, err
		}
		defer func() { _ = tx.Rollback() }()
		if _, err := tx.ExecContext(ctx, `DELETE FROM program_shape_sessions WHERE shape_key IN (SELECT shape_key FROM program_shapes WHERE state='observed' AND last_seen < ?)`, cutoff); err != nil {
			if strings.Contains(err.Error(), "no such table") {
				return 0, nil
			}
			return 0, err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM program_shapes WHERE state='observed' AND last_seen < ?`, cutoff)
		if err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM program_shape_sessions WHERE shape_key IN (SELECT shape_key FROM program_shapes WHERE state='observed' AND last_seen < ?)`, cutoff); err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return 0, nil
		}
		return 0, err
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM program_shapes WHERE state='observed' AND last_seen < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func provenanceCount(got, want programsv1.Provenance) int {
	if got == want {
		return 1
	}
	return 0
}

func (r *Repository) Get(ctx context.Context, key string) (Shape, error) {
	rows, err := r.List(ctx, Filter{})
	if err != nil {
		return Shape{}, err
	}
	for _, shape := range rows {
		if shape.ShapeKey == key {
			return shape, nil
		}
	}
	return Shape{}, ErrNotFound
}

func (r *Repository) List(ctx context.Context, filter Filter) ([]Shape, error) {
	query := `SELECT shape_key,binding_ids,binding_count,occurrences,agent_runs,operator_runs,test_runs,replay_runs,sessions,first_seen,last_seen,exemplar_program_id,exemplar_bytes,covered_by,covered_reason,state FROM program_shapes WHERE occurrences>=? AND sessions>=?`
	args := []any{filter.MinOccurrences, filter.MinSessions}
	if filter.UncoveredOnly {
		query += ` AND covered_by=''`
	}
	if filter.State != "" {
		query += ` AND state=?`
		args = append(args, filter.State)
	}
	query += ` ORDER BY occurrences DESC, shape_key`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Shape, 0)
	for rows.Next() {
		var shape Shape
		var encoded string
		if err := rows.Scan(&shape.ShapeKey, &encoded, &shape.BindingCount, &shape.Occurrences, &shape.AgentRuns, &shape.OperatorRuns, &shape.TestRuns, &shape.ReplayRuns, &shape.Sessions, &shape.FirstSeen, &shape.LastSeen, &shape.ExemplarProgramID, &shape.ExemplarBytes, &shape.CoveredBy, &shape.CoveredReason, &shape.State); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(encoded), &shape.BindingIDs); err != nil {
			return nil, err
		}
		shape.DominantScenario = DominantScenario(shape.BindingIDs)
		out = append(out, shape)
	}
	return out, rows.Err()
}

func DominantScenario(bindingIDs []string) string {
	counts := map[string]int{}
	for _, id := range bindingIDs {
		parts := strings.SplitN(id, "/", 2)
		if len(parts) == 2 && parts[0] != "" {
			counts[parts[0]]++
		}
	}
	best := ""
	for scenario, count := range counts {
		if count > counts[best] || (count == counts[best] && scenario < best) {
			best = scenario
		}
	}
	return best
}
