package models

import (
	"time"
)

// Threshold defines limits for anomaly detection
type Threshold struct {
	MetricName        string  `json:"metric_name"`
	Min               float64 `json:"min"`
	Max               float64 `json:"max"`
	WarningThreshold  float64 `json:"warning_threshold"`
	CriticalThreshold float64 `json:"critical_threshold"`
	StdDevs           float64 `json:"std_devs"`
	Duration          int     `json:"duration_seconds"`
	CheckInterval     int     `json:"check_interval"`
	Enabled           bool    `json:"enabled"`
}

// ThresholdViolation represents a threshold violation event
type ThresholdViolation struct {
	MetricName     string    `json:"metric_name"`
	CurrentValue   float64   `json:"current_value"`
	ThresholdValue float64   `json:"threshold_value"`
	Severity       string    `json:"severity"`
	ViolationType  string    `json:"violation_type"` // warning, critical
	Timestamp      time.Time `json:"timestamp"`
	Duration       string    `json:"duration"`
	PreviousValue  float64   `json:"previous_value"`
	Trend          string    `json:"trend"` // increasing, decreasing, stable
}

// ThresholdMonitorRequest represents a request to check thresholds
type ThresholdMonitorRequest struct {
	ForceCheck bool `json:"force_check"`
}

// ThresholdMonitorResponse represents the response from threshold monitoring
type ThresholdMonitorResponse struct {
	CheckID         string               `json:"check_id"`
	Status          string               `json:"status"`
	ViolationsFound int                  `json:"violations_found"`
	Violations      []ThresholdViolation `json:"violations"`
	SystemHealth    string               `json:"system_health"` // healthy, warning, critical
	CheckedAt       time.Time            `json:"checked_at"`
	NextCheck       time.Time            `json:"next_check"`
	Summary         string               `json:"summary"`
}

// Alert represents a system alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	MetricName  string                 `json:"metric_name"`
	MetricValue float64                `json:"metric_value"`
	Threshold   *Threshold             `json:"threshold,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	AckedAt     *time.Time             `json:"acked_at,omitempty"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	AckedBy     string                 `json:"acked_by,omitempty"`
}