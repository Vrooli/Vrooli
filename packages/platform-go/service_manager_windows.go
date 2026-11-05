//go:build windows

package platform

func serviceManagerCommand() string     { return "schtasks.exe" }
func serviceManagerCommandPath() string { return "schtasks.exe" }
