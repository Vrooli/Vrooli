// finite_installation.go validates and publishes the immutable finite-owner startup composition.
package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"context"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"sync"
)

type PreparedFiniteInstallation struct {
	config    effortauthority.Installation
	authority effortauthority.Engine
	factory   *runner.FiniteNativeFactory
	profiles  *effortauthority.ProfileContractPlan
	handler   *effortauthority.ProfileContractHandler
	mu        sync.Mutex
}

func PrepareFiniteInstallation(ctx context.Context, cfg effortauthority.Installation, authority effortauthority.Engine, manifest isolation.Manifest, witness *isolation.Witness, profiles map[string]map[string]effortauthority.ProfileContractGrant) (*PreparedFiniteInstallation, error) {
	return PrepareFiniteInstallationWithUnitOwner(ctx, cfg, authority, manifest, witness, nil, profiles)
}
func PrepareFiniteInstallationWithUnitOwner(ctx context.Context, cfg effortauthority.Installation, authority effortauthority.Engine, manifest isolation.Manifest, witness *isolation.Witness, unitOwner *isolation.FixedUnitOwner, profiles map[string]map[string]effortauthority.ProfileContractGrant) (*PreparedFiniteInstallation, error) {
	frozen, e := cfg.Freeze()
	if e != nil {
		return nil, e
	}
	if !frozen.Enabled {
		return &PreparedFiniteInstallation{config: frozen}, nil
	}
	if unitOwner == nil || authority == nil || witness == nil || len(profiles) == 0 || witness.Qualify(ctx) != nil || witness.ValidateManifest(manifest) != nil {
		return nil, effortauthority.ErrRefused
	}
	profilePlan, e := effortauthority.PrepareProfileContract(profiles, frozen.Policy)
	if e != nil {
		return nil, e
	}
	factory, e := runner.NewFiniteNativeFactoryWithOwners(manifest, witness, unitOwner)
	if e != nil {
		return nil, e
	}
	if len(manifest.Bindings) != 1 || len(frozen.Policy.Profiles) != 1 {
		return nil, effortauthority.ErrRefused
	}
	b := manifest.Bindings[frozen.Policy.ID]
	for _, digest := range frozen.Policy.Profiles {
		if digest != b.ProfileDigest {
			return nil, effortauthority.ErrRefused
		}
	}
	current, e := authority.CheckBinding(ctx, effortauthority.Binding{PolicyID: frozen.Policy.ID, PolicyDigest: effortauthority.Digest(frozen.Policy), Epoch: frozen.Policy.Epoch, Owner: frozen.Policy.Owner, Client: frozen.Policy.Client, Deadline: frozen.Policy.Deadline})
	if e != nil || effortauthority.Digest(current) != effortauthority.Digest(frozen.Policy) {
		return nil, effortauthority.ErrRefused
	}
	if b.PolicyDigest != effortauthority.Digest(frozen.Policy) || b.Repository != frozen.Policy.Repository || !b.Deadline.Equal(frozen.Policy.Deadline) {
		return nil, effortauthority.ErrRefused
	}
	return &PreparedFiniteInstallation{config: frozen, authority: authority, factory: factory, profiles: profilePlan}, nil
}
func (p *PreparedFiniteInstallation) Apply(o *Orchestrator) error {
	if p == nil || o == nil {
		return effortauthority.ErrRefused
	}
	if !p.config.Enabled {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if o.effortAuthority != nil || o.finiteNativeFactory != nil || p.handler != nil {
		return effortauthority.ErrRefused
	}
	handler, e := p.profiles.Handler(context.Background(), o)
	if e != nil {
		return e
	}
	if e := o.InstallFiniteNativeIsolation(p.factory); e != nil {
		return e
	}
	o.effortAuthority = p.authority
	p.handler = handler
	return nil
}
func (p *PreparedFiniteInstallation) ProfileHandler() *effortauthority.ProfileContractHandler {
	if p == nil || !p.config.Enabled {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.handler
}

// PrepareFiniteProfileReadContract is the read-only startup phase. It may be
// composed before policy consent/run activation, against the actual newly built
// native profile owner. It mounts no route and sets no factory or run authority.
func PrepareFiniteProfileReadContract(ctx context.Context, p effortauthority.Policy, o *Orchestrator, peers map[string]map[string]effortauthority.ProfileContractGrant) (*effortauthority.ProfileContractHandler, error) {
	if o == nil {
		return nil, effortauthority.ErrRefused
	}
	plan, e := effortauthority.PrepareProfileContract(peers, p)
	if e != nil {
		return nil, e
	}
	return plan.Handler(ctx, o)
}
