package signals

import (
	"context"
	"sort"
)

// Registry is the plug-in registry for signals. Production wiring
// instantiates one registry per process; tests construct ad-hoc ones.
type Registry struct {
	signals []Signal
}

// NewRegistry returns a registry holding the given signals in
// alphabetical order by Name() (deterministic invocation order).
func NewRegistry(in ...Signal) *Registry {
	out := append([]Signal(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return &Registry{signals: out}
}

// All returns the registered signals in deterministic order.
func (r *Registry) All() []Signal {
	out := make([]Signal, len(r.signals))
	copy(out, r.signals)
	return out
}

// Describe returns a SignalDescriptor for each registered signal,
// flagging signals that are currently unavailable.
func (r *Registry) Describe(ctx context.Context) []SignalDescriptor {
	out := make([]SignalDescriptor, 0, len(r.signals))
	for _, s := range r.signals {
		desc := SignalDescriptor{
			Name:          s.Name(),
			DefaultWeight: s.DefaultWeight(),
			Stability:     "beta",
		}
		if ok, reason := s.IsAvailable(ctx); !ok {
			desc.Disabled = true
			desc.DisabledReason = reason
		}
		out = append(out, desc)
	}
	return out
}
