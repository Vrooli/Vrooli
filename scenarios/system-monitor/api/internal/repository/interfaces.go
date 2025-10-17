package repository

import (
	"context"
	"time"

	"system-monitor-api/internal/models"
)

// Repository aggregates all repository interfaces
type Repository interface {
	MetricsRepository
	InvestigationRepository
	ReportRepository
	ThresholdRepository
	AlertRepository
}

// MetricsRepository handles metrics data persistence
type MetricsRepository interface {
	// SaveMetrics stores metrics data
	SaveMetrics(ctx context.Context, collectorName string, metrics map[string]interface{}) error
	
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
	Status     string
	AnomalyID  string
	TimeRange  TimeRange
	Limit      int
	Offset     int
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
	MetricName  string
	TimeRange   TimeRange
	Interval    string // 1m, 5m, 1h, etc.
	Function    string // avg, sum, max, min, count
	GroupBy     []string
}