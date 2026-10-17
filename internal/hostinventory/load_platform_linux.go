//go:build linux

package hostinventory

import "time"

func collectPlatformLoad(*Snapshot, time.Time) bool { return false }
