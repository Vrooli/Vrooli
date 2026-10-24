// Package testutil contains shared test infrastructure for the
// workspace-sandbox API: hand-written fakes (mocks/), domain object
// factories (fixtures/), test database helpers (db/), service builders
// (services/), live-HTTP harnesses (httpx/), and assertion helpers
// (assertx/).
//
// # Why this package exists
//
// Before Round 4, every test file authored its own set of mocks. The
// same Repository surface was implemented three times (with subtle
// drift), the same Driver surface four times, and so on. Cross-seam
// invariants (e.g., "what does the service look like when both a
// failing repo and a failing driver are wired in?") were impossible
// to express because the fakes never met.
//
// This package consolidates every fake into one place. Each fake has
// a `New*` constructor returning a sane default state, exported fields
// for inspecting state after a test, and per-method error knobs for
// failure-mode tests. The fakes are hand-written (no mockery, gomock,
// or other code generation) and intentionally small.
//
// # Greenfield contract
//
// No production code may import internal/testutil/... — this is
// asserted by no_prod_import_test.go. That guarantee lets us depend
// freely on `testing`, exercise concurrency knobs only tests need,
// and keep the fakes optimized for ergonomics rather than runtime
// performance.
//
// # Subpackages
//
//   - mocks/    — single canonical fake per seam (FakeRepository,
//     FakeDriver, FakeGitOps, FakeService, FakePinger,
//     FakeReconciler, FakeProfileStore, FakeProcFS).
//     Future-phase fakes (FakeClock, FakeMounter, FakeStarter,
//     FakeSSEWriter, FakeAuditEmitter) land in their respective
//     phases.
//   - fixtures/ — domain-object factories (NewSandbox,
//     NewIsolationProfile, NewExitInfo). Functional options keep
//     defaults out of test bodies.
//   - db/       — `NewSQLite(t)` returns a connected DB with the
//     production schema applied to a `t.TempDir()` file.
//   - services/ — Service builders that wire the fakes together
//     ergonomically (NewService(t, opts...)).
//   - httpx/    — Live-HTTP harness (Phase 3 expands this) and the
//     strict SSE frame parser used by ordering tests.
//   - assertx/  — Domain assertions that produce useful diff output
//     when they fail.
package testutil
