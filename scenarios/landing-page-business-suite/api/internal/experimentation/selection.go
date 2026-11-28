package experimentation

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

func WeightFingerprint(variants []*VariantSnapshot) string {
	h := sha256.New()
	for _, variant := range variants {
		fmt.Fprintf(h, "%s:%d;", variant.Variant.Slug, VariantWeight(variant))
	}
	return hex.EncodeToString(h.Sum(nil))
}

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
	return SelectVariantForVisitor(variants, "")
}

// SelectVariantForVisitor deterministically maps a non-empty visitor id into
// the current weighted range. Empty ids retain entropy-based assignment for
// callers that have not established an anonymous identity yet.
func SelectVariantForVisitor(variants []*VariantSnapshot, visitorID string) *VariantSnapshot {
	if len(variants) == 0 {
		return nil
	}
	totalWeight := new(big.Int)
	for _, variant := range variants {
		totalWeight.Add(totalWeight, big.NewInt(int64(VariantWeight(variant))))
	}
	if totalWeight.Sign() == 0 {
		return variants[0]
	}
	var picked *big.Int
	if strings.TrimSpace(visitorID) != "" {
		digest := sha256.Sum256([]byte(visitorID))
		bucket := new(big.Int).SetUint64(binary.BigEndian.Uint64(digest[:8]))
		if totalWeight.BitLen() > 64 {
			// Preserve existing 64-bit assignments for ordinary ranges. When the
			// range exceeds 64 bits, use the full digest so high weights remain
			// reachable instead of silently collapsing selection into the prefix.
			bucket.SetBytes(digest[:])
		}
		picked = bucket.Mod(bucket, totalWeight)
	} else {
		value, err := rand.Int(rand.Reader, totalWeight)
		if err != nil {
			return variants[0]
		}
		picked = value
	}
	cumulative := new(big.Int)
	for _, variant := range variants {
		cumulative.Add(cumulative, big.NewInt(int64(VariantWeight(variant))))
		if picked.Cmp(cumulative) < 0 {
			return variant
		}
	}
	return variants[0]
}
