package registry

import (
	"context"
	"fmt"
	"sort"

	"device-control/strategy"
	"device-control/strategy/androidadb"
	"device-control/strategy/androidtvremote"
	"device-control/strategy/homeassistant"
	"device-control/strategy/hostdesktop"
	"device-control/strategy/iosmirror"
	"device-control/strategy/iossimctl"
	"device-control/strategy/iosxcuitest"
)

type Registry struct{ items map[string]strategy.Strategy }

type DeviceDeclaration struct {
	Device      strategy.Device      `json:"device"`
	Declaration strategy.Declaration `json:"declaration"`
}

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
	return New(androidadb.New(), androidtvremote.New(), homeassistant.NewFromEnv(), hostdesktop.New(), iossimctl.New(), iosxcuitest.New(), iosmirror.New())
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

// ListDevices is the device-oriented view. Strategies remain the transport
// diagnostic view exposed by List; this method fans out enumerators and binds
// each discovered target before describing it.
func (r *Registry) ListDevices(ctx context.Context) []DeviceDeclaration {
	ids := make([]string, 0, len(r.items))
	for id := range r.items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]DeviceDeclaration, 0)
	for _, id := range ids {
		base := r.items[id]
		enumerator, ok := base.(strategy.Enumerator)
		if !ok {
			continue
		}
		devices, err := enumerator.Enumerate(ctx)
		if err != nil {
			continue
		}
		for _, device := range devices {
			driver := base
			if scoped, ok := base.(strategy.DeviceScoped); ok && device.Serial != "" {
				driver = scoped.ForDevice(device.Serial)
			}
			declaration, err := driver.Describe(ctx)
			if err != nil {
				declaration = strategy.UnavailableDeclaration(id, "device description failed", nil, err.Error())
			}
			declaration.DeviceID = device.ID
			declaration.Transport = device.Transport
			declaration.StrategyID = id
			out = append(out, DeviceDeclaration{Device: device, Declaration: declaration})
		}
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
