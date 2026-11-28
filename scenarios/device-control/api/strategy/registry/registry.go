package registry

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"device-control/internal/environment"
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

func Default() *Registry { return DefaultWithEnvironment(environment.OSReader{}) }

// DefaultWithEnvironment resolves operator settings once and passes explicit
// values into strategy adapters. Tests can supply a MapReader without mutating
// the process environment.
func DefaultWithEnvironment(env environment.Reader) *Registry {
	if env == nil {
		env = environment.MapReader{}
	}
	return New(
		androidadb.NewWithConfig(androidadb.Config{
			Serial:               strings.TrimSpace(environment.Value(env, "ANDROID_SERIAL")),
			RequireDefaultServer: hasADBServerOverride(env),
		}),
		configuredAndroidTVRemote(env), configuredGoogleCast(env),
		homeassistant.NewWithEnvironment(env),
		hostdesktop.NewWithConfig(hostdesktop.Config{
			Display:          strings.TrimSpace(environment.Value(env, "DISPLAY")),
			XAuthority:       strings.TrimSpace(environment.Value(env, "XAUTHORITY")),
			HelperConfigPath: strings.TrimSpace(environment.Value(env, "DEVICE_CONTROL_DESKTOP_HELPER_CONFIG")),
			HostNodeID:       hostNodeID(env),
			SessionID:        strings.TrimSpace(environment.Value(env, "XDG_SESSION_ID")),
			SessionType:      strings.TrimSpace(environment.Value(env, "XDG_SESSION_TYPE")),
			WaylandDisplay:   strings.TrimSpace(environment.Value(env, "WAYLAND_DISPLAY")),
		}),
		iossimctl.NewWithUDID(strings.TrimSpace(environment.Value(env, "IOS_SIMULATOR_UDID"))),
		iosxcuitest.New(), iosmirror.New(),
	)
}

func hostNodeID(env environment.Reader) string {
	if id := strings.TrimSpace(environment.Value(env, "VROOLI_NODE_ID")); id != "" {
		return id
	}
	if id, err := os.Hostname(); err == nil && strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	return "local"
}

func hasADBServerOverride(env environment.Reader) bool {
	for _, key := range []string{"ADB_SERVER_SOCKET", "ANDROID_ADB_SERVER_ADDRESS", "ANDROID_ADB_SERVER_PORT"} {
		if strings.TrimSpace(environment.Value(env, key)) != "" {
			return true
		}
	}
	return false
}

// Static endpoints are an owner-controlled escape hatch for networks where
// multicast is filtered. They are intentionally opt-in and carry no pairing
// material; normal operation remains DNS-SD discovery.
func configuredAndroidTVRemote(env environment.Reader) strategy.Strategy {
	endpoint := strings.TrimSpace(environment.Value(env, "DEVICE_CONTROL_ANDROID_TV_REMOTE_ENDPOINT"))
	if endpoint == "" {
		return androidtvremote.New()
	}
	configuredIdentity := strings.TrimSpace(environment.Value(env, "DEVICE_CONTROL_ANDROID_TV_REMOTE_IDENTITY"))
	device := androidtvremote.Device{Serial: configuredIdentity, IdentityKey: configuredIdentity, Endpoint: endpoint, Name: strings.TrimSpace(environment.Value(env, "DEVICE_CONTROL_ANDROID_TV_REMOTE_NAME"))}
	if device.Serial == "" {
		device.Serial = "configured:" + endpoint
	}
	if claim, err := identitydomain.NewClaim(string(identitydomain.BluetoothMAC), configuredIdentity, "android-tv-remote", "owner-configured"); err == nil {
		device.IdentityKey = claim.Value
		device.IdentityKind = string(claim.Kind)
	}
	return androidtvremote.New(androidtvremote.WithDevices(device))
}

func configuredGoogleCast(env environment.Reader) strategy.Strategy {
	endpoint := strings.TrimSpace(environment.Value(env, "DEVICE_CONTROL_GOOGLE_CAST_ENDPOINT"))
	if endpoint == "" {
		return googlecast.New()
	}
	configuredIdentity := strings.TrimSpace(environment.Value(env, "DEVICE_CONTROL_GOOGLE_CAST_IDENTITY"))
	device := googlecast.Device{ID: configuredIdentity, IdentityKey: configuredIdentity, Endpoint: endpoint, Name: strings.TrimSpace(environment.Value(env, "DEVICE_CONTROL_GOOGLE_CAST_NAME"))}
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
	out := make([]strategy.Declaration, len(ids))
	// Independent transports must not consume each other's probe budget.
	// Keep the work bounded and publish results in deterministic ID order.
	const maxConcurrentProbes = 8
	limit := make(chan struct{}, maxConcurrentProbes)
	var wait sync.WaitGroup
	for index, id := range ids {
		wait.Add(1)
		go func(index int, id string) {
			defer wait.Done()
			defer func() {
				if recover() != nil {
					out[index] = strategy.UnavailableDeclaration(id, "strategy probe failed", []strategy.Capability{{Name: strategy.CapInput}, {Name: strategy.CapScreenshot}}, "strategy probe panicked")
				}
			}()
			select {
			case limit <- struct{}{}:
				defer func() { <-limit }()
			case <-ctx.Done():
				out[index] = strategy.UnavailableDeclaration(id, "strategy probe failed", []strategy.Capability{{Name: strategy.CapInput}, {Name: strategy.CapScreenshot}}, ctx.Err().Error())
				return
			}
			if err := ctx.Err(); err != nil {
				out[index] = strategy.UnavailableDeclaration(id, "strategy probe failed", []strategy.Capability{{Name: strategy.CapInput}, {Name: strategy.CapScreenshot}}, err.Error())
				return
			}
			d, err := r.items[id].Describe(ctx)
			if err != nil {
				d = strategy.UnavailableDeclaration(id, "strategy probe failed", []strategy.Capability{{Name: strategy.CapInput}, {Name: strategy.CapScreenshot}}, err.Error())
			}
			out[index] = d
		}(index, id)
	}
	wait.Wait()
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
