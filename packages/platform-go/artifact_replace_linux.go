//go:build linux

package platform

func atomicReplace(staged, target string) error { return atomicRenameReplace(staged, target) }
