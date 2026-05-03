// Package testutil holds the test-only helpers the API uses for
// dependency-injected handler/middleware/repository tests.
//
// The package is split into focused sub-packages so consumers import only
// what they need and the surface area stays scannable as it grows:
//
//   - mocks/    canonical hand-written fakes for each interface (FakeClock,
//               FakePinger). One implementation per seam; tests arrange via
//               struct-field mutation, not method calls. Per-method error
//               knobs (`PingErr`, future `CreateErr`) are the standard
//               failure-injection idiom.
//   - fixtures/ domain factories using the functional-options pattern
//               (NewHealthResponse(WithHealthStatus(...))). Default values
//               are picked so the most common test path is `fixtures.NewX()`
//               with no opts. Factories take *testing.T only when they
//               validate inputs — current factories are plain-data
//               builders, so they don't.
//   - db/       per-test SQLite handles (modernc.org/sqlite, pure-Go,
//               CGO-clean) for repository-integration tests. Schema is
//               applied per handle so each test starts from a fresh,
//               production-equivalent database.
//   - httpx/    a live HTTP harness wrapping the production middleware in
//               an httptest.Server. Catches the class of bug where a
//               wrapper drops http.Flusher/http.Hijacker — ResponseRecorder
//               fakes those interfaces and would silently pass.
//   - assertx/  domain-aware test assertions (AssertStatus, MustDecodeJSON).
//               Stay focused; resist generalising into a god-helper grab
//               bag.
//
// # Production must not import testutil
//
// no_prod_import_test.go enforces this at the AST level: any non-test file
// under api/ that imports `<module>/internal/testutil/...` fails the test
// suite. This is what lets testutil freely depend on `testing`, mutate
// process-wide state in fakes, and expose concurrency knobs only tests
// need — production code can't see any of it.
package testutil
