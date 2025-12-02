// Package bundleruntime provides the desktop bundle runtime supervisor.
//
// This file contains shared utility functions used across multiple files.
package bundleruntime

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
