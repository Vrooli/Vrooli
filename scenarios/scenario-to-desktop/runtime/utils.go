// Package bundleruntime provides the desktop bundle runtime supervisor.
//
// This file contains shared utility functions used across multiple files.
package bundleruntime

import (
	"errors"
	"strconv"
	"strings"
)

// copyStringMap creates a shallow copy of a string map.
func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// envMapToList converts a map of environment variables to KEY=VALUE format.
func envMapToList(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
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
// Uses the injected FileSystem for testability.
func (s *Supervisor) tailFile(path string, lines int) ([]byte, error) {
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(data), "\n")
	if lines >= len(parts) {
		return []byte(strings.Join(parts, "\n")), nil
	}
	return []byte(strings.Join(parts[len(parts)-lines:], "\n")), nil
}

// intersection returns elements present in both slices.
func intersection(a []string, b []string) []string {
	set := map[string]bool{}
	for _, v := range b {
		set[v] = true
	}
	var out []string
	for _, v := range a {
		if set[v] {
			out = append(out, v)
		}
	}
	return out
}

