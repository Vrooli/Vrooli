package nativeisolation

import (
	"context"
	"github.com/vrooli/api-core/effortauthority"
	"github.com/vrooli/vrooli/internal/nativeisolation/probeprotocol"
	"strings"
	"testing"
	"time"
)

func firstObservationFixture() (ProbeSpec, time.Time) {
	s, now := probeFixture()
	s.Purpose = FirstQualification
	s.Deadline = now.Add(5 * time.Minute)
	s.FirstObservation = &FirstObservationContract{CanaryRoot: "/var/lib/vrooli-isolation-canary", CanaryID: strings.Repeat("a", 32), WorkspaceSHA256: probeprotocol.MarkerDigest(strings.Repeat("a", 32), "workspace"), DeniedSHA256: probeprotocol.MarkerDigest(strings.Repeat("a", 32), "denied")}
	return s, now
}
func TestFirstQualificationRequiresExactOwnerCanaryContract(t *testing.T) {
	s, now := firstObservationFixture()
	p, e := s.BootstrapPlan(context.Background(), now)
	if e != nil {
		t.Fatal(e)
	}
	args := strings.Join(p.Args, " ")
	if !strings.Contains(args, "--property=BindReadOnlyPaths="+s.FirstObservation.deniedSource()+":"+probeprotocol.DeniedMarker) {
		t.Fatal("denial marker not bound exactly")
	}
	if len(p.Args) < 5 || p.Args[len(p.Args)-3] != "--first" || p.Args[len(p.Args)-2] != "--contract" {
		t.Fatal("purpose not fixed")
	}
	o, e := probeprotocol.Decode(p.Args[len(p.Args)-1], now)
	if e != nil || o.Repository != s.repository() || !o.Deadline.Equal(s.Deadline) {
		t.Fatal("original contract changed", e)
	}
	for _, change := range []func(*ProbeSpec){func(s *ProbeSpec) { s.FirstObservation = nil }, func(s *ProbeSpec) { s.FirstObservation.CanaryID = "../foreign" }, func(s *ProbeSpec) { s.FirstObservation.CanaryRoot = "/var/lib/issuer/ledger/vrooli-isolation-canary" }, func(s *ProbeSpec) {
		s.FirstObservation.CanaryRoot = s.Manifest.WorkspaceSource + "/vrooli-isolation-canary"
	}, func(s *ProbeSpec) { s.FirstObservation.CanaryRoot = "/arbitrary-path" }, func(s *ProbeSpec) { s.Manifest.NetworkAllow = []string{"8.8.8.8/32"} }, func(s *ProbeSpec) { s.Deadline = now.Add(5*time.Minute + time.Second) }, func(s *ProbeSpec) { s.Purpose = ContinuingWitness }} {
		copy, _ := cloneSpec(s)
		change(&copy)
		if _, e := copy.BootstrapPlan(context.Background(), now); e == nil {
			t.Fatal("broader observation accepted")
		}
	}
}
func TestContinuingWitnessCheckPolicyHasNoHostEffects(t *testing.T) {
	s, now := probeFixture()
	binding := s.Manifest.Bindings["canary"]
	p := effortauthority.Policy{ID: "canary", Owner: "owner", Client: "client", ClientKey: make([]byte, 32), Epoch: 1, Repository: binding.Repository, Effort: "effort", Revision: "revision", ContentDigest: binding.PolicyDigest, Team: "team", TeamDigest: binding.PolicyDigest, BindingDigest: binding.PolicyDigest, Members: []string{"leader"}, Profiles: map[string]string{"bounded": binding.ProfileDigest}, Scopes: []string{"agent-manager:write"}, Effects: []string{"run.create"}, NotBefore: now, Deadline: binding.Deadline, MaxStarts: 1, MaxConcurrent: 1, MaxTurns: 1, MaxToolCalls: 1, MaxRunSeconds: 60, TotalTurns: 1, TotalToolCalls: 1, TotalRunSeconds: 60, SurviveLogout: true}
	binding.PolicyDigest = effortauthority.Digest(p)
	s.Manifest.Bindings["canary"] = binding
	reader := &fixtureWitnessReader{}
	w := &Witness{spec: s, reader: reader}
	if w.CheckPolicy(p) != nil {
		t.Fatal("exact startup policy refused")
	}
	for _, name := range []string{"nil", "first", "id", "digest", "repository", "deadline", "profile", "multi-profile"} {
		t.Run(name, func(t *testing.T) {
			copy, _ := cloneSpec(s)
			candidate := &Witness{spec: copy, reader: reader}
			offered := p
			offered.Profiles = map[string]string{"bounded": binding.ProfileDigest}
			switch name {
			case "nil":
				candidate = nil
			case "first":
				candidate.spec.Purpose = FirstQualification
			case "id":
				offered.ID = "foreign"
			case "digest":
				b := candidate.spec.Manifest.Bindings["canary"]
				b.PolicyDigest = effortauthority.Digest("foreign")
				candidate.spec.Manifest.Bindings["canary"] = b
			case "repository":
				offered.Repository = "/foreign"
			case "deadline":
				offered.Deadline = offered.Deadline.Add(time.Second)
			case "profile":
				offered.Profiles["bounded"] = effortauthority.Digest("foreign")
			case "multi-profile":
				offered.Profiles["foreign"] = binding.ProfileDigest
			}
			if candidate.CheckPolicy(offered) == nil {
				t.Fatal("different policy accepted")
			}
		})
	}
	if reader.reads != 0 || reader.qualifications != 0 {
		t.Fatal("policy comparison performed host effects")
	}
}
