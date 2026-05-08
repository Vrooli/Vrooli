// Package persistence provides database operations for health check results.
// SQLite is the single runtime backend for this scenario.
package persistence

import (
	"context"
	"database/sql"
	"time"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/hostinventory"
	"vrooli-autoheal/internal/incidents"
)

// Store handles database operations for health check data.
type Store struct {
	db *sql.DB
}

// NewStore creates a new SQLite-backed persistence store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Ping checks database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// SaveResult persists a health check result to the database.
func (s *Store) SaveResult(ctx context.Context, result checks.Result) error {
	return s.saveResultSQLite(ctx, result)
}

// GetLatestResultPerCheck retrieves the most recent result for each check.
func (s *Store) GetLatestResultPerCheck(ctx context.Context) ([]checks.Result, error) {
	return s.getLatestResultPerCheckSQLite(ctx)
}

// GetRecentResults retrieves recent health check results.
func (s *Store) GetRecentResults(ctx context.Context, checkID string, limit int) ([]checks.Result, error) {
	return s.getRecentResultsSQLite(ctx, checkID, limit)
}

// CleanupOldResults removes health check results older than the retention period.
func (s *Store) CleanupOldResults(ctx context.Context, retentionHours int) (int64, error) {
	return s.cleanupOldResultsSQLite(ctx, retentionHours)
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// TimelineEvent represents a single event in the timeline.
type TimelineEvent struct {
	CheckID   string                 `json:"checkId"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// GetTimelineEvents retrieves recent events across all checks, ordered by time.
func (s *Store) GetTimelineEvents(ctx context.Context, limit int) ([]TimelineEvent, error) {
	return s.getTimelineEventsSQLite(ctx, limit)
}

// UptimeStats represents uptime statistics over a time window.
type UptimeStats struct {
	TotalEvents      int     `json:"totalEvents"`
	OkEvents         int     `json:"okEvents"`
	WarningEvents    int     `json:"warningEvents"`
	CriticalEvents   int     `json:"criticalEvents"`
	UptimePercentage float64 `json:"uptimePercentage"`
	WindowHours      int     `json:"windowHours"`
}

// GetUptimeStats calculates uptime statistics over the given time window.
func (s *Store) GetUptimeStats(ctx context.Context, windowHours int) (*UptimeStats, error) {
	return s.getUptimeStatsSQLite(ctx, windowHours)
}

// UptimeHistoryBucket represents a time bucket with aggregated health status counts.
type UptimeHistoryBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Total     int       `json:"total"`
	Ok        int       `json:"ok"`
	Warning   int       `json:"warning"`
	Critical  int       `json:"critical"`
}

// UptimeHistory represents the full history response.
type UptimeHistory struct {
	Buckets     []UptimeHistoryBucket `json:"buckets"`
	Overall     UptimeStats           `json:"overall"`
	WindowHours int                   `json:"windowHours"`
	BucketCount int                   `json:"bucketCount"`
}

// GetUptimeHistory returns time-bucketed uptime data for charting.
func (s *Store) GetUptimeHistory(ctx context.Context, windowHours, bucketCount int) (*UptimeHistory, error) {
	return s.getUptimeHistorySQLite(ctx, windowHours, bucketCount)
}

// CheckTrend represents per-check trend data.
type CheckTrend struct {
	CheckID        string   `json:"checkId"`
	Total          int      `json:"total"`
	Ok             int      `json:"ok"`
	Warning        int      `json:"warning"`
	Critical       int      `json:"critical"`
	UptimePercent  float64  `json:"uptimePercent"`
	CurrentStatus  string   `json:"currentStatus"`
	RecentStatuses []string `json:"recentStatuses"`
	LastChecked    string   `json:"lastChecked"`
}

// CheckTrendsResponse contains all check trends.
type CheckTrendsResponse struct {
	Trends      []CheckTrend `json:"trends"`
	WindowHours int          `json:"windowHours"`
	TotalChecks int          `json:"totalChecks"`
}

// GetCheckTrends returns per-check trend data aggregated over the time window.
func (s *Store) GetCheckTrends(ctx context.Context, windowHours int) (*CheckTrendsResponse, error) {
	return s.getCheckTrendsSQLite(ctx, windowHours)
}

// Transition represents a status transition event.
type Transition struct {
	Timestamp  string `json:"timestamp"`
	CheckID    string `json:"checkId"`
	FromStatus string `json:"fromStatus"`
	ToStatus   string `json:"toStatus"`
	Message    string `json:"message"`
}

// TransitionsResponse contains status transitions.
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
	WindowHours int          `json:"windowHours"`
	Total       int          `json:"total"`
}

// GetTransitions returns status transition events over the time window.
func (s *Store) GetTransitions(ctx context.Context, windowHours, limit int) (*TransitionsResponse, error) {
	return s.getTransitionsSQLite(ctx, windowHours, limit)
}

func (s *Store) SaveHostInventorySnapshot(ctx context.Context, inv hostinventory.HostInventory) (*hostinventory.SnapshotRecord, []hostinventory.Change, error) {
	return s.saveHostInventorySnapshotSQLite(ctx, inv)
}

func (s *Store) GetLatestHostInventorySnapshot(ctx context.Context) (*hostinventory.SnapshotRecord, error) {
	return s.getLatestHostInventorySnapshotSQLite(ctx)
}

func (s *Store) GetRecentHostInventoryChanges(ctx context.Context, limit int) ([]hostinventory.Change, error) {
	return s.getRecentHostInventoryChangesSQLite(ctx, limit)
}

func (s *Store) UpsertIncident(ctx context.Context, input incidents.UpsertInput) (*incidents.Incident, error) {
	return s.upsertIncidentSQLite(ctx, input)
}

func (s *Store) ListIncidents(ctx context.Context, filters incidents.ListFilters) (*incidents.ListResponse, error) {
	return s.listIncidentsSQLite(ctx, filters)
}

func (s *Store) GetIncident(ctx context.Context, id string) (*incidents.Incident, error) {
	return s.getIncidentSQLite(ctx, id)
}

func (s *Store) ListIncidentObservations(ctx context.Context, incidentID string, limit int) ([]incidents.Observation, error) {
	return s.listIncidentObservationsSQLite(ctx, incidentID, limit)
}

func (s *Store) UpdateIncidentStatus(ctx context.Context, incidentID string, status incidents.Status, note string) (*incidents.Incident, error) {
	return s.updateIncidentStatusSQLite(ctx, incidentID, status, note)
}

// ActionLog represents a logged recovery action execution.
type ActionLog struct {
	ID         int64  `json:"id"`
	CheckID    string `json:"checkId"`
	ActionID   string `json:"actionId"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"durationMs"`
	Timestamp  string `json:"timestamp"`
}

// ActionLogsResponse contains action history.
type ActionLogsResponse struct {
	Logs  []ActionLog `json:"logs"`
	Total int         `json:"total"`
}

// SaveActionLog persists a recovery action execution to the database.
func (s *Store) SaveActionLog(ctx context.Context, checkID, actionID string, success bool, message, output, errMsg string, durationMs int64) error {
	return s.saveActionLogSQLite(ctx, checkID, actionID, success, message, output, errMsg, durationMs)
}

// GetActionLogs retrieves recent action logs.
func (s *Store) GetActionLogs(ctx context.Context, limit int) (*ActionLogsResponse, error) {
	return s.getActionLogsSQLite(ctx, limit)
}

// GetActionLogsForCheck retrieves action logs for a specific check.
func (s *Store) GetActionLogsForCheck(ctx context.Context, checkID string, limit int) (*ActionLogsResponse, error) {
	return s.getActionLogsForCheckSQLite(ctx, checkID, limit)
}

// SaveHealTracker persists a heal tracker state to the database.
func (s *Store) SaveHealTracker(ctx context.Context, checkID string, tracker *checks.HealTracker) error {
	return s.saveHealTrackerSQLite(ctx, checkID, tracker)
}

// GetAllHealTrackers retrieves all heal tracker states from the database.
func (s *Store) GetAllHealTrackers(ctx context.Context) (map[string]*checks.HealTracker, error) {
	return s.getAllHealTrackersSQLite(ctx)
}

// DeleteHealTracker removes a heal tracker from the database.
func (s *Store) DeleteHealTracker(ctx context.Context, checkID string) error {
	return s.deleteHealTrackerSQLite(ctx, checkID)
}
