// Package requirements implements native Go requirements synchronization.
// This file re-exports types from the types subpackage for backwards compatibility.
package requirements

import "test-genie/internal/requirements/types"

// Type aliases for requirement types
type (
	Requirement       = types.Requirement
	AggregatedStatus  = types.AggregatedStatus
	ValidationSummary = types.ValidationSummary
	Validation        = types.Validation
	LiveDetails       = types.LiveDetails
)

// Function re-exports
var ComputeValidationSummary = types.ComputeValidationSummary
