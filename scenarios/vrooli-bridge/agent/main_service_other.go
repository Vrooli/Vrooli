//go:build !windows

package main

func runWindowsService([]string) (bool, error) { return false, nil }
