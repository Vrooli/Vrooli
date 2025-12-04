// Package business provides requirements validation orchestration for the business phase.
// It is organized to follow "screaming architecture" principles where the package
// structure communicates domain concepts.
//
// The package exposes a Runner that orchestrates:
//   - Discovery: Finding requirement files in the scenario
//   - Parsing: Reading and parsing requirement modules
//   - Validation: Checking structural integrity (duplicates, cycles, orphans, etc.)
//
// This design mirrors the structure package, providing a clean separation between
// the phase orchestrator (phases/business.go) and the actual validation logic.
package business
