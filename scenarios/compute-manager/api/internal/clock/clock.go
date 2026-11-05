// Package clock defines the process-wide time seam used by scheduled and
// persistence code. Production wiring uses System; tests provide a fixed
// implementation without sleeping or reading wall-clock time.
package clock

import "time"

type Clock interface {
	Now() time.Time
}

type System struct{}

func (System) Now() time.Time { return time.Now() }
