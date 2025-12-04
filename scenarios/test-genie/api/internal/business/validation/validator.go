package validation

import (
	"context"
	"fmt"
	"io"
	"strings"

	"test-genie/internal/requirements/parsing"
	reqvalidation "test-genie/internal/requirements/validation"
)

// Validator validates requirement structure and returns validation results.
type Validator interface {
	// Validate checks all requirements for structural issues.
	Validate(ctx context.Context, index *parsing.ModuleIndex, scenarioRoot string) ValidationResult
}

// validator wraps requirements/validation with validation result handling.
type validator struct {
	underlying reqvalidation.Validator
	logWriter  io.Writer
}

// New creates a new structural validator.
func New(logWriter io.Writer) Validator {
	return &validator{
		underlying: reqvalidation.NewDefault(),
		logWriter:  logWriter,
	}
}

// NewWithValidator creates a validator with a custom underlying validator (for testing).
func NewWithValidator(underlying reqvalidation.Validator, logWriter io.Writer) Validator {
	return &validator{
		underlying: underlying,
		logWriter:  logWriter,
	}
}

// Validate implements Validator.
func (v *validator) Validate(ctx context.Context, index *parsing.ModuleIndex, scenarioRoot string) ValidationResult {
	logInfo(v.logWriter, "Running structural validation rules...")

	underlyingResult := v.underlying.Validate(ctx, index, scenarioRoot)

	errorCount := underlyingResult.ErrorCount()
	warningCount := underlyingResult.WarningCount()

	var observations []Observation

	// Add warnings as observations
	for _, issue := range underlyingResult.Warnings() {
		observations = append(observations, NewWarningObservation(issue.Error()))
	}

	// Check if there are errors
	if underlyingResult.HasErrors() {
		errorMsgs := make([]string, 0, errorCount)
		for _, issue := range underlyingResult.Errors() {
			errorMsgs = append(errorMsgs, issue.Error())
			observations = append(observations, NewErrorObservation(issue.Error()))
			logError(v.logWriter, "validation error: %s", issue.Error())
		}

		// Build detailed error message
		errDetail := strings.Join(errorMsgs, "; ")

		return ValidationResult{
			Result: FailMisconfiguration(
				fmt.Errorf("structural validation failed: %s", errDetail),
				"Fix the structural issues in requirement files: duplicate IDs, missing references, cycles in hierarchy.",
			).WithObservations(observations...),
			ErrorCount:   errorCount,
			WarningCount: warningCount,
			Issues:       underlyingResult.Issues,
		}
	}

	logSuccess(v.logWriter, "validation rules checked: %d issues (%d errors, %d warnings)",
		len(underlyingResult.Issues), errorCount, warningCount)

	result := OK()
	result.Observations = observations

	return ValidationResult{
		Result:       result,
		ErrorCount:   errorCount,
		WarningCount: warningCount,
		Issues:       underlyingResult.Issues,
	}
}

// Logging helpers

func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

func logSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] ✅ %s\n", msg)
}

func logError(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[ERROR] ❌ %s\n", msg)
}
