// Package uismoke provides UI smoke testing functionality for Vrooli scenarios.
//
// The UI smoke test validates that a scenario's web UI loads correctly,
// establishes communication with the host via iframe-bridge, and produces
// no critical errors. Results include screenshots, console logs, network
// failures, and DOM snapshots.
//
// # Architecture
//
// The package follows a layered architecture:
//   - uismoke: Shared types (Config, Result, Status)
//   - orchestrator: Coordinates the smoke test workflow
//   - preflight: Validates preconditions (browserless, bundle, bridge)
//   - browser: Browserless API client
//   - handshake: Bridge handshake detection
//   - artifacts: Result persistence
package uismoke
