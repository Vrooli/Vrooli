//go:build !linux && !darwin && !windows

package watchdoginstall

func CurrentBackend() string { return "unsupported" }
