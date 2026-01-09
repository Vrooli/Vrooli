// Package types provides shared types for structure validation sub-packages.
//
// This package exists to avoid circular imports between the structure package
// and its sub-packages (existence, content, smoke). All sub-packages import
// these shared types rather than defining their own.
//
// Types defined here:
//   - Observation: A single validation observation with formatting hints
//   - ObservationType: Categorizes the kind of observation
//   - Result: The outcome of a validation step
//   - FailureClass: Categorizes the type of validation failure
package types
