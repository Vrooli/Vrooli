// Package domain defines the core domain entities for agent-manager.
//
// This file defines Investigation - a self-investigation feature where
// agent-manager can analyze its own runs to diagnose errors, identify
// patterns, and recommend improvements.

package domain

import (
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Investigation - Self-analysis of agent runs
// -----------------------------------------------------------------------------

// Investigation represents an analysis session where agent-manager investigates
// its own runs to diagnose errors, analyze efficiency, and identify patterns.
type Investigation struct {
	ID     uuid.UUID           `json:"id" db:"id"`
	RunIDs []uuid.UUID         `json:"runIds" db:"run_ids"` // Runs being investigated
	Status InvestigationStatus `json:"status" db:"status"`

	// Configuration
	AnalysisType   AnalysisType   `json:"analysisType" db:"analysis_type"`
	ReportSections ReportSections `json:"reportSections" db:"report_sections"`
	CustomContext  string         `json:"customContext,omitempty" db:"custom_context"`

	// Progress tracking
	Progress int `json:"progress" db:"progress"` // 0-100

	// Agent execution
	AgentRunID *uuid.UUID `json:"agentRunId,omitempty" db:"agent_run_id"` // Run ID of investigator agent

	// Results
	Findings     *InvestigationReport `json:"findings,omitempty" db:"findings"`
	Metrics      *MetricsData         `json:"metrics,omitempty" db:"metrics"`
	ErrorMessage string               `json:"errorMessage,omitempty" db:"error_message"`

	// Fix chain linkage
	SourceInvestigationID *uuid.UUID `json:"sourceInvestigationId,omitempty" db:"source_investigation_id"`

	// Timestamps
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	StartedAt   *time.Time `json:"startedAt,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completedAt,omitempty" db:"completed_at"`
}

// IsTerminal returns whether the investigation is in a terminal state.
func (i *Investigation) IsTerminal() bool {
	switch i.Status {
	case InvestigationStatusCompleted, InvestigationStatusFailed, InvestigationStatusCancelled:
		return true
	default:
		return false
	}
}

// IsActive returns whether the investigation is currently running.
func (i *Investigation) IsActive() bool {
	return i.Status == InvestigationStatusPending || i.Status == InvestigationStatusRunning
}

// InvestigationStatus represents the current state of an investigation.
type InvestigationStatus string

const (
	InvestigationStatusPending   InvestigationStatus = "pending"
	InvestigationStatusRunning   InvestigationStatus = "running"
	InvestigationStatusCompleted InvestigationStatus = "completed"
	InvestigationStatusFailed    InvestigationStatus = "failed"
	InvestigationStatusCancelled InvestigationStatus = "cancelled"
)

// IsValid checks if the status is a valid investigation status.
func (s InvestigationStatus) IsValid() bool {
	switch s {
	case InvestigationStatusPending, InvestigationStatusRunning,
		InvestigationStatusCompleted, InvestigationStatusFailed, InvestigationStatusCancelled:
		return true
	default:
		return false
	}
}

// -----------------------------------------------------------------------------
// Investigation Configuration
// -----------------------------------------------------------------------------

// AnalysisType specifies which types of analysis to perform.
// Multiple can be selected simultaneously.
type AnalysisType struct {
	ErrorDiagnosis     bool `json:"errorDiagnosis"`
	EfficiencyAnalysis bool `json:"efficiencyAnalysis"`
	ToolUsagePatterns  bool `json:"toolUsagePatterns"`
}

// HasAny returns true if at least one analysis type is selected.
func (a AnalysisType) HasAny() bool {
	return a.ErrorDiagnosis || a.EfficiencyAnalysis || a.ToolUsagePatterns
}

// ReportSections specifies which sections to include in the report.
type ReportSections struct {
	RootCauseEvidence bool `json:"rootCauseEvidence"`
	Recommendations   bool `json:"recommendations"`
	MetricsSummary    bool `json:"metricsSummary"`
}

// HasAny returns true if at least one report section is selected.
func (r ReportSections) HasAny() bool {
	return r.RootCauseEvidence || r.Recommendations || r.MetricsSummary
}

// -----------------------------------------------------------------------------
// Investigation Report - Structured findings
// -----------------------------------------------------------------------------

// InvestigationReport contains the structured output from an investigation.
type InvestigationReport struct {
	Summary         string             `json:"summary"`
	RootCause       *RootCauseAnalysis `json:"rootCause,omitempty"`
	Recommendations []Recommendation   `json:"recommendations,omitempty"`
}

// RootCauseAnalysis contains the identified root cause and supporting evidence.
type RootCauseAnalysis struct {
	Description string     `json:"description"`
	Evidence    []Evidence `json:"evidence"`
	Confidence  string     `json:"confidence"` // "high", "medium", "low"
}

// Evidence represents a piece of evidence supporting the root cause analysis.
type Evidence struct {
	RunID       string `json:"runId"`
	EventSeq    int64  `json:"eventSeq,omitempty"`
	Description string `json:"description"`
	Snippet     string `json:"snippet,omitempty"`
}

// Recommendation represents a suggested action to address findings.
type Recommendation struct {
	ID          string `json:"id"`
	Priority    string `json:"priority"`   // "critical", "high", "medium", "low"
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionType  string `json:"actionType"` // "prompt_change", "profile_config", "code_fix"
}

// ValidPriorities returns the valid priority values.
func ValidPriorities() []string {
	return []string{"critical", "high", "medium", "low"}
}

// ValidActionTypes returns the valid action type values.
func ValidActionTypes() []string {
	return []string{"prompt_change", "profile_config", "code_fix"}
}

// -----------------------------------------------------------------------------
// Metrics - Computed values from investigation
// -----------------------------------------------------------------------------

// MetricsData contains metrics computed by the investigation agent.
// These are recorded via the record_metric tool to ensure accuracy.
type MetricsData struct {
	TotalRuns          int                `json:"totalRuns"`
	SuccessRate        float64            `json:"successRate"`
	AvgDurationSeconds float64            `json:"avgDurationSeconds"`
	TotalTokensUsed    int64              `json:"totalTokensUsed"`
	TotalCost          float64            `json:"totalCost"`
	ToolUsageBreakdown map[string]int     `json:"toolUsageBreakdown,omitempty"`
	ErrorTypeBreakdown map[string]int     `json:"errorTypeBreakdown,omitempty"`
	CustomMetrics      map[string]float64 `json:"customMetrics,omitempty"`
}

// RecordedMetric represents a single metric recorded via the record_metric tool.
type RecordedMetric struct {
	Name     string  `json:"name"`
	Value    float64 `json:"value"`
	Category string  `json:"category,omitempty"` // "rate", "count", "duration", "cost", "custom"
}

// ValidMetricCategories returns the valid metric category values.
func ValidMetricCategories() []string {
	return []string{"rate", "count", "duration", "cost", "custom"}
}

// -----------------------------------------------------------------------------
// Request/Response Types
// -----------------------------------------------------------------------------

// CreateInvestigationRequest contains the parameters for triggering an investigation.
type CreateInvestigationRequest struct {
	RunIDs         []uuid.UUID    `json:"runIds"`
	AnalysisType   AnalysisType   `json:"analysisType"`
	ReportSections ReportSections `json:"reportSections"`
	CustomContext  string         `json:"customContext,omitempty"`
}

// Validate checks if the request is valid.
func (r *CreateInvestigationRequest) Validate() error {
	if len(r.RunIDs) == 0 {
		return NewValidationError("runIds", "at least one run ID is required")
	}
	if !r.AnalysisType.HasAny() {
		return NewValidationError("analysisType", "at least one analysis type must be selected")
	}
	if !r.ReportSections.HasAny() {
		return NewValidationError("reportSections", "at least one report section must be selected")
	}
	return nil
}

// ApplyFixesRequest contains the parameters for applying investigation recommendations.
type ApplyFixesRequest struct {
	RecommendationIDs []string `json:"recommendationIds"`
	Note              string   `json:"note,omitempty"`
}

// Validate checks if the request is valid.
func (r *ApplyFixesRequest) Validate() error {
	if len(r.RecommendationIDs) == 0 {
		return NewValidationError("recommendationIds", "at least one recommendation ID is required")
	}
	return nil
}

// InvestigationSummary is a lightweight view of an investigation for list responses.
type InvestigationSummary struct {
	ID                    uuid.UUID           `json:"id"`
	RunIDs                []uuid.UUID         `json:"runIds"`
	Status                InvestigationStatus `json:"status"`
	Progress              int                 `json:"progress"`
	HasFindings           bool                `json:"hasFindings"`
	ErrorMessage          string              `json:"errorMessage,omitempty"`
	SourceInvestigationID *uuid.UUID          `json:"sourceInvestigationId,omitempty"`
	CreatedAt             time.Time           `json:"createdAt"`
	CompletedAt           *time.Time          `json:"completedAt,omitempty"`
}

// ToSummary converts an Investigation to its summary form.
func (i *Investigation) ToSummary() InvestigationSummary {
	return InvestigationSummary{
		ID:                    i.ID,
		RunIDs:                i.RunIDs,
		Status:                i.Status,
		Progress:              i.Progress,
		HasFindings:           i.Findings != nil,
		ErrorMessage:          i.ErrorMessage,
		SourceInvestigationID: i.SourceInvestigationID,
		CreatedAt:             i.CreatedAt,
		CompletedAt:           i.CompletedAt,
	}
}
