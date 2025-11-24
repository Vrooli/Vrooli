package validator

import (
	"context"
	"time"
)

const schemaVersion = "2025.11.22"

// Validator validates workflow definitions against the canonical schema and lint rules.
type Validator struct{}

// Options control linting behaviour.
type Options struct {
	Strict bool
}

// IssueSeverity conveys validation severity.
type IssueSeverity string

const (
	// SeverityError indicates a blocking issue.
	SeverityError IssueSeverity = "error"
	// SeverityWarning indicates a non-blocking issue.
	SeverityWarning IssueSeverity = "warning"
)

// Issue represents a validation finding.
type Issue struct {
	Severity IssueSeverity `json:"severity"`
	Code     string        `json:"code"`
	Message  string        `json:"message"`
	NodeID   string        `json:"node_id,omitempty"`
	NodeType string        `json:"node_type,omitempty"`
	Field    string        `json:"field,omitempty"`
	Pointer  string        `json:"pointer,omitempty"`
	Hint     string        `json:"hint,omitempty"`
}

// Stats describe the validated workflow.
type Stats struct {
	NodeCount            int  `json:"node_count"`
	EdgeCount            int  `json:"edge_count"`
	SelectorCount        int  `json:"selector_count"`
	UniqueSelectorCount  int  `json:"unique_selector_count"`
	ElementWaitCount     int  `json:"element_wait_count"`
	HasMetadata          bool `json:"has_metadata"`
	HasExecutionViewport bool `json:"has_execution_viewport"`
}

// Result is returned after validation.
type Result struct {
	Valid         bool      `json:"valid"`
	Errors        []Issue   `json:"errors"`
	Warnings      []Issue   `json:"warnings"`
	Stats         Stats     `json:"stats"`
	SchemaVersion string    `json:"schema_version"`
	CheckedAt     time.Time `json:"checked_at"`
	DurationMs    int64     `json:"duration_ms"`
}

// NewValidator creates a Validator instance.
func NewValidator() (*Validator, error) {
	if _, err := loadSchema(); err != nil {
		return nil, err
	}
	return &Validator{}, nil
}

// Validate validates a workflow definition.
func (v *Validator) Validate(ctx context.Context, definition map[string]any, opts Options) (*Result, error) {
	if definition == nil {
		definition = map[string]any{}
	}

	start := time.Now()
	result := &Result{
		SchemaVersion: schemaVersion,
		CheckedAt:     time.Now(),
	}

	schema, err := loadSchema()
	if err != nil {
		return nil, err
	}

	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	if err := schema.Validate(definition); err != nil {
		result.Errors = append(result.Errors, convertSchemaErrors(err)...)
	}

	stats, lintErrors, lintWarnings := runLint(definition)
	result.Stats = stats
	result.Errors = append(result.Errors, lintErrors...)
	result.Warnings = append(result.Warnings, lintWarnings...)

	if opts.Strict && len(result.Warnings) > 0 {
		for _, warn := range result.Warnings {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_STRICT_WARNING",
				Message:  "Warning promoted to error in strict mode: " + warn.Message,
				NodeID:   warn.NodeID,
				NodeType: warn.NodeType,
				Field:    warn.Field,
				Pointer:  warn.Pointer,
				Hint:     warn.Hint,
			})
		}
	}

	result.Valid = len(result.Errors) == 0
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}
