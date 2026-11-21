package execplan

import "testing"

func TestManagedDNSIsExplicitOptInAndPrecedesEdgeRoute(t *testing.T) {
	in := baseInputs()
	in.Manifest.Edge.ManagedDNSProfile = "cloudflare-production"
	plan := compile(t, in)
	dns := plan.Action(OpEdgeDNSEnsure)
	route := plan.Action(OpEdgeRouteApply)
	if dns == nil || route == nil {
		t.Fatalf("expected managed DNS and edge route actions, got %v", plan.ActionIDs())
	}
	if dns.Inputs["provider_profile"] != "cloudflare-production" || dns.Inputs["record_type"] != "A" || dns.Inputs["content"] != "203.0.113.10" {
		t.Fatalf("unexpected DNS inputs: %+v", dns.Inputs)
	}
	if len(route.DependsOn) != 2 || route.DependsOn[1] != OpEdgeDNSEnsure {
		t.Fatalf("edge route must wait for managed DNS: %+v", route.DependsOn)
	}

	in.Manifest.Edge.ManagedDNSProfile = ""
	without := compile(t, in)
	if without.Action(OpEdgeDNSEnsure) != nil {
		t.Fatal("managed DNS must remain absent without explicit profile")
	}
}
