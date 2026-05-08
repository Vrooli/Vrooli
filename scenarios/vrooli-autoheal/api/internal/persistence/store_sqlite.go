package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/hostinventory"
	"vrooli-autoheal/internal/incidents"
)

// rfc3339NanoCutoff returns now-windowHours as an RFC3339Nano UTC string,
// the canonical wire format for created_at columns. Computing the cutoff in
// Go (rather than in SQL via datetime('now', ?)) lets the SQLite planner use
// the indexes on created_at for direct string comparison.
func rfc3339NanoCutoff(now time.Time, windowHours int) string {
	return now.UTC().Add(-time.Duration(windowHours) * time.Hour).Format(time.RFC3339Nano)
}

func (s *Store) saveResultSQLite(ctx context.Context, result checks.Result) error {
	detailsJSON, err := json.Marshal(result.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO health_results (check_id, status, message, details, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.ExecContext(ctx, query,
		result.CheckID,
		result.Status,
		result.Message,
		detailsJSON,
		result.Duration.Milliseconds(),
		result.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) getLatestResultPerCheckSQLite(ctx context.Context) ([]checks.Result, error) {
	query := `
		SELECT check_id, status, message, details, duration_ms, created_at
		FROM (
			SELECT
				check_id,
				status,
				message,
				details,
				duration_ms,
				created_at,
				ROW_NUMBER() OVER (PARTITION BY check_id ORDER BY created_at DESC) AS rn
			FROM health_results
		)
		WHERE rn = 1
		ORDER BY check_id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []checks.Result
	for rows.Next() {
		var r checks.Result
		var detailsJSON []byte
		var durationMs int64
		var createdRaw any

		if err := rows.Scan(&r.CheckID, &r.Status, &r.Message, &detailsJSON, &durationMs, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}

		r.Timestamp = ts
		r.Duration = time.Duration(durationMs) * time.Millisecond
		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &r.Details); err != nil {
				return nil, fmt.Errorf("unmarshal details failed: %w", err)
			}
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) getRecentResultsSQLite(ctx context.Context, checkID string, limit int) ([]checks.Result, error) {
	query := `
		SELECT check_id, status, message, details, duration_ms, created_at
		FROM health_results
		WHERE check_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, checkID, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []checks.Result
	for rows.Next() {
		var r checks.Result
		var detailsJSON []byte
		var durationMs int64
		var createdRaw any

		if err := rows.Scan(&r.CheckID, &r.Status, &r.Message, &detailsJSON, &durationMs, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		r.Timestamp = ts
		_ = durationMs
		r.Duration = checks.Result{}.Duration

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &r.Details); err != nil {
				return nil, fmt.Errorf("unmarshal details failed: %w", err)
			}
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) cleanupOldResultsSQLite(ctx context.Context, retentionHours int) (int64, error) {
	query := `
		DELETE FROM health_results
		WHERE created_at < ?
	`
	cutoff := rfc3339NanoCutoff(time.Now(), retentionHours)
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) getTimelineEventsSQLite(ctx context.Context, limit int) ([]TimelineEvent, error) {
	query := `
		SELECT check_id, status, message, details, created_at
		FROM health_results
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var e TimelineEvent
		var detailsJSON []byte
		var createdRaw any

		if err := rows.Scan(&e.CheckID, &e.Status, &e.Message, &detailsJSON, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		e.Timestamp = ts.UTC().Format(time.RFC3339)

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &e.Details); err != nil {
				return nil, fmt.Errorf("unmarshal timeline details failed: %w", err)
			}
		}

		events = append(events, e)
	}

	return events, rows.Err()
}

func (s *Store) getUptimeStatsSQLite(ctx context.Context, windowHours int) (*UptimeStats, error) {
	if windowHours <= 0 {
		windowHours = 24
	}

	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'ok' THEN 1 ELSE 0 END), 0) as ok_count,
			COALESCE(SUM(CASE WHEN status = 'warning' THEN 1 ELSE 0 END), 0) as warning_count,
			COALESCE(SUM(CASE WHEN status = 'critical' THEN 1 ELSE 0 END), 0) as critical_count
		FROM health_results
		WHERE created_at >= ?
	`
	cutoff := rfc3339NanoCutoff(time.Now(), windowHours)

	var stats UptimeStats
	err := s.db.QueryRowContext(ctx, query, cutoff).Scan(
		&stats.TotalEvents,
		&stats.OkEvents,
		&stats.WarningEvents,
		&stats.CriticalEvents,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	stats.WindowHours = windowHours
	if stats.TotalEvents > 0 {
		stats.UptimePercentage = float64(stats.OkEvents) / float64(stats.TotalEvents) * 100
	} else {
		stats.UptimePercentage = 100
	}

	return &stats, nil
}

func (s *Store) getUptimeHistorySQLite(ctx context.Context, windowHours, bucketCount int) (*UptimeHistory, error) {
	if bucketCount <= 0 {
		bucketCount = 24
	}
	if windowHours <= 0 {
		windowHours = 24
	}

	bucketDuration := time.Duration(windowHours) * time.Hour / time.Duration(bucketCount)
	now := time.Now().UTC()
	start := now.Add(-time.Duration(windowHours) * time.Hour)

	startUnix := start.Unix()
	bucketSeconds := int64(bucketDuration / time.Second)
	if bucketSeconds <= 0 {
		bucketSeconds = 1
	}

	// Single aggregation: count health-result events per (bucket, status). The
	// previous implementation issued bucketCount × distinctCheckIDs prepared-
	// statement round-trips to read "status as of bucket boundary"; that
	// scaled linearly with bucket × check count and was the dominant source
	// of dashboard latency on populated databases. Counting events per slice
	// is the natural shape for the stacked-area trend chart, which normalizes
	// per render anyway.
	query := `
		SELECT
			CAST((CAST(strftime('%s', created_at) AS INTEGER) - ?) / ? AS INTEGER) AS bucket,
			status,
			COUNT(*) AS n
		FROM health_results
		WHERE created_at >= ? AND created_at <= ?
		GROUP BY bucket, status
	`
	startStr := start.Format(time.RFC3339Nano)
	endStr := now.Format(time.RFC3339Nano)

	rows, err := s.db.QueryContext(ctx, query, startUnix, bucketSeconds, startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("query bucket aggregation failed: %w", err)
	}
	defer rows.Close()

	buckets := make([]UptimeHistoryBucket, bucketCount)
	for i := 0; i < bucketCount; i++ {
		bucketTime := start.Add(time.Duration(i) * bucketDuration)
		if bucketTime.After(now) {
			bucketTime = now
		}
		buckets[i] = UptimeHistoryBucket{Timestamp: bucketTime}
	}

	var totalSnapshots, totalOk, totalWarning, totalCritical int
	for rows.Next() {
		var bucket int
		var status string
		var n int
		if err := rows.Scan(&bucket, &status, &n); err != nil {
			return nil, fmt.Errorf("scan bucket row failed: %w", err)
		}
		if bucket < 0 {
			bucket = 0
		}
		if bucket >= bucketCount {
			bucket = bucketCount - 1
		}
		buckets[bucket].Total += n
		switch status {
		case "ok":
			buckets[bucket].Ok += n
			totalOk += n
		case "warning":
			buckets[bucket].Warning += n
			totalWarning += n
		case "critical":
			buckets[bucket].Critical += n
			totalCritical += n
		}
		totalSnapshots += n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	uptimePercent := 100.0
	if totalSnapshots > 0 {
		uptimePercent = float64(totalOk) / float64(totalSnapshots) * 100
	}

	return &UptimeHistory{
		Buckets: buckets,
		Overall: UptimeStats{
			TotalEvents:      totalSnapshots,
			OkEvents:         totalOk,
			WarningEvents:    totalWarning,
			CriticalEvents:   totalCritical,
			UptimePercentage: uptimePercent,
			WindowHours:      windowHours,
		},
		WindowHours: windowHours,
		BucketCount: bucketCount,
	}, nil
}

func (s *Store) getCheckTrendsSQLite(ctx context.Context, windowHours int) (*CheckTrendsResponse, error) {
	if windowHours <= 0 {
		windowHours = 24
	}

	query := `
		SELECT
			check_id,
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'ok' THEN 1 END) as ok_count,
			COUNT(CASE WHEN status = 'warning' THEN 1 END) as warning_count,
			COUNT(CASE WHEN status = 'critical' THEN 1 END) as critical_count,
			MAX(created_at) as last_checked
		FROM health_results
		WHERE created_at >= ?
		GROUP BY check_id
		ORDER BY
			CASE WHEN COUNT(*) = 0 THEN 100 ELSE COUNT(CASE WHEN status = 'ok' THEN 1 END) * 100.0 / COUNT(*) END ASC,
			check_id ASC
	`
	cutoff := rfc3339NanoCutoff(time.Now(), windowHours)
	rows, err := s.db.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var trends []CheckTrend
	for rows.Next() {
		var t CheckTrend
		var lastCheckedRaw any

		if err := rows.Scan(&t.CheckID, &t.Total, &t.Ok, &t.Warning, &t.Critical, &lastCheckedRaw); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		lastChecked, err := parseDBTime(lastCheckedRaw)
		if err != nil {
			return nil, fmt.Errorf("parse last_checked failed: %w", err)
		}
		t.LastChecked = lastChecked.UTC().Format(time.RFC3339)

		if t.Total > 0 {
			t.UptimePercent = float64(t.Ok) / float64(t.Total) * 100
		} else {
			t.UptimePercent = 100
		}

		trends = append(trends, t)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("rows error: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close trends rows failed: %w", err)
	}

	currentStmt, err := s.db.PrepareContext(ctx, `
		SELECT status
		FROM health_results
		WHERE check_id = ? AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT 1
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare current status query failed: %w", err)
	}
	defer currentStmt.Close()

	recentStmt, err := s.db.PrepareContext(ctx, `
		SELECT status
		FROM health_results
		WHERE check_id = ? AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT 12
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare recent statuses query failed: %w", err)
	}
	defer recentStmt.Close()

	for i := range trends {
		if err := currentStmt.QueryRowContext(ctx, trends[i].CheckID, cutoff).Scan(&trends[i].CurrentStatus); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				trends[i].CurrentStatus = "ok"
			} else {
				return nil, fmt.Errorf("current status query failed: %w", err)
			}
		}

		recentRows, err := recentStmt.QueryContext(ctx, trends[i].CheckID, cutoff)
		if err != nil {
			return nil, fmt.Errorf("recent statuses query failed: %w", err)
		}
		for recentRows.Next() {
			var status string
			if err := recentRows.Scan(&status); err != nil {
				recentRows.Close()
				return nil, fmt.Errorf("scan recent status failed: %w", err)
			}
			trends[i].RecentStatuses = append(trends[i].RecentStatuses, status)
		}
		if err := recentRows.Err(); err != nil {
			recentRows.Close()
			return nil, fmt.Errorf("recent rows error: %w", err)
		}
		if err := recentRows.Close(); err != nil {
			return nil, fmt.Errorf("close recent rows failed: %w", err)
		}
	}

	return &CheckTrendsResponse{
		Trends:      trends,
		WindowHours: windowHours,
		TotalChecks: len(trends),
	}, nil
}

func (s *Store) getTransitionsSQLite(ctx context.Context, windowHours, limit int) (*TransitionsResponse, error) {
	if windowHours <= 0 {
		windowHours = 24
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `
		WITH ordered_results AS (
			SELECT
				check_id,
				status,
				message,
				created_at,
				LAG(status) OVER (PARTITION BY check_id ORDER BY created_at) as prev_status
			FROM health_results
			WHERE created_at >= ?
		)
		SELECT check_id, created_at, prev_status, status, message
		FROM ordered_results
		WHERE prev_status IS NOT NULL AND prev_status != status
		ORDER BY created_at DESC
		LIMIT ?
	`
	cutoff := rfc3339NanoCutoff(time.Now(), windowHours)
	rows, err := s.db.QueryContext(ctx, query, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var transitions []Transition
	for rows.Next() {
		var i Transition
		var createdRaw any
		if err := rows.Scan(&i.CheckID, &createdRaw, &i.FromStatus, &i.ToStatus, &i.Message); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp failed: %w", err)
		}
		i.Timestamp = ts.UTC().Format(time.RFC3339)
		transitions = append(transitions, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &TransitionsResponse{
		Transitions: transitions,
		WindowHours: windowHours,
		Total:       len(transitions),
	}, nil
}

func (s *Store) saveHostInventorySnapshotSQLite(ctx context.Context, inv hostinventory.HostInventory) (*hostinventory.SnapshotRecord, []hostinventory.Change, error) {
	if inv.CollectedAt.IsZero() {
		inv.CollectedAt = time.Now().UTC()
	}
	if inv.Fingerprint == "" {
		inv.Fingerprint = hostinventory.Fingerprint(inv)
	}
	id := fmt.Sprintf("hostinv_%s_%d", inv.Fingerprint, time.Now().UTC().UnixNano())
	inv.ID = id
	inventoryJSON, err := json.Marshal(inv)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal host inventory: %w", err)
	}
	previous, _ := s.getLatestHostInventorySnapshotSQLite(ctx)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO host_inventory_snapshots (id, collected_at, platform, os, arch, boot_id, kernel_release, fingerprint, inventory_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, inv.CollectedAt.UTC().Format(time.RFC3339Nano), inv.Platform, inv.OS, inv.Arch, inv.BootID, inv.Kernel.Release, inv.Fingerprint, inventoryJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("insert host inventory snapshot: %w", err)
	}
	record := &hostinventory.SnapshotRecord{
		ID:            id,
		CollectedAt:   inv.CollectedAt.UTC(),
		Platform:      inv.Platform,
		OS:            inv.OS,
		Arch:          inv.Arch,
		BootID:        inv.BootID,
		KernelRelease: inv.Kernel.Release,
		Fingerprint:   inv.Fingerprint,
		Inventory:     inv,
	}
	changes := deriveHostInventoryChanges(previous, record)
	for _, change := range changes {
		detailsJSON, _ := json.Marshal(change.Details)
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO host_inventory_changes (from_snapshot_id, to_snapshot_id, change_type, severity, summary, details_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, nullableString(change.FromSnapshotID), nullableString(change.ToSnapshotID), change.ChangeType, change.Severity, change.Summary, detailsJSON, change.CreatedAt.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return record, changes, fmt.Errorf("insert host inventory change: %w", err)
		}
	}
	return record, changes, nil
}

func (s *Store) getLatestHostInventorySnapshotSQLite(ctx context.Context) (*hostinventory.SnapshotRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, collected_at, platform, os, arch, COALESCE(boot_id, ''), COALESCE(kernel_release, ''), fingerprint, inventory_json
		FROM host_inventory_snapshots
		ORDER BY collected_at DESC
		LIMIT 1
	`)
	return scanHostInventorySnapshot(row)
}

func (s *Store) getRecentHostInventoryChangesSQLite(ctx context.Context, limit int) ([]hostinventory.Change, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(from_snapshot_id, ''), COALESCE(to_snapshot_id, ''), change_type, severity, summary, details_json, created_at
		FROM host_inventory_changes
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query host inventory changes: %w", err)
	}
	defer rows.Close()
	var changes []hostinventory.Change
	for rows.Next() {
		var change hostinventory.Change
		var detailsJSON []byte
		var createdRaw any
		if err := rows.Scan(&change.ID, &change.FromSnapshotID, &change.ToSnapshotID, &change.ChangeType, &change.Severity, &change.Summary, &detailsJSON, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan host inventory change: %w", err)
		}
		change.CreatedAt, err = parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse host inventory change time: %w", err)
		}
		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &change.Details)
		}
		changes = append(changes, change)
	}
	return changes, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanHostInventorySnapshot(row rowScanner) (*hostinventory.SnapshotRecord, error) {
	var record hostinventory.SnapshotRecord
	var collectedRaw any
	var inventoryJSON []byte
	if err := row.Scan(&record.ID, &collectedRaw, &record.Platform, &record.OS, &record.Arch, &record.BootID, &record.KernelRelease, &record.Fingerprint, &inventoryJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan host inventory snapshot: %w", err)
	}
	collectedAt, err := parseDBTime(collectedRaw)
	if err != nil {
		return nil, fmt.Errorf("parse host inventory snapshot time: %w", err)
	}
	record.CollectedAt = collectedAt.UTC()
	if err := json.Unmarshal(inventoryJSON, &record.Inventory); err != nil {
		return nil, fmt.Errorf("unmarshal host inventory snapshot: %w", err)
	}
	return &record, nil
}

func deriveHostInventoryChanges(previous, current *hostinventory.SnapshotRecord) []hostinventory.Change {
	if current == nil || previous == nil || previous.Fingerprint == current.Fingerprint {
		return nil
	}
	now := current.CollectedAt
	changes := []hostinventory.Change{{
		FromSnapshotID: previous.ID,
		ToSnapshotID:   current.ID,
		ChangeType:     "inventory_fingerprint_changed",
		Severity:       "info",
		Summary:        "Host inventory fingerprint changed",
		Details:        map[string]any{"fromFingerprint": previous.Fingerprint, "toFingerprint": current.Fingerprint},
		CreatedAt:      now,
	}}
	if previous.KernelRelease != current.KernelRelease {
		changes = append(changes, hostinventory.Change{
			FromSnapshotID: previous.ID,
			ToSnapshotID:   current.ID,
			ChangeType:     "kernel_release_changed",
			Severity:       "warning",
			Summary:        "Running kernel release changed",
			Details:        map[string]any{"from": previous.KernelRelease, "to": current.KernelRelease},
			CreatedAt:      now,
		})
	}
	return changes
}

func (s *Store) upsertIncidentSQLite(ctx context.Context, input incidents.UpsertInput) (*incidents.Incident, error) {
	if input.ObservedAt.IsZero() {
		input.ObservedAt = time.Now().UTC()
	}
	if input.Fingerprint == "" {
		input.Fingerprint = incidents.Fingerprint(string(input.Type), input.SourceCheckID, input.Title)
	}
	id := "inc_" + strings.TrimPrefix(input.Fingerprint, "incfp_")
	sourceCheckIDs, _ := json.Marshal(nonEmptyUnique([]string{input.SourceCheckID}))
	sourceResultIDs, _ := json.Marshal([]string{})
	evidenceJSON, _ := json.Marshal(input.Evidence)
	recommendationsJSON, _ := json.Marshal(input.Recommendations)
	evidenceItemsJSON, _ := json.Marshal(input.EvidenceItems)
	corroborationNeededJSON, _ := json.Marshal(input.CorroborationNeeded)
	safeActionsJSON, _ := json.Marshal(input.SafeActions)
	operatorActionsJSON, _ := json.Marshal(input.OperatorActions)
	rollbackOrFallbackJSON, _ := json.Marshal(input.RollbackOrFallback)
	postChecksJSON, _ := json.Marshal(input.PostChecks)
	remediationCandidatesJSON, _ := json.Marshal(input.RemediationCandidates)
	remediationArtifactsJSON, _ := json.Marshal(input.RemediationArtifacts)
	outcomeJSON, _ := json.Marshal(input.Outcome)
	var outcomeValue any
	if input.Outcome != nil {
		outcomeValue = outcomeJSON
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO incidents (
			id, fingerprint, type, severity, status, title, summary, detected_at, last_seen_at, updated_at,
			boot_id, previous_boot_id, source_check_ids_json, source_result_ids_json, evidence_json, recommendations_json,
			diagnosis, confidence, evidence_items_json, corroboration_needed_json, safe_actions_json, operator_actions_json,
			rollback_or_fallback_json, post_checks_json, remediation_candidates_json, remediation_artifacts_json, outcome_json,
			event_count, observation_count
		) VALUES (?, ?, ?, ?, 'open', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0)
		ON CONFLICT(fingerprint) DO UPDATE SET
			severity = CASE
				WHEN incidents.severity = 'critical' OR excluded.severity = 'critical' THEN 'critical'
				WHEN incidents.severity = 'warning' OR excluded.severity = 'warning' THEN 'warning'
				ELSE excluded.severity
			END,
			status = CASE WHEN incidents.status = 'resolved' THEN 'open' ELSE incidents.status END,
			title = excluded.title,
			summary = excluded.summary,
			last_seen_at = excluded.last_seen_at,
			updated_at = excluded.updated_at,
			resolved_at = CASE WHEN incidents.status = 'resolved' THEN NULL ELSE incidents.resolved_at END,
			boot_id = excluded.boot_id,
			previous_boot_id = excluded.previous_boot_id,
			source_check_ids_json = excluded.source_check_ids_json,
			evidence_json = excluded.evidence_json,
			recommendations_json = excluded.recommendations_json,
			diagnosis = excluded.diagnosis,
			confidence = excluded.confidence,
			evidence_items_json = excluded.evidence_items_json,
			corroboration_needed_json = excluded.corroboration_needed_json,
			safe_actions_json = excluded.safe_actions_json,
			operator_actions_json = excluded.operator_actions_json,
			rollback_or_fallback_json = excluded.rollback_or_fallback_json,
			post_checks_json = excluded.post_checks_json,
			remediation_candidates_json = excluded.remediation_candidates_json,
			remediation_artifacts_json = CASE
				WHEN excluded.remediation_artifacts_json IS NULL OR CAST(excluded.remediation_artifacts_json AS TEXT) IN ('null', '[]') THEN incidents.remediation_artifacts_json
				ELSE excluded.remediation_artifacts_json
			END,
			outcome_json = COALESCE(excluded.outcome_json, incidents.outcome_json),
			event_count = CASE WHEN incidents.status = 'resolved' THEN incidents.event_count + 1 ELSE incidents.event_count END
	`, id, input.Fingerprint, input.Type, input.Severity, input.Title, input.Summary,
		input.ObservedAt.UTC().Format(time.RFC3339Nano),
		input.ObservedAt.UTC().Format(time.RFC3339Nano),
		input.ObservedAt.UTC().Format(time.RFC3339Nano),
		nullableString(input.BootID),
		nullableString(input.PreviousBootID),
		sourceCheckIDs,
		sourceResultIDs,
		evidenceJSON,
		recommendationsJSON,
		input.Diagnosis,
		input.Confidence,
		evidenceItemsJSON,
		corroborationNeededJSON,
		safeActionsJSON,
		operatorActionsJSON,
		rollbackOrFallbackJSON,
		postChecksJSON,
		remediationCandidatesJSON,
		remediationArtifactsJSON,
		outcomeValue,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert incident: %w", err)
	}
	incident, err := s.getIncidentByFingerprintSQLite(ctx, input.Fingerprint)
	if err != nil {
		return nil, err
	}
	if incident == nil {
		return nil, sql.ErrNoRows
	}
	obsEvidenceJSON, _ := json.Marshal(input.Evidence)
	shouldRecord, err := s.shouldRecordIncidentObservationSQLite(ctx, incident.ID, input, obsEvidenceJSON)
	if err != nil {
		return nil, err
	}
	if shouldRecord {
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO incident_observations (incident_id, observed_at, source_check_id, severity, status, message, evidence_json)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, incident.ID, input.ObservedAt.UTC().Format(time.RFC3339Nano), input.SourceCheckID, input.Severity, string(incident.Status), input.Summary, obsEvidenceJSON); err != nil {
			return nil, fmt.Errorf("insert incident observation: %w", err)
		}
		if _, err := s.db.ExecContext(ctx, `
			UPDATE incidents
			SET observation_count = observation_count + 1
			WHERE id = ?
		`, incident.ID); err != nil {
			return nil, fmt.Errorf("update incident observation count: %w", err)
		}
		incident, err = s.getIncidentSQLite(ctx, incident.ID)
		if err != nil {
			return nil, err
		}
	}
	return incident, nil
}

func (s *Store) ensureIncidentContractColumns(ctx context.Context) error {
	columns := []struct {
		name string
		def  string
	}{
		{"diagnosis", "TEXT NOT NULL DEFAULT ''"},
		{"confidence", "TEXT NOT NULL DEFAULT ''"},
		{"evidence_items_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"corroboration_needed_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"safe_actions_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"operator_actions_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"rollback_or_fallback_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"post_checks_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"remediation_candidates_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"remediation_artifacts_json", "TEXT NOT NULL DEFAULT '[]'"},
		{"outcome_json", "TEXT"},
	}
	for _, column := range columns {
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE incidents ADD COLUMN %s %s", column.name, column.def)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column name") ||
				strings.Contains(strings.ToLower(err.Error()), "no such table") {
				continue
			}
			return fmt.Errorf("add incidents.%s column: %w", column.name, err)
		}
	}
	return nil
}

func (s *Store) shouldRecordIncidentObservationSQLite(ctx context.Context, incidentID string, input incidents.UpsertInput, evidenceJSON []byte) (bool, error) {
	const observationQuietWindow = 30 * time.Minute
	row := s.db.QueryRowContext(ctx, `
		SELECT observed_at, severity, COALESCE(status, ''), message, evidence_json
		FROM incident_observations
		WHERE incident_id = ? AND COALESCE(source_check_id, '') = COALESCE(?, '')
		ORDER BY observed_at DESC
		LIMIT 1
	`, incidentID, nullableString(input.SourceCheckID))
	var observedRaw any
	var severity incidents.Severity
	var status string
	var message string
	var previousEvidenceJSON []byte
	if err := row.Scan(&observedRaw, &severity, &status, &message, &previousEvidenceJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, fmt.Errorf("query latest incident observation: %w", err)
	}
	observedAt, err := parseDBTime(observedRaw)
	if err != nil {
		return false, fmt.Errorf("parse latest incident observation time: %w", err)
	}
	sameEvidence := string(previousEvidenceJSON) == string(evidenceJSON)
	sameObservation := severity == input.Severity && message == input.Summary && sameEvidence
	if sameObservation && input.ObservedAt.Sub(observedAt) < observationQuietWindow {
		return false, nil
	}
	return true, nil
}

func (s *Store) listIncidentsSQLite(ctx context.Context, filters incidents.ListFilters) (*incidents.ListResponse, error) {
	if filters.Limit <= 0 || filters.Limit > 200 {
		filters.Limit = 50
	}
	where := []string{"1=1"}
	args := []any{}
	if filters.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filters.Status)
	}
	if filters.Severity != "" {
		where = append(where, "severity = ?")
		args = append(args, filters.Severity)
	}
	if filters.Type != "" {
		where = append(where, "type = ?")
		args = append(args, filters.Type)
	}
	if filters.Since != nil {
		where = append(where, "updated_at >= ?")
		args = append(args, filters.Since.UTC().Format(time.RFC3339Nano))
	}
	if filters.Until != nil {
		where = append(where, "updated_at <= ?")
		args = append(args, filters.Until.UTC().Format(time.RFC3339Nano))
	}
	args = append(args, filters.Limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, fingerprint, type, severity, status, title, summary, detected_at, last_seen_at, updated_at,
			resolved_at, acknowledged_at, ignored_at, COALESCE(boot_id, ''), COALESCE(previous_boot_id, ''),
			source_check_ids_json, source_result_ids_json, evidence_json, recommendations_json,
			diagnosis, confidence, evidence_items_json, corroboration_needed_json, safe_actions_json, operator_actions_json,
			rollback_or_fallback_json, post_checks_json, remediation_candidates_json, remediation_artifacts_json, outcome_json,
			event_count, observation_count, operator_notes
		FROM incidents
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY updated_at DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()
	var list []incidents.Incident
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *incident)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &incidents.ListResponse{Incidents: list, Total: len(list), Filters: filters}, nil
}

func (s *Store) getIncidentSQLite(ctx context.Context, id string) (*incidents.Incident, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, fingerprint, type, severity, status, title, summary, detected_at, last_seen_at, updated_at,
			resolved_at, acknowledged_at, ignored_at, COALESCE(boot_id, ''), COALESCE(previous_boot_id, ''),
			source_check_ids_json, source_result_ids_json, evidence_json, recommendations_json,
			diagnosis, confidence, evidence_items_json, corroboration_needed_json, safe_actions_json, operator_actions_json,
			rollback_or_fallback_json, post_checks_json, remediation_candidates_json, remediation_artifacts_json, outcome_json,
			event_count, observation_count, operator_notes
		FROM incidents
		WHERE id = ?
	`, id)
	return scanIncident(row)
}

func (s *Store) getIncidentByFingerprintSQLite(ctx context.Context, fingerprint string) (*incidents.Incident, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, fingerprint, type, severity, status, title, summary, detected_at, last_seen_at, updated_at,
			resolved_at, acknowledged_at, ignored_at, COALESCE(boot_id, ''), COALESCE(previous_boot_id, ''),
			source_check_ids_json, source_result_ids_json, evidence_json, recommendations_json,
			diagnosis, confidence, evidence_items_json, corroboration_needed_json, safe_actions_json, operator_actions_json,
			rollback_or_fallback_json, post_checks_json, remediation_candidates_json, remediation_artifacts_json, outcome_json,
			event_count, observation_count, operator_notes
		FROM incidents
		WHERE fingerprint = ?
	`, fingerprint)
	return scanIncident(row)
}

func (s *Store) listIncidentObservationsSQLite(ctx context.Context, incidentID string, limit int) ([]incidents.Observation, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, incident_id, observed_at, COALESCE(source_check_id, ''), severity, COALESCE(status, ''), message, evidence_json
		FROM incident_observations
		WHERE incident_id = ?
		ORDER BY observed_at DESC
		LIMIT ?
	`, incidentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var observations []incidents.Observation
	for rows.Next() {
		var obs incidents.Observation
		var observedRaw any
		var evidenceJSON []byte
		if err := rows.Scan(&obs.ID, &obs.IncidentID, &observedRaw, &obs.SourceCheckID, &obs.Severity, &obs.Status, &obs.Message, &evidenceJSON); err != nil {
			return nil, err
		}
		observedAt, err := parseDBTime(observedRaw)
		if err != nil {
			return nil, err
		}
		obs.ObservedAt = observedAt.UTC()
		_ = json.Unmarshal(evidenceJSON, &obs.Evidence)
		observations = append(observations, obs)
	}
	return observations, rows.Err()
}

func (s *Store) updateIncidentStatusSQLite(ctx context.Context, incidentID string, status incidents.Status, note string) (*incidents.Incident, error) {
	current, err := s.getIncidentSQLite(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, sql.ErrNoRows
	}
	now := time.Now().UTC()
	var acknowledgedAt any
	var resolvedAt any
	var ignoredAt any
	if status == incidents.StatusAcknowledged {
		acknowledgedAt = now.Format(time.RFC3339Nano)
	}
	if status == incidents.StatusResolved {
		resolvedAt = now.Format(time.RFC3339Nano)
	}
	if status == incidents.StatusIgnored {
		ignoredAt = now.Format(time.RFC3339Nano)
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE incidents
		SET status = ?, updated_at = ?, acknowledged_at = COALESCE(?, acknowledged_at),
			resolved_at = COALESCE(?, resolved_at), ignored_at = COALESCE(?, ignored_at),
			operator_notes = CASE WHEN ? = '' THEN operator_notes ELSE trim(operator_notes || char(10) || ?) END
		WHERE id = ?
	`, status, now.Format(time.RFC3339Nano), acknowledgedAt, resolvedAt, ignoredAt, note, note, incidentID)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO incident_status_history (incident_id, from_status, to_status, note, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, incidentID, current.Status, status, note, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return s.getIncidentSQLite(ctx, incidentID)
}

func (s *Store) recordIncidentRemediationArtifactSQLite(ctx context.Context, incidentID string, artifact incidents.RemediationArtifact) (*incidents.Incident, error) {
	current, err := s.getIncidentSQLite(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, sql.ErrNoRows
	}
	artifacts := upsertRemediationArtifact(current.RemediationArtifacts, artifact)
	payload, err := json.Marshal(artifacts)
	if err != nil {
		return nil, fmt.Errorf("marshal remediation artifacts: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE incidents
		SET remediation_artifacts_json = ?, updated_at = ?
		WHERE id = ?
	`, payload, now, incidentID); err != nil {
		return nil, fmt.Errorf("record remediation artifact: %w", err)
	}
	return s.getIncidentSQLite(ctx, incidentID)
}

func (s *Store) recordIncidentRemediationOutcomeSQLite(ctx context.Context, incidentID string, outcome incidents.Outcome) (*incidents.Incident, error) {
	current, err := s.getIncidentSQLite(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, sql.ErrNoRows
	}
	if outcome.ReportedAt.IsZero() {
		outcome.ReportedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(outcome)
	if err != nil {
		return nil, fmt.Errorf("marshal remediation outcome: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE incidents
		SET outcome_json = ?, updated_at = ?,
			operator_notes = CASE WHEN ? = '' THEN operator_notes ELSE trim(operator_notes || char(10) || ?) END
		WHERE id = ?
	`, payload, now, outcome.Note, outcome.Note, incidentID); err != nil {
		return nil, fmt.Errorf("record remediation outcome: %w", err)
	}
	return s.getIncidentSQLite(ctx, incidentID)
}

func upsertRemediationArtifact(existing []incidents.RemediationArtifact, artifact incidents.RemediationArtifact) []incidents.RemediationArtifact {
	for i := range existing {
		if existing[i].ID == artifact.ID || existing[i].RemediationID == artifact.RemediationID {
			existing[i] = artifact
			return existing
		}
	}
	return append(existing, artifact)
}

func scanIncident(row rowScanner) (*incidents.Incident, error) {
	var incident incidents.Incident
	var detectedRaw, lastSeenRaw, updatedRaw any
	var resolvedRaw, acknowledgedRaw, ignoredRaw any
	var sourceChecksJSON, sourceResultsJSON, evidenceJSON, recommendationsJSON []byte
	var evidenceItemsJSON, corroborationNeededJSON, safeActionsJSON, operatorActionsJSON []byte
	var rollbackOrFallbackJSON, postChecksJSON, remediationCandidatesJSON, remediationArtifactsJSON []byte
	var outcomeJSON []byte
	if err := row.Scan(
		&incident.ID, &incident.Fingerprint, &incident.Type, &incident.Severity, &incident.Status,
		&incident.Title, &incident.Summary, &detectedRaw, &lastSeenRaw, &updatedRaw,
		&resolvedRaw, &acknowledgedRaw, &ignoredRaw, &incident.BootID, &incident.PreviousBootID,
		&sourceChecksJSON, &sourceResultsJSON, &evidenceJSON, &recommendationsJSON,
		&incident.Diagnosis, &incident.Confidence, &evidenceItemsJSON, &corroborationNeededJSON, &safeActionsJSON, &operatorActionsJSON,
		&rollbackOrFallbackJSON, &postChecksJSON, &remediationCandidatesJSON, &remediationArtifactsJSON, &outcomeJSON,
		&incident.EventCount, &incident.ObservationCount, &incident.OperatorNotes,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var err error
	if incident.DetectedAt, err = parseDBTime(detectedRaw); err != nil {
		return nil, err
	}
	if incident.LastSeenAt, err = parseDBTime(lastSeenRaw); err != nil {
		return nil, err
	}
	if incident.UpdatedAt, err = parseDBTime(updatedRaw); err != nil {
		return nil, err
	}
	incident.ResolvedAt = parseOptionalTimePtr(resolvedRaw)
	incident.AcknowledgedAt = parseOptionalTimePtr(acknowledgedRaw)
	incident.IgnoredAt = parseOptionalTimePtr(ignoredRaw)
	_ = json.Unmarshal(sourceChecksJSON, &incident.SourceCheckIDs)
	_ = json.Unmarshal(sourceResultsJSON, &incident.SourceResultIDs)
	_ = json.Unmarshal(evidenceJSON, &incident.Evidence)
	_ = json.Unmarshal(recommendationsJSON, &incident.Recommendations)
	_ = json.Unmarshal(evidenceItemsJSON, &incident.EvidenceItems)
	_ = json.Unmarshal(corroborationNeededJSON, &incident.CorroborationNeeded)
	_ = json.Unmarshal(safeActionsJSON, &incident.SafeActions)
	_ = json.Unmarshal(operatorActionsJSON, &incident.OperatorActions)
	_ = json.Unmarshal(rollbackOrFallbackJSON, &incident.RollbackOrFallback)
	_ = json.Unmarshal(postChecksJSON, &incident.PostChecks)
	_ = json.Unmarshal(remediationCandidatesJSON, &incident.RemediationCandidates)
	_ = json.Unmarshal(remediationArtifactsJSON, &incident.RemediationArtifacts)
	if len(outcomeJSON) > 0 && string(outcomeJSON) != "null" {
		var outcome incidents.Outcome
		if err := json.Unmarshal(outcomeJSON, &outcome); err == nil {
			incident.Outcome = &outcome
		}
	}
	return &incident, nil
}

func parseOptionalTimePtr(raw any) *time.Time {
	ts, ok := parseNullableDBTime(raw)
	if !ok {
		return nil
	}
	ts = ts.UTC()
	return &ts
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nonEmptyUnique(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func (s *Store) saveActionLogSQLite(ctx context.Context, checkID, actionID string, success bool, message, output, errMsg string, durationMs int64) error {
	query := `
		INSERT INTO action_logs (check_id, action_id, success, message, output, error, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		checkID,
		actionID,
		success,
		message,
		output,
		errMsg,
		durationMs,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) getActionLogsSQLite(ctx context.Context, limit int) (*ActionLogsResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `
		SELECT id, check_id, action_id, success, message, COALESCE(output, ''), COALESCE(error, ''), duration_ms, created_at
		FROM action_logs
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var logs []ActionLog
	for rows.Next() {
		var l ActionLog
		var createdRaw any
		if err := rows.Scan(&l.ID, &l.CheckID, &l.ActionID, &l.Success, &l.Message, &l.Output, &l.Error, &l.DurationMs, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp failed: %w", err)
		}
		l.Timestamp = ts.UTC().Format(time.RFC3339)
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &ActionLogsResponse{Logs: logs, Total: len(logs)}, nil
}

func (s *Store) getActionLogsForCheckSQLite(ctx context.Context, checkID string, limit int) (*ActionLogsResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}

	query := `
		SELECT id, check_id, action_id, success, message, COALESCE(output, ''), COALESCE(error, ''), duration_ms, created_at
		FROM action_logs
		WHERE check_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, checkID, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var logs []ActionLog
	for rows.Next() {
		var l ActionLog
		var createdRaw any
		if err := rows.Scan(&l.ID, &l.CheckID, &l.ActionID, &l.Success, &l.Message, &l.Output, &l.Error, &l.DurationMs, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp failed: %w", err)
		}
		l.Timestamp = ts.UTC().Format(time.RFC3339)
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &ActionLogsResponse{Logs: logs, Total: len(logs)}, nil
}

func (s *Store) saveHealTrackerSQLite(ctx context.Context, checkID string, tracker *checks.HealTracker) error {
	query := `
		INSERT INTO heal_trackers (check_id, last_attempt, last_success, consecutive_failures, total_attempts, total_successes, cooldown_until, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(check_id) DO UPDATE SET
			last_attempt = excluded.last_attempt,
			last_success = excluded.last_success,
			consecutive_failures = excluded.consecutive_failures,
			total_attempts = excluded.total_attempts,
			total_successes = excluded.total_successes,
			cooldown_until = excluded.cooldown_until,
			updated_at = excluded.updated_at
	`

	_, err := s.db.ExecContext(ctx, query,
		checkID,
		nullableTimeToDBText(tracker.LastAttempt),
		nullableTimeToDBText(tracker.LastSuccess),
		tracker.ConsecutiveFailures,
		tracker.TotalAttempts,
		tracker.TotalSuccesses,
		nullableTimeToDBText(tracker.CooldownUntil),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) getAllHealTrackersSQLite(ctx context.Context) (map[string]*checks.HealTracker, error) {
	query := `
		SELECT check_id, last_attempt, last_success, consecutive_failures, total_attempts, total_successes, cooldown_until
		FROM heal_trackers
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	trackers := make(map[string]*checks.HealTracker)
	for rows.Next() {
		var checkID string
		var lastAttemptRaw any
		var lastSuccessRaw any
		var cooldownUntilRaw any
		var tracker checks.HealTracker

		if err := rows.Scan(
			&checkID,
			&lastAttemptRaw,
			&lastSuccessRaw,
			&tracker.ConsecutiveFailures,
			&tracker.TotalAttempts,
			&tracker.TotalSuccesses,
			&cooldownUntilRaw,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		lastAttempt, ok := parseNullableDBTime(lastAttemptRaw)
		if ok {
			tracker.LastAttempt = lastAttempt
		}
		lastSuccess, ok := parseNullableDBTime(lastSuccessRaw)
		if ok {
			tracker.LastSuccess = lastSuccess
		}
		cooldownUntil, ok := parseNullableDBTime(cooldownUntilRaw)
		if ok {
			tracker.CooldownUntil = cooldownUntil
		}

		trackers[checkID] = &tracker
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return trackers, nil
}

func (s *Store) deleteHealTrackerSQLite(ctx context.Context, checkID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM heal_trackers WHERE check_id = ?`, checkID)
	return err
}

func parseDBTime(raw any) (time.Time, error) {
	switch v := raw.(type) {
	case time.Time:
		return v, nil
	case string:
		return parseTimeString(v)
	case []byte:
		return parseTimeString(string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type %T", raw)
	}
}

func parseTimeString(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse time %q", s)
}

func nullableTimeToDBText(ts time.Time) interface{} {
	if ts.IsZero() {
		return nil
	}
	return ts.UTC().Format(time.RFC3339Nano)
}

func parseNullableDBTime(raw any) (time.Time, bool) {
	if raw == nil {
		return time.Time{}, false
	}
	switch v := raw.(type) {
	case time.Time:
		if v.IsZero() {
			return time.Time{}, false
		}
		return v, true
	case string:
		if v == "" {
			return time.Time{}, false
		}
		ts, err := parseTimeString(v)
		if err != nil {
			return time.Time{}, false
		}
		return ts, true
	case []byte:
		if len(v) == 0 {
			return time.Time{}, false
		}
		ts, err := parseTimeString(string(v))
		if err != nil {
			return time.Time{}, false
		}
		return ts, true
	case sql.NullTime:
		if !v.Valid || v.Time.IsZero() {
			return time.Time{}, false
		}
		return v.Time, true
	default:
		return time.Time{}, false
	}
}
