package policy

import "testing"

func TestKeepAliveSecondsPrefersTheModelThenResidencyThenIdleUnload(t *testing.T) {
	p := Policy{
		Upstreams: map[string]Upstream{"local": {Locality: LocalityLocal, Protocol: ProtocolOllaya}},
		Roles:     map[string]Role{"decide.local": {Upstream: "local", Model: "router"}, "decide.big": {Upstream: "local", Model: "big:4b"}},
		Models: map[string]Model{
			"router":    {Routes: []string{"router:en"}},
			"router:en": servable(""),
			"big:4b":    {Placement: &ModelPlacement{Device: PlacementDeviceGPU, VRAMBytes: 1, KeepAlive: "90s"}},
			"other:1b":  servable(""),
		},
		Residency: Residency{ResidentRoles: []string{"decide.local"}, IdleUnloadSeconds: 300},
	}
	cases := map[string]int{
		// A model's own declaration is the most specific and wins.
		"big:4b@sha256:" + "ab": 90,
		// A resident role's model and the targets the daemon loads for it stay.
		"router":    KeepAliveForever,
		"router:en": KeepAliveForever,
		// Everything else, including a model the policy does not pin, unloads
		// when idle.
		"other:1b":       300,
		"OTHER:1b ":      300,
		"unlisted:model": 300,
	}
	for model, want := range cases {
		if got, ok := p.KeepAliveSeconds(model); !ok || got != want {
			t.Errorf("KeepAliveSeconds(%q) = %d, %v; want %d", model, got, ok, want)
		}
	}
	p.Residency = Residency{}
	if got, ok := p.KeepAliveSeconds("other:1b"); ok {
		t.Fatalf("with no idle unload declared, KeepAliveSeconds = %d; want the daemon default", got)
	}
	if got, ok := p.KeepAliveSeconds("big:4b"); !ok || got != 90 {
		t.Fatalf("a model's own keep_alive must apply without a residency block; got %d, %v", got, ok)
	}
}

// Every model the shipped policy can load must be on a policy schedule, so no
// model falls back to the daemon-wide default and holds memory for it.
func TestShippedPolicyPutsEveryModelOnAnIdleUnload(t *testing.T) {
	p := shippedPolicy(t)
	if p.Residency.IdleUnloadSeconds < minIdleUnloadSeconds || p.Residency.Rationale == "" {
		t.Fatalf("residency = %+v, want a measured idle unload with its rationale", p.Residency)
	}
	for name := range p.Models {
		got, ok := p.KeepAliveSeconds(name)
		if !ok || got == 0 {
			t.Errorf("model %s keep_alive = %d, %v; want a policy schedule", name, got, ok)
		}
		if got == KeepAliveForever && len(p.Residency.ResidentRoles) == 0 {
			t.Errorf("model %s stays resident although no role is resident", name)
		}
	}
}
