// two_profile_binding_test.go qualifies pure frozen profile matching without native effects.
package nativeisolation

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
	"time"
)

func twoProfileFixture(t *testing.T) (Manifest, Request, effortauthority.Policy) {
	t.Helper()
	owner, _, request, engine := unitOwnerFixture(t, &fixtureDriver{})
	copy, err := cloneSpec(ProbeSpec{Manifest: owner.manifest})
	if err != nil {
		t.Fatal(err)
	}
	policy := engine.policy
	policy.Profiles = map[string]string{"orchestrator": hash, "worker": digest("existing complete worker profile")}
	m := copy.Manifest
	m.ProfileBindings = map[string]string{"orchestrator": policy.Profiles["orchestrator"], "worker": policy.Profiles["worker"]}
	request.PolicyDigest = effortauthority.Digest(policy)
	b := m.Bindings[policy.ID]
	b.PolicyDigest = request.PolicyDigest
	m.Bindings[policy.ID] = b
	return m, request, policy
}
func TestTwoProfilePlanAndPolicyPreserveOnePolicyIsolation(t *testing.T) {
	m, req, policy := twoProfileFixture(t)
	if err := m.CheckPolicy(policy); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"orchestrator", "worker"} {
		req.ProfileDigest = policy.Profiles[key]
		plan, err := m.Plan(context.Background(), req, time.Now())
		if err != nil || plan.Request.ProfileDigest != req.ProfileDigest || len(m.Bindings) != 1 {
			t.Fatalf("%s profile plan refused: %v", key, err)
		}
		rt, err := NewRuntime(m)
		if err != nil || rt.CheckBinding(policy.ID, Binding{req.PolicyDigest, req.ProfileDigest, req.Repository, req.Deadline}) != nil {
			t.Fatalf("%s runtime binding refused: %v", key, err)
		}
		if rt.Ready() {
			t.Fatal("planning-only runtime became ready")
		}
	}
	req.ProfileDigest = digest("unapproved")
	if _, err := m.Plan(context.Background(), req, time.Now()); err == nil {
		t.Fatal("foreign profile planned")
	}
}
func TestTwoProfileCompleteTableRefusals(t *testing.T) {
	for _, name := range []string{"missing", "extra", "ambiguous", "unknown-key", "digest", "policy", "repository", "deadline", "second-policy", "base-missing"} {
		t.Run(name, func(t *testing.T) {
			m, req, p := twoProfileFixture(t)
			b := m.Bindings[p.ID]
			switch name {
			case "missing":
				delete(m.ProfileBindings, "worker")
			case "extra":
				m.ProfileBindings["extra"] = digest("extra")
			case "ambiguous":
				m.ProfileBindings["worker"] = hash
			case "unknown-key":
				delete(m.ProfileBindings, "worker")
				m.ProfileBindings["other"] = p.Profiles["worker"]
			case "digest":
				m.ProfileBindings["worker"] = digest("changed")
			case "policy":
				b.PolicyDigest = digest("other policy")
				m.Bindings[p.ID] = b
			case "repository":
				b.Repository = "/other/repository"
				m.Bindings[p.ID] = b
			case "deadline":
				b.Deadline = b.Deadline.Add(time.Second)
				m.Bindings[p.ID] = b
			case "second-policy":
				m.Bindings["other"] = b
			case "base-missing":
				b.ProfileDigest = digest("base absent")
				m.Bindings[p.ID] = b
			}
			if m.CheckPolicy(p) == nil {
				t.Fatal("incomplete or changed table accepted")
			}
			// Structural defects also refuse pure native planning; policy/key changes
			// are checked against the actual installed policy by startup/owner matching.
			if name == "missing" || name == "extra" || name == "ambiguous" || name == "second-policy" || name == "base-missing" {
				if _, err := m.Plan(context.Background(), req, time.Now()); err == nil {
					t.Fatal("malformed native table planned")
				}
			}
		})
	}
}
func TestTwoProfileWitnessAndOwnerCaptureCompleteTableWithoutUnitEffects(t *testing.T) {
	m, req, p := twoProfileFixture(t)
	spec := ProbeSpec{Manifest: m, HelperSHA256: hash, Purpose: ContinuingWitness, Deadline: p.Deadline}
	w := &Witness{spec: spec, reader: &fixtureWitnessReader{clock: time.Now()}}
	if w.CheckPolicy(p) != nil {
		t.Fatal("two-profile witness mismatch")
	}
	engine := &unitOwnerFixtureEngine{policy: p, slots: map[string]effortauthority.Reservation{}}
	addUnitOwnerReceipt(engine, req)
	cfg := FixedUnitOwnerConfig{Policy: p, Manifest: m, Continuing: spec, Witness: w, Authority: engine, Peers: map[string]UnitPeerRole{hash: NativeUnitPeer}}
	o, err := NewFixedUnitOwner(cfg)
	if err != nil {
		t.Fatal(err)
	}
	d := &fixtureDriver{}
	o.driver = d
	if o.CheckBinding(m, w) != nil {
		t.Fatal("whole manifest owner binding mismatch")
	}
	req.ProfileDigest = p.Profiles["worker"]
	req.ReservationKey = "worker-root-receipt"
	req.RunID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	addUnitOwnerReceipt(engine, req)
	if _, err = o.PrepareNative(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	slot := engine.slots[req.ReservationKey]
	slot.Intent.Effect = "run.child"
	slot.Intent.ParentRunID = "parent"
	engine.slots[req.ReservationKey] = slot
	if _, err = o.PrepareNative(context.Background(), req); err == nil {
		t.Fatal("two profiles enabled child launch")
	}
	cfg.Manifest.ProfileBindings["worker"] = digest("caller mutation")
	cfg.Policy.Profiles["worker"] = digest("caller mutation")
	if o.manifest.ProfileBindings["worker"] != req.ProfileDigest || o.policy.Profiles["worker"] != req.ProfileDigest {
		t.Fatal("owner aliases profile table")
	}
	if o.CheckBinding(cfg.Manifest, w) == nil {
		t.Fatal("changed whole manifest matched")
	}
	if d.started+d.shows+len(d.signals) != 0 {
		t.Fatal("profile checking caused unit effects")
	}
}
func TestTwoProfileRuntimeFreezeAndLegacyBinding(t *testing.T) {
	legacy, req := fixture()
	if legacy.Validate() != nil {
		t.Fatal("legacy manifest invalid")
	}
	old, err := legacy.Plan(context.Background(), req, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	rt, err := NewRuntime(legacy)
	if err != nil || rt.CheckBinding(req.PolicyID, legacy.Bindings[req.PolicyID]) != nil {
		t.Fatal("legacy binding changed", err)
	}
	again, err := rt.manifest.Plan(context.Background(), req, time.Now())
	if err != nil || old.Unit != again.Unit || digest(old.Args) != digest(again.Args) {
		t.Fatal("legacy plan changed", err)
	}
	m, req, p := twoProfileFixture(t)
	rt, err = NewRuntime(m)
	if err != nil {
		t.Fatal(err)
	}
	req.ProfileDigest = p.Profiles["worker"]
	m.ProfileBindings["worker"] = digest("changed after capture")
	if rt.CheckBinding(p.ID, Binding{req.PolicyDigest, req.ProfileDigest, req.Repository, req.Deadline}) != nil {
		t.Fatal("captured runtime table amended")
	}
	foreign := Binding{req.PolicyDigest, digest("unknown"), req.Repository, req.Deadline}
	if rt.CheckBinding(p.ID, foreign) == nil {
		t.Fatal("foreign digest matched runtime")
	}
}
