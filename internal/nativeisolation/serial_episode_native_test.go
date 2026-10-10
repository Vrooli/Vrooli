// serial_episode_native_test.go uses synthetic owner reads and fake drivers only.
package nativeisolation

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
	"time"
)

type serialNativeFixtureEngine struct {
	*unitOwnerFixtureEngine
	handoffs      map[string]effortauthority.SerialHandoff
	refuseHandoff bool
}

func (e *serialNativeFixtureEngine) PrepareSerialHandoff(context.Context, effortauthority.Binding, string, effortauthority.Intent) (effortauthority.SerialHandoff, bool, error) {
	return effortauthority.SerialHandoff{}, false, effortauthority.ErrRefused
}
func (e *serialNativeFixtureEngine) ReserveSerialEpisode(context.Context, effortauthority.Binding, string) (effortauthority.Reservation, bool, error) {
	return effortauthority.Reservation{}, false, effortauthority.ErrRefused
}
func (e *serialNativeFixtureEngine) ReadSerialHandoff(_ context.Context, _ effortauthority.Binding, key string) (effortauthority.SerialHandoff, error) {
	h, ok := e.handoffs[key]
	if !ok || e.refuseHandoff {
		return effortauthority.SerialHandoff{}, effortauthority.ErrRefused
	}
	return h, nil
}
func serialNativeFixture(t *testing.T) (*FixedUnitOwner, *Runtime, Request, *serialNativeFixtureEngine, *fixtureDriver) {
	t.Helper()
	m, r, p := twoProfileFixture(t)
	p.AllowChildren = true
	p.Effects = []string{"run.create", "run.child"}
	p.SerialEdges = []effortauthority.SerialEdge{{FromMember: "leader", FromProfile: "orchestrator", ToMember: "leader", ToProfile: "worker"}, {FromMember: "leader", FromProfile: "worker", ToMember: "leader", ToProfile: "orchestrator"}}
	r.PolicyDigest = effortauthority.Digest(p)
	b := m.Bindings[p.ID]
	b.PolicyDigest = r.PolicyDigest
	m.Bindings[p.ID] = b
	r.ProfileDigest = p.Profiles["worker"]
	r.ReservationKey = "reserved-worker"
	sourceRun := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	intent := effortauthority.Intent{Audience: "agent-manager", Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "leader", Profile: "orchestrator", ProfileDigest: p.Profiles["orchestrator"], Effect: "run.create", IdempotencyKey: "reserved-parent", InputDigest: hash, Turns: p.MaxTurns, ToolCalls: p.MaxToolCalls, RunSeconds: p.MaxRunSeconds}
	source := effortauthority.Reservation{NativeBound: true, Terminal: true, RunID: sourceRun, Intent: intent, IntentDigest: effortauthority.Digest(intent)}
	next := intent
	next.IdempotencyKey = r.ReservationKey
	next.Effect = "run.child"
	next.ParentRunID = sourceRun
	next.Profile = "worker"
	next.ProfileDigest = r.ProfileDigest
	next.InputDigest = digest("exact worker prompt")
	child := effortauthority.Reservation{NativeBound: true, RunID: r.RunID, Intent: next, IntentDigest: effortauthority.Digest(next)}
	engine := &serialNativeFixtureEngine{unitOwnerFixtureEngine: &unitOwnerFixtureEngine{policy: p, slots: map[string]effortauthority.Reservation{intent.IdempotencyKey: source, next.IdempotencyKey: child}}, handoffs: map[string]effortauthority.SerialHandoff{next.IdempotencyKey: {SourceKey: intent.IdempotencyKey, SourceRunID: sourceRun, Next: next}}}
	spec := ProbeSpec{Manifest: m, HelperSHA256: hash, Purpose: ContinuingWitness, Deadline: p.Deadline}
	w := &Witness{spec: spec, reader: &fixtureWitnessReader{clock: time.Now()}}
	o, err := NewFixedUnitOwner(FixedUnitOwnerConfig{Policy: p, Manifest: m, Continuing: spec, Witness: w, Authority: engine, Peers: map[string]UnitPeerRole{hash: NativeUnitPeer}})
	if err != nil {
		t.Fatal(err)
	}
	d := &fixtureDriver{}
	o.driver = d
	o.qualify = func(context.Context) error { return nil }
	rt, err := NewRuntimeWithWitnessAndUnitOwner(m, w, o)
	if err != nil {
		t.Fatal(err)
	}
	rt.qualify = func(context.Context) error { return nil }
	return o, rt, r, engine, d
}
func TestSerialNativeExactAcceptedTerminalHandoffStartsFreshUnit(t *testing.T) {
	o, rt, r, e, d := serialNativeFixture(t)
	child := e.slots[r.ReservationKey]
	if err := rt.CheckSerialEpisode(context.Background(), o.binding(), child.Intent, r.RunID); err != nil {
		t.Fatal(err)
	}
	if d.started+d.shows != 0 {
		t.Fatal("read gate touched unit")
	}
	plan, err := o.PrepareNative(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.start(context.Background(), plan.Selector(), nil); err != nil || d.started != 1 {
		t.Fatal("serial fake-driver start", err)
	}
	if _, err = o.start(context.Background(), plan.Selector(), nil); err == nil || d.started != 1 {
		t.Fatal("episode restarted")
	}
	if !e.slots["reserved-parent"].Terminal {
		t.Fatal("source reservation reopened")
	}
}
func TestSerialNativeWrongOrUnknownHandoffHasZeroUnitEffects(t *testing.T) {
	for _, name := range []string{"missing", "unavailable", "unaccepted", "terminal-child", "live-source", "unaccepted-source", "wrong-source-run", "same-run", "wrong-source-key", "changed-next", "wrong-parent", "wrong-profile", "wrong-member", "source-digest", "child-digest", "recovery", "revoked", "expired", "wrong-binding", "wrong-offered-intent", "no-carrier"} {
		t.Run(name, func(t *testing.T) {
			o, rt, r, e, d := serialNativeFixture(t)
			h := e.handoffs[r.ReservationKey]
			child := e.slots[r.ReservationKey]
			source := e.slots[h.SourceKey]
			offered := child.Intent
			binding := o.binding()
			switch name {
			case "missing":
				delete(e.handoffs, r.ReservationKey)
			case "unavailable":
				e.refuseHandoff = true
			case "unaccepted":
				child.NativeBound = false
			case "terminal-child":
				child.Terminal = true
			case "live-source":
				source.Terminal = false
			case "unaccepted-source":
				source.NativeBound = false
			case "wrong-source-run":
				source.RunID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
			case "same-run":
				child.RunID = source.RunID
				r.RunID = source.RunID
			case "wrong-source-key":
				h.SourceKey = "missing"
			case "changed-next":
				h.Next.InputDigest = digest("different prompt")
			case "wrong-parent":
				child.Intent.ParentRunID = "cccccccc-cccc-cccc-cccc-cccccccccccc"
				child.IntentDigest = effortauthority.Digest(child.Intent)
			case "wrong-profile":
				source.Intent.Profile = "worker"
				source.Intent.ProfileDigest = o.policy.Profiles["worker"]
				source.IntentDigest = effortauthority.Digest(source.Intent)
			case "wrong-member":
				source.Intent.Member = "foreign"
				source.IntentDigest = effortauthority.Digest(source.Intent)
			case "source-digest":
				source.IntentDigest = hash
			case "child-digest":
				child.IntentDigest = hash
			case "recovery":
				child.Intent.Effect = "run.recover"
				child.IntentDigest = effortauthority.Digest(child.Intent)
			case "revoked":
				e.deny = true
			case "expired":
				o.clock = func() time.Time { return o.policy.Deadline }
			case "wrong-binding":
				binding.Epoch++
			case "wrong-offered-intent":
				offered.InputDigest = digest("different offered input")
			case "no-carrier":
				o.authority = e.unitOwnerFixtureEngine
			}
			if name != "missing" {
				e.handoffs[r.ReservationKey] = h
			}
			e.slots[r.ReservationKey] = child
			e.slots["reserved-parent"] = source
			if rt.CheckSerialEpisode(context.Background(), binding, offered, r.RunID) == nil {
				t.Fatal("invalid serial read accepted")
			}
			// Offered-only defects are caller gate checks; all owner defects refuse plan.
			if name != "wrong-binding" && name != "wrong-offered-intent" {
				if _, err := o.PrepareNative(context.Background(), r); err == nil {
					t.Fatal("invalid serial plan accepted")
				}
			}
			if d.started+d.shows+len(d.signals) != 0 {
				t.Fatal("refusal touched native driver")
			}
		})
	}
}
func TestSerialNativeRechecksRetainedSourceBeforeStartAndPinsRequest(t *testing.T) {
	o, _, r, e, d := serialNativeFixture(t)
	plan, err := o.PrepareNative(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	r.Args[1] = "/foreign"
	if o.units[plan.Selector()].request.Args[1] == r.Args[1] {
		t.Fatal("request alias")
	}
	source := e.slots["reserved-parent"]
	source.Terminal = false
	e.slots["reserved-parent"] = source
	if _, err = o.start(context.Background(), plan.Selector(), nil); err == nil || d.started+d.shows != 0 {
		t.Fatal("lost terminal proof reached driver")
	}
}
