// Package testutil holds the test-only helpers the API uses for
// dependency-injected handler/middleware/repository tests.
//
// The package is split into focused sub-packages so consumers import only
// what they need and the surface area stays scannable as it grows:
//
//   - mocks/    canonical hand-written fakes for scenario-owned interfaces
//     (FakeClock and domain dependencies). One implementation per seam;
//     tests arrange via struct-field mutation, not method calls. Per-operation
//     error fields are the standard failure-injection idiom.
//   - fixtures/ add domain factories when tests need reusable inputs.
//     The template health factory is a teaching example excluded from
//     generated scenarios; handler tests decode actual responses.
//   - db/       per-test SQLite handles (modernc.org/sqlite, pure-Go,
//     CGO-clean) for repository-integration tests. Schema is
//     applied per handle so each test starts from a fresh,
//     production-equivalent database.
//   - httpx/    a live HTTP harness wrapping the production middleware in
//     an httptest.Server. Catches the class of bug where a
//     wrapper drops http.Flusher/http.Hijacker — ResponseRecorder
//     fakes those interfaces and would silently pass.
//   - assertx/  domain-aware test assertions (AssertStatus, MustDecodeJSON).
//     Stay focused; resist generalising into a god-helper grab
//     bag.
//
// # Production must not import testutil
//
// no_prod_import_test.go enforces this at the AST level: any non-test file
// under api/ that imports `<module>/internal/testutil/...` fails the test
// suite. This is what lets testutil freely depend on `testing`, mutate
// process-wide state in fakes, and expose concurrency knobs only tests
// need — production code can't see any of it.
package testutil

import (
	"os"
	"testing"
)

// SocketDir keeps Unix socket paths within platform limits even when the test
// runner supplies a long TMPDIR. Cleanup is registered before the caller uses it.
func SocketDir(t testing.TB) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "dc-sock-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
