package broker

import "github.com/vrooli/vrooli/scenarios/vrooli-events/internal/match"

// Match checks if value matches the given glob pattern using segment-aware matching.
// Segments are separated by ".". "*" matches exactly one segment. "**" matches one or more segments.
// An empty pattern matches everything.
func Match(pattern, value string) bool {
	return match.Glob(pattern, value)
}
