// Package bats provides BATS test suite discovery and execution for the integration phase.
//
// It handles:
//   - Verifying bats is available on the system
//   - Discovering the primary BATS suite (e.g., scenario-name.bats)
//   - Discovering additional BATS suites in cli/test/
//   - Executing BATS suites with TAP output
package bats
