// Package phasebatch decides which phases may run together.
//
// It was extracted from the suite orchestrator, where batching sat beside
// admission, cache identity, and orchestration in one 2,900-line file. The
// concern is worth its own package because it is PURE: given a phase list and a
// policy it returns a boundary, with no host, database, or broker involved. That
// makes every scheduling rule here testable as a table rather than through a
// live run — which is how a batching rule earns trust.
package phasebatch

import (
	"strings"

	"test-genie/internal/orchestrator/phases"
)

// Policy carries the per-run predicates the batcher consults. They
// are injected rather than looked up inline so the batcher stays a pure
// function of the phase list, and so a run resolves each phase's duration
// history once instead of once per batch proposal.
type Policy struct {
	ForceSerial bool
	// TimeoutRisk reports whether the phase runs close enough to its own
	// deadline that contention could turn a pass into a timeout.
	TimeoutRisk func(phases.Definition) bool
	// AdmissionEnabled distinguishes an unavailable broker from a phase whose
	// individual measurement is missing. The latter uses a conservative
	// reservation and remains eligible for a batch.
	AdmissionEnabled bool
}

// CanBatch reports whether a phase may share a batch.
func (p Policy) CanBatch(def phases.Definition) bool {
	if !p.AdmissionEnabled {
		return false
	}
	return p.TimeoutRisk == nil || !p.TimeoutRisk(def)
}

// Next returns the exclusive end of the next contiguous batch.
//
// It excludes two kinds of phase from a batch. A phase whose provider
// declares `exclusive` never shares a batch. A phase whose duration sits close
// to its own timeout is kept alone, because a phase that already consumes most
// of its budget has no contention headroom and concurrency could turn a passing
// run into a timeout with no source change. A phase with no resource estimate is
// not excluded: admitPhaseBatch uses the documented fallback reservation.
func Next(defs []phases.Definition, start int, policy Policy) int {
	if start >= len(defs) {
		return start
	}
	if policy.ForceSerial || ConcurrencyMode(defs[start]) == "exclusive" {
		return start + 1
	}
	// Collect deferred phases in one pass. Repeatedly moving a deferred phase
	// and retrying the same index can cycle forever when all remaining phases
	// are non-batchable. Reorder once after the scan so the scheduler always
	// consumes work.
	deferred := make([]phases.Definition, 0)
	providers := map[string]struct{}{}
	batch := make([]phases.Definition, 0, len(defs)-start)
	boundary := len(defs)
	for end := start; end < len(defs); {
		def := defs[end]
		mode := ConcurrencyMode(def)
		if mode == "exclusive" {
			boundary = end
			break
		}
		if !policy.CanBatch(def) {
			deferred = append(deferred, def)
			end++
			continue
		}
		if mode == "provider-serial" {
			provider := strings.TrimSpace(def.ProviderScenario)
			if provider == "" {
				provider = def.Name.Key()
			}
			if _, exists := providers[provider]; exists {
				boundary = end
				break
			}
			providers[provider] = struct{}{}
		}
		batch = append(batch, def)
		end++
	}
	if len(deferred) > 0 {
		reordered := append(append([]phases.Definition(nil), batch...), deferred...)
		copy(defs[start:boundary], reordered)
		if len(batch) == 0 {
			return start + 1
		}
		return start + len(batch)
	}
	if boundary < len(defs) {
		return boundary
	}
	return start + len(batch)
}

// ContentionAllowance is how much slower a phase is assumed to run when it
// shares the host with its batch. A phase is kept out of a batch when its
// measured duration times this allowance would reach its timeout.
//
// The original guard used 2x because it compared an upward-biased planner
// prediction against half the timeout. The scheduler now uses observed p90
// wall-clock, and the post-change full-suite evidence showed that 2x still
// serialized a roughly 100 s security phase against its 180 s timeout despite
// the capacity broker granting the batch. 1.5x preserves a meaningful timeout
// margin while allowing measured p90 phases with genuine headroom to overlap.
// Revisit if a fresh 200-run window records timeout escapes under contention or
// remains below the 2.5x parallelism target.
const ContentionAllowance = 1.5

// These reservations approximate the fleet p90 of observed phase resource
// claims. They keep an unmeasured phase eligible for a safe batch without
// pretending its cost is known. Revisit when durable fallback admissions
// exceed 10% for two consecutive weeks or the host capacity profile changes.
const (
	DefaultReservationRAMBytes int64 = 512 * 1024 * 1024
	DefaultReservationCPUMilli int64 = 500
)

// TimeoutRisk reports whether concurrency could push the phase past its
// own deadline. It prefers measured history and falls back to the planner's
// prediction when a phase has none, so a phase nobody has run yet is still
// guarded — conservatively, since the fallback input is the biased one.
func TimeoutRisk(def phases.Definition, predicted map[string]int64, measured func(phases.Definition) (int64, bool)) bool {
	timeout := def.Timeout
	if timeout <= 0 {
		timeout = phases.DefaultTimeout
	}
	if measured != nil {
		if observed, ok := measured(def); ok && observed > 0 {
			return float64(observed)*ContentionAllowance >= float64(timeout.Milliseconds())
		}
	}
	if len(predicted) == 0 {
		return false
	}
	prediction := predicted[strings.ToLower(strings.TrimSpace(def.Name.String()))]
	if prediction <= 0 {
		return false
	}
	return float64(prediction)*ContentionAllowance >= float64(timeout.Milliseconds())
}

// ConcurrencyMode normalizes a phase's declared concurrency mode. Anything
// unrecognized is treated as exclusive, so an unclear declaration costs
// parallelism rather than correctness.
func ConcurrencyMode(def phases.Definition) string {
	mode := strings.ToLower(strings.TrimSpace(def.Concurrency.Mode))
	if mode == "parallel-safe" || mode == "provider-serial" {
		return mode
	}
	return "exclusive"
}
