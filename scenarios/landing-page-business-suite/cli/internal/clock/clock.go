// Package clock provides wall-clock time to CLI domains through an injectable seam.
package clock

import "time"

// seam: Clock supplies wall-clock time to CLI commands. Production uses System;
// tests can provide a deterministic implementation.
type Clock interface {
	Now() time.Time
}

type System struct{}

func (System) Now() time.Time { return time.Now() }

var _ Clock = System{}
