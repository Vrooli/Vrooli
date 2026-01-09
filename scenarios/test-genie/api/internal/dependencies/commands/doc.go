// Package commands validates that required commands are available in PATH.
//
// This includes baseline commands needed for local phase execution:
//   - bash: Shell scripting and CLI operations
//   - curl: HTTP requests for health checks
//   - jq: JSON processing for configuration parsing
//
// The package provides an interface for testing seams and supports dependency
// injection for isolated unit tests.
package commands
