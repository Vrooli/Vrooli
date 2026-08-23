//go:build !linux

package executor

func childLeakDetectionAvailable() bool { return false }

func waitForProcessGroupExit(int) bool { return false }
