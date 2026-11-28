package versionledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"react-component-library/internal/sqliterows"
	"strings"
	"time"
)

func (r *Repository) List(ctx context.Context, libraryID string) ([]VersionLedger, error) {
	return r.ListWindow(ctx, libraryID, "")
}

func (r *Repository) ListWindow(ctx context.Context, libraryID, window string) ([]VersionLedger, error) {
	query := `SELECT l.library_id, l.version, l.created_at, l.released_at, l.retired_at, l.lifecycle_state, l.gate_pass_count, l.gate_fail_count, l.test_runs, l.test_pass_rate, l.adoption_current, l.adoption_peak, l.file_count, l.lines_of_code, l.dependency_count, COALESCE(v.presence, 'materialized') FROM version_ledger l LEFT JOIN component_versions v ON v.library_id = l.library_id AND v.version = l.version`
	args := []any{}
	filters := []string{}
	if libraryID != "" {
		filters = append(filters, "l.library_id = ?")
		args = append(args, libraryID)
	}
	if since := windowSince(window); !since.IsZero() {
		filters = append(filters, "l.created_at >= ?")
		args = append(args, since.Format(time.RFC3339Nano))
	}
	if len(filters) > 0 {
		query += " WHERE " + strings.Join(filters, " AND ")
	}
	query += " ORDER BY l.library_id, l.version"
	out, err := sqliterows.Collect(ctx, r.db, query, args, false, func(row sqliterows.Scanner) (VersionLedger, error) {
		var version VersionLedger
		var created, released, retired string
		if err := row.Scan(&version.LibraryID, &version.Version, &created, &released, &retired, &version.LifecycleState, &version.GatePassCount, &version.GateFailCount, &version.TestRuns, &version.TestPassRate, &version.AdoptionCurrent, &version.AdoptionPeak, &version.FileCount, &version.LinesOfCode, &version.DependencyCount, &version.Presence); err != nil {
			return VersionLedger{}, err
		}
		version.CreatedAt = parseTime(created)
		version.ReleasedAt = parseTime(released)
		version.RetiredAt = parseTime(retired)
		return version, nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func windowSince(window string) time.Time {
	switch strings.ToLower(strings.TrimSpace(window)) {
	case "last_7d", "this_week":
		return time.Now().UTC().Add(-7 * 24 * time.Hour)
	case "last_30d", "this_month":
		return time.Now().UTC().Add(-30 * 24 * time.Hour)
	case "last_month":
		return time.Now().UTC().Add(-60 * 24 * time.Hour)
	case "this_quarter":
		return time.Now().UTC().Add(-90 * 24 * time.Hour)
	}
	return time.Time{}
}

func (r *Repository) Rebuild(ctx context.Context) error {
	return r.rebuild(ctx, nil)
}

// rebuild observes the fixed set of source queries in tests. Production calls
// Rebuild without an observer; keeping the hook here makes the read budget
// explicit without changing database ownership or introducing a query wrapper.
func (r *Repository) rebuild(ctx context.Context, observeQuery func(string)) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	query := func(statement string, args ...any) (*sql.Rows, error) {
		if observeQuery != nil {
			observeQuery(statement)
		}
		return tx.QueryContext(ctx, statement, args...)
	}
	catalogByLibrary := r.catalogIDs()
	type sourceVersion struct {
		id, componentID, libraryID, version, status, created, released, content string
	}
	rows, err := query(`SELECT id, component_id, library_id, version, status, created_at, released_at, content FROM component_versions`)
	if err != nil {
		return fmt.Errorf("read version rebuild component versions: %w", err)
	}
	versions, err := sqliterows.CollectRows(rows, false, func(row sqliterows.Scanner) (sourceVersion, error) {
		var version sourceVersion
		if err := row.Scan(&version.id, &version.componentID, &version.libraryID, &version.version, &version.status, &version.created, &version.released, &version.content); err != nil {
			return sourceVersion{}, err
		}
		return version, nil
	})
	if err != nil {
		return fmt.Errorf("scan version rebuild component versions: %w", err)
	}
	type versionKey struct{ owner, version string }
	// Every projection below reads its source table once, so rebuild read cost
	// stays fixed as the number of historical versions grows.
	readGroupedInt := func(source, statement string, keyColumns int) (map[versionKey]int, error) {
		rows, err := query(statement)
		if err != nil {
			return nil, fmt.Errorf("read version rebuild %s: %w", source, err)
		}
		returnMap := make(map[versionKey]int)
		read, err := sqliterows.CollectRows(rows, false, func(row sqliterows.Scanner) (struct {
			key   versionKey
			count int
		}, error,
		) {
			var item struct {
				key   versionKey
				count int
			}
			if keyColumns == 1 {
				err := row.Scan(&item.key.owner, &item.count)
				return item, err
			}
			err := row.Scan(&item.key.owner, &item.key.version, &item.count)
			return item, err
		})
		if err != nil {
			return nil, fmt.Errorf("scan version rebuild %s: %w", source, err)
		}
		for _, item := range read {
			returnMap[item.key] = item.count
		}
		return returnMap, nil
	}
	type gateKey struct{ assetID, version string }
	type gateCounts struct{ pass, fail int }
	gateRows, gateErr := query(`SELECT asset_id, version,
		SUM(CASE WHEN lower(result) = 'pass' THEN 1 ELSE 0 END),
		SUM(CASE WHEN lower(result) IN ('fail', 'failed') THEN 1 ELSE 0 END)
		FROM catalog_gate_evidence GROUP BY asset_id, version`)
	gates := make(map[gateKey]gateCounts)
	if gateErr != nil {
		return fmt.Errorf("read version rebuild catalog gate evidence: %w", gateErr)
	}
	{
		gateValues, scanErr := sqliterows.CollectRows(gateRows, false, func(row sqliterows.Scanner) (struct {
			key    gateKey
			counts gateCounts
		}, error,
		) {
			var value struct {
				key    gateKey
				counts gateCounts
			}
			err := row.Scan(&value.key.assetID, &value.key.version, &value.counts.pass, &value.counts.fail)
			return value, err
		})
		if scanErr != nil {
			return fmt.Errorf("scan version rebuild catalog gate evidence: %w", scanErr)
		}
		for _, value := range gateValues {
			gates[value.key] = value.counts
		}
	}
	type testCounts struct{ runs, passed int }
	testRows, testErr := query(`SELECT library_id, version, runs_total, runs_passed FROM component_version_test_rollup`)
	if testErr != nil {
		return fmt.Errorf("read version rebuild test rollups: %w", testErr)
	}
	tests := make(map[versionKey]testCounts)
	{
		testValues, scanErr := sqliterows.CollectRows(testRows, false, func(row sqliterows.Scanner) (struct {
			key    versionKey
			counts testCounts
		}, error,
		) {
			var value struct {
				key    versionKey
				counts testCounts
			}
			err := row.Scan(&value.key.owner, &value.key.version, &value.counts.runs, &value.counts.passed)
			return value, err
		})
		if scanErr != nil {
			return fmt.Errorf("scan version rebuild test rollups: %w", scanErr)
		}
		for _, value := range testValues {
			tests[value.key] = value.counts
		}
	}
	adoptions, err := readGroupedInt("adoption records", `SELECT library_id, adopted_version, COUNT(*) FROM adoption_records GROUP BY library_id, adopted_version`, 2)
	if err != nil {
		return err
	}
	dependencies, err := readGroupedInt("component dependencies", `SELECT component_id, version, COUNT(*) FROM component_asset_dependencies GROUP BY component_id, version`, 2)
	if err != nil {
		return err
	}
	files, err := readGroupedInt("version files", `SELECT version_id, COUNT(*) FROM component_version_files GROUP BY version_id`, 1)
	if err != nil {
		return err
	}
	retiredRows, err := query(`SELECT library_id, version, retired_at FROM version_ledger`)
	if err != nil {
		return fmt.Errorf("read version rebuild retirement records: %w", err)
	}
	retired := make(map[versionKey]string)
	{
		retiredValues, scanErr := sqliterows.CollectRows(retiredRows, false, func(row sqliterows.Scanner) (struct {
			key versionKey
			at  string
		}, error,
		) {
			var value struct {
				key versionKey
				at  string
			}
			err := row.Scan(&value.key.owner, &value.key.version, &value.at)
			return value, err
		})
		if scanErr != nil {
			return fmt.Errorf("scan version rebuild retirement records: %w", scanErr)
		}
		for _, value := range retiredValues {
			retired[value.key] = value.at
		}
	}
	for _, version := range versions {
		catalogID := catalogByLibrary[version.libraryID]
		gate := gates[gateKey{assetID: catalogID, version: version.version}]
		key := versionKey{owner: version.libraryID, version: version.version}
		test := tests[key]
		runs := test.runs
		passed := test.passed
		adoption := adoptions[key]
		dependencyCount := dependencies[versionKey{owner: version.componentID, version: version.version}]
		fileCount := files[versionKey{owner: version.id}]
		if fileCount == 0 {
			fileCount = 1
		}
		lines := lineCount(version.content)
		retiredAt := retired[key]
		state := version.status
		if retiredAt != "" {
			state = "retired"
		}
		rate := 0.0
		if runs > 0 {
			rate = float64(passed) / float64(runs)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO version_ledger (library_id, version, created_at, released_at, retired_at, lifecycle_state, gate_pass_count, gate_fail_count, test_runs, test_pass_rate, adoption_current, adoption_peak, file_count, lines_of_code, dependency_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(library_id, version) DO UPDATE SET created_at=excluded.created_at, released_at=excluded.released_at, lifecycle_state=CASE WHEN version_ledger.retired_at<>'' THEN 'retired' ELSE excluded.lifecycle_state END, gate_pass_count=excluded.gate_pass_count, gate_fail_count=excluded.gate_fail_count, test_runs=excluded.test_runs, test_pass_rate=excluded.test_pass_rate, adoption_current=excluded.adoption_current, adoption_peak=CASE WHEN version_ledger.adoption_peak > excluded.adoption_peak THEN version_ledger.adoption_peak ELSE excluded.adoption_peak END, file_count=excluded.file_count, lines_of_code=excluded.lines_of_code, dependency_count=excluded.dependency_count`, version.libraryID, version.version, version.created, version.released, retiredAt, state, gate.pass, gate.fail, runs, rate, adoption, adoption, fileCount, lines, dependencyCount)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) catalogIDs() map[string]string {
	out := map[string]string{}
	paths, _ := filepath.Glob(filepath.Join(r.sourceRoot, "*", "*", "component.json"))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc struct {
			LibraryID string `json:"libraryId"`
			CatalogID string `json:"catalogId"`
		}
		if json.Unmarshal(data, &doc) == nil && doc.LibraryID != "" {
			out[doc.LibraryID] = doc.CatalogID
		}
	}
	return out
}

func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	value, _ := time.Parse(time.RFC3339Nano, raw)
	return value
}

func lineCount(raw string) int {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	return strings.Count(raw, "\n") + 1
}
