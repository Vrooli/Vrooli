package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PostgresRepository implements all repository interfaces for PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// GetApps retrieves all applications from the database
func (r *PostgresRepository) GetApps(ctx context.Context) ([]App, error) {
	query := `
		SELECT
			a.id,
			a.name,
			a.scenario_name,
			a.path,
			a.created_at,
			a.updated_at,
			a.status,
			COALESCE(a.port_mappings, '{}'),
			COALESCE(a.environment, '{}'),
			COALESCE(a.config, '{}'),
			COALESCE(v.view_count, 0) AS view_count,
			v.first_viewed_at,
			v.last_viewed_at
		FROM apps a
		LEFT JOIN app_view_stats v ON v.scenario_name = a.scenario_name
		ORDER BY a.name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query apps: %w", err)
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var app App
		var portMappingsJSON, environmentJSON, configJSON string
		var viewCount int64
		var firstViewed, lastViewed sql.NullTime

		err := rows.Scan(
			&app.ID, &app.Name, &app.ScenarioName, &app.Path,
			&app.CreatedAt, &app.UpdatedAt, &app.Status,
			&portMappingsJSON, &environmentJSON, &configJSON,
			&viewCount, &firstViewed, &lastViewed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan app row: %w", err)
		}

		// Parse JSON fields
		if err := json.Unmarshal([]byte(portMappingsJSON), &app.PortMappings); err != nil {
			app.PortMappings = make(map[string]interface{})
		}
		if err := json.Unmarshal([]byte(environmentJSON), &app.Environment); err != nil {
			app.Environment = make(map[string]interface{})
		}
		if err := json.Unmarshal([]byte(configJSON), &app.Config); err != nil {
			app.Config = make(map[string]interface{})
		}

		app.ViewCount = viewCount
		if firstViewed.Valid {
			value := firstViewed.Time
			app.FirstViewed = &value
		}
		if lastViewed.Valid {
			value := lastViewed.Time
			app.LastViewed = &value
		}

		apps = append(apps, app)
	}

	return apps, rows.Err()
}

// GetApp retrieves a single application by ID
func (r *PostgresRepository) GetApp(ctx context.Context, id string) (*App, error) {
	query := `
		SELECT
			a.id,
			a.name,
			a.scenario_name,
			a.path,
			a.created_at,
			a.updated_at,
			a.status,
			COALESCE(a.port_mappings, '{}'),
			COALESCE(a.environment, '{}'),
			COALESCE(a.config, '{}'),
			COALESCE(v.view_count, 0) AS view_count,
			v.first_viewed_at,
			v.last_viewed_at
		FROM apps a
		LEFT JOIN app_view_stats v ON v.scenario_name = a.scenario_name
		WHERE a.id = $1 OR a.scenario_name = $1
		LIMIT 1
	`

	var app App
	var portMappingsJSON, environmentJSON, configJSON string
	var viewCount int64
	var firstViewed, lastViewed sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&app.ID, &app.Name, &app.ScenarioName, &app.Path,
		&app.CreatedAt, &app.UpdatedAt, &app.Status,
		&portMappingsJSON, &environmentJSON, &configJSON,
		&viewCount, &firstViewed, &lastViewed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("app not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// Parse JSON fields
	json.Unmarshal([]byte(portMappingsJSON), &app.PortMappings)
	json.Unmarshal([]byte(environmentJSON), &app.Environment)
	json.Unmarshal([]byte(configJSON), &app.Config)

	app.ViewCount = viewCount
	if firstViewed.Valid {
		value := firstViewed.Time
		app.FirstViewed = &value
	}
	if lastViewed.Valid {
		value := lastViewed.Time
		app.LastViewed = &value
	}

	return &app, nil
}

// CreateApp creates a new application in the database
func (r *PostgresRepository) CreateApp(ctx context.Context, app *App) error {
	portMappingsJSON, _ := json.Marshal(app.PortMappings)
	environmentJSON, _ := json.Marshal(app.Environment)
	configJSON, _ := json.Marshal(app.Config)

	query := `
		INSERT INTO apps (name, scenario_name, path, status, port_mappings, environment, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx, query,
		app.Name, app.ScenarioName, app.Path, app.Status,
		string(portMappingsJSON), string(environmentJSON), string(configJSON),
	).Scan(&app.ID, &app.CreatedAt, &app.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	return nil
}

// UpdateApp updates an existing application
func (r *PostgresRepository) UpdateApp(ctx context.Context, app *App) error {
	portMappingsJSON, _ := json.Marshal(app.PortMappings)
	environmentJSON, _ := json.Marshal(app.Environment)
	configJSON, _ := json.Marshal(app.Config)

	query := `
		UPDATE apps 
		SET name = $2, scenario_name = $3, path = $4, status = $5,
		    port_mappings = $6, environment = $7, config = $8,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx, query,
		app.ID, app.Name, app.ScenarioName, app.Path, app.Status,
		string(portMappingsJSON), string(environmentJSON), string(configJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to update app: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("app not found: %s", app.ID)
	}

	return nil
}

// UpdateAppStatus updates only the status field of an application
func (r *PostgresRepository) UpdateAppStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE apps 
		SET status = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update app status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("app not found: %s", id)
	}

	return nil
}

// DeleteApp removes an application from the database
func (r *PostgresRepository) DeleteApp(ctx context.Context, id string) error {
	query := `DELETE FROM apps WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("app not found: %s", id)
	}

	return nil
}

// CreateAppStatus records a new status entry for an application
func (r *PostgresRepository) CreateAppStatus(ctx context.Context, status *AppStatus) error {
	query := `
		INSERT INTO app_status (app_id, status, cpu_usage, memory_usage, disk_usage, network_in, network_out)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, timestamp
	`

	err := r.db.QueryRowContext(
		ctx, query,
		status.AppID, status.Status, status.CPUUsage, status.MemUsage,
		status.DiskUsage, status.NetworkIn, status.NetworkOut,
	).Scan(&status.ID, &status.Timestamp)

	if err != nil {
		return fmt.Errorf("failed to create app status: %w", err)
	}

	return nil
}

// GetAppStatus retrieves the latest status for an application
func (r *PostgresRepository) GetAppStatus(ctx context.Context, appID string) (*AppStatus, error) {
	query := `
		SELECT id, app_id, status, cpu_usage, memory_usage, disk_usage, network_in, network_out, timestamp
		FROM app_status
		WHERE app_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var status AppStatus
	err := r.db.QueryRowContext(ctx, query, appID).Scan(
		&status.ID, &status.AppID, &status.Status,
		&status.CPUUsage, &status.MemUsage, &status.DiskUsage,
		&status.NetworkIn, &status.NetworkOut, &status.Timestamp,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no status found for app: %s", appID)
		}
		return nil, fmt.Errorf("failed to get app status: %w", err)
	}

	return &status, nil
}

// GetAppStatusHistory retrieves historical status entries for an application
func (r *PostgresRepository) GetAppStatusHistory(ctx context.Context, appID string, hours int) ([]AppStatus, error) {
	query := `
		SELECT id, app_id, status, cpu_usage, memory_usage, disk_usage, network_in, network_out, timestamp
		FROM app_status
		WHERE app_id = $1 AND timestamp > NOW() - INTERVAL '%d hours'
		ORDER BY timestamp DESC
	`

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(query, hours), appID)
	if err != nil {
		return nil, fmt.Errorf("failed to query status history: %w", err)
	}
	defer rows.Close()

	var statuses []AppStatus
	for rows.Next() {
		var status AppStatus
		err := rows.Scan(
			&status.ID, &status.AppID, &status.Status,
			&status.CPUUsage, &status.MemUsage, &status.DiskUsage,
			&status.NetworkIn, &status.NetworkOut, &status.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status row: %w", err)
		}
		statuses = append(statuses, status)
	}

	return statuses, rows.Err()
}

// CreateAppLog creates a new log entry for an application
func (r *PostgresRepository) CreateAppLog(ctx context.Context, log *AppLog) error {
	query := `
		INSERT INTO app_logs (app_id, level, message, source)
		VALUES ($1, $2, $3, $4)
		RETURNING id, timestamp
	`

	err := r.db.QueryRowContext(
		ctx, query,
		log.AppID, log.Level, log.Message, log.Source,
	).Scan(&log.ID, &log.Timestamp)

	if err != nil {
		return fmt.Errorf("failed to create app log: %w", err)
	}

	return nil
}

// GetAppLogs retrieves logs for an application with pagination
func (r *PostgresRepository) GetAppLogs(ctx context.Context, appID string, limit, offset int) ([]AppLog, error) {
	query := `
		SELECT id, app_id, level, message, source, timestamp
		FROM app_logs
		WHERE app_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, appID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query app logs: %w", err)
	}
	defer rows.Close()

	var logs []AppLog
	for rows.Next() {
		var log AppLog
		err := rows.Scan(
			&log.ID, &log.AppID, &log.Level,
			&log.Message, &log.Source, &log.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log row: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetAppLogsByLevel retrieves logs of a specific level for an application
func (r *PostgresRepository) GetAppLogsByLevel(ctx context.Context, appID string, level string, limit int) ([]AppLog, error) {
	query := `
		SELECT id, app_id, level, message, source, timestamp
		FROM app_logs
		WHERE app_id = $1 AND level = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, appID, level, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query app logs by level: %w", err)
	}
	defer rows.Close()

	var logs []AppLog
	for rows.Next() {
		var log AppLog
		err := rows.Scan(
			&log.ID, &log.AppID, &log.Level,
			&log.Message, &log.Source, &log.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log row: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// RecordAppView increments the view counters for a scenario and returns updated stats
func (r *PostgresRepository) RecordAppView(ctx context.Context, scenarioName string) (*AppViewStats, error) {
	normalized := strings.TrimSpace(scenarioName)
	if normalized == "" {
		return nil, fmt.Errorf("scenario name is required")
	}

	query := `
		INSERT INTO app_view_stats (scenario_name, view_count, first_viewed_at, last_viewed_at, updated_at)
		VALUES ($1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (scenario_name)
		DO UPDATE SET
			view_count = app_view_stats.view_count + 1,
			last_viewed_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		RETURNING scenario_name, view_count, first_viewed_at, last_viewed_at, updated_at
	`

	var stats AppViewStats
	var firstViewed, lastViewed, updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, normalized).Scan(
		&stats.ScenarioName,
		&stats.ViewCount,
		&firstViewed,
		&lastViewed,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record app view: %w", err)
	}

	if firstViewed.Valid {
		value := firstViewed.Time
		stats.FirstViewed = &value
	}
	if lastViewed.Valid {
		value := lastViewed.Time
		stats.LastViewed = &value
	}
	if updatedAt.Valid {
		value := updatedAt.Time
		stats.UpdatedAt = &value
	}

	return &stats, nil
}

// GetAppViewStats fetches aggregated view metrics for all scenarios
func (r *PostgresRepository) GetAppViewStats(ctx context.Context) (map[string]AppViewStats, error) {
	query := `
		SELECT scenario_name, view_count, first_viewed_at, last_viewed_at, updated_at
		FROM app_view_stats
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query app view stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]AppViewStats)

	for rows.Next() {
		var entry AppViewStats
		var firstViewed, lastViewed, updatedAt sql.NullTime

		err := rows.Scan(
			&entry.ScenarioName,
			&entry.ViewCount,
			&firstViewed,
			&lastViewed,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan view stats row: %w", err)
		}

		if firstViewed.Valid {
			value := firstViewed.Time
			entry.FirstViewed = &value
		}
		if lastViewed.Valid {
			value := lastViewed.Time
			entry.LastViewed = &value
		}
		if updatedAt.Valid {
			value := updatedAt.Time
			entry.UpdatedAt = &value
		}

		stats[entry.ScenarioName] = entry
		stats[strings.ToLower(entry.ScenarioName)] = entry
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate view stats rows: %w", err)
	}

	return stats, nil
}

// RecordMetric records a metric value for an application
func (r *PostgresRepository) RecordMetric(ctx context.Context, appID string, metricType string, value float64, metadata map[string]interface{}) error {
	metadataJSON, _ := json.Marshal(metadata)

	query := `
		INSERT INTO app_metrics (app_id, metric_type, value, metadata)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.ExecContext(ctx, query, appID, metricType, value, string(metadataJSON))
	if err != nil {
		return fmt.Errorf("failed to record metric: %w", err)
	}

	return nil
}

// GetMetrics retrieves metrics for an application since a given time
func (r *PostgresRepository) GetMetrics(ctx context.Context, appID string, metricType string, since time.Time) ([]map[string]interface{}, error) {
	query := `
		SELECT id, metric_type, value, metadata, timestamp
		FROM app_metrics
		WHERE app_id = $1 AND metric_type = $2 AND timestamp > $3
		ORDER BY timestamp DESC
	`

	rows, err := r.db.QueryContext(ctx, query, appID, metricType, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []map[string]interface{}
	for rows.Next() {
		var id, metricType string
		var value float64
		var metadataJSON string
		var timestamp time.Time

		err := rows.Scan(&id, &metricType, &value, &metadataJSON, &timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric row: %w", err)
		}

		var metadata map[string]interface{}
		json.Unmarshal([]byte(metadataJSON), &metadata)

		metric := map[string]interface{}{
			"id":          id,
			"metric_type": metricType,
			"value":       value,
			"metadata":    metadata,
			"timestamp":   timestamp,
		}
		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}

// GetAggregatedMetrics retrieves aggregated metrics for an application
func (r *PostgresRepository) GetAggregatedMetrics(ctx context.Context, appID string, interval string) (map[string]interface{}, error) {
	// Validate interval
	validIntervals := map[string]string{
		"1h":  "1 hour",
		"24h": "24 hours",
		"7d":  "7 days",
		"30d": "30 days",
	}

	intervalSQL, ok := validIntervals[interval]
	if !ok {
		intervalSQL = "24 hours"
	}

	query := fmt.Sprintf(`
		SELECT 
			AVG(cpu_usage) as avg_cpu,
			MAX(cpu_usage) as max_cpu,
			AVG(memory_usage) as avg_memory,
			MAX(memory_usage) as max_memory,
			SUM(request_count) as total_requests,
			SUM(error_count) as total_errors,
			AVG(response_time) as avg_response_time
		FROM app_metrics
		WHERE app_id = $1 AND timestamp > NOW() - INTERVAL '%s'
	`, intervalSQL)

	var result map[string]interface{}
	var avgCPU, maxCPU, avgMemory, maxMemory sql.NullFloat64
	var totalRequests, totalErrors, avgResponseTime sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, appID).Scan(
		&avgCPU, &maxCPU, &avgMemory, &maxMemory,
		&totalRequests, &totalErrors, &avgResponseTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated metrics: %w", err)
	}

	result = map[string]interface{}{
		"avg_cpu":           avgCPU.Float64,
		"max_cpu":           maxCPU.Float64,
		"avg_memory":        avgMemory.Float64,
		"max_memory":        maxMemory.Float64,
		"total_requests":    totalRequests.Int64,
		"total_errors":      totalErrors.Int64,
		"avg_response_time": avgResponseTime.Int64,
		"interval":          interval,
	}

	return result, nil
}

// UpdateHealthStatus updates the health status for an application
func (r *PostgresRepository) UpdateHealthStatus(ctx context.Context, appID string, status string, responseTime int) error {
	query := `
		INSERT INTO app_health_status (app_id, status, response_time, last_check)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (app_id) DO UPDATE
		SET status = $2, response_time = $3, last_check = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.ExecContext(ctx, query, appID, status, responseTime)
	if err != nil {
		return fmt.Errorf("failed to update health status: %w", err)
	}

	return nil
}

// GetHealthStatus retrieves the current health status for an application
func (r *PostgresRepository) GetHealthStatus(ctx context.Context, appID string) (map[string]interface{}, error) {
	query := `
		SELECT status, health_score, response_time, consecutive_failures, last_check, error_message
		FROM app_health_status
		WHERE app_id = $1
	`

	var status string
	var healthScore, responseTime, consecutiveFailures sql.NullInt64
	var lastCheck sql.NullTime
	var errorMessage sql.NullString

	err := r.db.QueryRowContext(ctx, query, appID).Scan(
		&status, &healthScore, &responseTime, &consecutiveFailures, &lastCheck, &errorMessage,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no health status found for app: %s", appID)
		}
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}

	result := map[string]interface{}{
		"status":               status,
		"health_score":         healthScore.Int64,
		"response_time":        responseTime.Int64,
		"consecutive_failures": consecutiveFailures.Int64,
		"last_check":           lastCheck.Time,
		"error_message":        errorMessage.String,
	}

	return result, nil
}

// RecordHealthAlert records a health alert for an application
func (r *PostgresRepository) RecordHealthAlert(ctx context.Context, appID string, alertType string, severity string, message string) error {
	query := `
		INSERT INTO app_health_alerts (app_id, alert_type, severity, message)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.ExecContext(ctx, query, appID, alertType, severity, message)
	if err != nil {
		return fmt.Errorf("failed to record health alert: %w", err)
	}

	return nil
}

// GetUnresolvedAlerts retrieves all unresolved health alerts
func (r *PostgresRepository) GetUnresolvedAlerts(ctx context.Context) ([]map[string]interface{}, error) {
	query := `
		SELECT id, app_id, app_name, alert_type, severity, message, created_at
		FROM app_health_alerts
		WHERE resolved = FALSE
		ORDER BY severity DESC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unresolved alerts: %w", err)
	}
	defer rows.Close()

	var alerts []map[string]interface{}
	for rows.Next() {
		var id, appID, appName, alertType, severity, message string
		var createdAt time.Time

		err := rows.Scan(&id, &appID, &appName, &alertType, &severity, &message, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert row: %w", err)
		}

		alert := map[string]interface{}{
			"id":         id,
			"app_id":     appID,
			"app_name":   appName,
			"alert_type": alertType,
			"severity":   severity,
			"message":    message,
			"created_at": createdAt,
		}
		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}
