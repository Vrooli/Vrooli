// Package clock defines the injectable wall-clock seam used by API services.
package clock

import "time"

// Clock is the narrow time dependency shared by production code and tests.
type Clock interface {
	Now() time.Time
}

// System returns the process clock for the composition root.
type system struct{}

func (system) Now() time.Time { return time.Now() }

func System() Clock { return system{} }
