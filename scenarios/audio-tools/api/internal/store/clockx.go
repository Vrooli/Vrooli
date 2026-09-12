package store

import (
	"time"

	"github.com/vrooli/api-core/schedule"
)

// defaultClock is the package-level clock seam consulted by every Store
// for CreatedAt / EmittedAt / UpdatedAt stamping. Production lets the
// default stand (schedule.System()); tests override via SetClockForTest in
// a sub-test that does not run in parallel with other store_test files.
//
// Per-Store Clock fields would be more idiomatic, but every store
// constructor in this package has its own signature; threading clock
// through them all would balloon the diff well past the value of the
// substitution for these stamp-only callsites. The wrap function below
// stays the only callsite for time.Now-equivalent reads.
var defaultClock schedule.Clock = schedule.System()

// now returns the current UTC time via the package-level seam.
func now() time.Time {
	if defaultClock == nil {
		return schedule.System().Now().UTC()
	}
	return defaultClock.Now().UTC()
}

// SetClockForTest replaces the package-level clock for the duration of
// the test. The returned func restores the previous clock; tests pair
// it with t.Cleanup. NOT safe for parallel test packages within the
// same process.
func SetClockForTest(c schedule.Clock) func() {
	prev := defaultClock
	defaultClock = c
	return func() { defaultClock = prev }
}
