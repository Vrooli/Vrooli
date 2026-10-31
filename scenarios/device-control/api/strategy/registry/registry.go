package registry

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	identitydomain "device-control/internal/identity"
	"device-control/strategy"
	"device-control/strategy/androidadb"
	"device-control/strategy/androidtvremote"
	"device-control/strategy/googlecast"
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
	return New(androidadb.New(), configuredAndroidTVRemote(), configuredGoogleCast(), homeassistant.NewFromEnv(), hostdesktop.New(), iossimctl.New(), iosxcuitest.New(), iosmirror.New())
}

// Static endpoints are an owner-controlled escape hatch for networks where
// multicast is filtered. They are intentionally opt-in and carry no pairing
// material; normal operation remains DNS-SD discovery.
func configuredAndroidTVRemote() strategy.Strategy {
	endpoint := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_ANDROID_TV_REMOTE_ENDPOINT"))
	if endpoint == "" {
		return androidtvremote.New()
	}
	configuredIdentity := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_ANDROID_TV_REMOTE_IDENTITY"))
	device := androidtvremote.Device{Serial: configuredIdentity, IdentityKey: configuredIdentity, Endpoint: endpoint, Name: strings.TrimSpace(os.Getenv("DEVICE_CONTROL_ANDROID_TV_REMOTE_NAME"))}
	if device.Serial == "" {
		device.Serial = "configured:" + endpoint
	}
	if claim, err := identitydomain.NewClaim(string(identitydomain.BluetoothMAC), configuredIdentity, "android-tv-remote", "owner-configured"); err == nil {
		device.IdentityKey = claim.Value
		device.IdentityKind = string(claim.Kind)
	}
	return androidtvremote.New(androidtvremote.WithDevices(device))
}

func configuredGoogleCast() strategy.Strategy {
	endpoint := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_GOOGLE_CAST_ENDPOINT"))
	if endpoint == "" {
		return googlecast.New()
	}
	configuredIdentity := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_GOOGLE_CAST_IDENTITY"))
	device := googlecast.Device{ID: configuredIdentity, IdentityKey: configuredIdentity, Endpoint: endpoint, Name: strings.TrimSpace(os.Getenv("DEVICE_CONTROL_GOOGLE_CAST_NAME"))}
	if claim, err := identitydomain.NewClaim(string(identitydomain.CastID), configuredIdentity, "google-cast", "owner-configured"); err == nil {
		device.IdentityKey = claim.Value
		device.IdentityKind = string(claim.Kind)
	}
	if device.ID == "" {
		device.ID = endpoint
	}
	return googlecast.New(googlecast.WithDevices(device))
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
			if endpointScoped, ok := driver.(interface {
				ForEndpoint(string) strategy.Strategy
			}); ok && device.Endpoint != "" {
				driver = endpointScoped.ForEndpoint(device.Endpoint)
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
