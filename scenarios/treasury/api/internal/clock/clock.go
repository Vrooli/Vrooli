// Package clock provides the Treasury-owned wall-clock seam.
package clock

import "time"

// Clock is the wall clock used by authorization and expiry decisions.
type Clock interface {
	Now() time.Time
}

// System supplies the production wall clock at the composition boundary.
type System struct{}

func (System) Now() time.Time { return time.Now() }
