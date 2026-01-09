// Package preflight provides precondition checks for UI smoke tests.
//
// The preflight checker validates:
//   - Browserless service availability and health
//   - UI bundle freshness
//   - iframe-bridge dependency presence
//   - UI port discovery
package preflight
