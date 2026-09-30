//go:build !linux

package collectors

import "time"

func collectFragmentation(_ *counterRateTracker, _ map[string]uint64, _ time.Time) fragmentationReading {
	return fragmentationReading{
		status:     "unsupported",
		reason:     "buddy-allocator fragmentation is a Linux concept",
		provenance: "platform capability",
	}
}
