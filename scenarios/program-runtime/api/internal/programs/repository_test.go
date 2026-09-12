package programs

import (
	"context"
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

func newProgramsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	_, err := d.ExecContext(context.Background(), `CREATE TABLE refusals (binding_id TEXT, reason TEXT, provenance TEXT, occurred_at TEXT)`)
	require.NoError(t, err)
	_, err = d.ExecContext(context.Background(), `CREATE TABLE unresolved_binding_attempts (attempted_name TEXT, provenance TEXT, occurred_at TEXT)`)
	require.NoError(t, err)
	return d
}

func TestStartupReconcilesOnlyInterruptedPrograms(t *testing.T) { // [REQ:PRT-P1-006]
	ctx := context.Background()
	d := newProgramsTestDB(t)
	for _, repo := range []Repository{NewRepository(d), newMemoryRepository()} {
		for id, status := range map[string]programsv1.ProgramStatus{
			"accepted":  programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED,
			"running":   programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING,
			"completed": programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED,
		} {
			require.NoError(t, repo.Save(ctx, &programsv1.Program{
				Id: id, SessionId: "old-session", Source: "owner operation",
				Status: status, CreatedAt: "2026-09-06T05:20:15Z", Stdout: "retained evidence",
			}))
		}
		count, err := repo.InterruptUnfinished(ctx, "2026-09-06T05:23:18Z")
		require.NoError(t, err)
		require.EqualValues(t, 2, count)
		for _, id := range []string{"accepted", "running"} {
			p, err := repo.Get(ctx, id)
			require.NoError(t, err)
			require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, p.Status)
			require.Equal(t, programsv1.FailureCause_FAILURE_CAUSE_RUNTIME_INTERRUPTED, p.FailureCause)
			require.Equal(t, "retained evidence", p.Stdout)
			require.Contains(t, p.FailureDetail, "effects")
			require.Equal(t, "2026-09-06T05:23:18Z", p.CompletedAt)
		}
		p, err := repo.Get(ctx, "completed")
		require.NoError(t, err)
		require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, p.Status)
		count, err = repo.InterruptUnfinished(ctx, "2026-09-06T05:24:00Z")
		require.NoError(t, err)
		require.Zero(t, count)
	}
}

func TestSQLiteRepositoryRoundTripAfterRepositoryRestart(t *testing.T) { // [REQ:PRT-P1-006]
	ctx := context.Background()
	d := newProgramsTestDB(t)
	want := &programsv1.Program{Id: "prog_persisted", SessionId: "sess_1", Source: "raise ValueError()", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Status: programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, Stdout: "partial", FailureDetail: "field title: invalid", FailureShape: "field title", ContextBytes: 128, CreatedAt: time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC).Format(time.RFC3339Nano), OutputLimitBytes: 4096, ProgramName: "example.program", ProgramDigest: "digest-1", CallerRunId: "run-1", CallerAgentProfile: "profile", CallerSkillId: "skill", CallerHarness: "cli"}
	require.NoError(t, NewRepository(d).Save(ctx, want))

	got, err := NewRepository(d).Get(ctx, want.Id)
	require.NoError(t, err)
	require.Equal(t, want.Id, got.Id)
	require.Equal(t, want.Source, got.Source)
	require.Equal(t, want.FailureDetail, got.FailureDetail)
	require.Equal(t, want.Provenance, got.Provenance)
	require.Equal(t, want.ContextBytes, got.ContextBytes)
	require.Equal(t, want.ProgramName, got.ProgramName)
	require.Equal(t, want.ProgramDigest, got.ProgramDigest)
	require.Equal(t, want.CallerRunId, got.CallerRunId)
	require.Equal(t, want.CallerAgentProfile, got.CallerAgentProfile)
	require.Equal(t, want.CallerSkillId, got.CallerSkillId)
	require.Equal(t, want.CallerHarness, got.CallerHarness)
}

func TestSQLiteRepositoryLeavesAbsentCallerEmpty(t *testing.T) { // [REQ:PRT-P1-008]
	ctx := context.Background()
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: "prog_no_caller", SessionId: "sess", Source: "print(1)", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, CreatedAt: "2026-08-11T14:00:00Z"}))
	got, err := repo.Get(ctx, "prog_no_caller")
	require.NoError(t, err)
	require.Empty(t, got.GetCallerRunId())
	require.Empty(t, got.GetCallerAgentProfile())
	require.Empty(t, got.GetCallerSkillId())
	require.Empty(t, got.GetCallerHarness())
}

func TestSQLiteRepositoryMineFailuresExcludesOperatorByDefault(t *testing.T) { // [REQ:PRT-P1-008]
	ctx := context.Background()
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	for i, provenance := range []programsv1.Provenance{programsv1.Provenance_PROVENANCE_OPERATOR, programsv1.Provenance_PROVENANCE_AGENT, programsv1.Provenance_PROVENANCE_AGENT} {
		require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: "prog_" + string(rune('a'+i)), SessionId: "sess_1", Source: "x", Provenance: provenance, Status: programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, FailureShape: "same failure", CreatedAt: time.Date(2026, 8, 11, 14, i, 0, 0, time.UTC).Format(time.RFC3339Nano)}))
	}
	shapes, err := repo.MineFailures(ctx, false, time.Time{})
	require.NoError(t, err)
	require.Len(t, shapes, 1)
	require.Equal(t, int64(2), shapes[0].Count)
	shapes, err = repo.MineFailures(ctx, true, time.Time{})
	require.NoError(t, err)
	require.Equal(t, int64(3), shapes[0].Count)
}

func TestSQLiteRepositoryMineFailuresHonorsTimeWindow(t *testing.T) {
	ctx := context.Background()
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	old := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	for id, created := range map[string]time.Time{"old": old, "new": now} {
		require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: id, SessionId: "sess_1", Source: "x", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Status: programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, FailureShape: "window", CreatedAt: created.Format(time.RFC3339Nano)}))
	}
	shapes, err := repo.MineFailures(ctx, false, now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, shapes, 1)
	require.Equal(t, int64(1), shapes[0].Count)
}

func TestSQLiteRepositoryListFilteredHonorsProvenanceBoundsAndLimit(t *testing.T) {
	ctx := context.Background()
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	old := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	for id, item := range map[string]struct {
		at time.Time
		p  programsv1.Provenance
	}{
		"old-agent":    {old, programsv1.Provenance_PROVENANCE_AGENT},
		"new-agent":    {now, programsv1.Provenance_PROVENANCE_AGENT},
		"new-operator": {now, programsv1.Provenance_PROVENANCE_OPERATOR},
	} {
		require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: id, SessionId: "s", Source: "x", Provenance: item.p, Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, CreatedAt: item.at.Format(time.RFC3339Nano)}))
	}
	rows, err := repo.ListFiltered(ctx, ListFilter{SessionID: "s", IncludeOperator: true, Provenance: []programsv1.Provenance{programsv1.Provenance_PROVENANCE_AGENT}, Since: now.Add(-time.Hour), Until: now.Add(time.Hour), Limit: 1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "new-agent", rows[0].GetId())
}

func TestSQLiteRepositoryPortfolioStatsGroupsPercentilesAndUnattributedRows(t *testing.T) { // [REQ:PRT-P1-006]
	ctx := context.Background()
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	created := func(minute int) string {
		return time.Date(2026, 8, 11, 14, minute, 0, 0, time.UTC).Format(time.RFC3339Nano)
	}
	for i, wall := range []int64{10, 20, 30, 40, 50} {
		status := programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED
		cause := ""
		if i == 4 {
			status = programsv1.ProgramStatus_PROGRAM_STATUS_FAILED
			cause = programsv1.FailureCause_FAILURE_CAUSE_KERNEL_RUNTIME.String()
		}
		require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: "named-" + strconv.Itoa(i), SessionId: "s", Source: "x", ProgramName: "demo.program", ProgramDigest: "digest", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Status: status, FailureCause: programsv1.FailureCause(programsv1.FailureCause_value[cause]), FailureShape: cause, CallerRunId: "caller-" + strconv.Itoa(i%2), CallerHarness: "program", CreatedAt: created(i), WallTimeMillis: wall}))
	}
	require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: "ad-hoc", SessionId: "s", Source: "x", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, CreatedAt: created(6), WallTimeMillis: 7}))
	aggregate, err := repo.PortfolioStats(ctx, PortfolioFilter{IncludeAdHoc: true})
	require.NoError(t, err)
	require.EqualValues(t, 6, aggregate.ProgramsExecuted)
	require.EqualValues(t, 1, aggregate.RowsWithoutIdentity)
	require.EqualValues(t, 1, aggregate.UnattributedAgentRuns)
	require.Len(t, aggregate.Groups, 2)
	row := aggregate.Groups[0].Row
	require.Equal(t, "demo.program", row.GetName())
	require.EqualValues(t, 5, row.GetRuns())
	require.EqualValues(t, 4, row.GetSucceeded())
	require.EqualValues(t, 1, row.GetFailed())
	require.EqualValues(t, 30, row.GetP50Millis())
	require.EqualValues(t, 50, row.GetP95Millis())
	require.EqualValues(t, 2, row.GetDistinctCallers())
	require.EqualValues(t, 5, row.GetCalledByPrograms())
	require.Equal(t, programsv1.FailureCause_FAILURE_CAUSE_KERNEL_RUNTIME.String(), row.GetTopFailureCause())
}

// The library search corpus ranks by usage on every Search Hub query, so the
// usage read must agree with PortfolioStats without paying for its full scan.
func TestSQLiteRepositoryUsageByNameMatchesPortfolioStatsRuns(t *testing.T) {
	ctx := context.Background()
	d := newProgramsTestDB(t)
	created := func(day, minute int) string {
		return time.Date(2026, 8, day, 14, minute, 0, 0, time.UTC).Format(time.RFC3339Nano)
	}
	fixture := []*programsv1.Program{
		{Id: "a1", ProgramName: "demo.program", CreatedAt: created(11, 0)},
		{Id: "a2", ProgramName: "demo.program", CreatedAt: created(11, 1), Status: programsv1.ProgramStatus_PROGRAM_STATUS_FAILED},
		{Id: "a3", ProgramName: "demo.program", CreatedAt: created(12, 0)},
		{Id: "b1", ProgramName: "other.program", CreatedAt: created(12, 5)},
		{Id: "old", ProgramName: "other.program", CreatedAt: created(1, 0)},
		{Id: "ad-hoc", CreatedAt: created(12, 6)},
	}
	for _, repo := range []Repository{NewRepository(d), newMemoryRepository()} {
		for _, p := range fixture {
			p.SessionId, p.Source, p.Provenance = "s", "x", programsv1.Provenance_PROVENANCE_AGENT
			if p.Status == programsv1.ProgramStatus_PROGRAM_STATUS_UNSPECIFIED {
				p.Status = programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED
			}
			require.NoError(t, repo.Save(ctx, clone(p)))
		}
		since := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
		aggregate, err := repo.PortfolioStats(ctx, PortfolioFilter{Since: since})
		require.NoError(t, err)
		want := map[string]int64{}
		for _, group := range aggregate.Groups {
			want[group.Row.GetName()] = group.Row.GetRuns()
		}
		got, err := repo.UsageByName(ctx, since)
		require.NoError(t, err)
		require.Equal(t, map[string]int64{"demo.program": 3, "other.program": 1}, want)
		require.Equal(t, want, got)
		unbounded, err := repo.UsageByName(ctx, time.Time{})
		require.NoError(t, err)
		require.Equal(t, map[string]int64{"demo.program": 3, "other.program": 2}, unbounded)
	}
}

// On the production table (664 MB, dominated by source/stdout) the usage read
// is only cheap if SQLite never visits table pages: it must be answered from
// idx_programs_name_created alone.
func TestSQLiteRepositoryUsageByNameUsesCoveringIndex(t *testing.T) {
	ctx := context.Background()
	d := newProgramsTestDB(t)
	rows, err := d.QueryContext(ctx, `EXPLAIN QUERY PLAN SELECT program_name, COUNT(*) FROM programs WHERE program_name != '' AND created_at >= ? GROUP BY program_name`, "2026-08-10T00:00:00Z")
	require.NoError(t, err)
	defer rows.Close()
	var plan []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &notUsed, &detail))
		plan = append(plan, detail)
	}
	require.NoError(t, rows.Err())
	require.NotEmpty(t, plan)
	for _, step := range plan {
		require.Contains(t, step, "COVERING INDEX idx_programs_name_created", "plan step %q must not read table pages; full plan: %v", step, plan)
		require.NotContains(t, step, "TEMP B-TREE", "grouping must come from index order; full plan: %v", plan)
	}
}

func TestSQLiteRepositoryMineRefusalsFiltersOperatorByDefault(t *testing.T) {
	ctx := context.Background()
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	_, err := d.ExecContext(ctx, `INSERT INTO refusals (binding_id, reason, provenance, occurred_at) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`, "ops/delete", "missing grant", programsv1.Provenance_PROVENANCE_OPERATOR.String(), "2026-08-11T14:00:00Z", "ops/delete", "missing grant", programsv1.Provenance_PROVENANCE_AGENT.String(), "2026-08-11T14:01:00Z")
	require.NoError(t, err)
	shapes, err := repo.MineRefusals(ctx, false)
	require.NoError(t, err)
	require.Len(t, shapes, 1)
	require.Equal(t, int64(1), shapes[0].Count)
	shapes, err = repo.MineRefusals(ctx, true)
	require.NoError(t, err)
	require.Equal(t, int64(2), shapes[0].Count)
}

func TestSQLiteRepositoryMineUnresolvedFiltersOperatorByDefault(t *testing.T) {
	ctx := context.Background()
	d := newProgramsTestDB(t)
	repo := NewRepository(d)
	_, err := d.ExecContext(ctx, `INSERT INTO unresolved_binding_attempts (attempted_name, provenance, occurred_at) VALUES (?, ?, ?), (?, ?, ?)`, "missing.binding", programsv1.Provenance_PROVENANCE_OPERATOR.String(), "2026-08-11T14:00:00Z", "missing.binding", programsv1.Provenance_PROVENANCE_AGENT.String(), "2026-08-11T14:01:00Z")
	require.NoError(t, err)
	shapes, err := repo.MineUnresolvedBindings(ctx, false)
	require.NoError(t, err)
	require.Len(t, shapes, 1)
	require.Equal(t, int64(1), shapes[0].Count)
	shapes, err = repo.MineUnresolvedBindings(ctx, true)
	require.NoError(t, err)
	require.Equal(t, int64(2), shapes[0].Count)
}
