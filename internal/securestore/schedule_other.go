//go:build !linux && !darwin && !windows

package securestore

import "time"

const (
	nativeScheduleProvider  = "unsupported"
	nativeScheduleSupported = false
)

func installNativeCopySchedule(string, time.Duration, bool) error { return nil }
