//go:build linux

package watchdoginstall

func CurrentBackend() string { return "systemd-user-timer" }
