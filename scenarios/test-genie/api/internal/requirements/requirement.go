// Package requirements implements native Go requirements synchronization.
// This file re-exports types from the types subpackage for backwards compatibility.
package requirements

import "test-genie/internal/requirements/types"

// Type aliases for requirement types
type Requirement = types.Requirement
type AggregatedStatus = types.AggregatedStatus
type ValidationSummary = types.ValidationSummary
type Validation = types.Validation
type LiveDetails = types.LiveDetails

// Function re-exports
var ComputeValidationSummary = types.ComputeValidationSummary
