// Package budget evaluates the aggregate byte reservation declared by storage
// manifests against the capacity of the device that storage-manager governs.
// A per-owner ceiling can be locally reasonable while the collection of
// ceilings is impossible to satisfy simultaneously; this package makes that
// contradiction visible without treating declarations as live usage.
package budget

import (
	"strings"

	coreRetention "github.com/vrooli/api-core/retention"
	coreStorage "github.com/vrooli/api-core/storage"
)

const (
	// WarningFraction leaves room for durable data that is not represented by a
	// byte ceiling and for declaration overlap. A reservation above this level
	// deserves attention even when it is not yet mathematically impossible.
	WarningFraction = 0.80

	StatusCapacityUnknown = "capacity_unknown"
	StatusHealthy         = "healthy"
	StatusWarning         = "warning"
	StatusUnreasonable    = "unreasonable"
)

// Report is the machine-level view of all parsed budget.max_bytes values. Age
// budgets are intentionally excluded: they bound time, not reserved bytes.
type Report struct {
	DeclaredBytes      int64   `json:"declared_bytes"`
	CapacityBytes      int64   `json:"capacity_bytes"`
	WarningBytes       int64   `json:"warning_bytes"`
	Utilization        float64 `json:"utilization"`
	EntryCount         int     `json:"entry_count"`
	OwnerCount         int     `json:"owner_count"`
	Status             string  `json:"status"`
	OverCapacity       bool    `json:"over_capacity"`
	UnreasonableReason string  `json:"unreasonable_reason,omitempty"`
}

// Aggregate parses every storage entry with a byte ceiling. Invalid values are
// omitted from the arithmetic because the inventory/retention validators own
// their parse finding; silently treating them as zero would make the aggregate
// look safer than the declarations are.
func Aggregate(inventory coreStorage.OwnerInventory, capacityBytes int64) Report {
	var out Report
	owners := make(map[string]struct{})
	for _, owner := range inventory.Owners {
		for _, entry := range owner.StorageEntries {
			if entry.Budget == nil || strings.TrimSpace(entry.Budget.MaxBytes) == "" {
				continue
			}
			max, err := coreRetention.ParseBytes(entry.Budget.MaxBytes)
			if err != nil || max < 0 {
				continue
			}
			out.DeclaredBytes += max
			out.EntryCount++
			owners[string(owner.Kind)+"/"+owner.ID] = struct{}{}
		}
	}
	out.OwnerCount = len(owners)
	out.CapacityBytes = capacityBytes
	if capacityBytes <= 0 {
		out.Status = StatusCapacityUnknown
		return out
	}
	out.WarningBytes = int64(float64(capacityBytes) * WarningFraction)
	out.Utilization = float64(out.DeclaredBytes) / float64(capacityBytes)
	switch {
	case out.DeclaredBytes > capacityBytes:
		out.Status = StatusUnreasonable
		out.OverCapacity = true
		out.UnreasonableReason = "aggregate byte ceilings exceed device capacity; the declarations cannot all be satisfied simultaneously"
	case out.DeclaredBytes > out.WarningBytes:
		out.Status = StatusWarning
		out.UnreasonableReason = "aggregate byte ceilings consume more than 80% of device capacity"
	default:
		out.Status = StatusHealthy
	}
	return out
}
