package repository

// DOC: docs/internal/SEAMS.md#repository-interface

import (
	"context"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// Repository aggregates all repository interfaces
type Repository interface {
	MetricsRepository
	ProcessSampleRepository
	InvestigationRepository
	ReportRepository
	ThresholdRepository
	AlertRepository
	MaintenanceRepository
}

// ProcessSampleRepository persists and queries per-process samples used for the
// "top consumers over time, attributed to scenario" timeline. It is additive to
// the opaque metrics blob storage (MetricsRepository) and never replaces it.
type ProcessSampleRepository interface {
	// SaveProcessSamples writes one cycle's worth of per-process rows.
	SaveProcessSamples(ctx context.Context, samples []ProcessSample) error

	// QueryProcessTimeline returns ranked consumers over the window described by
	// the query. Rows come from the raw table when available and the per-owner
	// minute rollups for the older portion of the window.
	QueryProcessTimeline(ctx context.Context, q ProcessTimelineQuery) ([]ProcessTimelineEntry, error)

	// PruneProcessSamplesBefore deletes raw process rows older than cutoff.
	PruneProcessSamplesBefore(ctx context.Context, cutoff time.Time) (int64, error)

	// RollupProcessSamples downsamples raw rows in [from, to) into per-owner /
	// per-minute aggregates, then deletes the raw rows it rolled up. It returns
	// the number of raw rows consumed so the caller can log what was collapsed.
	RollupProcessSamples(ctx context.Context, from, to time.Time) (RollupResult, error)

	// PruneProcessRollupsBefore deletes per-minute rollup rows older than cutoff.
	PruneProcessRollupsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// ProcessSample is one observed process within a single sampling cycle, ready
// to persist. CPUPct is share-of-one-CPU since the prior sample (may exceed
// 100 for multi-threaded processes).
type ProcessSample struct {
	Timestamp            time.Time
	PID                  int
	PPID                 int
	Comm                 string
	Cmdline              string
	Cwd                  string
	Owner                string
	CPUPct               float64
	CPUSeconds           float64
	CPUSecondsStatus     string
	CPUSecondsReason     string
	RSSKB                int64
	SwapKB               int64
	MajorFaultsPerSecond float64
	MetricsStatus        string
	MetricsReason        string
	Threads              int
	GPUVRAMMB            float64
}

// ProcessTimelineQuery parameterizes a timeline read.
type ProcessTimelineQuery struct {
	Start time.Time
	End   time.Time
	Owner string // optional scenario filter; "" means all owners
	Top   int    // optional cap on ranked rows returned; <=0 means a default
	Rank  string // "cpu" (default), "cpu_seconds", "rss", or "gpu"
}

// ProcessTimelineEntry is one ranked consumer over the queried window,
// aggregated across the samples that fell inside it.
type ProcessTimelineEntry struct {
	Owner       string
	Comm        string
	PID         int  // representative pid; 0 when aggregated across rollups
	Aggregated  bool // true when the entry spans rollup (per-minute) rows
	CPUPct      float64
	MaxCPUPct   float64
	CPUSeconds  float64
	RSSKB       int64
	GPUVRAMMB   float64
	SampleCount int64
	FirstSeen   time.Time
	LastSeen    time.Time
}

// RollupResult reports the outcome of a downsampling pass.
type RollupResult struct {
	RawRowsConsumed int64
	RollupRows      int64
	From            time.Time
	To              time.Time
}

// MetricsRepository handles metrics data persistence
type MetricsRepository interface {
	// SaveMetricCycle persists all collector observations from one logical
	// collection cycle. CycleID and ObservedAt are caller-owned.
	SaveMetricCycle(ctx context.Context, cycleID string, observedAt time.Time, observations []MetricObservation) error
	// GetMetrics retrieves metrics with optional filtering
	GetMetrics(ctx context.Context, filter MetricsFilter) ([]*models.MetricsResponse, error)

	// GetLatestMetrics gets the most recent metrics
	GetLatestMetrics(ctx context.Context) (*models.MetricsResponse, error)

	// GetDetailedMetrics retrieves comprehensive metrics
	GetDetailedMetrics(ctx context.Context, timeRange TimeRange) (*models.DetailedMetrics, error)

	// GetHistoricalMetrics retrieves metrics for a time range
	GetHistoricalMetrics(ctx context.Context, metricName string, timeRange TimeRange) ([]MetricDataPoint, error)

	// GetAggregatedMetrics gets aggregated metrics
	GetAggregatedMetrics(ctx context.Context, aggregation AggregationQuery) (map[string]interface{}, error)

	// GetEarliestMetricTime returns the timestamp of the earliest stored metric
	GetEarliestMetricTime(ctx context.Context) (time.Time, error)
}

// MetricObservation is one collector's result inside a logical cycle.
// Values must retain an explicit status for failed and unsupported readings.
type MetricObservation struct {
	CollectorName string
	Values        map[string]interface{}
}

// InvestigationRepository handles investigation data persistence
type InvestigationRepository interface {
	// CreateInvestigation creates a new investigation
	CreateInvestigation(ctx context.Context, investigation *models.Investigation) error

	// GetInvestigation retrieves an investigation by ID
	GetInvestigation(ctx context.Context, id string) (*models.Investigation, error)

	// UpdateInvestigation updates an existing investigation
	UpdateInvestigation(ctx context.Context, investigation *models.Investigation) error

	// ListInvestigations lists investigations with filtering
	ListInvestigations(ctx context.Context, filter InvestigationFilter) ([]*models.Investigation, error)

	// GetLatestInvestigation gets the most recent investigation
	GetLatestInvestigation(ctx context.Context) (*models.Investigation, error)

	// SaveInvestigationStep adds a step to an investigation
	SaveInvestigationStep(ctx context.Context, investigationID string, step *models.InvestigationStep) error
}

// ReportRepository handles report data persistence
type ReportRepository interface {
	// CreateReport creates a new report
	CreateReport(ctx context.Context, report *models.Report) error

	// GetReport retrieves a report by ID
	GetReport(ctx context.Context, id string) (*models.Report, error)

	// ListReports lists reports with filtering
	ListReports(ctx context.Context, filter ReportFilter) ([]*models.Report, error)

	// SaveDetailedReport saves a comprehensive system report
	SaveDetailedReport(ctx context.Context, report *models.DetailedSystemReport) error

	// GetDetailedReport retrieves a detailed report
	GetDetailedReport(ctx context.Context, id string) (*models.DetailedSystemReport, error)

	// SaveEnhancedReport saves an enhanced system report
	SaveEnhancedReport(ctx context.Context, report *models.EnhancedSystemReport) error

	// GetEnhancedReport retrieves an enhanced report
	GetEnhancedReport(ctx context.Context, id string) (*models.EnhancedSystemReport, error)

	// ListEnhancedReports retrieves all enhanced reports
	ListEnhancedReports(ctx context.Context) ([]*models.EnhancedSystemReport, error)
}

// ThresholdRepository handles threshold configuration persistence
type ThresholdRepository interface {
	// GetActiveThresholds retrieves all active thresholds
	GetActiveThresholds(ctx context.Context) ([]*models.Threshold, error)

	// GetThreshold retrieves a specific threshold
	GetThreshold(ctx context.Context, metricName string) (*models.Threshold, error)

	// SaveThreshold saves or updates a threshold
	SaveThreshold(ctx context.Context, threshold *models.Threshold) error

	// DeleteThreshold removes a threshold
	DeleteThreshold(ctx context.Context, metricName string) error

	// SaveThresholdViolation records a threshold violation
	SaveThresholdViolation(ctx context.Context, violation *models.ThresholdViolation) error

	// GetThresholdViolations retrieves violations for a time range
	GetThresholdViolations(ctx context.Context, timeRange TimeRange) ([]*models.ThresholdViolation, error)
}

// AlertRepository handles alert data persistence
type AlertRepository interface {
	// CreateAlert creates a new alert
	CreateAlert(ctx context.Context, alert *models.Alert) error

	// GetAlert retrieves an alert by ID
	GetAlert(ctx context.Context, id string) (*models.Alert, error)

	// UpdateAlert updates an existing alert
	UpdateAlert(ctx context.Context, alert *models.Alert) error

	// ListAlerts lists alerts with filtering
	ListAlerts(ctx context.Context, filter AlertFilter) ([]*models.Alert, error)

	// AcknowledgeAlert marks an alert as acknowledged
	AcknowledgeAlert(ctx context.Context, id string, ackedBy string) error

	// ResolveAlert marks an alert as resolved
	ResolveAlert(ctx context.Context, id string) error

	// GetActiveAlerts retrieves all unresolved alerts
	GetActiveAlerts(ctx context.Context) ([]*models.Alert, error)
}

// AnomalyRepository handles anomaly data persistence
type AnomalyRepository interface {
	// SaveAnomaly stores an anomaly
	SaveAnomaly(ctx context.Context, anomaly *models.Anomaly) error

	// GetAnomaly retrieves an anomaly by ID
	GetAnomaly(ctx context.Context, id string) (*models.Anomaly, error)

	// ListAnomalies lists anomalies with filtering
	ListAnomalies(ctx context.Context, filter AnomalyFilter) ([]*models.Anomaly, error)

	// UpdateAnomalyStatus updates the status of an anomaly
	UpdateAnomalyStatus(ctx context.Context, id string, status string) error

	// GetAnomaliesByTimeRange retrieves anomalies for a time range
	GetAnomaliesByTimeRange(ctx context.Context, timeRange TimeRange) ([]*models.Anomaly, error)
}

// Filter types

// MetricsFilter defines filtering options for metrics queries
type MetricsFilter struct {
	CollectorName string
	MetricNames   []string
	TimeRange     TimeRange
	Limit         int
	Offset        int
}

// InvestigationFilter defines filtering options for investigation queries
type InvestigationFilter struct {
	Status    string
	AnomalyID string
	TimeRange TimeRange
	Limit     int
	Offset    int
}

// ReportFilter defines filtering options for report queries
type ReportFilter struct {
	Type      string
	TimeRange TimeRange
	Format    string
	Limit     int
	Offset    int
}

// AlertFilter defines filtering options for alert queries
type AlertFilter struct {
	Type       string
	Severity   string
	Status     string // active, acknowledged, resolved
	TimeRange  TimeRange
	MetricName string
	Limit      int
	Offset     int
}

// AnomalyFilter defines filtering options for anomaly queries
type AnomalyFilter struct {
	Type      string
	Severity  string
	Status    string
	TimeRange TimeRange
	Limit     int
	Offset    int
}

// TimeRange defines a time range for queries
type TimeRange struct {
	StartTime time.Time
	EndTime   time.Time
}

// MetricDataPoint represents a single metric data point
type MetricDataPoint struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

// AggregationQuery defines parameters for metric aggregation
type AggregationQuery struct {
	MetricName string
	TimeRange  TimeRange
	Interval   string // 1m, 5m, 1h, etc.
	Function   string // avg, sum, max, min, count
	GroupBy    []string
}

// DefaultThresholds returns the built-in threshold set used by repository
// implementations when no thresholds have been configured. Centralising the
// defaults keeps the in-memory and SQLite repositories in sync.
func DefaultThresholds() []*models.Threshold {
	return []*models.Threshold{
		{
			MetricName:        "cpu_usage",
			Min:               0,
			Max:               100,
			WarningThreshold:  80,
			CriticalThreshold: 95,
			CheckInterval:     60,
			Enabled:           true,
		},
		{
			MetricName:        "memory_usage",
			Min:               0,
			Max:               100,
			WarningThreshold:  85,
			CriticalThreshold: 95,
			CheckInterval:     60,
			Enabled:           true,
		},
	}
}
