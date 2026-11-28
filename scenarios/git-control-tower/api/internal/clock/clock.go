package clock

import "time"

// ApplyOverride replaces an injected clock only when an override is supplied.
func ApplyOverride(target *func() time.Time, override func() time.Time) {
	if override != nil {
		*target = override
	}
}
