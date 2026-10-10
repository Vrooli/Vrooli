// unit_owner_setup.go owns fixed nonmodel bootstrap and witness setup capabilities.
package nativeisolation

import (
	"context"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"path/filepath"
	"time"
)

// NewFixedBootstrapUnitOwner is separate pre-installation authority: an exact
// owner-approved first spec plus existing authenticated peer pins. It has no
// finite engine/policy/native capability and cannot start a continuing probe.
func NewFixedBootstrapUnitOwner(spec ProbeSpec, peers map[string]UnitPeerRole) (*FixedUnitOwner, error) {
	s, e := cloneSpec(spec)
	if e != nil || s.Purpose != FirstQualification || s.validate(context.Background(), time.Now().UTC()) != nil || len(peers) == 0 {
		return nil, ErrRefused
	}
	frozen := map[string]UnitPeerRole{}
	for pin, role := range peers {
		if !validHash(pin) || role != BootstrapUnitPeer {
			return nil, ErrRefused
		}
		frozen[pin] = role
	}
	o := &FixedUnitOwner{manifest: s.Manifest, first: s, peers: frozen, driver: &systemdDriver{manifest: s.Manifest}, clock: func() time.Time { return time.Now().UTC() }, units: map[string]*ownedUnit{}, bootstrapOnly: true, setupPurpose: FirstQualification}
	o.probePreflight = o.checkProbeHost
	p, e := s.BootstrapPlan(context.Background(), o.clock())
	if e != nil {
		return nil, ErrRefused
	}
	o.units[digest(struct{ Bootstrap, Plan string }{"first-only:v1", p.Digest})] = &ownedUnit{plan: p, probe: &s}
	return o, nil
}
func (o *FixedUnitOwner) checkProbeHost(ctx context.Context, s ProbeSpec) error {
	if s.Purpose == FirstQualification && s.CheckObservationInstallation(ctx) != nil {
		return ErrRefused
	}
	if checkExecutable(s.Manifest.SystemdRun, s.Manifest.NativeUID) != nil || checkExecutable(s.Manifest.Systemctl, s.Manifest.NativeUID) != nil || trustedTree(s.Manifest.RootFS, s.Manifest.NativeUID) != nil {
		return ErrRefused
	}
	image, e := treeDigest(s.Manifest.RootFS)
	if e != nil || image != s.Manifest.RootFSDigest {
		return ErrRefused
	}
	if checkExecutable(Executable{Path: filepath.Join(s.Manifest.RootFS, ProbeHelperPath), SHA256: s.HelperSHA256}, s.Manifest.NativeUID) != nil {
		return ErrRefused
	}
	return nil
}

// NewFixedWitnessUnitOwner is nonmodel setup authority for one exact frozen
// continuing spec/proposal. It starts no policy or model, consults no finite
// ledger and never treats helper evidence as human consent or approval.
func NewFixedWitnessUnitOwner(proposal effortauthority.Policy, spec ProbeSpec, peers map[string]UnitPeerRole) (*FixedUnitOwner, error) {
	raw, e := json.Marshal(proposal)
	if e != nil {
		return nil, ErrRefused
	}
	var p effortauthority.Policy
	if json.Unmarshal(raw, &p) != nil || p.Validate() != nil || p.Deadline.Sub(p.NotBefore) > 8*time.Hour {
		return nil, ErrRefused
	}
	s, e := cloneSpec(spec)
	if e != nil || s.Purpose != ContinuingWitness || s.validate(context.Background(), time.Now().UTC()) != nil || !s.Deadline.Equal(p.Deadline) || len(peers) == 0 {
		return nil, ErrRefused
	}
	if (&Witness{spec: s, reader: newWitnessReader()}).CheckPolicy(p) != nil {
		return nil, ErrRefused
	}
	frozen := map[string]UnitPeerRole{}
	for pin, role := range peers {
		if !validHash(pin) || role != WitnessUnitPeer {
			return nil, ErrRefused
		}
		frozen[pin] = role
	}
	o := &FixedUnitOwner{policy: p, manifest: s.Manifest, continuing: s, peers: frozen, driver: &systemdDriver{manifest: s.Manifest}, clock: func() time.Time { return time.Now().UTC() }, units: map[string]*ownedUnit{}, bootstrapOnly: true, setupPurpose: ContinuingWitness}
	o.probePreflight = o.checkProbeHost
	plan, e := s.BootstrapPlan(context.Background(), o.clock())
	if e != nil {
		return nil, ErrRefused
	}
	o.units[digest(struct{ Setup, Proposal, Plan string }{"witness-only:v1", effortauthority.Digest(p), plan.Digest})] = &ownedUnit{plan: plan, probe: &s}
	return o, nil
}
