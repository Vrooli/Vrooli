package tasks

import (
	"encoding/json"
	"fmt"
)

// TaskResults provides structured, type-safe task execution results
// This replaces the loose map[string]any pattern with compile-time safety
type TaskResults struct {
	// Execution outcome
	Success bool   `json:"success" yaml:"success"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
	Output  string `json:"output,omitempty" yaml:"output,omitempty"`
	Error   string `json:"error,omitempty" yaml:"error,omitempty"`

	// Timing information
	ExecutionTime  string `json:"execution_time,omitempty" yaml:"execution_time,omitempty"`
	TimeoutAllowed string `json:"timeout_allowed,omitempty" yaml:"timeout_allowed,omitempty"`
	StartedAt      string `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	CompletedAt    string `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	FailedAt       string `json:"failed_at,omitempty" yaml:"failed_at,omitempty"`

	// Prompt metadata
	PromptSize string `json:"prompt_size,omitempty" yaml:"prompt_size,omitempty"`

	// Failure modes
	TimeoutFailure   bool `json:"timeout_failure,omitempty" yaml:"timeout_failure,omitempty"`
	MaxTurnsExceeded bool `json:"max_turns_exceeded,omitempty" yaml:"max_turns_exceeded,omitempty"`

	// Rate limiting
	RateLimitInfo *RateLimitInfo `json:"rate_limit_info,omitempty" yaml:"rate_limit_info,omitempty"`

	// Recycler metadata
	RecyclerClassification string `json:"recycler_classification,omitempty" yaml:"recycler_classification,omitempty"`
	RecyclerUpdatedAt      string `json:"recycler_updated_at,omitempty" yaml:"recycler_updated_at,omitempty"`

	// Extension point for backward compatibility and future fields
	Extensions map[string]any `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

// RateLimitInfo contains rate limit hit information
type RateLimitInfo struct {
	HitAt      string `json:"hit_at" yaml:"hit_at"`
	RetryAfter int    `json:"retry_after" yaml:"retry_after"` // Seconds
	Message    string `json:"message" yaml:"message"`
}

// ToMap converts structured results to map[string]any for backward compatibility
// This ensures existing code that expects map[string]any continues to work
func (r *TaskResults) ToMap() map[string]any {
	data, _ := json.Marshal(r)
	var result map[string]any
	json.Unmarshal(data, &result)

	// Merge extensions into top level for backward compatibility
	if r.Extensions != nil {
		for k, v := range r.Extensions {
			result[k] = v
		}
	}

	return result
}

// FromMap creates TaskResults from a legacy map[string]any
func FromMap(m map[string]any) *TaskResults {
	if m == nil {
		return &TaskResults{}
	}

	// Marshal and unmarshal through JSON for type conversion
	data, _ := json.Marshal(m)
	var result TaskResults
	json.Unmarshal(data, &result)

	// Store unmapped fields in Extensions
	result.Extensions = make(map[string]any)
	knownFields := map[string]bool{
		"success": true, "message": true, "output": true, "error": true,
		"execution_time": true, "timeout_allowed": true, "started_at": true,
		"completed_at": true, "failed_at": true, "prompt_size": true,
		"timeout_failure": true, "max_turns_exceeded": true, "rate_limit_info": true,
		"recycler_classification": true, "recycler_updated_at": true, "extensions": true,
	}

	for k, v := range m {
		if !knownFields[k] {
			result.Extensions[k] = v
		}
	}

	return &result
}

// NewSuccessResults creates results for a successful task execution
func NewSuccessResults(message, output, executionTime, timeoutAllowed, startedAt, completedAt, promptSize string) *TaskResults {
	return &TaskResults{
		Success:        true,
		Message:        message,
		Output:         output,
		ExecutionTime:  executionTime,
		TimeoutAllowed: timeoutAllowed,
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
		PromptSize:     promptSize,
	}
}

// NewFailureResults creates results for a failed task execution
func NewFailureResults(errorMsg, output, executionTime, timeoutAllowed, startedAt, failedAt, promptSize string, isTimeout bool, extras map[string]any) *TaskResults {
	result := &TaskResults{
		Success:        false,
		Error:          errorMsg,
		Output:         output,
		ExecutionTime:  executionTime,
		TimeoutAllowed: timeoutAllowed,
		StartedAt:      startedAt,
		FailedAt:       failedAt,
		PromptSize:     promptSize,
		TimeoutFailure: isTimeout,
		Extensions:     extras,
	}

	// Extract max_turns_exceeded from extras if present
	if extras != nil {
		if maxTurns, ok := extras["max_turns_exceeded"].(bool); ok {
			result.MaxTurnsExceeded = maxTurns
		}
	}

	return result
}

// NewRateLimitResults creates results for a rate-limited task
func NewRateLimitResults(hitAt string, retryAfter int) *TaskResults {
	return &TaskResults{
		Success: false,
		RateLimitInfo: &RateLimitInfo{
			HitAt:      hitAt,
			RetryAfter: retryAfter,
			Message:    fmt.Sprintf("Rate limited at %s. Will retry after %d seconds.", hitAt, retryAfter),
		},
	}
}

// SetRecyclerInfo updates recycler metadata
func (r *TaskResults) SetRecyclerInfo(classification, updatedAt string) {
	r.RecyclerClassification = classification
	r.RecyclerUpdatedAt = updatedAt
}

// MergeExtensions adds additional fields to the Extensions map
func (r *TaskResults) MergeExtensions(extras map[string]any) {
	if r.Extensions == nil {
		r.Extensions = make(map[string]any)
	}
	for k, v := range extras {
		r.Extensions[k] = v
	}
}
