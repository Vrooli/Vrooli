// Package testutil contains shared test infrastructure for the
// prompt-manager API: domain fixtures, hand-written fakes, HTTP
// helpers, and focused assertions.
//
// # Why this package exists
//
// Prompt-manager tests historically grew package-local helpers for the
// same seams. Team fixtures are the first consolidation target because
// heartbeat, teams, and store tests each built nearly identical team
// objects with small undocumented drift.
//
// Test utilities live under internal/testutil so they are available to
// API tests but hidden from production packages outside this module.
// no_prod_import_test.go enforces the stronger local rule: non-test Go
// files must not import prompt-manager/internal/testutil.
//
// # Subpackages
//
//   - fixtures/ — domain-object factories with functional options.
//   - mocks/    — canonical hand-written fakes, one per production seam.
//   - httpx/    — request/response helpers for handler tests.
//   - assertx/  — domain assertions with clear failure messages.
package testutil
