// Package persistence provides database operations for health check results.
// SQLite is the single runtime backend for this scenario.
package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/systemevents"
)

// Store handles database operations for health check data.
type Store struct {
	db *sql.DB
}

// NewStore creates a new SQLite-backed persistence store.
func NewStore(db *sql.DB) *Store {
	store := &Store{db: db}
	_ = store.ensureIncidentContractColumns(context.Background())
	return store
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

// RetentionResult reports a bounded operational-history prune. Counts are
// deliberately per data class so callers can surface retention work rather
// than silently shrinking forensic history.
type RetentionResult struct {
	HealthResults int64 `json:"health_results"`
	ActionLogs    int64 `json:"action_logs"`
	Actions       int64 `json:"actions"`
	SystemEvents  int64 `json:"system_events"`
}

// RetentionStatus is a read-only operational summary used by status surfaces.
// It deliberately reports metadata rather than scanning retained evidence.
type RetentionStatus struct {
	DatabaseBytes int64      `json:"databaseBytes"`
	OldestAt      *time.Time `json:"oldestAt,omitempty"`
	NewestAt      *time.Time `json:"newestAt,omitempty"`
}

func (s *Store) OperationalRetentionStatus(ctx context.Context) (RetentionStatus, error) {
	return s.operationalRetentionStatusSQLite(ctx)
}

// PruneOperationalHistory removes at most batchSize old rows from each
// high-volume operational table. Incident records are intentionally retained:
// they are the compact forensic rollup that survives raw-history pruning.
func (s *Store) PruneOperationalHistory(ctx context.Context, before time.Time, batchSize int) (RetentionResult, error) {
	return s.pruneOperationalHistorySQLite(ctx, before, batchSize)
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

func (s *Store) UpsertSystemEvents(ctx context.Context, events []systemevents.Event) (int, int, error) {
	return s.upsertSystemEventsSQLite(ctx, events)
}

func (s *Store) UpsertSystemEventSource(ctx context.Context, source systemevents.SourceStatus) error {
	return s.upsertSystemEventSourceSQLite(ctx, source)
}

func (s *Store) ListSystemEvents(ctx context.Context, filters systemevents.Filters) (*systemevents.Response, error) {
	return s.listSystemEventsSQLite(ctx, filters)
}

func (s *Store) GetSystemEventSources(ctx context.Context) ([]systemevents.SourceStatus, error) {
	return s.getSystemEventSourcesSQLite(ctx)
}

// GetJournalCursor returns the persisted incremental-ingest cursor for the
// given logical source key (empty state if none recorded).
func (s *Store) GetJournalCursor(ctx context.Context, sourceKey string) (systemevents.CursorState, error) {
	return s.getJournalCursorSQLite(ctx, sourceKey)
}

// SetJournalCursor persists the incremental-ingest cursor for the source key.
func (s *Store) SetJournalCursor(ctx context.Context, sourceKey string, state systemevents.CursorState) error {
	return s.setJournalCursorSQLite(ctx, sourceKey, state)
}

// IsBootScanned reports whether the (sourceKey, bootID) pair has already been
// scanned to completion.
func (s *Store) IsBootScanned(ctx context.Context, sourceKey, bootID string) (bool, error) {
	return s.isBootScannedSQLite(ctx, sourceKey, bootID)
}

// MarkBootScanned records that (sourceKey, bootID) has been scanned.
func (s *Store) MarkBootScanned(ctx context.Context, sourceKey, bootID string) error {
	return s.markBootScannedSQLite(ctx, sourceKey, bootID)
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

func (s *Store) RecordIncidentRemediationArtifact(ctx context.Context, incidentID string, artifact incidents.RemediationArtifact) (*incidents.Incident, error) {
	return s.recordIncidentRemediationArtifactSQLite(ctx, incidentID, artifact)
}

func (s *Store) RecordIncidentRemediationOutcome(ctx context.Context, incidentID string, outcome incidents.Outcome) (*incidents.Incident, error) {
	return s.recordIncidentRemediationOutcomeSQLite(ctx, incidentID, outcome)
}

// ActionLog represents a logged recovery action execution.
type ActionLog struct {
	ID         int64  `json:"id"`
	CheckID    string `json:"checkId"`
	ActionID   string `json:"actionId"`
	Success    bool   `json:"success"`
	TimedOut   bool   `json:"timedOut,omitempty"`
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
func (s *Store) SaveActionLog(ctx context.Context, checkID, actionID string, success, timedOut bool, message, output, errMsg string, durationMs int64) error {
	return s.saveActionLogSQLite(ctx, checkID, actionID, success, timedOut, message, output, errMsg, durationMs)
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
