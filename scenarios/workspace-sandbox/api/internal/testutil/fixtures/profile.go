package fixtures

import (
	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/types"
)

// ProfileOpt mutates an IsolationProfile during construction.
type ProfileOpt func(*config.IsolationProfile)

// NewIsolationProfile returns a builtin-style profile with full
// network access and no binds. Tests override with options for
// network=none, $HOME-required profiles, etc.
func NewIsolationProfile(opts ...ProfileOpt) config.IsolationProfile {
	p := config.IsolationProfile{
		ID:                     "test-profile",
		Name:                   "Test Profile",
		Description:            "Default test profile",
		Builtin:                false,
		NetworkAccess:          "full",
		HomeOverlayRequirement: types.HomeOverlayNotNeeded,
		ReadOnlyBinds:          map[string]string{},
		ReadWriteBinds:         map[string]string{},
		Environment:            map[string]string{},
	}
	for _, opt := range opts {
		opt(&p)
	}
	return p
}

func WithProfileID(id string) ProfileOpt {
	return func(p *config.IsolationProfile) { p.ID = id }
}

func WithProfileName(name string) ProfileOpt {
	return func(p *config.IsolationProfile) { p.Name = name }
}

func WithProfileBuiltin(b bool) ProfileOpt {
	return func(p *config.IsolationProfile) { p.Builtin = b }
}

func WithProfileNetworkAccess(net string) ProfileOpt {
	return func(p *config.IsolationProfile) { p.NetworkAccess = net }
}

func WithProfileEnv(k, v string) ProfileOpt {
	return func(p *config.IsolationProfile) {
		if p.Environment == nil {
			p.Environment = map[string]string{}
		}
		p.Environment[k] = v
	}
}

func WithProfileReadOnlyBind(host, sandbox string) ProfileOpt {
	return func(p *config.IsolationProfile) {
		if p.ReadOnlyBinds == nil {
			p.ReadOnlyBinds = map[string]string{}
		}
		p.ReadOnlyBinds[host] = sandbox
	}
}

func WithProfileReadWriteBind(host, sandbox string) ProfileOpt {
	return func(p *config.IsolationProfile) {
		if p.ReadWriteBinds == nil {
			p.ReadWriteBinds = map[string]string{}
		}
		p.ReadWriteBinds[host] = sandbox
	}
}

func WithProfileHomeOverlayRequirement(r types.HomeOverlayRequirement) ProfileOpt {
	return func(p *config.IsolationProfile) { p.HomeOverlayRequirement = r }
}
