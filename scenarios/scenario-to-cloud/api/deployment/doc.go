// Package deployment provides the core deployment orchestration for scenario-to-cloud.
// It coordinates the full deployment pipeline: secrets fetching, bundle building,
// preflight checks, VPS setup, and VPS deployment.
//
// The package also manages progress tracking via SSE (Server-Sent Events),
// allowing clients to subscribe to real-time deployment progress updates.
//
// Key types:
//   - Orchestrator: Coordinates the full deployment pipeline
//   - Hub: Manages SSE subscriptions for progress events
//   - Event: Represents a progress update sent to clients
package deployment
