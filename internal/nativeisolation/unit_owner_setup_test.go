// unit_owner_setup_test.go qualifies fixed nonmodel setup separate from finite approval.
package nativeisolation

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFixedBootstrapOwnerIsPeerBoundWithoutFiniteLedger(t *testing.T) {
	spec, _ := firstObservationFixture()
	now := time.Now().UTC()
	spec.Deadline = now.Add(5 * time.Minute)
	b := spec.Manifest.Bindings["canary"]
	b.Deadline = now.Add(time.Hour)
	spec.Manifest.Bindings["canary"] = b
	pin := sha256.Sum256([]byte("synthetic-bootstrap-peer"))
	owner, e := NewFixedBootstrapUnitOwner(spec, map[string]UnitPeerRole{hex.EncodeToString(pin[:]): BootstrapUnitPeer})
	if e != nil {
		t.Fatal(e)
	}
	driver := &fixtureDriver{}
	owner.driver = driver
	preflights := 0
	owner.probePreflight = func(context.Context, ProbeSpec) error { preflights++; return nil }
	plan, e := owner.ProbePlan(FirstQualification)
	if e != nil {
		t.Fatal(e)
	}
	h, _ := NewFixedUnitOwnerHandler(owner)
	request := syntheticUnitRequest(plan.selector, "probe-start")
	request.TLS.PeerCertificates[0] = &x509.Certificate{Raw: []byte("synthetic-bootstrap-peer")}
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != 204 || driver.started != 1 || preflights != 1 || owner.authority != nil {
		t.Fatal("first probe incorrectly required finite ledger or refused exact peer", response.Code)
	}
	if _, e := owner.ProbePlan(ContinuingWitness); e == nil {
		t.Fatal("bootstrap acquired continuing authority")
	}
	if owner.current(context.Background()) == nil {
		t.Fatal("bootstrap acquired finite policy authority")
	}
	if _, e = owner.PrepareNative(context.Background(), Request{}); e == nil {
		t.Fatal("bootstrap acquired native plan authority")
	}
	response = httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != 403 || driver.started != 1 {
		t.Fatal("first bootstrap repeated")
	}
}
func TestFixedBootstrapOwnerAbsentPeerAndExpiredSpecHaveZeroUnitEffects(t *testing.T) {
	spec, _ := firstObservationFixture()
	now := time.Now().UTC()
	spec.Deadline = now.Add(5 * time.Minute)
	b := spec.Manifest.Bindings["canary"]
	b.Deadline = now.Add(time.Hour)
	spec.Manifest.Bindings["canary"] = b
	pin := sha256.Sum256([]byte("synthetic-bootstrap-peer"))
	owner, e := NewFixedBootstrapUnitOwner(spec, map[string]UnitPeerRole{hex.EncodeToString(pin[:]): BootstrapUnitPeer})
	if e != nil {
		t.Fatal(e)
	}
	driver := &fixtureDriver{}
	owner.driver = driver
	owner.probePreflight = func(context.Context, ProbeSpec) error { t.Fatal("refusal reached host preflight"); return nil }
	plan, _ := owner.ProbePlan(FirstQualification)
	h, _ := NewFixedUnitOwnerHandler(owner)
	for _, name := range []string{"absent-peer", "wrong-peer", "expired"} {
		t.Run(name, func(t *testing.T) {
			r := syntheticUnitRequest(plan.selector, "probe-start")
			if name == "absent-peer" {
				r.TLS = nil
				r.Body = refuseBody{}
			} else if name == "wrong-peer" {
				r.Body = refuseBody{}
			} else {
				r.TLS.PeerCertificates[0] = &x509.Certificate{Raw: []byte("synthetic-bootstrap-peer")}
				owner.clock = func() time.Time { return spec.Deadline }
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 403 || driver.started+driver.shows+len(driver.signals) != 0 {
				t.Fatal("invalid bootstrap peer/time caused unit effects")
			}
		})
	}
}

func TestFixedWitnessSetupOwnerPrecedesConsentWithoutNativeGrant(t *testing.T) {
	_, rt, _, engine := unitOwnerFixture(t, &fixtureDriver{})
	proposal := engine.policy
	spec := rt.witness.spec
	pin := sha256.Sum256([]byte("synthetic-witness-peer"))
	owner, e := NewFixedWitnessUnitOwner(proposal, spec, map[string]UnitPeerRole{hex.EncodeToString(pin[:]): WitnessUnitPeer})
	if e != nil {
		t.Fatal(e)
	}
	driver := &fixtureDriver{}
	owner.driver = driver
	preflights := 0
	owner.probePreflight = func(context.Context, ProbeSpec) error { preflights++; return nil }
	plan, e := owner.ProbePlan(ContinuingWitness)
	if e != nil {
		t.Fatal(e)
	}
	handler, _ := NewFixedUnitOwnerHandler(owner)
	r := syntheticUnitRequest(plan.selector, "probe-start")
	r.TLS.PeerCertificates[0] = &x509.Certificate{Raw: []byte("synthetic-witness-peer")}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 204 || driver.started != 1 || preflights != 1 || owner.authority != nil {
		t.Fatal("witness setup required live finite approval or changed purpose", w.Code)
	}
	if owner.current(context.Background()) == nil {
		t.Fatal("proposal became finite grant")
	}
	if _, e := owner.PrepareNative(context.Background(), Request{}); e == nil {
		t.Fatal("setup acquired native capability")
	}
	if _, e := owner.ProbePlan(FirstQualification); e == nil {
		t.Fatal("witness setup acquired first authority")
	}
	if _, e := NewRuntimeWithWitnessAndUnitOwner(spec.Manifest, rt.witness, owner); e == nil {
		t.Fatal("setup became model unit owner")
	}
	changed := proposal
	changed.Revision = "foreign"
	if _, e := NewFixedWitnessUnitOwner(changed, spec, map[string]UnitPeerRole{hex.EncodeToString(pin[:]): WitnessUnitPeer}); e == nil {
		t.Fatal("different proposal selected existing spec")
	}
	if _, e := NewFixedWitnessUnitOwner(proposal, spec, map[string]UnitPeerRole{hex.EncodeToString(pin[:]): NativeUnitPeer}); e == nil {
		t.Fatal("native peer acquired setup role")
	}
}
