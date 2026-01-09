// Package golang provides a unit test runner for Go projects.
//
// The runner detects Go projects by checking for go.mod in the api/ directory,
// then executes `go test ./...` to run all tests.
//
// Detection:
//   - Checks for api/go.mod
//   - Verifies `go` command is available
//
// Execution:
//   - Runs `go test ./...` in the api/ directory
//   - Captures and reports test output
//   - Extracts coverage data if available
package golang
