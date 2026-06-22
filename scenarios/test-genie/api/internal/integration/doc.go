// Package integration provides validators for the integration testing phase.
// This phase validates CLI functionality, API health, and WebSocket connectivity.
//
// The package follows a screaming architecture pattern where each validation
// concern is isolated in its own subpackage:
//   - cli: CLI binary discovery and command validation (help, version)
//   - api: API health-endpoint validation
//   - websocket: WebSocket connectivity validation
//
// The main Runner orchestrates these validators and supports dependency injection
// for testing via functional options.
package integration
