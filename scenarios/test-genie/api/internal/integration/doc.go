// Package integration provides validators for the integration testing phase.
// This phase validates CLI functionality and executes BATS acceptance tests.
//
// The package follows a screaming architecture pattern where each validation
// concern is isolated in its own subpackage:
//   - cli: CLI binary discovery and command validation (help, version)
//   - bats: BATS test suite discovery and execution
//
// The main Runner orchestrates these validators and supports dependency injection
// for testing via functional options.
package integration
