package targetmodel

import "testing"

func TestSelectUsesStableIdentityAndSharedAvailabilityReasons(t *testing.T) {
	inventory := Inventory{Targets: []Target{
		{ID: "node-b", Platform: "desktop", OS: "linux", Available: true, Capabilities: []string{CapabilityCDP}, Transport: Transport{Kind: TransportBridge}},
		{ID: "node-a", Platform: "desktop", OS: "linux", Available: true, Capabilities: []string{CapabilityCDP}, Transport: Transport{Kind: TransportBridge}},
		{ID: "node-offline", Platform: "desktop", OS: "darwin", Available: false, Reason: "bridge node is offline or not dispatchable", MissingCapability: "bridge dispatch reachability", NextAction: "restore bridge dispatchability", Transport: Transport{Kind: TransportBridge}},
	}}

	selected := Select(inventory, SelectionRequest{OS: "linux", RequiredCapabilities: []string{CapabilityCDP}})
	if !selected.Found || !selected.Available || selected.Target.ID != "node-a" {
		t.Fatalf("linux selection = %+v, want available node-a", selected)
	}

	offline := Select(inventory, SelectionRequest{OS: "darwin"})
	if !offline.Found || offline.Available || offline.Reason != "bridge node is offline or not dispatchable" {
		t.Fatalf("darwin selection = %+v, want the inventory availability reason", offline)
	}
	if offline.NextAction != "restore bridge dispatchability" {
		t.Fatalf("darwin next action = %q", offline.NextAction)
	}
}

func TestSelectReportsMissingCapabilityAndNoTarget(t *testing.T) {
	inventory := Inventory{Targets: []Target{{
		ID: "node-1", Platform: "desktop", OS: "linux", Available: true,
		Capabilities: []string{CapabilityCDP}, Transport: Transport{Kind: TransportBridge},
	}}}

	missing := Select(inventory, SelectionRequest{OS: "linux", RequiredCapabilities: []string{CapabilityNativeWindow}})
	if !missing.Found || missing.Available || missing.Reason == "" || missing.NextAction == "" {
		t.Fatalf("missing capability selection = %+v", missing)
	}

	none := Select(inventory, SelectionRequest{OS: "windows"})
	if none.Found || none.Available || none.Reason == "" || none.NextAction == "" {
		t.Fatalf("no target selection = %+v", none)
	}
}

func TestScopeAllowsUsesBridgeCommandGrammar(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		verb   string
		want   bool
	}{
		{name: "exact", scopes: []string{"scenario status"}, verb: "scenario status", want: true},
		{name: "prefix", scopes: []string{"scenario test*"}, verb: "scenario test", want: true},
		{name: "prefix does not cross", scopes: []string{"scenario test*"}, verb: "scenario start", want: false},
		{name: "universal", scopes: []string{"*"}, verb: "scenario status", want: true},
		{name: "empty", scopes: []string{"*"}, verb: "", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ScopeAllows(tc.scopes, tc.verb); got != tc.want {
				t.Fatalf("ScopeAllows(%q, %q) = %v, want %v", tc.scopes, tc.verb, got, tc.want)
			}
		})
	}
}
