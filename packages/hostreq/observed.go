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

// ObserveSafeguard samples one named safeguard's unprivileged inspection
// handler. It is the focused twin of ListObservedSafeguards, for consumers that
// already know which safeguard they are reporting on and must not pay for the
// whole roster: sampling every handler to answer a question about one of them
// charges that safeguard for every other handler's probe cost.
//
// An unknown name is an error, never an empty observation. Like the list form,
// this never calls Apply.
func ObserveSafeguard(root, name string, now func() time.Time) (ObservedSafeguard, error) {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	item, err := hostruntime.ObserveSafeguardAt(root, name, now)
	if err != nil {
		return ObservedSafeguard{}, err
	}
	return ObservedSafeguard{
		Name:           item.Name,
		Capability:     item.Capability,
		CapabilityRole: item.CapabilityRole,
		Platforms:      append([]string(nil), item.Platforms...),
		SupportClass:   string(item.SupportClass),
		ExecutionState: string(item.ExecutionState),
		Notes:          append([]string(nil), item.Notes...),
		ObservedAt:     item.ObservedAt,
	}, nil
}
