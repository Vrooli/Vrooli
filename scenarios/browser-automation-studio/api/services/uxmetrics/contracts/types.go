// Package contracts defines the domain types for UX metrics.
// These types are the atomic building blocks of the UX metrics subsystem,
// kept in a separate package to enforce clean dependencies.
package contracts

import (
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

// ActionType enumerates the types of user interactions tracked.
type ActionType string

const (
	ActionClick      ActionType = "click"
	ActionType_      ActionType = "type"
	ActionScroll     ActionType = "scroll"
	ActionNavigation ActionType = "navigation"
	ActionWait       ActionType = "wait"
	ActionHover      ActionType = "hover"
	ActionDrag       ActionType = "drag"
)

// FrictionType enumerates the types of friction signals detected.
type FrictionType string

const (
	FrictionExcessiveTime   FrictionType = "excessive_time"
	FrictionZigZagPath      FrictionType = "zigzag_path"
	FrictionMultipleRetries FrictionType = "multiple_retries"
	FrictionRapidClicks     FrictionType = "rapid_clicks"
	FrictionLongHesitation  FrictionType = "long_hesitation"
	FrictionBackNavigation  FrictionType = "back_navigation"
	FrictionElementMiss     FrictionType = "element_miss"
)

// Severity indicates the severity level of a friction signal.
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// Point is an alias for the canonical automation contracts Point type.
// This avoids type duplication while allowing uxmetrics to use Point in its API.
type Point = autocontracts.Point

// TimedPoint represents a 2D coordinate with a timestamp.
type TimedPoint struct {
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	Timestamp time.Time `json:"timestamp"`
}

// InteractionTrace captures a single user interaction event.
// These are the atomic building blocks of UX metrics.
type InteractionTrace struct {
	ID          uuid.UUID      `json:"id"`
	ExecutionID uuid.UUID      `json:"execution_id"`
	StepIndex   int            `json:"step_index"`
	ActionType  ActionType     `json:"action_type"`
	ElementID   string         `json:"element_id,omitempty"`
	Selector    string         `json:"selector,omitempty"`
	Position    *Point         `json:"position,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
	DurationMs  int64          `json:"duration_ms,omitempty"`
	Success     bool           `json:"success"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// CursorPath captures the full trajectory of cursor movement for a step.
type CursorPath struct {
	StepIndex       int          `json:"step_index"`
	Points          []TimedPoint `json:"points"`
	TotalDistancePx float64      `json:"total_distance_px"`
	DurationMs      int64        `json:"duration_ms"`
	DirectDistance  float64      `json:"direct_distance_px"` // Straight-line start->end
	Directness      float64      `json:"directness"`         // 0-1, 1 = perfectly straight
	ZigZagScore     float64      `json:"zigzag_score"`       // Higher = more erratic
	AverageSpeed    float64      `json:"average_speed_px_ms"`
	MaxSpeed        float64      `json:"max_speed_px_ms"`
	Hesitations     int          `json:"hesitation_count"` // Pauses > 200ms
}

// FrictionSignal indicates a potential usability issue detected in the flow.
type FrictionSignal struct {
	Type        FrictionType   `json:"type"`
	StepIndex   int            `json:"step_index"`
	Severity    Severity       `json:"severity"`
	Score       float64        `json:"score"` // Normalized 0-100
	Description string         `json:"description"`
	Evidence    map[string]any `json:"evidence,omitempty"`
}

// StepMetrics aggregates UX metrics for a single step.
type StepMetrics struct {
	StepIndex        int              `json:"step_index"`
	NodeID           string           `json:"node_id"`
	StepType         string           `json:"step_type"`
	TimeToActionMs   int64            `json:"time_to_action_ms"`
	ActionDurationMs int64            `json:"action_duration_ms"`
	TotalDurationMs  int64            `json:"total_duration_ms"`
	CursorPath       *CursorPath      `json:"cursor_path,omitempty"`
	RetryCount       int              `json:"retry_count"`
	FrictionSignals  []FrictionSignal `json:"friction_signals,omitempty"`
	FrictionScore    float64          `json:"friction_score"` // 0-100, lower is better
}

// MetricsSummary provides human-readable insights.
type MetricsSummary struct {
	HighFrictionSteps  []int    `json:"high_friction_steps,omitempty"`
	SlowestSteps       []int    `json:"slowest_steps,omitempty"`
	TopFrictionTypes   []string `json:"top_friction_types,omitempty"`
	RecommendedActions []string `json:"recommended_actions,omitempty"`
}

// ExecutionMetrics aggregates UX metrics for an entire execution.
type ExecutionMetrics struct {
	ExecutionID       uuid.UUID        `json:"execution_id"`
	WorkflowID        uuid.UUID        `json:"workflow_id"`
	ComputedAt        time.Time        `json:"computed_at"`
	TotalDurationMs   int64            `json:"total_duration_ms"`
	StepCount         int              `json:"step_count"`
	SuccessfulSteps   int              `json:"successful_steps"`
	FailedSteps       int              `json:"failed_steps"`
	TotalRetries      int              `json:"total_retries"`
	AvgStepDurationMs float64          `json:"avg_step_duration_ms"`
	TotalCursorDist   float64          `json:"total_cursor_distance_px"`
	OverallFriction   float64          `json:"overall_friction_score"` // 0-100
	FrictionSignals   []FrictionSignal `json:"friction_signals"`
	StepMetrics       []StepMetrics    `json:"step_metrics"`
	Summary           *MetricsSummary  `json:"summary,omitempty"`
}

// WorkflowMetricsAggregate provides workflow-level trends across executions.
type WorkflowMetricsAggregate struct {
	WorkflowID           uuid.UUID   `json:"workflow_id"`
	ExecutionCount       int         `json:"execution_count"`
	AvgFrictionScore     float64     `json:"avg_friction_score"`
	AvgDurationMs        float64     `json:"avg_duration_ms"`
	TrendDirection       string      `json:"trend_direction"` // improving, degrading, stable
	HighFrictionStepFreq map[int]int `json:"high_friction_step_frequency"`
}
