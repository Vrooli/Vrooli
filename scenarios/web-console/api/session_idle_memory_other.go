//go:build !linux

package main

// sessionAgentMemory is not measured without /proc; the archive note says so
// by omitting the figure.
func sessionAgentMemory(string) (int64, bool) { return 0, false }
