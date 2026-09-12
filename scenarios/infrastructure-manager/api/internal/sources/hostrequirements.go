package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/packages/hostreq"
)

// HostRequirementsReader reads the control plane's unprivileged safeguard
// observations. It never calls an apply path; the scenario only reports the
// state the control plane observed.
type HostRequirementsReader struct {
	Root string
	Now  func() time.Time
}

func (r HostRequirementsReader) Read(ctx context.Context) ([]Observation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := r.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	items, err := hostreq.ListObservedSafeguards(r.Root, now)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("control-plane host requirement read returned no safeguards")
	}
	satisfied, observedAt := 0, now().UTC()
	for _, item := range items {
		if item.ObservedAt.After(observedAt) {
			observedAt = item.ObservedAt
		}
		if item.ExecutionState == "already_present" || item.ExecutionState == "applied" || item.ExecutionState == "installed" || item.ExecutionState == "not_applicable" {
			satisfied++
		}
	}
	value := float64(satisfied) / float64(len(items)) * 100
	return []Observation{{
		ID: "commissioning-CM2", CellRef: "commissioning/CM2", Value: value, Unit: "percent",
		Source: "control-plane/host-requirements", ObservedAt: observedAt,
		TrustHints: TrustHints{UnitMatches: true},
	}}, nil
}
