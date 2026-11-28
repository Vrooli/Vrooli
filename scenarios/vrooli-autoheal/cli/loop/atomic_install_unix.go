//go:build !windows

package main

import "os"

func atomicReplace(source, target string) error {
	return os.Rename(source, target)
}
