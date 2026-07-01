package hygiene

import (
	"context"
	"fmt"
)

// Provider is one registered hygiene check surface. Providers append their
// checks/findings/actions into the shared report so the root hygiene command can
// aggregate scenario- or domain-specific hygiene without hardcoding every rule
// in Service.Run.
type Provider interface {
	ID() string
	Run(ctx context.Context, req Request, report *Report) error
}

// Registry owns the ordered hygiene provider set.
type Registry struct {
	order    []string
	provider map[string]Provider
}

func NewRegistry(providers ...Provider) Registry {
	r := Registry{provider: map[string]Provider{}}
	for _, p := range providers {
		r.Register(p)
	}
	return r
}

func (r *Registry) Register(p Provider) {
	if p == nil || p.ID() == "" {
		return
	}
	id := p.ID()
	if _, exists := r.provider[id]; !exists {
		r.order = append(r.order, id)
	}
	r.provider[id] = p
}

func (r Registry) Run(ctx context.Context, req Request, report *Report, ids ...string) error {
	if len(ids) == 0 {
		ids = r.order
	}
	for _, id := range ids {
		p, ok := r.provider[id]
		if !ok {
			return fmt.Errorf("hygiene provider %q is not registered", id)
		}
		if err := p.Run(ctx, req, report); err != nil {
			return err
		}
	}
	return nil
}
