//go:build darwin

package watchdoginstall

func CurrentBackend() string { return "launchd-user-agent" }
