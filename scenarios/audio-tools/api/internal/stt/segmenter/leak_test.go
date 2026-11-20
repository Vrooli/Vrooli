package segmenter

import "go.uber.org/goleak"

// ignoreDependencyGoroutines exempts process-lifetime goroutines owned by
// imported dependencies from this package's leak checks.
//
// packages/api-core/database constructs its foreign-pin pool at package init
// with sql.OpenDB, which starts database/sql's connectionOpener for the life of
// the process. That goroutine is created before any test runs and is not owned
// by the segmenter, so goleak reports it as a leak in every test here. The
// ignore can be removed once the dependency opens that pool lazily or exposes a
// close.
func ignoreDependencyGoroutines() goleak.Option {
	return goleak.IgnoreTopFunction("database/sql.(*DB).connectionOpener")
}
