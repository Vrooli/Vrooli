package targetmodel

import (
	"testing"
	"time"
)

func TestSelectUsesStableIdentityAndSharedAvailabilityReasons(t *testing.T) {
	inventory := Inventory{Targets: []Target{
		{ID: "node-b", Platform: "desktop", OS: "linux", Available: true, Capabilities: []string{"cdp"}, Transport: Transport{Kind: TransportBridge}},
		{ID: "node-a", Platform: "desktop", OS: "linux", Available: true, Capabilities: []string{"cdp"}, Transport: Transport{Kind: TransportBridge}},
		{ID: "node-offline", Platform: "desktop", OS: "darwin", Available: false, Reason: "bridge node is offline or not dispatchable", MissingCapability: "bridge dispatch reachability", NextAction: "restore bridge dispatchability", Transport: Transport{Kind: TransportBridge}},
	}}

	selected := Select(inventory, SelectionRequest{OS: "linux", RequiredCapabilities: []string{"cdp"}})
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
		Capabilities: []string{"cdp"}, Transport: Transport{Kind: TransportBridge},
	}}}

	missing := Select(inventory, SelectionRequest{OS: "linux", RequiredCapabilities: []string{"native-window"}})
	if !missing.Found || missing.Available || missing.Reason == "" || missing.NextAction == "" {
		t.Fatalf("missing capability selection = %+v", missing)
	}

	none := Select(inventory, SelectionRequest{OS: "windows"})
	if none.Found || none.Available || none.Reason == "" || none.NextAction == "" {
		t.Fatalf("no target selection = %+v", none)
	}
}

func TestHeartbeatFreshRequiresTimestampAndBoundsAge(t *testing.T) {
	now := time.Date(2026, 8, 26, 20, 0, 0, 0, time.UTC)
	if fresh, age := HeartbeatFresh(time.Time{}, now, time.Minute); fresh || age != 0 {
		t.Fatalf("missing timestamp = (%t, %s), want false, zero", fresh, age)
	}
	if fresh, _ := HeartbeatFresh(now.Add(-2*time.Minute), now, time.Minute); fresh {
		t.Fatal("stale heartbeat reported fresh")
	}
	if fresh, age := HeartbeatFresh(now.Add(-30*time.Second), now, time.Minute); !fresh || age != 30*time.Second {
		t.Fatalf("fresh heartbeat = (%t, %s)", fresh, age)
	}
}

func TestReadinessCheckUsesStableIdentityAndRecovery(t *testing.T) {
	check := ReadinessCheckFor(ReadinessHeartbeat, false, "last seen 7 days ago")
	if check.Label != "Heartbeat fresh" || check.Passed || check.RecoveryAction == "" {
		t.Fatalf("heartbeat check = %+v", check)
	}
	unknown := ReadinessCheckFor("future_check", false, "detail")
	if unknown.Label != "future_check" || unknown.RecoveryAction == "" {
		t.Fatalf("unknown check = %+v", unknown)
	}
}
