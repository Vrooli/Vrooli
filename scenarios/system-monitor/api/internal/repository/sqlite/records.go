package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func (r *Repository) CreateInvestigation(ctx context.Context, inv *models.Investigation) error {
	details, _ := json.Marshal(inv.Details)
	steps, _ := json.Marshal(inv.Steps)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO investigations (id, status, anomaly_id, start_time, end_time, findings, progress, details, steps)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.Status, inv.AnomalyID, inv.StartTime.UTC(), nullTime(inv.EndTime),
		inv.Findings, inv.Progress, string(details), string(steps),
	)
	return err
}

func (r *Repository) GetInvestigation(ctx context.Context, id string) (*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scanInvestigation(r.db.QueryRowContext(ctx,
		"SELECT id, status, anomaly_id, start_time, end_time, findings, progress, details, steps FROM investigations WHERE id = ?", id,
	))
}

func (r *Repository) UpdateInvestigation(ctx context.Context, inv *models.Investigation) error {
	details, _ := json.Marshal(inv.Details)
	steps, _ := json.Marshal(inv.Steps)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		`UPDATE investigations SET status=?, anomaly_id=?, start_time=?, end_time=?, findings=?, progress=?, details=?, steps=?
		 WHERE id=?`,
		inv.Status, inv.AnomalyID, inv.StartTime.UTC(), nullTime(inv.EndTime),
		inv.Findings, inv.Progress, string(details), string(steps), inv.ID,
	)
	return err
}

func (r *Repository) ListInvestigations(ctx context.Context, filter repository.InvestigationFilter) ([]*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT id, status, anomaly_id, start_time, end_time, findings, progress, details, steps FROM investigations WHERE 1=1"
	args := []interface{}{}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	query += " ORDER BY start_time DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Investigation
	for rows.Next() {
		inv, err := r.scanInvestigationRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, inv)
	}
	return results, rows.Err()
}

func (r *Repository) GetLatestInvestigation(ctx context.Context) (*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scanInvestigation(r.db.QueryRowContext(ctx,
		"SELECT id, status, anomaly_id, start_time, end_time, findings, progress, details, steps FROM investigations ORDER BY start_time DESC LIMIT 1",
	))
}

func (r *Repository) SaveInvestigationStep(ctx context.Context, investigationID string, step *models.InvestigationStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var stepsJSON string
	err := r.db.QueryRowContext(ctx, "SELECT steps FROM investigations WHERE id = ?", investigationID).Scan(&stepsJSON)
	if err != nil {
		return apierrors.NotFound("investigation", investigationID)
	}

	var steps []models.InvestigationStep
	if stepsJSON != "" {
		json.Unmarshal([]byte(stepsJSON), &steps) //nolint:errcheck // malformed legacy step JSON falls back to an empty step list
	}
	steps = append(steps, *step)

	newSteps, _ := json.Marshal(steps)
	_, err = r.db.ExecContext(ctx, "UPDATE investigations SET steps = ? WHERE id = ?", string(newSteps), investigationID)
	return err
}

// ---------------------------------------------------------------------------
// ReportRepository
// ---------------------------------------------------------------------------

func (r *Repository) CreateReport(ctx context.Context, report *models.Report) error {
	data, _ := json.Marshal(report.Data)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO reports (id, type, generated_at, time_range_start, time_range_end, time_range_duration, data, format)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID, report.Type, report.GeneratedAt.UTC(),
		report.TimeRange.StartTime.UTC(), report.TimeRange.EndTime.UTC(), report.TimeRange.Duration,
		string(data), report.Format,
	)
	return err
}

func (r *Repository) GetReport(ctx context.Context, id string) (*models.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var report models.Report
	var data string
	err := r.db.QueryRowContext(ctx,
		"SELECT id, type, generated_at, time_range_start, time_range_end, time_range_duration, data, format FROM reports WHERE id = ?", id,
	).Scan(&report.ID, &report.Type, &report.GeneratedAt,
		&report.TimeRange.StartTime, &report.TimeRange.EndTime, &report.TimeRange.Duration,
		&data, &report.Format,
	)
	if err != nil {
		return nil, apierrors.NotFound("report", id)
	}
	json.Unmarshal([]byte(data), &report.Data) //nolint:errcheck // malformed legacy report data falls back to an empty payload
	return &report, nil
}

func (r *Repository) ListReports(ctx context.Context, filter repository.ReportFilter) ([]*models.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT id, type, generated_at, time_range_start, time_range_end, time_range_duration, data, format FROM reports WHERE 1=1"
	args := []interface{}{}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	query += " ORDER BY generated_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Report
	for rows.Next() {
		var report models.Report
		var data string
		if err := rows.Scan(&report.ID, &report.Type, &report.GeneratedAt,
			&report.TimeRange.StartTime, &report.TimeRange.EndTime, &report.TimeRange.Duration,
			&data, &report.Format,
		); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(data), &report.Data) //nolint:errcheck // malformed legacy report data falls back to an empty payload
		results = append(results, &report)
	}
	return results, rows.Err()
}

func (r *Repository) SaveDetailedReport(ctx context.Context, report *models.DetailedSystemReport) error {
	basicReport := &models.Report{
		ID:          report.ReportID,
		Type:        report.ReportType,
		GeneratedAt: report.GeneratedAt,
		TimeRange: models.ReportTimeRange{
			StartTime: report.TimeRange.StartTime,
			EndTime:   report.TimeRange.EndTime,
			Duration:  report.TimeRange.Duration,
		},
		Data: map[string]interface{}{"detailed": report},
	}
	return r.CreateReport(ctx, basicReport)
}

func (r *Repository) GetDetailedReport(ctx context.Context, id string) (*models.DetailedSystemReport, error) {
	report, err := r.GetReport(ctx, id)
	if err != nil {
		return nil, err
	}

	if detailed, ok := report.Data["detailed"]; ok {
		// Re-marshal and unmarshal to get proper type.
		b, err := json.Marshal(detailed)
		if err != nil {
			return nil, apierrors.Internal("failed to read report", err)
		}
		var result models.DetailedSystemReport
		if err := json.Unmarshal(b, &result); err != nil {
			return nil, apierrors.Internal("failed to read report", err)
		}
		return &result, nil
	}

	return nil, apierrors.NotFound("report", id)
}

func (r *Repository) SaveEnhancedReport(ctx context.Context, report *models.EnhancedSystemReport) error {
	data, _ := json.Marshal(report)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO enhanced_reports (report_id, type, generated_at, report_data) VALUES (?, ?, ?, ?)",
		report.ReportID, report.ReportType, report.GeneratedAt.UTC(), string(data),
	)
	return err
}

func (r *Repository) GetEnhancedReport(ctx context.Context, id string) (*models.EnhancedSystemReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var data string
	err := r.db.QueryRowContext(ctx, "SELECT report_data FROM enhanced_reports WHERE report_id = ?", id).Scan(&data)
	if err != nil {
		return nil, apierrors.NotFound("report", id)
	}

	var report models.EnhancedSystemReport
	if err := json.Unmarshal([]byte(data), &report); err != nil {
		return nil, apierrors.Internal("failed to read report", err)
	}
	return &report, nil
}

func (r *Repository) ListEnhancedReports(ctx context.Context) ([]*models.EnhancedSystemReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.QueryContext(ctx, "SELECT report_data FROM enhanced_reports ORDER BY generated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.EnhancedSystemReport
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var report models.EnhancedSystemReport
		if err := json.Unmarshal([]byte(data), &report); err != nil {
			continue
		}
		results = append(results, &report)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// ThresholdRepository (in-memory)
// ---------------------------------------------------------------------------

func (r *Repository) GetActiveThresholds(ctx context.Context) ([]*models.Threshold, error) {
	r.thMu.RLock()
	defer r.thMu.RUnlock()

	var results []*models.Threshold
	for _, t := range r.th {
		if t.Enabled {
			results = append(results, t)
		}
	}

	if len(results) == 0 {
		results = repository.DefaultThresholds()
	}

	return results, nil
}

func (r *Repository) GetThreshold(ctx context.Context, metricName string) (*models.Threshold, error) {
	r.thMu.RLock()
	defer r.thMu.RUnlock()

	if t, exists := r.th[metricName]; exists {
		return t, nil
	}
	return nil, apierrors.NotFound("threshold", metricName)
}

func (r *Repository) SaveThreshold(ctx context.Context, threshold *models.Threshold) error {
	r.thMu.Lock()
	defer r.thMu.Unlock()

	r.th[threshold.MetricName] = threshold
	return nil
}

func (r *Repository) DeleteThreshold(ctx context.Context, metricName string) error {
	r.thMu.Lock()
	defer r.thMu.Unlock()

	delete(r.th, metricName)
	return nil
}

func (r *Repository) SaveThresholdViolation(ctx context.Context, violation *models.ThresholdViolation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO threshold_violations (metric_name, current_value, threshold_value, severity, violation_type, timestamp, duration, previous_value, trend)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		violation.MetricName, violation.CurrentValue, violation.ThresholdValue,
		violation.Severity, violation.ViolationType, violation.Timestamp.UTC(),
		violation.Duration, violation.PreviousValue, violation.Trend,
	)
	return err
}

func (r *Repository) GetThresholdViolations(ctx context.Context, timeRange repository.TimeRange) ([]*models.ThresholdViolation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.QueryContext(ctx,
		`SELECT metric_name, current_value, threshold_value, severity, violation_type, timestamp, duration, previous_value, trend
		 FROM threshold_violations WHERE timestamp > ? AND timestamp < ? ORDER BY timestamp ASC`,
		timeRange.StartTime.UTC(), timeRange.EndTime.UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.ThresholdViolation
	for rows.Next() {
		var v models.ThresholdViolation
		if err := rows.Scan(&v.MetricName, &v.CurrentValue, &v.ThresholdValue,
			&v.Severity, &v.ViolationType, &v.Timestamp,
			&v.Duration, &v.PreviousValue, &v.Trend,
		); err != nil {
			return nil, err
		}
		results = append(results, &v)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// AlertRepository
// ---------------------------------------------------------------------------

func (r *Repository) CreateAlert(ctx context.Context, alert *models.Alert) error {
	threshold, _ := json.Marshal(alert.Threshold)
	details, _ := json.Marshal(alert.Details)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO alerts (id, type, severity, message, metric_name, metric_value, threshold, details, timestamp, acked_at, resolved_at, acked_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		alert.ID, alert.Type, alert.Severity, alert.Message,
		alert.MetricName, alert.MetricValue, string(threshold), string(details),
		alert.Timestamp.UTC(), nullTime(alert.AckedAt), nullTime(alert.ResolvedAt), alert.AckedBy,
	)
	return err
}

func (r *Repository) GetAlert(ctx context.Context, id string) (*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alert models.Alert
	var threshold, details string
	var ackedAt, resolvedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		"SELECT id, type, severity, message, metric_name, metric_value, threshold, details, timestamp, acked_at, resolved_at, acked_by FROM alerts WHERE id = ?", id,
	).Scan(&alert.ID, &alert.Type, &alert.Severity, &alert.Message,
		&alert.MetricName, &alert.MetricValue, &threshold, &details,
		&alert.Timestamp, &ackedAt, &resolvedAt, &alert.AckedBy,
	)
	if err != nil {
		return nil, apierrors.NotFound("alert", id)
	}

	if threshold != "" && threshold != "null" {
		var th models.Threshold
		if json.Unmarshal([]byte(threshold), &th) == nil {
			alert.Threshold = &th
		}
	}
	json.Unmarshal([]byte(details), &alert.Details) //nolint:errcheck // malformed optional alert details fall back to an empty map

	if ackedAt.Valid {
		alert.AckedAt = &ackedAt.Time
	}
	if resolvedAt.Valid {
		alert.ResolvedAt = &resolvedAt.Time
	}

	return &alert, nil
}

func (r *Repository) UpdateAlert(ctx context.Context, alert *models.Alert) error {
	threshold, _ := json.Marshal(alert.Threshold)
	details, _ := json.Marshal(alert.Details)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		`UPDATE alerts SET type=?, severity=?, message=?, metric_name=?, metric_value=?, threshold=?, details=?, timestamp=?, acked_at=?, resolved_at=?, acked_by=?
		 WHERE id=?`,
		alert.Type, alert.Severity, alert.Message,
		alert.MetricName, alert.MetricValue, string(threshold), string(details),
		alert.Timestamp.UTC(), nullTime(alert.AckedAt), nullTime(alert.ResolvedAt), alert.AckedBy,
		alert.ID,
	)
	return err
}

func (r *Repository) ListAlerts(ctx context.Context, filter repository.AlertFilter) ([]*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT id, type, severity, message, metric_name, metric_value, threshold, details, timestamp, acked_at, resolved_at, acked_by FROM alerts WHERE 1=1"
	args := []interface{}{}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.Severity != "" {
		query += " AND severity = ?"
		args = append(args, filter.Severity)
	}
	switch filter.Status {
	case "active":
		query += " AND resolved_at IS NULL"
	case "resolved":
		query += " AND resolved_at IS NOT NULL"
	case "acknowledged":
		query += " AND acked_at IS NOT NULL"
	}
	if filter.MetricName != "" {
		query += " AND metric_name = ?"
		args = append(args, filter.MetricName)
	}
	query += " ORDER BY timestamp DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Alert
	for rows.Next() {
		alert, err := r.scanAlertRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, alert)
	}
	return results, rows.Err()
}

func (r *Repository) AcknowledgeAlert(ctx context.Context, id string, ackedBy string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.db.ExecContext(ctx, "UPDATE alerts SET acked_at = ?, acked_by = ? WHERE id = ?",
		time.Now().UTC(), ackedBy, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apierrors.NotFound("alert", id)
	}
	return nil
}

func (r *Repository) ResolveAlert(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.db.ExecContext(ctx, "UPDATE alerts SET resolved_at = ? WHERE id = ?",
		time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apierrors.NotFound("alert", id)
	}
	return nil
}

func (r *Repository) GetActiveAlerts(ctx context.Context) ([]*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.QueryContext(ctx,
		"SELECT id, type, severity, message, metric_name, metric_value, threshold, details, timestamp, acked_at, resolved_at, acked_by FROM alerts WHERE resolved_at IS NULL ORDER BY timestamp DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Alert
	for rows.Next() {
		alert, err := r.scanAlertRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, alert)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseTime tries multiple formats to handle SQLite's string timestamp storage.
