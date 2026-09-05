//go:build !linux && !darwin && !windows

package hostinventory

import "time"

func collectPlatformLoad(*Snapshot, time.Time) bool { return false }
