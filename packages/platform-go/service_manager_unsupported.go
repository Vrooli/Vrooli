//go:build !linux && !darwin && !windows

package platform

func serviceManagerCommand() string     { return "" }
func serviceManagerCommandPath() string { return "" }
