// Package requirements implements native Go requirements synchronization.
// This file re-exports types from the types subpackage for backwards compatibility.
package requirements

import "test-genie/internal/requirements/types"

// Sentinel errors re-exported from types
var (
	ErrNoRequirementsDir = types.ErrNoRequirementsDir
	ErrInvalidJSON       = types.ErrInvalidJSON
	ErrCycleDetected     = types.ErrCycleDetected
	ErrDuplicateID       = types.ErrDuplicateID
	ErrMissingReference  = types.ErrMissingReference
	ErrMissingID         = types.ErrMissingID
	ErrInvalidImport     = types.ErrInvalidImport
	ErrIndexNotFound     = types.ErrIndexNotFound
)

// Type aliases for error types
type ParseError = types.ParseError
type ValidationIssue = types.ValidationIssue
type IssueSeverity = types.IssueSeverity
type ValidationResult = types.ValidationResult
type DiscoveryError = types.DiscoveryError
type SyncError = types.SyncError

// Severity constants
const (
	SeverityError   = types.SeverityError
	SeverityWarning = types.SeverityWarning
	SeverityInfo    = types.SeverityInfo
)

// Function re-exports
var (
	NewParseError         = types.NewParseError
	NewParseErrorAt       = types.NewParseErrorAt
	NewValidationResult   = types.NewValidationResult
	NewDiscoveryError     = types.NewDiscoveryError
	NewSyncError          = types.NewSyncError
	NewSyncErrorWithPhase = types.NewSyncErrorWithPhase
	IsNotFound            = types.IsNotFound
	IsValidationError     = types.IsValidationError
	IsParseError          = types.IsParseError
)
