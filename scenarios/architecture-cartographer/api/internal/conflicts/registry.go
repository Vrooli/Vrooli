package conflicts

import (
	"context"
	"sort"
)

// Registry is the plug-in registry for detectors. Production wires
// every day-one detector here; tests construct ad-hoc registries.
type Registry struct {
	detectors []Detector
	profiles  []SurfaceProfile
}

// NewRegistry returns a registry holding the given detectors in
// alphabetical order by Name() (deterministic invocation order — see
// REQ-P0-003).
func NewRegistry(in ...Detector) *Registry {
	return NewRegistryWithProfiles(nil, in...)
}

// NewRegistryWithProfiles returns a registry that only invokes detectors
// selected by the active surface/language profile. A nil profile slice keeps
// the legacy flat behavior used by narrow unit tests.
func NewRegistryWithProfiles(profiles []SurfaceProfile, in ...Detector) *Registry {
	out := append([]Detector(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return &Registry{detectors: out, profiles: append([]SurfaceProfile(nil), profiles...)}
}

// All returns the registered detectors in deterministic order.
func (r *Registry) All() []Detector {
	out := make([]Detector, len(r.detectors))
	copy(out, r.detectors)
	return out
}

// DetectAll runs every detector and concatenates their results in
// detector order (then in each detector's emission order).
func (r *Registry) DetectAll(ctx context.Context, in DetectInput) ([]Conflict, error) {
	var out []Conflict
	for _, d := range r.activeDetectors(in) {
		conflicts, err := d.Detect(ctx, in)
		if err != nil {
			return nil, err
		}
		out = append(out, conflicts...)
	}
	return out, nil
}

func (r *Registry) activeDetectors(in DetectInput) []Detector {
	if len(r.profiles) == 0 {
		return r.detectors
	}
	allowed := detectorsForProfiles(in, r.profiles)
	out := make([]Detector, 0, len(r.detectors))
	for _, d := range r.detectors {
		if _, ok := allowed[d.Name()]; ok {
			out = append(out, d)
		}
	}
	return out
}

// Describe returns one descriptor per registered detector.
func (r *Registry) Describe() []DetectorDescriptor {
	out := make([]DetectorDescriptor, 0, len(r.detectors))
	for _, d := range r.detectors {
		out = append(out, DetectorDescriptor{
			Name:        d.Name(),
			Description: d.Description(),
			Stability:   "beta",
			EmitsTypes:  d.EmitsTypes(),
		})
	}
	return out
}
