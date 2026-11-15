// Package clock provides deterministic wall-clock access for Treasury CLI domains.
package clock

import "time"

// Clock is the timestamp source used when signing a run-owned settlement proof.
type Clock interface {
	Now() time.Time
}

// System is the production wall clock.
type System struct{}

func (System) Now() time.Time { return time.Now() }
