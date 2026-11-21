//go:build !linux && !darwin && !windows

package platform

func atomicReplace(staged, target string) error { return atomicRenameReplace(staged, target) }
