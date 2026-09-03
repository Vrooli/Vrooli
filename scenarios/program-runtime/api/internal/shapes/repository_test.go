package shapes

import (
	"context"
	"database/sql"
	"testing"
	"time"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	internalbindings "program-runtime/internal/bindings"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func newShapeDB(t *testing.T) *sql.DB {
	t.Helper()
	d := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internalbindings.Schema), apidb.SchemaProviderFunc(Schema)))
	_, err := d.Exec(`CREATE TABLE programs (id TEXT PRIMARY KEY, session_id TEXT, source TEXT, provenance TEXT, status TEXT, created_at TEXT)`)
	require.NoError(t, err)
	return d
}

func seedShapeProgram(t *testing.T, d *sql.DB, id, source string) {
	_, err := d.Exec(`INSERT INTO programs (id,session_id,source,provenance,status,created_at) VALUES (?,?,?,?,?,?)`, id, "session", source, "1", "succeeded", "2026-09-03T00:00:00Z")
	require.NoError(t, err)
	for n, binding := range []string{"demo/ops/read", "demo/ops/list"} {
		_, err = d.Exec(`INSERT INTO binding_invocations (invocation_id,binding_id,target_scenario,session_id,program_id,provenance,outcome,reason,latency_ms,occurred_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, id+string(rune('a'+n)), binding, "demo", "session", id, "agent", "success", "", 1, "2026-09-03T00:00:00Z")
		require.NoError(t, err)
	}
}

func TestObserveAggregatesAndTracksSessions(t *testing.T) {
	d := newShapeDB(t)
	seedShapeProgram(t, d, "program-one", "short")
	repo := NewRepository(d)
	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	outcome, err := repo.Observe(context.Background(), "program-one", "session", programsv1.Provenance_PROVENANCE_AGENT, now)
	require.NoError(t, err)
	require.Equal(t, OutcomeInserted, outcome)
	outcome, err = repo.Observe(context.Background(), "program-one", "session", programsv1.Provenance_PROVENANCE_AGENT, now.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, OutcomeExisting, outcome)
	rows, err := repo.List(context.Background(), Filter{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.EqualValues(t, 1, rows[0].Occurrences)
	require.EqualValues(t, 1, rows[0].Sessions)
}

func TestObserveSkipsInsufficientBindingsAndExcludesTestSessions(t *testing.T) {
	d := newShapeDB(t)
	seedShapeProgram(t, d, "program-two", "source")
	_, err := d.Exec(`DELETE FROM binding_invocations WHERE program_id=? AND binding_id=?`, "program-two", "demo/ops/list")
	require.NoError(t, err)
	repo := NewRepository(d)
	outcome, err := repo.Observe(context.Background(), "program-two", "test-session", programsv1.Provenance_PROVENANCE_TEST, time.Now())
	require.NoError(t, err)
	require.Equal(t, OutcomeSkipped, outcome)
}

type shapeCoverage struct{ value string }

type shapeEvents struct{ events []ShapeEvent }

func (s *shapeEvents) AppendShapeEvent(event ShapeEvent) { s.events = append(s.events, event) }

func (c shapeCoverage) CoveredBy([]string) string { return c.value }

func TestResolveCoverageMovesBothDirections(t *testing.T) {
	d := newShapeDB(t)
	seedShapeProgram(t, d, "program-three", "source")
	repo := NewRepository(d)
	_, err := repo.Observe(context.Background(), "program-three", "session", programsv1.Provenance_PROVENANCE_AGENT, time.Now())
	require.NoError(t, err)
	_, err = d.Exec(`UPDATE program_shapes SET state='nominated' WHERE shape_key=?`, "demo/ops/list|demo/ops/read")
	require.NoError(t, err)
	changed, err := repo.ResolveCoverage(context.Background(), shapeCoverage{value: "demo.contract"})
	require.NoError(t, err)
	require.EqualValues(t, 1, changed)
	row, err := repo.Get(context.Background(), "demo/ops/list|demo/ops/read")
	require.NoError(t, err)
	require.Equal(t, "covered", row.State)
	require.Equal(t, "demo.contract", row.CoveredBy)
	_, err = repo.ResolveCoverage(context.Background(), shapeCoverage{})
	require.NoError(t, err)
	row, err = repo.Get(context.Background(), "demo/ops/list|demo/ops/read")
	require.NoError(t, err)
	require.Equal(t, "observed", row.State)
}

func TestApplyGateRequiresRecurrenceAndSessions(t *testing.T) {
	d := newShapeDB(t)
	_, err := d.Exec(`INSERT INTO program_shapes(shape_key,binding_ids,binding_count,occurrences,sessions,first_seen,last_seen,exemplar_program_id,state) VALUES('gate','["a","b"]',2,3,2,'2026-01-01','2026-01-01','p','observed')`)
	require.NoError(t, err)
	repo := NewRepository(d)
	result, err := repo.ApplyGate(context.Background(), time.Now())
	require.NoError(t, err)
	require.EqualValues(t, 1, result.Nominated)
	row, err := repo.Get(context.Background(), "gate")
	require.NoError(t, err)
	require.Equal(t, "nominated", row.State)
}

func TestShapeTransitionsEmitOnceAndCarryCoverage(t *testing.T) {
	d := newShapeDB(t)
	for _, id := range []string{"p1", "p2", "p3"} {
		seedShapeProgram(t, d, id, "source")
	}
	events := &shapeEvents{}
	repo := NewRepository(d)
	repo.SetEventSink(events)
	for n, id := range []string{"p1", "p2", "p3"} {
		_, err := repo.Observe(context.Background(), id, "session-"+string(rune('1'+n)), programsv1.Provenance_PROVENANCE_AGENT, time.Now())
		require.NoError(t, err)
	}
	require.Len(t, events.events, 1)
	require.Equal(t, telemetryv1.EventKind_NOMINATION, events.events[0].Kind)
	_, err := repo.ResolveCoverage(context.Background(), shapeCoverage{value: "demo.contract"})
	require.NoError(t, err)
	require.Len(t, events.events, 2)
	require.Equal(t, telemetryv1.EventKind_COVERAGE_MISS, events.events[1].Kind)
	require.Equal(t, "demo.contract", events.events[1].CoveringContractID)
	_, err = repo.ResolveCoverage(context.Background(), shapeCoverage{value: "demo.contract"})
	require.NoError(t, err)
	require.Len(t, events.events, 2)
}

func TestCoverageMissRequiresRecurrenceThreshold(t *testing.T) {
	d := newShapeDB(t)
	seedShapeProgram(t, d, "below-threshold", "source")
	events := &shapeEvents{}
	repo := NewRepository(d)
	repo.SetEventSink(events)
	_, err := repo.Observe(context.Background(), "below-threshold", "session-1", programsv1.Provenance_PROVENANCE_AGENT, time.Now())
	require.NoError(t, err)
	_, err = d.Exec(`UPDATE program_shapes SET occurrences=2, sessions=2 WHERE shape_key=?`, "demo/ops/list|demo/ops/read")
	require.NoError(t, err)
	_, err = repo.ResolveCoverage(context.Background(), shapeCoverage{value: "demo.contract"})
	require.NoError(t, err)
	require.Empty(t, events.events)
}
