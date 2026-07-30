package experimentation

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"
)

// Axis is the assignable vocabulary for one experiment dimension.
type Axis struct {
	Variants []string
}

// Space is the normalized experiment-assignment policy used by persisted
// variant configuration and request validation.
type Space struct {
	Axes                   map[string]Axis
	DisallowedCombinations []map[string]string
}

// ValidateSelection rejects unknown, incomplete, invalid, and explicitly
// disallowed experiment assignments.
func ValidateSelection(space Space, selection map[string]string) error {
	for axis := range selection {
		if _, ok := space.Axes[axis]; !ok {
			return fmt.Errorf("unknown axis %s", axis)
		}
	}
	axisIDs := make([]string, 0, len(space.Axes))
	for axis := range space.Axes {
		axisIDs = append(axisIDs, axis)
	}
	sort.Strings(axisIDs)
	for _, axis := range axisIDs {
		value, ok := selection[axis]
		if !ok || value == "" {
			return fmt.Errorf("axis %s is required", axis)
		}
		valid := false
		for _, candidate := range space.Axes[axis].Variants {
			if candidate == value {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value '%s' for axis %s", value, axis)
		}
	}
	for _, combination := range space.DisallowedCombinations {
		matches := true
		for axis, value := range combination {
			if selection[axis] != value {
				matches = false
				break
			}
		}
		if matches {
			return fmt.Errorf("selection is disallowed by experiment policy")
		}
	}
	return nil
}

// NormalizeVariantStatus maps legacy and missing persisted values to the only
// statuses understood by the experiment domain.
func NormalizeVariantStatus(status string) string {
	if status == "archived" {
		return "archived"
	}
	return "active"
}

// VariantWeight returns zero for archived or disabled variants, so callers
// cannot accidentally select an inactive variant.
func VariantWeight(snapshot *VariantSnapshot) int {
	if snapshot == nil || NormalizeVariantStatus(snapshot.Variant.Status) != "active" || snapshot.Variant.Weight <= 0 {
		return 0
	}
	return snapshot.Variant.Weight
}

// SelectWeightedRandomVariant chooses from active weighted variants. It keeps
// the historic deterministic fallback for zero-weight and entropy failures so
// landing configuration remains available when selection cannot be randomized.
func SelectWeightedRandomVariant(variants []*VariantSnapshot) *VariantSnapshot {
	if len(variants) == 0 {
		return nil
	}
	totalWeight := 0
	for _, variant := range variants {
		totalWeight += VariantWeight(variant)
	}
	if totalWeight == 0 {
		return variants[0]
	}
	picked, err := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	if err != nil {
		return variants[0]
	}
	cumulative := 0
	for _, variant := range variants {
		cumulative += VariantWeight(variant)
		if picked.Int64() < int64(cumulative) {
			return variant
		}
	}
	return variants[0]
}
