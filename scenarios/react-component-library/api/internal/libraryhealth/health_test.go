package libraryhealth

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func newSyntheticSchema(t *testing.T, tables ...string) *sql.DB {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	definitions := map[string]string{
		"components":            `CREATE TABLE components (id TEXT PRIMARY KEY)`,
		"component_versions":    `CREATE TABLE component_versions (id TEXT PRIMARY KEY, status TEXT NOT NULL)`,
		"workspaces":            `CREATE TABLE workspaces (id TEXT PRIMARY KEY)`,
		"findings":              `CREATE TABLE findings (id TEXT PRIMARY KEY, status TEXT NOT NULL)`,
		"catalog_gate_evidence": `CREATE TABLE catalog_gate_evidence (id TEXT PRIMARY KEY, result TEXT NOT NULL)`,
		"activity_events":       `CREATE TABLE activity_events (id TEXT PRIMARY KEY, actor TEXT NOT NULL, action TEXT NOT NULL, at TEXT NOT NULL)`,
	}
	for _, table := range tables {
		statement, ok := definitions[table]
		require.True(t, ok, "unknown synthetic table %q", table)
		_, err := sqlDB.ExecContext(context.Background(), statement)
		require.NoError(t, err)
	}
	return sqlDB
}

func exec(t *testing.T, sqlDB *sql.DB, statement string, args ...any) {
	t.Helper()
	_, err := sqlDB.ExecContext(context.Background(), statement, args...)
	require.NoError(t, err)
}

func seedAvailable(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB := newSyntheticSchema(t, "components", "component_versions", "workspaces", "findings", "catalog_gate_evidence", "activity_events")
	for _, id := range []string{"a", "b", "c", "d"} {
		exec(t, sqlDB, `INSERT INTO components(id) VALUES (?)`, id)
	}
	for id, status := range map[string]string{"v1": "released", "v2": "released", "v3": "released", "v4": "draft"} {
		exec(t, sqlDB, `INSERT INTO component_versions(id, status) VALUES (?, ?)`, id, status)
	}
	exec(t, sqlDB, `INSERT INTO workspaces(id) VALUES ('w1'), ('w2')`)
	exec(t, sqlDB, `INSERT INTO findings(id, status) VALUES ('f1','open'), ('f2','open'), ('f3','accepted')`)
	for index, result := range []string{"pass", "pass", "pass", "pass", "pass", "fail", "fail", "unmeasured"} {
		exec(t, sqlDB, `INSERT INTO catalog_gate_evidence(id, result) VALUES (?, ?)`, index, result)
	}
	now := time.Now().UTC()
	exec(t, sqlDB, `INSERT INTO activity_events(id, actor, action, at) VALUES ('e1','ada','asset.publish',?)`, now.Add(-time.Hour).Format(timestampLayout))
	exec(t, sqlDB, `INSERT INTO activity_events(id, actor, action, at) VALUES ('e2','lin','workspace.save',?)`, now.Add(-2*time.Hour).Format(timestampLayout))
	exec(t, sqlDB, `INSERT INTO activity_events(id, actor, action, at) VALUES ('e3','ada','asset.publish',?)`, now.Add(-3*time.Hour).Format(timestampLayout))
	exec(t, sqlDB, `INSERT INTO activity_events(id, actor, action, at) VALUES ('old','old','asset.publish',?)`, now.Add(-40*24*time.Hour).Format(timestampLayout))
	return sqlDB
}

func metricByKey(t *testing.T, metrics []Metric, key string) Metric {
	t.Helper()
	for _, metric := range metrics {
		if metric.Key == key {
			return metric
		}
	}
	t.Fatalf("metric %q not found in %+v", key, metrics)
	return Metric{}
}

func TestDBAvailableSections(t *testing.T) {
	sqlDB := seedAvailable(t)
	service := NewService(DBProviders(sqlDB))
	service.WithClock(func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) })

	health := service.Get(context.Background(), 30)
	require.Equal(t, StateAvailable, health.Overview.State)
	require.Equal(t, OverviewCounts{Assets: 4, Published: 3, Workspaces: 2, FindingsOpen: 2, CoverageRows: 8}, health.Counts)
	require.Equal(t, 4, metricByKey(t, health.Overview.Metrics, "assets").Value)
	require.Equal(t, 3, metricByKey(t, health.Overview.Metrics, "published").Value)

	require.Equal(t, StateAvailable, health.Quality.State)
	require.Equal(t, 8, metricByKey(t, health.Quality.Metrics, "coverage_rows").Value)
	require.Equal(t, 5, metricByKey(t, health.Quality.Metrics, "pass").Value)
	require.Equal(t, 2, metricByKey(t, health.Quality.Metrics, "fail").Value)
	require.Equal(t, 1, metricByKey(t, health.Quality.Metrics, "unmeasured").Value)
	require.Equal(t, 8, metricByKey(t, health.Quality.Metrics, "pass").Total)

	require.Equal(t, StateAvailable, health.Activity.State)
	require.Equal(t, 3, metricByKey(t, health.Activity.Metrics, "events").Value)
	require.Equal(t, 2, metricByKey(t, health.Activity.Metrics, "actors").Value)
	require.Equal(t, 2, metricByKey(t, health.Activity.Metrics, "actions").Value)

	// Services is not wired by DBProviders; unavailable is the honest answer.
	require.Equal(t, StateUnavailable, health.Services.State)
	require.Contains(t, health.Services.Reason, "not configured")
	require.Empty(t, health.Services.Metrics)
	require.Equal(t, "2026-11-19T12:00:00.000000000Z", health.GeneratedAt)
}

func TestDBProvidersRouteMarkedRequestsToInstalledTestPool(t *testing.T) {
	ctx := context.Background()
	routed, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          filepath.Join(t.TempDir(), "primary.db"),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })

	schema := []string{
		`CREATE TABLE components (id TEXT PRIMARY KEY)`,
		`CREATE TABLE component_versions (id TEXT PRIMARY KEY, status TEXT NOT NULL)`,
		`CREATE TABLE workspaces (id TEXT PRIMARY KEY)`,
		`CREATE TABLE findings (id TEXT PRIMARY KEY, status TEXT NOT NULL)`,
		`CREATE TABLE catalog_gate_evidence (id TEXT PRIMARY KEY, result TEXT NOT NULL)`,
		`CREATE TABLE activity_events (id TEXT PRIMARY KEY, actor TEXT NOT NULL, action TEXT NOT NULL, at TEXT NOT NULL)`,
	}
	for _, statement := range schema {
		_, err = routed.ExecContext(ctx, statement)
		require.NoError(t, err)
	}
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		for _, statement := range schema {
			if _, err := pool.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "test.db"), "health-test", 0))

	_, err = routed.ExecContext(ctx, `INSERT INTO components(id) VALUES ('primary-a'), ('primary-b')`)
	require.NoError(t, err)
	requestCtx := database.WithTestMode(ctx)
	_, err = routed.ExecContext(requestCtx, `INSERT INTO components(id) VALUES ('test-only')`)
	require.NoError(t, err)

	service := NewService(DBProviders(routed))
	primary := service.Get(ctx, 30)
	testPool := service.Get(requestCtx, 30)
	require.Equal(t, 2, primary.Counts.Assets)
	require.Equal(t, 1, testPool.Counts.Assets, "request context must route the provider reads to the installed test pool")
}

func TestDBEmptyTablesAreMeasuredEmpty(t *testing.T) {
	sqlDB := newSyntheticSchema(t, "components", "component_versions", "workspaces", "findings", "catalog_gate_evidence", "activity_events")
	service := NewService(DBProviders(sqlDB))

	health := service.Get(context.Background(), 30)
	require.Equal(t, StateEmpty, health.Overview.State)
	require.Equal(t, StateEmpty, health.Quality.State)
	require.Equal(t, StateEmpty, health.Activity.State)
	require.Equal(t, OverviewCounts{}, health.Counts)
	require.Len(t, health.Overview.Metrics, 5)
	require.Len(t, health.Quality.Metrics, 4)
	require.Len(t, health.Activity.Metrics, 3)
}

func TestDBMissingTablesAreUnavailableNotZero(t *testing.T) {
	emptyDB := db.NewSQLite(t)
	service := NewService(DBProviders(emptyDB))
	health := service.Get(context.Background(), 30)

	require.Equal(t, StateUnavailable, health.Overview.State)
	require.Contains(t, health.Overview.Reason, "unavailable")
	require.Empty(t, health.Overview.Metrics)
	require.Equal(t, OverviewCounts{}, health.Counts)

	require.Equal(t, StateUnavailable, health.Quality.State)
	require.Contains(t, health.Quality.Reason, "catalog_gate_evidence")
	require.Empty(t, health.Quality.Metrics)

	require.Equal(t, StateUnavailable, health.Activity.State)
	require.Contains(t, health.Activity.Reason, "activity_events")
	require.Empty(t, health.Activity.Metrics)
}

func TestDBOverviewIsPartialWhenOnlySomeSourcesExist(t *testing.T) {
	sqlDB := newSyntheticSchema(t, "components")
	exec(t, sqlDB, `INSERT INTO components(id) VALUES ('a'), ('b')`)
	service := NewService(DBProviders(sqlDB))

	health := service.Get(context.Background(), 30)
	require.Equal(t, StatePartial, health.Overview.State)
	require.Contains(t, health.Overview.Reason, "workspaces")
	require.Len(t, health.Overview.Metrics, 1, "unavailable counts must not be rendered as zero metrics")
	require.Equal(t, "assets", health.Overview.Metrics[0].Key)
	require.Equal(t, 2, health.Overview.Metrics[0].Value)
	require.Equal(t, 2, health.Counts.Assets)

	require.Equal(t, StateUnavailable, health.Quality.State)
	require.Empty(t, health.Quality.Metrics)
}

func TestInjectedPartialSectionKeepsAvailableMetrics(t *testing.T) {
	providers := Providers{
		Overview: func(context.Context) (OverviewCounts, error) {
			return OverviewCounts{Assets: 2, Workspaces: 1}, nil
		},
		Quality: func(context.Context) ([]Metric, error) {
			return []Metric{
				{Key: "pass", Label: "Passing", Value: 4, Total: 5},
				{Key: "unmeasured", Label: "Unmeasured", Value: 1, Total: 5},
			}, &PartialError{Reason: "measurement source is unavailable", Unavailable: []string{"unmeasured"}}
		},
		Activity: func(context.Context, int) ([]Metric, error) {
			return []Metric{{Key: "events", Label: "Events", Value: 3}}, nil
		},
		Services: func(context.Context) ([]Metric, error) {
			return []Metric{{Key: "capabilities", Label: "Capabilities", Value: 9}}, nil
		},
	}
	health := NewService(providers).Get(context.Background(), 30)

	require.Equal(t, StatePartial, health.Quality.State)
	require.Equal(t, "measurement source is unavailable", health.Quality.Reason)
	require.Len(t, health.Quality.Metrics, 1)
	require.Equal(t, "pass", health.Quality.Metrics[0].Key)
	require.Equal(t, StateAvailable, health.Activity.State)
	require.Equal(t, StateAvailable, health.Services.State)
}

func TestNonPartialErrorNeverLeaksMetricsOrCounts(t *testing.T) {
	providers := Providers{
		Overview: func(context.Context) (OverviewCounts, error) {
			return OverviewCounts{Assets: 7}, errors.New("overview database is offline")
		},
		Quality: func(context.Context) ([]Metric, error) {
			return []Metric{{Key: "pass", Label: "Passing", Value: 3}}, errors.New("quality database is offline")
		},
		Activity: func(context.Context, int) ([]Metric, error) {
			return []Metric{{Key: "events", Label: "Events", Value: 5}}, errors.New("activity database is offline")
		},
		Services: func(context.Context) ([]Metric, error) {
			return []Metric{{Key: "capabilities", Label: "Capabilities", Value: 2}}, errors.New("registry is offline")
		},
	}
	health := NewService(providers).Get(context.Background(), 30)

	for name, section := range map[string]Section{
		"overview": health.Overview, "quality": health.Quality,
		"activity": health.Activity, "services": health.Services,
	} {
		require.Equal(t, StateUnavailable, section.State, name)
		require.NotEmpty(t, section.Reason, name)
		require.Empty(t, section.Metrics, name)
	}
	require.Equal(t, OverviewCounts{}, health.Counts)
}

func TestEverySectionSupportsPartialState(t *testing.T) {
	providers := Providers{
		Overview: func(context.Context) (OverviewCounts, error) {
			return OverviewCounts{Assets: 3}, &PartialError{Reason: "workspaces are unavailable", Unavailable: []string{"workspaces"}}
		},
		Quality: func(context.Context) ([]Metric, error) {
			return []Metric{{Key: "pass", Label: "Passing", Value: 2}}, &PartialError{Reason: "some gate results are missing"}
		},
		Activity: func(context.Context, int) ([]Metric, error) {
			return []Metric{{Key: "events", Label: "Events", Value: 4}}, &PartialError{Reason: "activity retention is incomplete"}
		},
		Services: func(context.Context) ([]Metric, error) {
			return []Metric{{Key: "capabilities", Label: "Capabilities", Value: 5}}, &PartialError{Reason: "one capability probe timed out"}
		},
	}
	health := NewService(providers).Get(context.Background(), 30)

	for name, section := range map[string]Section{
		"overview": health.Overview, "quality": health.Quality,
		"activity": health.Activity, "services": health.Services,
	} {
		require.Equal(t, StatePartial, section.State, name)
		require.NotEmpty(t, section.Reason, name)
	}
	require.Len(t, health.Overview.Metrics, 4, "the unavailable workspaces metric is omitted")
	require.Equal(t, 3, health.Counts.Assets)
}

func TestEverySectionSupportsEmptyState(t *testing.T) {
	providers := Providers{
		Overview: func(context.Context) (OverviewCounts, error) { return OverviewCounts{}, nil },
		Quality:  func(context.Context) ([]Metric, error) { return nil, nil },
		Activity: func(context.Context, int) ([]Metric, error) { return []Metric{{Key: "events", Value: 0}}, nil },
		Services: func(context.Context) ([]Metric, error) { return nil, nil },
	}
	health := NewService(providers).Get(context.Background(), 30)

	require.Equal(t, StateEmpty, health.Overview.State)
	require.Equal(t, StateEmpty, health.Quality.State)
	require.Equal(t, StateEmpty, health.Activity.State)
	require.Equal(t, StateEmpty, health.Services.State)
}

func TestActivityDaysAreBounded(t *testing.T) {
	var seen []int
	providers := Providers{
		Overview: func(context.Context) (OverviewCounts, error) { return OverviewCounts{}, nil },
		Quality:  func(context.Context) ([]Metric, error) { return nil, nil },
		Activity: func(_ context.Context, days int) ([]Metric, error) {
			seen = append(seen, days)
			return nil, nil
		},
	}
	service := NewService(providers)
	service.Get(context.Background(), 0)
	service.Get(context.Background(), 900)
	service.Get(context.Background(), 45)
	require.Equal(t, []int{defaultActivityDays, maxActivityDays, 45}, seen)
}
