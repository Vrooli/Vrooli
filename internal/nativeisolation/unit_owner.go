// unit_owner.go owns immutable admitted native unit plans and exact lifecycle effects.
package nativeisolation

import (
	"context"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	"io"
	"sync"
	"time"
)

type UnitPeerRole string

const NativeUnitPeer UnitPeerRole = "native-owner"
const BootstrapUnitPeer UnitPeerRole = "bootstrap-owner"
const WitnessUnitPeer UnitPeerRole = "witness-setup-owner"

// FixedUnitOwnerConfig contains owner-selected existing peer pins and protected
// provider handles. Construction installs nothing, grants nothing and starts
// no unit. Protected cross-process publication of new plans is a separate owner
// integration; the HTTP endpoint cannot register or amend an executable plan.
type FixedUnitOwnerConfig struct {
	Policy     effortauthority.Policy
	Manifest   Manifest
	First      ProbeSpec
	Continuing ProbeSpec
	Witness    *Witness
	Authority  effortauthority.Engine
	Peers      map[string]UnitPeerRole
}
type ownedUnit struct {
	plan    Plan
	request Request
	probe   *ProbeSpec
	started bool
	unknown bool
}
type FixedUnitOwner struct {
	policy            effortauthority.Policy
	manifest          Manifest
	first, continuing ProbeSpec
	witness           *Witness
	authority         effortauthority.Engine
	peers             map[string]UnitPeerRole
	driver            driver
	qualify           func(context.Context) error
	clock             func() time.Time
	mu                sync.Mutex
	units             map[string]*ownedUnit
	nativeBusy        bool
	occupiedSelector  string
	bootstrapOnly     bool
	setupPurpose      ProbePurpose
	probePreflight    func(context.Context, ProbeSpec) error
}

// UnitPlan is an opaque process-local owner association. Selector is nonsecret
// data; it is never sufficient authentication or permission to operate a unit.
type UnitPlan struct {
	owner    *FixedUnitOwner
	selector string
}

func (p *UnitPlan) Selector() string {
	if p == nil {
		return ""
	}
	return p.selector
}

func NewFixedUnitOwner(cfg FixedUnitOwnerConfig) (*FixedUnitOwner, error) {
	raw, e := json.Marshal(struct {
		Policy            effortauthority.Policy
		Manifest          Manifest
		First, Continuing ProbeSpec
		Peers             map[string]UnitPeerRole
	}{cfg.Policy, cfg.Manifest, cfg.First, cfg.Continuing, cfg.Peers})
	if e != nil {
		return nil, ErrRefused
	}
	var copy struct {
		Policy            effortauthority.Policy
		Manifest          Manifest
		First, Continuing ProbeSpec
		Peers             map[string]UnitPeerRole
	}
	if json.Unmarshal(raw, &copy) != nil || copy.Policy.Validate() != nil || !copy.Manifest.Enabled || copy.Manifest.Validate() != nil || cfg.Authority == nil || len(copy.Peers) == 0 {
		return nil, ErrRefused
	}
	if copy.Manifest.CheckPolicy(copy.Policy) != nil {
		return nil, ErrRefused
	}
	for pin, role := range copy.Peers {
		if !validHash(pin) || (role != NativeUnitPeer && role != BootstrapUnitPeer) {
			return nil, ErrRefused
		}
	}
	now := time.Now().UTC()
	if copy.Continuing.Purpose != ContinuingWitness || digest(copy.Continuing.Manifest) != digest(copy.Manifest) || copy.Continuing.validate(context.Background(), now) != nil {
		return nil, ErrRefused
	}
	// Optional historical first descriptor is non-executable here. Its actual
	// bounded bootstrap authority exists only in the separate bootstrap owner.
	if copy.First.Purpose != "" && (copy.First.Purpose != FirstQualification || sameFirstManifest(copy.First.Manifest, copy.Manifest) != nil || copy.First.validate(context.Background(), copy.First.Deadline.Add(-time.Nanosecond)) != nil) {
		return nil, ErrRefused
	}
	if cfg.Witness != nil && (cfg.Witness.ValidateManifest(copy.Manifest) != nil || cfg.Witness.CheckPolicy(copy.Policy) != nil) {
		return nil, ErrRefused
	}
	o := &FixedUnitOwner{policy: copy.Policy, manifest: copy.Manifest, first: copy.First, continuing: copy.Continuing, witness: cfg.Witness, authority: cfg.Authority, peers: copy.Peers, driver: &systemdDriver{manifest: copy.Manifest}, clock: func() time.Time { return time.Now().UTC() }, units: map[string]*ownedUnit{}}
	o.probePreflight = o.checkProbeHost
	o.qualify = func(ctx context.Context) error {
		if o.witness == nil {
			return ErrRefused
		}
		rt, e := NewRuntimeWithWitness(o.manifest, o.witness)
		if e != nil {
			return ErrRefused
		}
		return rt.qualifyHost(ctx)
	}
	for _, s := range []ProbeSpec{o.continuing} {
		p, e := s.BootstrapPlan(context.Background(), now)
		if e != nil {
			return nil, ErrRefused
		}
		copy := s
		o.units[digest(struct{ Policy, Plan string }{effortauthority.Digest(o.policy), p.Digest})] = &ownedUnit{plan: p, probe: &copy}
	}
	return o, nil
}
func (o *FixedUnitOwner) CheckBinding(m Manifest, w *Witness) error {
	if o == nil || w == nil || o.witness != w || w.CheckPolicy(o.policy) != nil || w.ValidateManifest(m) != nil || digest(m) != digest(o.manifest) {
		return ErrRefused
	}
	return nil
}
func (o *FixedUnitOwner) binding() effortauthority.Binding {
	return effortauthority.Binding{PolicyID: o.policy.ID, PolicyDigest: effortauthority.Digest(o.policy), Epoch: o.policy.Epoch, Owner: o.policy.Owner, Client: o.policy.Client, Deadline: o.policy.Deadline}
}
func (o *FixedUnitOwner) current(ctx context.Context) error {
	if o == nil || o.authority == nil || o.bootstrapOnly || ctx.Err() != nil || !o.clock().Before(o.policy.Deadline) {
		return ErrRefused
	}
	p, e := o.authority.CheckBinding(ctx, o.binding())
	if e != nil || effortauthority.Digest(p) != effortauthority.Digest(o.policy) {
		return ErrRefused
	}
	return nil
}
func (o *FixedUnitOwner) native(ctx context.Context, r Request) error {
	if o.current(ctx) != nil || r.ReservationKey == "" || r.PolicyID != o.policy.ID || r.Timeout > time.Duration(o.policy.MaxRunSeconds)*time.Second {
		return ErrRefused
	}
	slot, e := o.authority.ReadReservation(ctx, o.binding(), r.ReservationKey)
	if e != nil || !slot.NativeBound || slot.Terminal || slot.RunID != r.RunID || slot.Intent.IdempotencyKey != r.ReservationKey || slot.Intent.SourceRunID != "" || slot.Intent.Repository != r.Repository || slot.Intent.ProfileDigest != r.ProfileDigest || slot.Intent.RunSeconds <= 0 || r.Timeout > time.Duration(slot.Intent.RunSeconds)*time.Second {
		return ErrRefused
	}
	switch slot.Intent.Effect {
	case "run.create":
		if slot.Intent.ParentRunID != "" {
			return ErrRefused
		}
	case "run.child":
		if o.serialEpisode(ctx, slot) != nil {
			return ErrRefused
		}
	default:
		return ErrRefused
	}
	return nil
}

// serialEpisodeEngine is the exact optional protected owner carrier. Its read
// does not acquire authority; the binding and reservation are current owner data.
type serialEpisodeEngine interface {
	PrepareSerialHandoff(context.Context, effortauthority.Binding, string, effortauthority.Intent) (effortauthority.SerialHandoff, bool, error)
	ReserveSerialEpisode(context.Context, effortauthority.Binding, string) (effortauthority.Reservation, bool, error)
	ReadSerialHandoff(context.Context, effortauthority.Binding, string) (effortauthority.SerialHandoff, error)
}

func (o *FixedUnitOwner) serialEpisode(ctx context.Context, slot effortauthority.Reservation) error {
	engine, ok := o.authority.(serialEpisodeEngine)
	if !ok || len(o.policy.SerialEdges) == 0 || !o.policy.AllowChildren || slot.Intent.Effect != "run.child" || slot.Intent.Endpoint != "/api/v1/runs" || slot.Intent.ParentRunID == "" || slot.Intent.SourceRunID != "" || slot.IntentDigest != effortauthority.Digest(slot.Intent) {
		return ErrRefused
	}
	h, err := engine.ReadSerialHandoff(ctx, o.binding(), slot.Intent.IdempotencyKey)
	if err != nil || h.SourceKey == slot.Intent.IdempotencyKey || h.SourceRunID != slot.Intent.ParentRunID || effortauthority.Digest(h.Next) != effortauthority.Digest(slot.Intent) {
		return ErrRefused
	}
	source, err := o.authority.ReadReservation(ctx, o.binding(), h.SourceKey)
	if err != nil || !source.NativeBound || !source.Terminal || source.RunID == "" || source.RunID != h.SourceRunID || source.RunID == slot.RunID || source.Intent.IdempotencyKey != h.SourceKey || source.IntentDigest != effortauthority.Digest(source.Intent) || source.Intent.Repository != o.policy.Repository || source.Intent.SourceRunID != "" {
		return ErrRefused
	}
	if !runID.MatchString(slot.RunID) || !runID.MatchString(source.RunID) || (source.Intent.Effect != "run.create" && source.Intent.Effect != "run.child") {
		return ErrRefused
	}
	if slot.Intent.PolicyID != o.policy.ID || slot.Intent.Epoch != o.policy.Epoch || slot.Intent.Client != o.policy.Client || source.Intent.PolicyID != o.policy.ID || source.Intent.Epoch != o.policy.Epoch || source.Intent.Client != o.policy.Client || slot.Intent.ProfileDigest != o.policy.Profiles[slot.Intent.Profile] || source.Intent.ProfileDigest != o.policy.Profiles[source.Intent.Profile] || slot.Intent.ProfileDigest == "" || source.Intent.ProfileDigest == "" {
		return ErrRefused
	}
	for _, intent := range []effortauthority.Intent{source.Intent, slot.Intent} {
		if intent.Effort != o.policy.Effort || intent.Revision != o.policy.Revision || intent.ContentDigest != o.policy.ContentDigest || intent.Team != o.policy.Team || intent.Method != "POST" || intent.Turns < 1 || intent.Turns > o.policy.MaxTurns || intent.ToolCalls < 1 || intent.ToolCalls > o.policy.MaxToolCalls || intent.RunSeconds < 1 || intent.RunSeconds > o.policy.MaxRunSeconds {
			return ErrRefused
		}
	}
	if slot.Intent.Turns != o.policy.MaxTurns || slot.Intent.ToolCalls != o.policy.MaxToolCalls || slot.Intent.RunSeconds != o.policy.MaxRunSeconds {
		return ErrRefused
	}
	for _, edge := range o.policy.SerialEdges {
		if source.Intent.Member == edge.FromMember && source.Intent.Profile == edge.FromProfile && slot.Intent.Member == edge.ToMember && slot.Intent.Profile == edge.ToProfile {
			return nil
		}
	}
	return ErrRefused
}

// PrepareNative is a local owning-process operation, not an HTTP route. It
// freezes the admitted native request after exact live ledger receipt checks.
func (o *FixedUnitOwner) PrepareNative(ctx context.Context, r Request) (*UnitPlan, error) {
	if o == nil || o.native(ctx, r) != nil {
		return nil, ErrRefused
	}
	raw, e := json.Marshal(r)
	if e != nil {
		return nil, ErrRefused
	}
	var copy Request
	if json.Unmarshal(raw, &copy) != nil {
		return nil, ErrRefused
	}
	p, e := o.manifest.Plan(ctx, copy, o.clock())
	if e != nil {
		return nil, ErrRefused
	}
	selector := digest(struct {
		Policy  string
		Request Request
	}{effortauthority.Digest(o.policy), copy})
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, existing := range o.units {
		if existing.probe == nil && existing.request.RunID == copy.RunID && digest(existing.request) != digest(copy) {
			return nil, ErrRefused
		}
	}
	if old := o.units[selector]; old == nil {
		o.units[selector] = &ownedUnit{plan: p, request: copy}
	} else if digest(old.request) != digest(copy) {
		return nil, ErrRefused
	}
	return &UnitPlan{o, selector}, nil
}
func (o *FixedUnitOwner) ProbePlan(purpose ProbePurpose) (*UnitPlan, error) {
	if o == nil {
		return nil, ErrRefused
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	for selector, u := range o.units {
		if u.probe != nil && u.probe.Purpose == purpose {
			return &UnitPlan{o, selector}, nil
		}
	}
	return nil, ErrRefused
}
func (o *FixedUnitOwner) read(ctx context.Context, selector string) (map[string]string, error) {
	o.mu.Lock()
	u := o.units[selector]
	if u == nil || !u.started {
		o.mu.Unlock()
		return nil, ErrRefused
	}
	unit, native := u.plan.Unit, u.probe == nil
	o.mu.Unlock()
	state, e := o.driver.Show(ctx, unit)
	if e == nil && terminalState(state) && native {
		o.mu.Lock()
		if o.occupiedSelector == selector {
			o.nativeBusy = false
			o.occupiedSelector = ""
		}
		o.mu.Unlock()
	}
	return state, e
}
func (o *FixedUnitOwner) stop(ctx context.Context, selector string, kill bool) error {
	o.mu.Lock()
	u := o.units[selector]
	if u == nil || !u.started {
		o.mu.Unlock()
		return ErrRefused
	}
	unit := u.plan.Unit
	o.mu.Unlock()
	signal := "SIGTERM"
	if kill {
		signal = "SIGKILL"
	}
	return o.driver.Signal(ctx, unit, signal)
}
func (o *FixedUnitOwner) start(ctx context.Context, selector string, input io.Reader) (child, error) {
	if o == nil || ctx.Err() != nil {
		return nil, ErrRefused
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	u := o.units[selector]
	if u == nil || u.started || u.unknown {
		return nil, ErrRefused
	}
	first := u.probe != nil && u.probe.Purpose == FirstQualification
	setup := o.bootstrapOnly && u.probe != nil && u.probe.Purpose == o.setupPurpose
	if setup {
		if u.probe.validate(ctx, o.clock()) != nil {
			return nil, ErrRefused
		}
	} else if o.bootstrapOnly || first || o.current(ctx) != nil {
		return nil, ErrRefused
	}
	var p Plan
	var e error
	if u.probe == nil {
		if o.nativeBusy || o.native(ctx, u.request) != nil || o.qualify(ctx) != nil {
			return nil, ErrRefused
		}
		p, e = o.manifest.Plan(ctx, u.request, o.clock())
	} else {
		if input != nil {
			return nil, ErrRefused
		}
		p, e = u.probe.BootstrapPlan(ctx, o.clock())
		if e == nil {
			e = o.probePreflight(ctx, *u.probe)
		}
	}
	if e != nil {
		return nil, ErrRefused
	}
	state, e := o.driver.Show(ctx, p.Unit)
	if e != nil || state["LoadState"] != "not-found" {
		return nil, ErrRefused
	}
	// Revalidate current authority immediately before the retained start effect.
	if setup {
		if u.probe.validate(ctx, o.clock()) != nil {
			return nil, ErrRefused
		}
	} else if o.current(ctx) != nil || (u.probe == nil && o.native(ctx, u.request) != nil) {
		return nil, ErrRefused
	}
	u.plan = p
	u.started = true
	if u.probe == nil {
		o.nativeBusy = true
		o.occupiedSelector = selector
	}
	c, e := o.driver.Start(ctx, p, input)
	if e != nil {
		u.unknown = true
		return c, ErrUnknown
	}
	return c, nil
}

// First qualification may only narrow the installed production network allow
// list to deny-all. Every identity/image/path/policy/binary field stays exact.
func sameFirstManifest(first, production Manifest) error {
	if len(first.NetworkAllow) != 0 {
		return ErrRefused
	}
	copy, e := cloneSpec(ProbeSpec{Manifest: first})
	if e != nil {
		return ErrRefused
	}
	copy.Manifest.NetworkAllow = append([]string(nil), production.NetworkAllow...)
	if digest(copy.Manifest) != digest(production) {
		return ErrRefused
	}
	return nil
}
