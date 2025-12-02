package types

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure modes.
var (
	ErrNoRequirementsDir = errors.New("requirements directory not found")
	ErrInvalidJSON       = errors.New("invalid JSON in requirement file")
	ErrCycleDetected     = errors.New("cycle detected in requirement hierarchy")
	ErrDuplicateID       = errors.New("duplicate requirement ID")
	ErrMissingReference  = errors.New("validation references non-existent file")
	ErrMissingID         = errors.New("requirement is missing ID field")
	ErrInvalidImport     = errors.New("invalid import path")
	ErrIndexNotFound     = errors.New("index.json not found")
)

// ParseError wraps parsing errors with file context.
type ParseError struct {
	FilePath string
	Line     int
	Column   int
	Err      error
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		if e.Column > 0 {
			return fmt.Sprintf("%s:%d:%d: %v", e.FilePath, e.Line, e.Column, e.Err)
		}
		return fmt.Sprintf("%s:%d: %v", e.FilePath, e.Line, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.FilePath, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// NewParseError creates a ParseError with file context.
func NewParseError(filePath string, err error) *ParseError {
	return &ParseError{
		FilePath: filePath,
		Err:      err,
	}
}

// NewParseErrorAt creates a ParseError with line information.
func NewParseErrorAt(filePath string, line int, err error) *ParseError {
	return &ParseError{
		FilePath: filePath,
		Line:     line,
		Err:      err,
	}
}

// ValidationIssue represents a structural validation problem.
type ValidationIssue struct {
	FilePath      string
	RequirementID string
	Field         string
	Message       string
	Severity      IssueSeverity
}

// IssueSeverity indicates the importance of a validation issue.
type IssueSeverity string

const (
	SeverityError   IssueSeverity = "error"
	SeverityWarning IssueSeverity = "warning"
	SeverityInfo    IssueSeverity = "info"
)

func (i *ValidationIssue) Error() string {
	if i.RequirementID != "" {
		return fmt.Sprintf("%s: %s [%s]: %s", i.FilePath, i.RequirementID, i.Field, i.Message)
	}
	return fmt.Sprintf("%s: %s", i.FilePath, i.Message)
}

// IsError returns true if the issue is an error severity.
func (i *ValidationIssue) IsError() bool {
	return i != nil && i.Severity == SeverityError
}

// IsWarning returns true if the issue is a warning severity.
func (i *ValidationIssue) IsWarning() bool {
	return i != nil && i.Severity == SeverityWarning
}

// ValidationResult collects multiple validation issues.
type ValidationResult struct {
	Issues []ValidationIssue
}

// NewValidationResult creates an empty validation result.
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Issues: make([]ValidationIssue, 0),
	}
}

// AddError adds an error-severity issue.
func (r *ValidationResult) AddError(filePath, reqID, field, message string) {
	r.Issues = append(r.Issues, ValidationIssue{
		FilePath:      filePath,
		RequirementID: reqID,
		Field:         field,
		Message:       message,
		Severity:      SeverityError,
	})
}

// AddWarning adds a warning-severity issue.
func (r *ValidationResult) AddWarning(filePath, reqID, field, message string) {
	r.Issues = append(r.Issues, ValidationIssue{
		FilePath:      filePath,
		RequirementID: reqID,
		Field:         field,
		Message:       message,
		Severity:      SeverityWarning,
	})
}

// AddInfo adds an info-severity issue.
func (r *ValidationResult) AddInfo(filePath, reqID, field, message string) {
	r.Issues = append(r.Issues, ValidationIssue{
		FilePath:      filePath,
		RequirementID: reqID,
		Field:         field,
		Message:       message,
		Severity:      SeverityInfo,
	})
}

// HasErrors returns true if any error-severity issues exist.
func (r *ValidationResult) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if any warning-severity issues exist.
func (r *ValidationResult) HasWarnings() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of error-severity issues.
func (r *ValidationResult) ErrorCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warning-severity issues.
func (r *ValidationResult) WarningCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

// Errors returns only error-severity issues.
func (r *ValidationResult) Errors() []ValidationIssue {
	result := make([]ValidationIssue, 0)
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			result = append(result, issue)
		}
	}
	return result
}

// Warnings returns only warning-severity issues.
func (r *ValidationResult) Warnings() []ValidationIssue {
	result := make([]ValidationIssue, 0)
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			result = append(result, issue)
		}
	}
	return result
}

// Merge combines another validation result into this one.
func (r *ValidationResult) Merge(other *ValidationResult) {
	if other == nil {
		return
	}
	r.Issues = append(r.Issues, other.Issues...)
}

// DiscoveryError represents an error during file discovery.
type DiscoveryError struct {
	Path string
	Err  error
}

func (e *DiscoveryError) Error() string {
	return fmt.Sprintf("discovery error at %s: %v", e.Path, e.Err)
}

func (e *DiscoveryError) Unwrap() error {
	return e.Err
}

// NewDiscoveryError creates a DiscoveryError.
func NewDiscoveryError(path string, err error) *DiscoveryError {
	return &DiscoveryError{
		Path: path,
		Err:  err,
	}
}

// SyncError represents an error during synchronization.
type SyncError struct {
	FilePath string
	Phase    string
	Err      error
}

func (e *SyncError) Error() string {
	if e.Phase != "" {
		return fmt.Sprintf("sync error in %s for %s: %v", e.Phase, e.FilePath, e.Err)
	}
	return fmt.Sprintf("sync error for %s: %v", e.FilePath, e.Err)
}

func (e *SyncError) Unwrap() error {
	return e.Err
}

// NewSyncError creates a SyncError.
func NewSyncError(filePath string, err error) *SyncError {
	return &SyncError{
		FilePath: filePath,
		Err:      err,
	}
}

// NewSyncErrorWithPhase creates a SyncError with phase context.
func NewSyncErrorWithPhase(filePath, phase string, err error) *SyncError {
	return &SyncError{
		FilePath: filePath,
		Phase:    phase,
		Err:      err,
	}
}

// IsNotFound returns true if the error indicates a missing resource.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNoRequirementsDir) ||
		errors.Is(err, ErrIndexNotFound) ||
		errors.Is(err, ErrMissingReference)
}

// IsValidationError returns true if the error is a validation-related error.
func IsValidationError(err error) bool {
	return errors.Is(err, ErrDuplicateID) ||
		errors.Is(err, ErrMissingID) ||
		errors.Is(err, ErrCycleDetected) ||
		errors.Is(err, ErrInvalidImport)
}

// IsParseError returns true if the error is a parsing error.
func IsParseError(err error) bool {
	var parseErr *ParseError
	return errors.As(err, &parseErr) || errors.Is(err, ErrInvalidJSON)
}
