//go:build !windows

package main

import "os"

func currentEUID() int {
	return os.Geteuid()
}
