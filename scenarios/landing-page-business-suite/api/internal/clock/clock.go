// Package clock provides the wall-clock substrate shared by domain services.
package clock

import "time"

// seam: Clock provides wall-clock time to domain services. Production uses
// System; deterministic tests use testutil/mocks.FakeClock.
type Clock interface {
	Now() time.Time
}

type System struct{}

func (System) Now() time.Time { return time.Now() }

var _ Clock = System{}
