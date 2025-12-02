// Package bundleruntime provides the desktop bundle runtime supervisor.
//
// This file re-exports utility functions from internal packages for use
// within the bundleruntime package. External code should import the
// internal packages directly if needed.
package bundleruntime

import (
	"errors"
	"strconv"
	"strings"

	"scenario-to-desktop-runtime/internal/fileutil"
	"scenario-to-desktop-runtime/internal/strutil"
)

// copyStringMap creates a shallow copy of a string map.
// Delegates to strutil.CopyStringMap.
func copyStringMap(in map[string]string) map[string]string {
	return strutil.CopyStringMap(in)
}

// envMapToList converts a map of environment variables to KEY=VALUE format.
// Delegates to strutil.EnvMapToList.
func envMapToList(env map[string]string) []string {
	return strutil.EnvMapToList(env)
}

// intersection returns elements present in both slices.
// Delegates to strutil.Intersection.
func intersection(a, b []string) []string {
	return strutil.Intersection(a, b)
}

// parsePositiveInt parses a string as a positive integer.
func parsePositiveInt(s string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	if v <= 0 {
		return 0, errors.New("must be positive")
	}
	return v, nil
}

// tailFile returns the last N lines from a file.
// Delegates to fileutil.TailFile.
func (s *Supervisor) tailFile(path string, lines int) ([]byte, error) {
	return fileutil.TailFile(s.fs, path, lines)
}
