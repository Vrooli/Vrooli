// Package errors provides structured error types for the scoring API.
// [REQ:SCS-CORE-003] Graceful degradation error categorization
// [REQ:SCS-CORE-004] Partial results with error context
package errors

import (
	"fmt"
	"time"
)

// Severity indicates how severe an error is
type Severity string

const (
	// SeverityInfo indicates informational, non-blocking errors
	SeverityInfo Severity = "info"
	// SeverityWarning indicates issues that allow degraded operation
	SeverityWarning Severity = "warning"
	// SeverityError indicates failures requiring attention
	SeverityError Severity = "error"
	// SeverityCritical indicates system-wide failures
	SeverityCritical Severity = "critical"
)

// Category indicates the type of error for routing and handling
type Category string

const (
	// CategoryCollector errors from data collection
	CategoryCollector Category = "collector"
	// CategoryDatabase errors from SQLite operations
	CategoryDatabase Category = "database"
	// CategoryConfig errors from configuration loading/saving
	CategoryConfig Category = "config"
	// CategoryValidation errors from input validation
	CategoryValidation Category = "validation"
	// CategoryNetwork errors from network operations
	CategoryNetwork Category = "network"
	// CategoryFileSystem errors from file operations
	CategoryFileSystem Category = "filesystem"
	// CategoryInternal unexpected internal errors
	CategoryInternal Category = "internal"
)

// CollectorError represents a failure in a specific collector
// [REQ:SCS-CB-001] Collector-specific failure tracking
type CollectorError struct {
	Collector    string    `json:"collector"`
	Message      string    `json:"message"`
	Severity     Severity  `json:"severity"`
	Recoverable  bool      `json:"recoverable"`
	Timestamp    time.Time `json:"timestamp"`
	PartialData  bool      `json:"partial_data,omitempty"`
	Cause        error     `json:"-"`
}

func (e *CollectorError) Error() string {
	return fmt.Sprintf("[%s] collector '%s': %s", e.Severity, e.Collector, e.Message)
}

// Unwrap returns the underlying error
func (e *CollectorError) Unwrap() error {
	return e.Cause
}

// NewCollectorError creates a new collector error
func NewCollectorError(collector, message string, cause error) *CollectorError {
	return &CollectorError{
		Collector:   collector,
		Message:     message,
		Severity:    SeverityError,
		Recoverable: true,
		Timestamp:   time.Now(),
		Cause:       cause,
	}
}

// NewCollectorWarning creates a warning-level collector error
func NewCollectorWarning(collector, message string) *CollectorError {
	return &CollectorError{
		Collector:   collector,
		Message:     message,
		Severity:    SeverityWarning,
		Recoverable: true,
		Timestamp:   time.Now(),
		PartialData: true,
	}
}

// ScoringError represents a failure during score calculation
type ScoringError struct {
	Scenario       string             `json:"scenario"`
	Message        string             `json:"message"`
	Category       Category           `json:"category"`
	Severity       Severity           `json:"severity"`
	PartialResults bool               `json:"partial_results"`
	CollectorErrs  []*CollectorError  `json:"collector_errors,omitempty"`
	Timestamp      time.Time          `json:"timestamp"`
	Cause          error              `json:"-"`
}

func (e *ScoringError) Error() string {
	if len(e.CollectorErrs) > 0 {
		return fmt.Sprintf("scoring '%s': %s (%d collector errors)", e.Scenario, e.Message, len(e.CollectorErrs))
	}
	return fmt.Sprintf("scoring '%s': %s", e.Scenario, e.Message)
}

// Unwrap returns the underlying error
func (e *ScoringError) Unwrap() error {
	return e.Cause
}

// NewScoringError creates a new scoring error
func NewScoringError(scenario, message string, category Category, cause error) *ScoringError {
	return &ScoringError{
		Scenario:  scenario,
		Message:   message,
		Category:  category,
		Severity:  SeverityError,
		Timestamp: time.Now(),
		Cause:     cause,
	}
}

// WithPartialResults marks the error as having partial results available
func (e *ScoringError) WithPartialResults() *ScoringError {
	e.PartialResults = true
	return e
}

// WithCollectorErrors attaches collector-specific errors
func (e *ScoringError) WithCollectorErrors(errs ...*CollectorError) *ScoringError {
	e.CollectorErrs = append(e.CollectorErrs, errs...)
	return e
}

// PartialResult represents partial scoring data when some collectors fail
// [REQ:SCS-CORE-004] Return partial scores when some collectors fail
type PartialResult struct {
	Available       map[string]bool     `json:"available"`       // Which data sources succeeded
	Missing         []string            `json:"missing"`         // Which collectors failed
	CollectorErrors []*CollectorError   `json:"collector_errors,omitempty"`
	IsComplete      bool                `json:"is_complete"`
	Confidence      float64             `json:"confidence"`      // 0-1, how reliable the score is
	Message         string              `json:"message,omitempty"`
}

// NewPartialResult creates a new partial result tracker
func NewPartialResult() *PartialResult {
	return &PartialResult{
		Available:  make(map[string]bool),
		Missing:    []string{},
		IsComplete: true,
		Confidence: 1.0,
	}
}

// MarkAvailable marks a collector as successfully providing data
func (p *PartialResult) MarkAvailable(collector string) {
	p.Available[collector] = true
}

// MarkFailed marks a collector as failed and reduces confidence
func (p *PartialResult) MarkFailed(collector string, err *CollectorError) {
	p.Available[collector] = false
	p.Missing = append(p.Missing, collector)
	p.IsComplete = false

	if err != nil {
		p.CollectorErrors = append(p.CollectorErrors, err)
	}

	// Reduce confidence based on collector importance
	switch collector {
	case "requirements", "tests":
		p.Confidence *= 0.7 // Major impact
	case "ui":
		p.Confidence *= 0.85 // Moderate impact
	case "service":
		p.Confidence *= 0.95 // Minor impact
	default:
		p.Confidence *= 0.9
	}
}

// GetMessage generates a user-friendly message about the partial state
func (p *PartialResult) GetMessage() string {
	if p.IsComplete {
		return ""
	}

	if len(p.Missing) == 1 {
		return fmt.Sprintf("Partial results: %s collector unavailable", p.Missing[0])
	}

	return fmt.Sprintf("Partial results: %d collectors unavailable (%s)",
		len(p.Missing), joinStrings(p.Missing, ", "))
}

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += sep + s[i]
	}
	return result
}

// APIError represents a user-facing API error response
type APIError struct {
	Code       string          `json:"code"`
	Message    string          `json:"message"`
	Details    string          `json:"details,omitempty"`
	Category   Category        `json:"category"`
	Severity   Severity        `json:"severity"`
	Recoverable bool           `json:"recoverable"`
	NextSteps  []string        `json:"next_steps,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// NewAPIError creates a user-facing API error
func NewAPIError(code, message string, category Category) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Category:  category,
		Severity:  SeverityError,
		Timestamp: time.Now(),
	}
}

// WithDetails adds technical details (for debugging)
func (e *APIError) WithDetails(details string) *APIError {
	e.Details = details
	return e
}

// WithNextSteps adds actionable guidance for users
func (e *APIError) WithNextSteps(steps ...string) *APIError {
	e.NextSteps = steps
	return e
}

// AsRecoverable marks the error as recoverable
func (e *APIError) AsRecoverable() *APIError {
	e.Recoverable = true
	return e
}

// Common error codes for consistent client handling
const (
	ErrCodeScenarioNotFound     = "SCENARIO_NOT_FOUND"
	ErrCodeCollectorFailed      = "COLLECTOR_FAILED"
	ErrCodeDatabaseError        = "DATABASE_ERROR"
	ErrCodeConfigInvalid        = "CONFIG_INVALID"
	ErrCodeCircuitOpen          = "CIRCUIT_BREAKER_OPEN"
	ErrCodePartialResults       = "PARTIAL_RESULTS"
	ErrCodeInternalError        = "INTERNAL_ERROR"
	ErrCodeValidationFailed     = "VALIDATION_FAILED"
)

// Pre-defined error responses for common scenarios
var (
	ErrScenarioNotFound = NewAPIError(
		ErrCodeScenarioNotFound,
		"Scenario not found",
		CategoryValidation,
	).WithNextSteps(
		"Check the scenario name spelling",
		"Verify the scenario exists in the scenarios directory",
		"Run 'vrooli scenario list' to see available scenarios",
	)

	ErrDatabaseUnavailable = NewAPIError(
		ErrCodeDatabaseError,
		"History database is temporarily unavailable",
		CategoryDatabase,
	).AsRecoverable().WithNextSteps(
		"Scores will still be calculated without historical data",
		"Wait a few seconds and try again",
		"Check disk space if issue persists",
	)

	ErrCollectorCircuitOpen = NewAPIError(
		ErrCodeCircuitOpen,
		"One or more collectors are temporarily disabled",
		CategoryCollector,
	).AsRecoverable().WithNextSteps(
		"Scores are calculated with available data",
		"View collector health at /api/v1/health/collectors",
		"Reset circuit breakers at /api/v1/health/circuit-breaker/reset",
	)
)
