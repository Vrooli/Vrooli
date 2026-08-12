package registry

import (
	"context"
	"fmt"
	"sort"

	"device-control/strategy"
	"device-control/strategy/androidadb"
	"device-control/strategy/hostdesktop"
	"device-control/strategy/iosmirror"
	"device-control/strategy/iossimctl"
	"device-control/strategy/iosxcuitest"
)

type Registry struct{ items map[string]strategy.Strategy }

func New(items ...strategy.Strategy) *Registry {
	r := &Registry{items: map[string]strategy.Strategy{}}
	for _, item := range items {
		if item != nil {
			r.items[item.ID()] = item
		}
	}
	return r
}

func Default() *Registry {
	return New(androidadb.New(), hostdesktop.New(), iossimctl.New(), iosxcuitest.New(), iosmirror.New())
}
func (r *Registry) Get(id string) (strategy.Strategy, bool) { s, ok := r.items[id]; return s, ok }
func (r *Registry) List(ctx context.Context) []strategy.Declaration {
	ids := make([]string, 0, len(r.items))
	for id := range r.items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]strategy.Declaration, 0, len(ids))
	for _, id := range ids {
		d, err := r.items[id].Describe(ctx)
		if err != nil {
			d = strategy.UnavailableDeclaration(id, "strategy probe failed", []strategy.Capability{{Name: strategy.CapInput}, {Name: strategy.CapScreenshot}}, err.Error())
		}
		out = append(out, d)
	}
	return out
}

func (r *Registry) Verify(ctx context.Context, id string) (strategy.ConformanceReport, error) {
	s, ok := r.Get(id)
	if !ok {
		return strategy.ConformanceReport{}, fmt.Errorf("unknown strategy %q", id)
	}
	return strategy.Verify(ctx, s), nil
}
