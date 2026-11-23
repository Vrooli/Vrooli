package binaryfetch

import (
	"errors"
	"fmt"
)

// CompositionCapacity is the conservative staging envelope for a sequence of
// independently materialized URL archive inputs.
type CompositionCapacity struct {
	DownloadBytes int64
	ExpandedBytes int64
	PeakBytes     int64
}

// PlanCompositionCapacity accepts only URL archive steps whose compressed,
// expanded and entry limits are explicit. Other step kinds have no enforced
// per-step write ceiling in their current executors and remain unsupported.
func PlanCompositionCapacity(steps []ComposeStep) (CompositionCapacity, error) {
	var plan CompositionCapacity
	if len(steps) == 0 {
		return plan, errors.New("composed capacity requires at least one step")
	}
	for index, step := range steps {
		if err := step.Validate(); err != nil {
			return CompositionCapacity{}, fmt.Errorf("compose step %d (%s): %w", index, step.Role, err)
		}
		if step.Kind != "url" {
			return CompositionCapacity{}, fmt.Errorf("compose step %d (%s) kind %q has no enforceable capacity bound", index, step.Role, step.Kind)
		}
		if step.DownloadBytes <= 0 || step.MaxExpandedBytes <= 0 || step.MaxEntries <= 0 {
			return CompositionCapacity{}, fmt.Errorf("compose step %d (%s) requires positive download_bytes, max_expanded_bytes, and max_entries", index, step.Role)
		}
		var err error
		plan.DownloadBytes, err = addCapacity(plan.DownloadBytes, step.DownloadBytes)
		if err != nil {
			return CompositionCapacity{}, fmt.Errorf("compose download byte sum: %w", err)
		}
		plan.ExpandedBytes, err = addCapacity(plan.ExpandedBytes, step.MaxExpandedBytes)
		if err != nil {
			return CompositionCapacity{}, fmt.Errorf("compose expanded byte sum: %w", err)
		}
	}
	// Reserve the same conservative three-copy envelope used for single-source
	// archives, summed over every step, plus one bounded resolver allowance.
	total, err := addCapacity(plan.DownloadBytes, plan.ExpandedBytes)
	if err != nil {
		return CompositionCapacity{}, fmt.Errorf("compose total byte sum: %w", err)
	}
	if total > (maxInt64-(1<<20))/3 {
		return CompositionCapacity{}, errors.New("composed staging peak overflows int64")
	}
	plan.PeakBytes = 3*total + (1 << 20)
	return plan, nil
}

func addCapacity(left, right int64) (int64, error) {
	if right < 0 || left > maxInt64-right {
		return 0, errors.New("byte sum overflows int64")
	}
	return left + right, nil
}

const maxInt64 = int64(^uint64(0) >> 1)
