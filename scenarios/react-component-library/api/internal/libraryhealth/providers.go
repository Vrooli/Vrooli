package libraryhealth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DBProviders builds the honest default reads over tables owned by other
// domains. It never creates or writes anything: if an owning table is absent,
// the affected section reports unavailable (or partial) with a reason.
//
// Services is intentionally unset. The capability registry is a separate
// source; callers that want the section inject Providers.Services on the
// returned value.
// Database is the query-only SQL surface used by the read providers. A
// *sql.DB and api-core's RoutedDB satisfy it; production passes RoutedDB so
// request contexts select the appropriate pool for each snapshot.
type Database interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func DBProviders(db Database) Providers {
	return Providers{
		Overview: func(ctx context.Context) (OverviewCounts, error) {
			return dbOverview(ctx, db)
		},
		Quality: func(ctx context.Context) ([]Metric, error) {
			return dbQuality(ctx, db)
		},
		Activity: func(ctx context.Context, days int) ([]Metric, error) {
			return dbActivity(ctx, db, days)
		},
	}
}

func dbOverview(ctx context.Context, db Database) (OverviewCounts, error) {
	if db == nil {
		return OverviewCounts{}, errors.New("library health database is not configured")
	}
	counts := OverviewCounts{}
	missing := make([]string, 0, 5)

	assets, ok, err := countTable(ctx, db, "components", `SELECT COUNT(*) FROM components`)
	if err != nil {
		return OverviewCounts{}, fmt.Errorf("count components: %w", err)
	}
	if ok {
		counts.Assets = assets
	} else {
		missing = append(missing, "assets")
	}

	published, ok, err := publishedCount(ctx, db)
	if err != nil {
		return OverviewCounts{}, err
	}
	if ok {
		counts.Published = published
	} else {
		missing = append(missing, "published")
	}

	workspaces, ok, err := countTable(ctx, db, "workspaces", `SELECT COUNT(*) FROM workspaces`)
	if err != nil {
		return OverviewCounts{}, fmt.Errorf("count workspaces: %w", err)
	}
	if ok {
		counts.Workspaces = workspaces
	} else {
		missing = append(missing, "workspaces")
	}

	findingsOpen, ok, err := countTable(ctx, db, "findings", `SELECT COUNT(*) FROM findings WHERE status = 'open'`)
	if err != nil {
		return OverviewCounts{}, fmt.Errorf("count open findings: %w", err)
	}
	if ok {
		counts.FindingsOpen = findingsOpen
	} else {
		missing = append(missing, "findings_open")
	}

	coverageRows, ok, err := countTable(ctx, db, "catalog_gate_evidence", `SELECT COUNT(*) FROM catalog_gate_evidence`)
	if err != nil {
		return OverviewCounts{}, fmt.Errorf("count coverage rows: %w", err)
	}
	if ok {
		counts.CoverageRows = coverageRows
	} else {
		missing = append(missing, "coverage_rows")
	}

	switch {
	case len(missing) == 0:
		return counts, nil
	case len(missing) == 5:
		return counts, fmt.Errorf("overview sources are unavailable: %s", strings.Join(missing, ", "))
	default:
		return counts, &PartialError{
			Reason:      "overview is partial; unavailable: " + strings.Join(missing, ", "),
			Unavailable: missing,
		}
	}
}

func dbQuality(ctx context.Context, db Database) ([]Metric, error) {
	if db == nil {
		return nil, errors.New("library health database is not configured")
	}
	exists, err := tableExists(ctx, db, "catalog_gate_evidence")
	if err != nil {
		return nil, fmt.Errorf("inspect catalog_gate_evidence: %w", err)
	}
	if !exists {
		return nil, errors.New("catalog_gate_evidence table is missing; quality is unavailable")
	}
	rows, err := db.QueryContext(ctx,
		`SELECT LOWER(TRIM(result)), COUNT(*) FROM catalog_gate_evidence GROUP BY 1`)
	if err != nil {
		return nil, fmt.Errorf("read coverage results: %w", err)
	}
	defer rows.Close()

	var total, pass, fail, unmeasured, other int
	for rows.Next() {
		var (
			result string
			count  int
		)
		if err := rows.Scan(&result, &count); err != nil {
			return nil, fmt.Errorf("scan coverage result: %w", err)
		}
		total += count
		switch result {
		case "pass":
			pass += count
		case "fail":
			fail += count
		case "unmeasured":
			unmeasured += count
		default:
			other += count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate coverage results: %w", err)
	}

	metrics := []Metric{
		{Key: "coverage_rows", Label: "Coverage rows", Value: total, Total: total},
		{Key: "pass", Label: "Passing", Value: pass, Total: total},
		{Key: "fail", Label: "Failing", Value: fail, Total: total},
		{Key: "unmeasured", Label: "Unmeasured", Value: unmeasured, Total: total},
	}
	if other > 0 {
		metrics = append(metrics, Metric{Key: "other", Label: "Other results", Value: other, Total: total})
	}
	return metrics, nil
}

func dbActivity(ctx context.Context, db Database, days int) ([]Metric, error) {
	if db == nil {
		return nil, errors.New("library health database is not configured")
	}
	exists, err := tableExists(ctx, db, "activity_events")
	if err != nil {
		return nil, fmt.Errorf("inspect activity_events: %w", err)
	}
	if !exists {
		return nil, errors.New("activity_events table is missing; activity is unavailable")
	}
	days = clampActivityDays(days)
	cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format(timestampLayout)
	var total, actors, actions int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*), COUNT(DISTINCT actor), COUNT(DISTINCT action) FROM activity_events WHERE at >= ?`,
		cutoff).Scan(&total, &actors, &actions); err != nil {
		return nil, fmt.Errorf("read activity totals: %w", err)
	}
	return []Metric{
		{Key: "events", Label: "Events", Value: total, Total: total, Detail: fmt.Sprintf("last %d days", days)},
		{Key: "actors", Label: "Actors", Value: actors, Total: total},
		{Key: "actions", Label: "Actions", Value: actions, Total: total},
	}, nil
}

// publishedCount treats the components schema's released status as published.
// A legacy or alternate schema with a boolean published column is supported;
// if neither exists the count is unavailable rather than zero.
func publishedCount(ctx context.Context, db Database) (int, bool, error) {
	exists, err := tableExists(ctx, db, "component_versions")
	if err != nil {
		return 0, false, fmt.Errorf("inspect component_versions: %w", err)
	}
	if !exists {
		return 0, false, nil
	}
	columns, err := tableColumns(ctx, db, "component_versions")
	if err != nil {
		return 0, false, fmt.Errorf("inspect component_versions columns: %w", err)
	}
	switch {
	case columns["status"]:
		var count int
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM component_versions WHERE LOWER(TRIM(status)) = 'released'`).Scan(&count); err != nil {
			return 0, false, fmt.Errorf("count published versions: %w", err)
		}
		return count, true, nil
	case columns["published"]:
		var count int
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM component_versions WHERE published = 1`).Scan(&count); err != nil {
			return 0, false, fmt.Errorf("count published versions: %w", err)
		}
		return count, true, nil
	default:
		return 0, false, nil
	}
}

func countTable(ctx context.Context, db Database, table, query string) (int, bool, error) {
	exists, err := tableExists(ctx, db, table)
	if err != nil {
		return 0, false, err
	}
	if !exists {
		return 0, false, nil
	}
	var count int
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, true, err
	}
	return count, true, nil
}

func tableExists(ctx context.Context, db Database, table string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func tableColumns(ctx context.Context, db Database, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var (
			cid       int
			name      string
			columnTyp string
			notNull   int
			defaultV  any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &columnTyp, &notNull, &defaultV, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
}
