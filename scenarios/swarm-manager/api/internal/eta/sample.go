// Package eta holds the estimation substrate for swarm-manager: the coarse
// per-effort-class duration samples the ETA engine folds into completion
// bands. This file defines the sample vocabulary and the historical backfill
// that seeds the cold-start distribution from spec timestamps. The Monte Carlo
// rollup and confidence-labelled bands are built on top of these samples by the
// ETA engine.
package eta

import (
	"strings"
	"time"
)

// EffortClasses are the t-shirt sizes an item can carry, smallest to largest.
// The empty string denotes an unsized item, folded into the global
// distribution by the ETA engine.
var EffortClasses = []string{"XS", "S", "M", "L", "XL"}

// NormalizeEffort upper-cases and validates an effort value, returning the
// canonical class or "" for unsized / unrecognized input.
func NormalizeEffort(raw string) string {
	v := strings.ToUpper(strings.TrimSpace(raw))
	switch v {
	case "XS", "S", "M", "L", "XL":
		return v
	default:
		return ""
	}
}

// LeadTimeHours computes the wall-clock hours between two RFC3339 timestamps.
// It returns (hours, true) only for a strictly positive, parseable span;
// missing, unparseable, or non-positive spans return (0, false) so degenerate
// same-instant records contribute no signal.
func LeadTimeHours(createdRFC3339, completedRFC3339 string) (float64, bool) {
	created, err := parseTime(createdRFC3339)
	if err != nil {
		return 0, false
	}
	completed, err := parseTime(completedRFC3339)
	if err != nil {
		return 0, false
	}
	hours := completed.Sub(created).Hours()
	if hours <= 0 {
		return 0, false
	}
	return hours, true
}

func parseTime(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return t, nil
	}
	// Some historical specs carry sub-second precision or a space separator.
	return time.Parse(time.RFC3339Nano, trimmed)
}
