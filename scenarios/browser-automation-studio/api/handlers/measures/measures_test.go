package measures

import (
	"context"
	"database/sql"
	"testing"
	"time"

	coredb "github.com/vrooli/api-core/database"
	measurelib "github.com/vrooli/measures-go"
	_ "modernc.org/sqlite"
)

type fakeMetrics struct{ aggregate Aggregate }

func (f fakeMetrics) Aggregate(context.Context, time.Time, time.Time) (Aggregate, error) {
	return f.aggregate, nil
}

func newMeasuresSQLTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE executions (status TEXT, started_at TEXT, completed_at TEXT, created_at TEXT)`,
		`CREATE TABLE ux_execution_metrics (step_count INTEGER, failed_steps INTEGER, computed_at TEXT)`,
		`CREATE TABLE ux_interaction_traces (success INTEGER, timestamp TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestRegistryComputesDeclaredMeasuresFromOneAggregate(t *testing.T) {
	registry, err := declarationRegistry(fakeMetrics{aggregate: Aggregate{
		TerminalExecutions: 10, SuccessfulExecutions: 9, DurationP95Ms: 1234.5,
		StepCount: 20, FailedSteps: 2, SelectorTraces: 8, FailedSelectors: 1,
	}}, func() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		ExecutionSuccessRate: "0.9",
		ExecutionDurationP95: "1234.5",
		StepFailureRate:      "0.1",
		SelectorFailureRate:  "0.125",
	}
	for name, want := range cases {
		got, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: name, Params: map[string]string{"window": "last_7d"}})
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got.Value != want {
			t.Errorf("%s: got %q want %q", name, got.Value, want)
		}
		if got.Provenance.ExecutedQuery == "" || got.Provenance.ComputedAt.IsZero() {
			t.Errorf("%s: missing provenance: %+v", name, got.Provenance)
		}
	}
}

func TestSQLRepositoryDurationP95UsesNearestRank(t *testing.T) {
	for _, tc := range []struct {
		name  string
		count int
		want  float64
	}{
		{name: "single sample", count: 1, want: 1000},
		{name: "ten samples rounds up", count: 10, want: 10000},
		{name: "twenty-one samples rounds up", count: 21, want: 20000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMeasuresSQLTestDB(t)
			for durationSeconds := 1; durationSeconds <= tc.count; durationSeconds++ {
				completed := time.Date(2026, 9, 3, 0, 0, durationSeconds, 0, time.UTC)
				if _, err := db.Exec(`INSERT INTO executions(status, started_at, completed_at, created_at) VALUES('completed', ?, ?, ?)`,
					"2026-09-03T00:00:00Z", completed.Format(time.RFC3339), "2026-09-03T00:00:00Z"); err != nil {
					t.Fatal(err)
				}
			}

			routed := coredb.NewFromPrimary(db)
			got, err := NewSQLRepository(routed).Aggregate(context.Background(),
				time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
				time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatal(err)
			}
			if delta := got.DurationP95Ms - tc.want; delta < -1 || delta > 1 {
				t.Fatalf("duration p95 = %.3fms, want nearest-rank %.3fms (count=%d)", got.DurationP95Ms, tc.want, tc.count)
			}
			if got.TerminalExecutions != int64(tc.count) || got.SuccessfulExecutions != int64(tc.count) {
				t.Fatalf("execution counts = %d terminal, %d successful; want %d", got.TerminalExecutions, got.SuccessfulExecutions, tc.count)
			}
		})
	}
}

func TestSQLRepositoryDurationP95ExcludesFailedExecutions(t *testing.T) {
	db := newMeasuresSQLTestDB(t)
	started := "2026-09-03T00:00:00Z"
	for _, execution := range []struct {
		status    string
		completed string
	}{
		{status: "completed", completed: "2026-09-03T00:00:01Z"},
		{status: "completed", completed: "2026-09-03T00:00:02Z"},
		{status: "failed", completed: "2026-09-03T00:01:40Z"},
		{status: "running", completed: ""},
	} {
		if _, err := db.Exec(`INSERT INTO executions(status, started_at, completed_at, created_at) VALUES(?, ?, NULLIF(?, ''), ?)`,
			execution.status, started, execution.completed, started); err != nil {
			t.Fatal(err)
		}
	}

	got, err := NewSQLRepository(coredb.NewFromPrimary(db)).Aggregate(context.Background(),
		time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if delta := got.DurationP95Ms - 2000; delta < -1 || delta > 1 {
		t.Fatalf("duration p95 = %.3fms, want 2000ms from completed-only cohort", got.DurationP95Ms)
	}
	if got.TerminalExecutions != 3 || got.SuccessfulExecutions != 2 {
		t.Fatalf("execution counts = %d terminal, %d successful; want 3 terminal and 2 successful", got.TerminalExecutions, got.SuccessfulExecutions)
	}
}
