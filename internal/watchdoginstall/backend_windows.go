//go:build windows

package watchdoginstall

func CurrentBackend() string { return "windows-task-scheduler" }
