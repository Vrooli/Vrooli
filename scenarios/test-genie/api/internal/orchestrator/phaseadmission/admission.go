// Package phaseadmission acquires host capacity for a batch of phases.
//
// It was extracted from the suite orchestrator, where admission sat beside
// batching, cache identity, and orchestration in one 2,900-line file. Splitting
// it out makes the prefix rule — the reason this code exists in its current
// shape — reviewable on its own.
package phaseadmission

import (
	"context"
	"strings"

	"test-genie/internal/orchestrator/phasebatch"
	"test-genie/internal/orchestrator/phases"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

// Broker is the host-capacity admission surface this package needs.
type Broker interface {
	Acquire(context.Context, string, int64, int64) (sharedcapacity.Lease, sharedcapacity.Verdict, error)
}

// Estimator supplies a phase's measured resource cost.
type Estimator interface {
	PhaseCostEstimate(context.Context, string, string) (ramBytes, cpuMilli int64, reliable bool)
}

// Deps carries the two collaborators admission needs. They are a struct rather
// than package state so a test can supply either one without a live host.
type Deps struct {
	Broker    Broker
	Estimator Estimator
}

// Admit acquires capacity for the longest prefix of batch the host
// grants, and returns that prefix length along with the leases backing it.
//
// It admits a prefix rather than all-or-nothing because the previous behaviour
// let one phase veto every phase beside it: a broker denial on the last member
// released the grants already held for the others and dropped the whole batch
// to serial, and the caller then re-proposed the remainder, denied again, and
// walked the run one phase at a time. Sizing is no longer a failure mode here
// at all — missing estimates use the named fallback reservation — so the only
// reason a prefix is short now is that the host is genuinely full, which is a
// reason to run what fits rather than to run nothing.
//
// Every executed phase attempts a claim, including a batch of one. A solo
// phase is often solo because it is expensive or exclusive, so omitting its
// claim hides exactly the capacity pressure operators need to measure.
//
// The returned length is always at least 1, so the caller always makes
// progress. A prefix of exactly 1 carries the broker's reason so the run
// records why it serialized.
func Admit(ctx context.Context, deps Deps, scenario, runID string, batch []phases.Definition) ([]sharedcapacity.Lease, string, int, int) {
	if len(batch) == 0 || deps.Broker == nil || deps.Estimator == nil {
		return nil, "", len(batch), 0
	}
	leases := make([]sharedcapacity.Lease, 0, len(batch))
	estimated := 0
	release := func() {
		for _, acquired := range leases {
			if acquired != nil {
				_ = acquired.Release(context.Background())
			}
		}
	}
	for _, phase := range batch {
		ramBytes, cpuMilli, reliable := deps.Estimator.PhaseCostEstimate(ctx, scenario, phase.Name.Key())
		fallback := false
		var (
			lease   sharedcapacity.Lease
			verdict sharedcapacity.Verdict
			err     error
			reason  string
		)
		if !reliable || ramBytes <= 0 || cpuMilli <= 0 {
			ramBytes = phasebatch.DefaultReservationRAMBytes
			cpuMilli = phasebatch.DefaultReservationCPUMilli
			fallback = true
		}
		ownerID := sharedcapacity.OwnerIDFor("test-genie", strings.TrimSpace(runID), phase.Name.Key())
		lease, verdict, err = deps.Broker.Acquire(ctx, ownerID, ramBytes, cpuMilli)
		switch {
		case err != nil:
			reason = err.Error()
		case verdict.Kind != "grant" && verdict.Kind != "degrade":
			reason = verdict.Reason
			if reason == "" {
				reason = verdict.Kind
			}
		}
		if reason != "" {
			if len(leases) >= 2 {
				// A partial batch still beats a serial walk. The phases past
				// the stopping point are re-proposed on the next iteration.
				return leases, reason, len(leases), estimated
			}
			release()
			return nil, reason, 1, 0
		}
		leases = append(leases, lease)
		if fallback {
			estimated++
		}
	}
	return leases, "", len(leases), estimated
}
