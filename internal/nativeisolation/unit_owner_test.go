// unit_owner_test.go qualifies synthetic owner/provider/TLS boundaries without host effects.
package nativeisolation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/vrooli/api-core/effortauthority"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type unitOwnerFixtureEngine struct {
	effortauthority.Engine
	policy        effortauthority.Policy
	slots         map[string]effortauthority.Reservation
	deny          bool
	checks, reads int
}

func (e *unitOwnerFixtureEngine) CheckBinding(context.Context, effortauthority.Binding) (effortauthority.Policy, error) {
	e.checks++
	if e.deny {
		return effortauthority.Policy{}, effortauthority.ErrRefused
	}
	return e.policy, nil
}
func (e *unitOwnerFixtureEngine) ReadReservation(_ context.Context, _ effortauthority.Binding, key string) (effortauthority.Reservation, error) {
	e.reads++
	if e.deny {
		return effortauthority.Reservation{}, effortauthority.ErrRefused
	}
	slot, ok := e.slots[key]
	if !ok {
		return effortauthority.Reservation{}, effortauthority.ErrRefused
	}
	return slot, nil
}

// Package-private fixtures do not establish host isolation or credential
// possession. Witness readers, current provider and unit effects are synthetic.
func unitOwnerFixture(t *testing.T, d driver) (*FixedUnitOwner, *Runtime, Request, *unitOwnerFixtureEngine) {
	t.Helper()
	m, r := fixture()
	now := time.Now().UTC()
	m.NetworkAllow = nil
	m.WorkspaceSource = t.TempDir()
	marker := filepath.Join(t.TempDir(), "public-marker")
	if os.WriteFile(marker, []byte("nonsecret fixture marker"), 0600) != nil {
		t.Fatal("marker setup")
	}
	m.Isolation.ProtectedPaths = []string{marker}
	m.Isolation.NativeProbePID = 42
	deadline := now.Add(time.Hour)
	p := effortauthority.Policy{ID: r.PolicyID, Owner: "owner", Client: "client", ClientKey: make([]byte, 32), Epoch: 1, Repository: r.Repository, Effort: "effort", Revision: "revision", ContentDigest: hash, Team: "team", TeamDigest: hash, BindingDigest: hash, Members: []string{"leader"}, Profiles: map[string]string{"bounded": hash}, Scopes: []string{"agent-manager:write"}, Effects: []string{"run.create"}, NotBefore: now, Deadline: deadline, MaxStarts: 4, MaxConcurrent: 1, MaxTurns: 1, MaxToolCalls: 1, MaxRunSeconds: 3600, TotalTurns: 4, TotalToolCalls: 4, TotalRunSeconds: 14400, SurviveLogout: true}
	r.PolicyDigest = effortauthority.Digest(p)
	r.Deadline = deadline
	r.ReservationKey = "first-receipt"
	m.Bindings[p.ID] = Binding{PolicyDigest: r.PolicyDigest, ProfileDigest: r.ProfileDigest, Repository: r.Repository, Deadline: deadline}
	first, _ := firstObservationFixture()
	first.Manifest = m
	first.Deadline = now.Add(5 * time.Minute)
	continuing := ProbeSpec{Manifest: m, HelperSHA256: hash, Purpose: ContinuingWitness, Deadline: deadline}
	witness := &Witness{spec: continuing, reader: &fixtureWitnessReader{clock: now}}
	pin := sha256.Sum256([]byte("synthetic-native-peer"))
	bootstrap := sha256.Sum256([]byte("synthetic-bootstrap-peer"))
	engine := &unitOwnerFixtureEngine{policy: p, slots: map[string]effortauthority.Reservation{}}
	addUnitOwnerReceipt(engine, r)
	owner, e := NewFixedUnitOwner(FixedUnitOwnerConfig{Policy: p, Manifest: m, First: first, Continuing: continuing, Witness: witness, Authority: engine, Peers: map[string]UnitPeerRole{hex.EncodeToString(pin[:]): NativeUnitPeer, hex.EncodeToString(bootstrap[:]): BootstrapUnitPeer}})
	if e != nil {
		t.Fatal(e)
	}
	owner.driver = d
	owner.qualify = func(context.Context) error { return nil }
	rt, e := NewRuntimeWithWitnessAndUnitOwner(m, witness, owner)
	if e != nil {
		t.Fatal(e)
	}
	rt.qualify = func(context.Context) error { return nil }
	return owner, rt, r, engine
}
func addUnitOwnerReceipt(e *unitOwnerFixtureEngine, r Request) {
	e.slots[r.ReservationKey] = effortauthority.Reservation{NativeBound: true, RunID: r.RunID, Intent: effortauthority.Intent{IdempotencyKey: r.ReservationKey, Effect: "run.create", Repository: r.Repository, ProfileDigest: r.ProfileDigest, RunSeconds: 3600}}
}
func TestFixedUnitOwnerRejectsUnacceptedOrRevokedPlansWithoutUnitEffects(t *testing.T) {
	for _, name := range []string{"key", "run", "terminal", "unbound", "profile", "repository", "revoked", "policy", "runtime", "child", "recovery"} {
		t.Run(name, func(t *testing.T) {
			d := &fixtureDriver{}
			o, _, r, e := unitOwnerFixture(t, d)
			slot := e.slots[r.ReservationKey]
			switch name {
			case "key":
				r.ReservationKey = ""
			case "run":
				slot.RunID = "foreign"
			case "terminal":
				slot.Terminal = true
			case "unbound":
				slot.NativeBound = false
			case "profile":
				slot.Intent.ProfileDigest = digest("foreign")
			case "repository":
				slot.Intent.Repository = "/foreign"
			case "revoked":
				e.deny = true
			case "policy":
				r.PolicyID = "foreign"
			case "child":
				slot.Intent.Effect = "run.child"
				slot.Intent.ParentRunID = "parent"
			case "recovery":
				slot.Intent.Effect = "run.recover"
				slot.Intent.SourceRunID = "source"
			case "runtime":
				r.Timeout = 2 * time.Hour
			}
			e.slots["first-receipt"] = slot
			if _, err := o.PrepareNative(context.Background(), r); err == nil {
				t.Fatal("unaccepted plan allowed")
			}
			if d.started+d.shows+len(d.signals) != 0 {
				t.Fatal("refusal caused unit effects")
			}
		})
	}
}
func TestFixedUnitOwnerPinsLocalPlansAndRefusesWitnessOnlyRuntime(t *testing.T) {
	d := &fixtureDriver{}
	o, rt, r, _ := unitOwnerFixture(t, d)
	plan, e := o.PrepareNative(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	original := o.units[plan.selector].request.Args[1]
	r.Args[1] = "/foreign"
	if o.units[plan.selector].request.Args[1] != original {
		t.Fatal("owner plan aliases offered request")
	}
	plain, e := NewRuntimeWithWitness(rt.manifest, rt.witness)
	if e != nil {
		t.Fatal(e)
	}
	if plain.Ready() {
		t.Fatal("witness implicitly granted unit operations")
	}
	if _, e = plain.Start(context.Background(), r, nil); e == nil {
		t.Fatal("ownerless launch accepted")
	}
	if d.started+d.shows != 0 {
		t.Fatal("ownerless launch touched units")
	}
	changed, _ := cloneSpec(ProbeSpec{Manifest: rt.manifest})
	changed.Manifest.RootFSDigest = digest("foreign")
	if _, e = NewRuntimeWithWitnessAndUnitOwner(changed.Manifest, rt.witness, o); e == nil {
		t.Fatal("foreign owner attached")
	}
}

func TestFixedUnitOwnerRevalidationHoldsLostReplyAndRevokedStart(t *testing.T) {
	for _, name := range []string{"revoked", "unknown"} {
		t.Run(name, func(t *testing.T) {
			d := &fixtureDriver{failStart: true}
			o, _, r, engine := unitOwnerFixture(t, d)
			plan, e := o.PrepareNative(context.Background(), r)
			if e != nil {
				t.Fatal(e)
			}
			if name == "revoked" {
				engine.deny = true
			}
			_, e = o.start(context.Background(), plan.selector, nil)
			if name == "revoked" {
				if e == nil || d.started+d.shows != 0 {
					t.Fatal("revoked prepared plan had effects")
				}
			} else {
				if !errors.Is(e, ErrUnknown) || !o.nativeBusy || !o.units[plan.selector].unknown {
					t.Fatal("unknown outcome released owner slot")
				}
				if _, e = o.start(context.Background(), plan.selector, nil); e == nil || d.started != 1 {
					t.Fatal("unknown plan redispatched")
				}
			}
		})
	}
}

func TestFixedUnitOwnerFirstManifestMayOnlyNarrowNetwork(t *testing.T) {
	m, _ := fixture()
	first, _ := cloneSpec(ProbeSpec{Manifest: m})
	first.Manifest.NetworkAllow = nil
	if sameFirstManifest(first.Manifest, m) != nil {
		t.Fatal("deny-all first manifest refused")
	}
	for _, change := range []func(*Manifest){func(m *Manifest) { m.NetworkAllow = []string{"203.0.113.1/32"} }, func(m *Manifest) { m.RootFSDigest = digest("foreign") }, func(m *Manifest) { m.WorkspaceSource = "/foreign" }, func(m *Manifest) { m.NativeUID++ }, func(m *Manifest) { m.EnvKeys = append(m.EnvKeys, "FOREIGN") }} {
		copy, _ := cloneSpec(first)
		change(&copy.Manifest)
		if sameFirstManifest(copy.Manifest, m) == nil {
			t.Fatal("first qualification widened or changed installation")
		}
	}
}
func TestFixedUnitOwnerReadyChecksCurrentGrantBeforeHostRead(t *testing.T) {
	d := &fixtureDriver{}
	_, rt, _, engine := unitOwnerFixture(t, d)
	calls := 0
	rt.qualify = func(context.Context) error { calls++; return nil }
	if rt.CheckReady(context.Background()) != nil || calls != 1 {
		t.Fatal("exact current owner not ready")
	}
	engine.deny = true
	if rt.CheckReady(context.Background()) == nil || calls != 1 || d.started+d.shows != 0 {
		t.Fatal("revoked owner reached host/unit effects")
	}
}
func TestFixedUnitOwnerLocalDriverPreservesExistingStreamingCarrier(t *testing.T) {
	d := &unitStreamingFixtureDriver{}
	o, rt, r, _ := unitOwnerFixture(t, d)
	p, e := rt.manifest.Plan(context.Background(), r, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	driver := rt.driver.(*fixedUnitDriver)
	if driver.prepare(context.Background(), p) != nil {
		t.Fatal("accepted local plan refused")
	}
	c, e := driver.Start(context.Background(), p, strings.NewReader("exact disposable stdin"))
	if e != nil {
		t.Fatal(e)
	}
	out, e := io.ReadAll(c.Stdout())
	if e != nil || string(out) != "disposable-native-output" || d.stdin != "exact disposable stdin" {
		t.Fatal("native streaming carrier changed", e)
	}
	if _, e = o.start(context.Background(), driverSelectorForTest(t, driver, p.Unit), nil); e == nil || d.starts != 1 {
		t.Fatal("local plan repeated")
	}
}
func driverSelectorForTest(t *testing.T, d *fixedUnitDriver, unit string) string {
	t.Helper()
	h, e := d.planForUnit(unit)
	if e != nil {
		t.Fatal(e)
	}
	return h.selector
}

type unitStreamingFixtureDriver struct {
	stdin  string
	starts int
}

func (d *unitStreamingFixtureDriver) Start(_ context.Context, _ Plan, input io.Reader) (child, error) {
	d.starts++
	raw, e := io.ReadAll(input)
	d.stdin = string(raw)
	return &fixtureChild{}, e
}
func (*unitStreamingFixtureDriver) Show(context.Context, string) (map[string]string, error) {
	return map[string]string{"LoadState": "not-found"}, nil
}
func (*unitStreamingFixtureDriver) Signal(context.Context, string, string) error { return nil }

func TestFixedUnitOwnerSameNativeUnitCannotAmendFrozenPlan(t *testing.T) {
	d := &fixtureDriver{}
	o, _, r, _ := unitOwnerFixture(t, d)
	if _, e := o.PrepareNative(context.Background(), r); e != nil {
		t.Fatal(e)
	}
	r.Args = append(append([]string{}, r.Args...), "--foreign-input")
	if _, e := o.PrepareNative(context.Background(), r); e == nil || d.started+d.shows != 0 {
		t.Fatal("accepted native plan amended or had effects")
	}
}
