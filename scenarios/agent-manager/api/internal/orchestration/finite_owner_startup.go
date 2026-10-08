// finite_owner_startup.go publishes the actual profile owner before live finite installation.
package orchestration

import (
	"context"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"time"
)

type FiniteOwnerStartup struct {
	config      effortauthority.Installation
	engine      *effortauthority.BrokerClient
	manifest    isolation.Manifest
	witness     *isolation.Witness
	unitOwner   *isolation.FixedUnitOwner
	profiles    *effortauthority.ProfileContractPlan
	publication *effortauthority.ProtectedPublicationPlan
	grants      map[string]map[string]effortauthority.ProfileContractGrant
}

func PrepareFiniteOwnerStartup(ctx context.Context, cfg effortauthority.Installation, engine *effortauthority.BrokerClient, manifest isolation.Manifest, witness *isolation.Witness, unitOwner *isolation.FixedUnitOwner, grants map[string]map[string]effortauthority.ProfileContractGrant, publication *effortauthority.ProtectedPublicationPlan) (*FiniteOwnerStartup, error) {
	descriptor := cfg
	descriptor.Enabled = true
	frozen, e := descriptor.Freeze()
	if e != nil {
		return nil, e
	}
	frozen.Enabled = cfg.Enabled
	rawManifest, e := json.Marshal(manifest)
	if e != nil {
		return nil, effortauthority.ErrRefused
	}
	var frozenManifest isolation.Manifest
	if json.Unmarshal(rawManifest, &frozenManifest) != nil {
		return nil, effortauthority.ErrRefused
	}
	manifest = frozenManifest
	plan, e := effortauthority.PrepareProfileContract(grants, frozen.Policy)
	if e != nil || publication == nil || publication.Role() != "am" || publication.CheckInstallation(descriptor) != nil {
		return nil, effortauthority.ErrRefused
	}
	if frozen.Enabled {
		if engine == nil || witness == nil || unitOwner == nil || witness.CheckPolicy(frozen.Policy) != nil || witness.ValidateManifest(manifest) != nil || unitOwner.CheckBinding(manifest, witness) != nil || witness.Qualify(ctx) != nil {
			return nil, effortauthority.ErrRefused
		}
	} else if engine != nil || witness != nil || unitOwner != nil || manifest.Enabled {
		return nil, effortauthority.ErrRefused
	}
	// ProfileContractPlan freezes grant maps; freeze a second owner copy for the
	// live Prepare call so caller mutations cannot change later installation.
	raw, e := json.Marshal(grants)
	if e != nil {
		return nil, effortauthority.ErrRefused
	}
	var copy map[string]map[string]effortauthority.ProfileContractGrant
	if json.Unmarshal(raw, &copy) != nil {
		return nil, effortauthority.ErrRefused
	}
	return &FiniteOwnerStartup{config: frozen, engine: engine, manifest: manifest, witness: witness, unitOwner: unitOwner, profiles: plan, publication: publication, grants: copy}, nil
}
func (s *FiniteOwnerStartup) Live() bool { return s != nil && s.config.Enabled }
func (s *FiniteOwnerStartup) PublishReadPhase(ctx context.Context, o *Orchestrator) (*effortauthority.ProtectedPublication, error) {
	if s == nil || o == nil {
		return nil, effortauthority.ErrRefused
	}
	handler, e := s.profiles.Handler(ctx, o)
	if e != nil {
		return nil, e
	}
	return s.publication.Start(handler)
}
func (s *FiniteOwnerStartup) Install(ctx context.Context, o *Orchestrator) error {
	if s == nil || !s.Live() {
		return effortauthority.ErrRefused
	}
	prepared, e := PrepareFiniteInstallationWithUnitOwner(ctx, s.config, s.engine, s.manifest, s.witness, s.unitOwner, s.grants)
	if e != nil {
		return e
	}
	return prepared.Apply(o)
}

// CheckStartup rechecks the original owner window before graph construction;
// it never renews a deadline or treats a readiness boolean as authority.
func (s *FiniteOwnerStartup) CheckStartup() error {
	if s == nil {
		return effortauthority.ErrRefused
	}
	now := time.Now()
	if !now.Before(s.config.Policy.Deadline) || (s.config.Enabled && now.Before(s.config.Policy.NotBefore)) {
		return effortauthority.ErrRefused
	}
	return nil
}
