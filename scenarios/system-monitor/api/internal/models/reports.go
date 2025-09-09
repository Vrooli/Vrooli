package models

import (
	"time"
)

// Report represents a system report
type Report struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	GeneratedAt time.Time              `json:"generated_at"`
	TimeRange   ReportTimeRange        `json:"time_range"`
	Data        map[string]interface{} `json:"data"`
	Format      string                 `json:"format"`
}

// ReportRequest represents a request to generate a report
type ReportRequest struct {
	Type      string                 `json:"type"`
	TimeRange string                 `json:"time_range"`
	StartTime *time.Time             `json:"start_time,omitempty"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Metrics   []string               `json:"metrics,omitempty"`
	Format    string                 `json:"format"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// DetailedSystemReport contains comprehensive system information
type DetailedSystemReport struct {
	ReportID         string              `json:"report_id"`
	ReportType       string              `json:"report_type"`
	GeneratedAt      time.Time           `json:"generated_at"`
	TimeRange        ReportTimeRange     `json:"time_range"`
	ExecutiveSummary ExecutiveSummary    `json:"executive_summary"`
	MetricsSummary   MetricsSummary      `json:"metrics_summary"`
	Anomalies        []AnomalyReport     `json:"anomalies"`
	Performance      PerformanceReport   `json:"performance"`
	Availability     AvailabilityReport  `json:"availability"`
	Trends           TrendAnalysis       `json:"trends"`
	Recommendations  []string            `json:"recommendations"`
	NextSteps        []string            `json:"next_steps"`
}

// ReportTimeRange defines the time range for a report
type ReportTimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  string    `json:"duration"`
}

// ExecutiveSummary provides high-level report summary
type ExecutiveSummary struct {
	OverallHealth    string             `json:"overall_health"`
	KeyMetrics       []KeyMetricSummary `json:"key_metrics"`
	CriticalIssues   int                `json:"critical_issues"`
	ResolvedIssues   int                `json:"resolved_issues"`
	Uptime           float64            `json:"uptime"`
	PerformanceScore int                `json:"performance_score"`
}

// KeyMetricSummary summarizes a key metric
type KeyMetricSummary struct {
	Name    string  `json:"name"`
	Current float64 `json:"current"`
	Average float64 `json:"average"`
	Peak    float64 `json:"peak"`
	Status  string  `json:"status"`
}

// MetricsSummary summarizes all system metrics
type MetricsSummary struct {
	CPU     MetricSummary `json:"cpu"`
	Memory  MetricSummary `json:"memory"`
	Network MetricSummary `json:"network"`
	Storage MetricSummary `json:"storage"`
}

// MetricSummary provides summary statistics for a metric
type MetricSummary struct {
	Average    float64 `json:"average"`
	Peak       float64 `json:"peak"`
	Minimum    float64 `json:"minimum"`
	Current    float64 `json:"current"`
	Violations int     `json:"violations"`
	Trend      string  `json:"trend"`
}

// AnomalyReport summarizes anomalies for reporting
type AnomalyReport struct {
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Resolved  int       `json:"resolved"`
	Status    string    `json:"status"`
}

// PerformanceReport contains performance metrics
type PerformanceReport struct {
	ResponseTime MetricSummary `json:"response_time"`
	Throughput   MetricSummary `json:"throughput"`
	ErrorRate    MetricSummary `json:"error_rate"`
	SLA          SLAReport     `json:"sla"`
}

// AvailabilityReport contains availability metrics
type AvailabilityReport struct {
	Uptime    float64           `json:"uptime"`
	Downtime  string            `json:"downtime"`
	Incidents []IncidentSummary `json:"incidents"`
	MTTR      string            `json:"mttr"` // Mean Time To Recovery
	MTBF      string            `json:"mtbf"` // Mean Time Between Failures
}

// SLAReport tracks SLA compliance
type SLAReport struct {
	Target   float64 `json:"target"`
	Actual   float64 `json:"actual"`
	Status   string  `json:"status"`
	Breaches int     `json:"breaches"`
}

// IncidentSummary summarizes an incident
type IncidentSummary struct {
	ID          string     `json:"id"`
	Severity    string     `json:"severity"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Duration    string     `json:"duration"`
	Status      string     `json:"status"`
	Description string     `json:"description"`
}

// TrendAnalysis analyzes metric trends
type TrendAnalysis struct {
	CPU     TrendData `json:"cpu"`
	Memory  TrendData `json:"memory"`
	Network TrendData `json:"network"`
	Storage TrendData `json:"storage"`
}

// TrendData contains trend analysis for a metric
type TrendData struct {
	Direction  string  `json:"direction"` // increasing, decreasing, stable
	Slope      float64 `json:"slope"`
	Confidence float64 `json:"confidence"`
	Forecast   float64 `json:"forecast"` // predicted value for next period
	Alert      bool    `json:"alert"`    // if trend indicates potential issues
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Service   string                 `json:"service"`
	Timestamp int64                  `json:"timestamp"`
	Uptime    float64                `json:"uptime"`
	Checks    map[string]interface{} `json:"checks"`
	Version   string                 `json:"version"`
}

// LogEntry represents a log entry for debugging
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}