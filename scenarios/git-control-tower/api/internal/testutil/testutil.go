// Package testutil contains shared test infrastructure for the
// git-control-tower API: canonical fakes (mocks/), domain fixtures
// (fixtures/), HTTP/client harnesses (httpx/), test database helpers
// (db/), and assertion helpers (assertx/).
//
// No production code may import internal/testutil/...; that contract is
// enforced by no_prod_import_test.go. Keeping this package test-only lets
// helpers depend on testing.T, httptest, and deterministic fake state without
// leaking those choices into runtime code.
package testutil
