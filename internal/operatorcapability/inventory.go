package operatorcapability

import (
	"context"
	"fmt"
	"sort"
)

// InventoryEntry is the stable, secret-free projection consumed by catalog,
// settings, CLI prompts, and search. The complete descriptor remains attached
// so a consumer never needs a capability-name switch to render a control.
type InventoryEntry struct {
	Descriptor Descriptor `json:"descriptor"`
	State      State      `json:"state"`
	Reason     string     `json:"reason,omitempty"`
}

type Inventory struct {
	Version string           `json:"version"`
	Entries []InventoryEntry `json:"entries"`
}

// BuildInventory validates and sorts the provider-owned statuses. It is the
// generated inventory boundary: callers provide declarations/statuses from a
// trusted composition root and receive a deterministic read model.
func BuildInventory(statuses []Status) (Inventory, error) {
	entries := make([]InventoryEntry, 0, len(statuses))
	seen := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		if err := status.Descriptor.Validate(); err != nil {
			return Inventory{}, err
		}
		if _, exists := seen[status.Descriptor.ID]; exists {
			return Inventory{}, fmt.Errorf("capability inventory contains duplicate %q", status.Descriptor.ID)
		}
		seen[status.Descriptor.ID] = struct{}{}
		reason := status.Remediation
		if reason == "" {
			reason = status.Descriptor.DispositionReason
		}
		entries = append(entries, InventoryEntry{Descriptor: status.Descriptor, State: status.State, Reason: reason})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Descriptor.ID < entries[j].Descriptor.ID })
	return Inventory{Version: ContractVersion, Entries: entries}, nil
}

// Inventory returns the same provider-owned statuses as Discover, but in the
// stable catalog shape used by non-wizard consumers.
func (r *Registry) Inventory(ctx context.Context) (Inventory, error) {
	if r == nil {
		return Inventory{}, fmt.Errorf("capability registry is nil")
	}
	statuses, err := r.Discover(ctx)
	if err != nil {
		return Inventory{}, err
	}
	return BuildInventory(statuses)
}
