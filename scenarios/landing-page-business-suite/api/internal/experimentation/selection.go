// Package experimentation owns the policy used to decide whether a landing
// variant-axis selection can be served. Transport and config decoding stay at
// the API boundary; this package deliberately has no HTTP or filesystem deps.
package experimentation

import (
	"fmt"
	"strings"
)

// Axis lists the selectable identifiers for one experiment dimension.
type Axis struct {
	Variants []string
}

// Space is the policy-relevant view of a landing variant space.
type Space struct {
	Axes                   map[string]Axis
	DisallowedCombinations []map[string]string
}

// ValidateSelection rejects unknown, incomplete, invalid, and explicitly
// disallowed variant selections. A complete selection is required so a served
// landing experience always has a reproducible experiment assignment.
func ValidateSelection(space Space, selection map[string]string) error {
	if len(space.Axes) == 0 {
		return fmt.Errorf("variant space has no axes defined")
	}

	for axisID := range selection {
		if _, ok := space.Axes[axisID]; !ok {
			return fmt.Errorf("unknown axis %s", axisID)
		}
	}

	for axisID, axis := range space.Axes {
		value, ok := selection[axisID]
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("axis %s is required", axisID)
		}
		if !contains(axis.Variants, value) {
			return fmt.Errorf("invalid value '%s' for axis %s", value, axisID)
		}
	}

	for _, combination := range space.DisallowedCombinations {
		if matches(selection, combination) {
			return fmt.Errorf("axis combination %v is disallowed", combination)
		}
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func matches(selection, combination map[string]string) bool {
	if len(combination) == 0 {
		return false
	}
	for axisID, expected := range combination {
		if actual, ok := selection[axisID]; !ok || actual != expected {
			return false
		}
	}
	return true
}
