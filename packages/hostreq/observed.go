// Package hostreq exposes the control plane's read-only host observation
// boundary to scenario consumers. It deliberately contains no apply path.
package hostreq

import (
	"time"

	hostruntime "github.com/vrooli/vrooli/internal/runtime"
)

// ObservedSafeguard is a typed snapshot of one control-plane safeguard. The
// execution state is observed only; consumers must not interpret it as proof
// that a remediation was applied by this package.
type ObservedSafeguard struct {
	Name           string
	Capability     string
	CapabilityRole string
	Platforms      []string
	SupportClass   string
	ExecutionState string
	Notes          []string
	ObservedAt     time.Time
}

// ListObservedSafeguards reads every safeguard manifest and samples its
// unprivileged inspection handler through the control plane. A missing or
// failed sample is an explicit error; consumers must not silently treat an
// unavailable control-plane read as a healthy observation.
func ListObservedSafeguards(root string, now func() time.Time) ([]ObservedSafeguard, error) {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	items, err := hostruntime.ListObservedSafeguardsAt(root, now)
	if err != nil {
		return nil, err
	}
	result := make([]ObservedSafeguard, 0, len(items))
	for _, item := range items {
		result = append(result, ObservedSafeguard{Name: item.Name, Capability: item.Capability, CapabilityRole: item.CapabilityRole, Platforms: append([]string(nil), item.Platforms...), SupportClass: string(item.SupportClass), ExecutionState: string(item.ExecutionState), Notes: append([]string(nil), item.Notes...), ObservedAt: item.ObservedAt})
	}
	return result, nil
}
