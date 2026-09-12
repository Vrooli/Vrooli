//go:build darwin

package platform

func serviceManagerCommand() string     { return "launchctl" }
func serviceManagerCommandPath() string { return "/bin/launchctl" }
