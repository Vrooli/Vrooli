// Package testutil is the home for test-only helpers the API's tests share.
// It holds none today: shared fakes come from api-core (apihttptest.FakeDoer
// for outbound HTTP, scheduletest.FakeClock, databasetest.FakePinger) and
// one-domain fakes live beside their domain under internal/<domain>/mocks/.
//
// # Production must not import testutil
//
// no_prod_import_test.go enforces this at the AST level: any non-test file
// under api/ that imports `<module>/internal/testutil/...` fails the test
// suite. This is what lets testutil freely depend on `testing`, mutate
// process-wide state in fakes, and expose concurrency knobs only tests
// need — production code can't see any of it.
package testutil
