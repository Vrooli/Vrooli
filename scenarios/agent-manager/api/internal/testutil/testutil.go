// Package testutil contains shared test infrastructure for the agent-manager
// API: hand-written fakes (mocks/), domain object factories (fixtures/), test
// database helpers, HTTP helpers (httpx/), and assertion helpers (assertx/).
//
// # Why this package exists
//
// Agent Manager has stable seams for runner execution, sandbox lifecycle,
// event storage, repositories, and tool discovery. Tests should reuse one
// canonical fake per broad seam instead of redefining mocks in each test file.
// Shared fakes should have sane defaults, explicit error knobs, and inspection
// helpers for calls made by the code under test.
//
// # Test-only contract
//
// No production code may import internal/testutil or child packages. The
// no_prod_import_test.go meta-test enforces that boundary so this package can
// depend on testing-oriented behavior without leaking it into runtime builds.
//
// # Subpackages
//
//   - mocks/    - canonical fakes for broad interfaces.
//   - fixtures/ - domain object factories with explicit override options.
//   - assertx/  - domain-aware assertions with useful failure messages.
//   - httpx/    - HTTP handler and protocol test helpers.
//
// Existing SQLite helpers remain in this package for compatibility while DB
// helpers are migrated incrementally.
package testutil
