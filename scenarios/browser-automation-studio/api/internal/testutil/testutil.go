// Package testutil contains shared test infrastructure for the
// browser-automation-studio API: domain factories, hand-written fakes,
// test database helpers, HTTP harnesses, and BAS-specific assertions.
//
// # Why this package exists
//
// Browser Automation Studio has broad API coverage, but many tests still
// carry package-local setup and mocks. This package is the canonical home
// for recurring test seams so new tests can stay small, deterministic, and
// focused on expected behavior instead of wiring.
//
// Fakes in this tree should be hand-written, inspectable, and deterministic
// by default. Constructors should return useful defaults, and tests should
// opt into only the differences needed for the behavior under test.
//
// # Test-only contract
//
// No production code may import internal/testutil/...; that boundary is
// enforced by no_prod_import_test.go. The guarantee lets this package depend
// freely on testing-only behavior and keeps helper APIs optimized for test
// readability instead of runtime use.
//
// # Intended subpackages
//
//   - fixtures/ — domain-object factories for projects, workflows,
//     executions, recordings, session profiles, timeline entries, and export
//     specs.
//   - mocks/ — one canonical fake per recurring seam, such as repositories,
//     catalog/execution services, storage, hubs, clocks, drivers, AI/vision
//     clients, scenario-port resolvers, and import scanners/indexers.
//   - db/ — temp database setup and cleanup helpers.
//   - httpx/ — handler harnesses and response assertions.
//   - assertx/ — BAS-domain assertions for workflow definitions, protocol
//     responses, event ordering, exports, and recording timelines.
//   - integration/ — shared skip gates for optional local services and tools
//     such as Playwright, Ollama, MinIO, and FFmpeg.
package testutil
