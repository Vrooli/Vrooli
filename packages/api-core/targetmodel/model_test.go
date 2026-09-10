package targetmodel

import (
	"strings"
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

func TestSelectByIDUsesStableIdentityWhenLabelsCollide(t *testing.T) {
	inventory := Inventory{Targets: []Target{
		{ID: "target-b", Label: "Shared host", Platform: "desktop", OS: "linux", Available: true, Transport: Transport{Kind: TransportBridge}},
		{ID: "target-a", Label: "Shared host", Platform: "desktop", OS: "linux", Available: true, Transport: Transport{Kind: TransportBridge}},
	}}
	for _, id := range []string{"target-a", "target-b"} {
		selected := SelectByID(inventory, id)
		if !selected.Found || !selected.Available || selected.Target.ID != id || selected.Target.Label != "Shared host" {
			t.Fatalf("selection for %q = %+v, want the exact durable identity", id, selected)
		}
	}
	if err := inventory.Validate(); err != nil {
		t.Fatalf("same-label inventory should validate: %v", err)
	}
}

func TestSelectByIDRefusesAmbiguousDurableIdentity(t *testing.T) {
	inventory := Inventory{Targets: []Target{
		{ID: "duplicate", Label: "A", Platform: "desktop", OS: "linux", Available: true, Transport: Transport{Kind: TransportBridge}},
		{ID: "duplicate", Label: "B", Platform: "desktop", OS: "linux", Available: true, Transport: Transport{Kind: TransportBridge}},
	}}
	selected := SelectByID(inventory, "duplicate")
	if !selected.Found || selected.Available || selected.Reason != "target identity \"duplicate\" is ambiguous" {
		t.Fatalf("ambiguous selection = %+v, want fail-closed refusal", selected)
	}
	if err := inventory.Validate(); err == nil {
		t.Fatal("inventory validation accepted duplicate durable identity")
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

func TestCanHostSessionRequiresBridgeAgentAndWriteGrant(t *testing.T) {
	base := Target{
		ID: "node-1", Platform: "bridge", DeviceKind: "bridge-node",
		Transport:   Transport{Kind: TransportBridge, Available: true},
		BridgeTrust: &BridgeTrust{Registered: true, Online: true},
	}
	if ok, reason := CanHostSession(base); ok || reason == "" {
		t.Fatalf("missing grant = (%t, %q), want refusal with reason", ok, reason)
	}
	base.Scopes = []string{"vrooli-bridge:write"}
	if ok, reason := CanHostSession(base); !ok || reason != "" {
		t.Fatalf("write grant = (%t, %q), want admitted", ok, reason)
	}
	base.DeviceKind = "ssh"
	if ok, reason := CanHostSession(base); ok || reason == "" {
		t.Fatalf("ssh target = (%t, %q), want explicit unsupported reason", ok, reason)
	}
}

func TestOperationReadinessSeparatesHeadlessAndVisualPrerequisites(t *testing.T) {
	now := time.Date(2026, 9, 9, 23, 0, 0, 0, time.UTC)
	target := Target{
		ID: "mac-node", OS: "darwin", DeviceKind: "bridge-node",
		LastSeenAt:  now.Add(-time.Second),
		Transport:   Transport{Kind: TransportBridge, Available: true},
		BridgeTrust: &BridgeTrust{Registered: true, Online: true},
		Scopes:      []string{"vrooli-bridge:write"},
		Readiness: []ReadinessCheck{
			ReadinessCheckFor(ReadinessRegistry, true, "registered"),
			ReadinessCheckFor(ReadinessHeartbeat, true, "fresh"),
			ReadinessCheckFor(ReadinessChannel, true, "held"),
			ReadinessCheckFor(ReadinessProtocol, true, "compatible"),
			ReadinessCheckFor(ReadinessDispatch, true, "dispatchable"),
			ReadinessCheckFor(ReadinessBridgeScope, true, "approved"),
		},
	}
	headless := EvaluateOperationReadiness(target, OperationHeadlessExecution, now)
	if !headless.Ready || headless.State != ReadinessReady {
		t.Fatalf("headless readiness = %+v, want ready without GUI", headless)
	}
	visual := EvaluateOperationReadiness(target, OperationVisualValidation, now)
	if visual.Ready || visual.ReasonCode != "gui_session_missing" || !strings.Contains(visual.Detail, "GUI") {
		t.Fatalf("visual readiness = %+v, want explicit missing GUI session", visual)
	}
}

func TestOperationReadinessRefusesStaleFactsAndReportsFreshness(t *testing.T) {
	now := time.Date(2026, 9, 9, 23, 0, 0, 0, time.UTC)
	target := Target{
		ID: "node", OS: "linux", DeviceKind: "bridge-node", LastSeenAt: now.Add(-time.Minute),
		Transport:   Transport{Kind: TransportBridge, Available: true},
		BridgeTrust: &BridgeTrust{Registered: true, Online: true}, Scopes: []string{"vrooli-bridge:write"},
		Readiness: []ReadinessCheck{
			ReadinessCheckFor(ReadinessRegistry, true, "registered"),
			ReadinessCheckFor(ReadinessHeartbeat, false, "last heartbeat is stale"),
			ReadinessCheckFor(ReadinessChannel, true, "held"),
			ReadinessCheckFor(ReadinessProtocol, true, "compatible"),
			ReadinessCheckFor(ReadinessDispatch, true, "dispatchable"),
			ReadinessCheckFor(ReadinessBridgeScope, true, "approved"),
		},
	}
	decision := EvaluateOperationReadiness(target, OperationHeadlessExecution, now)
	if decision.Ready || decision.State != ReadinessUnknown || decision.ReasonCode != "heartbeat_stale" {
		t.Fatalf("stale readiness = %+v, want unknown heartbeat refusal", decision)
	}
	if !decision.ObservedAt.Equal(target.LastSeenAt) || !decision.FreshUntil.Equal(target.LastSeenAt.Add(DefaultReadinessStaleAfter)) {
		t.Fatalf("freshness window = %s..%s, want %s..%s", decision.ObservedAt, decision.FreshUntil, target.LastSeenAt, target.LastSeenAt.Add(DefaultReadinessStaleAfter))
	}
}

func TestOperationReadinessUsesOneRevokedGrantDecisionAcrossOperations(t *testing.T) {
	now := time.Date(2026, 9, 9, 23, 0, 0, 0, time.UTC)
	target := Target{
		ID: "revoked", OS: "linux", DeviceKind: "bridge-node", Revoked: true,
		Transport:   Transport{Kind: TransportBridge, Available: true},
		BridgeTrust: &BridgeTrust{Registered: true, Online: true},
		Scopes:      []string{"vrooli-bridge:write"},
	}
	for _, decision := range EvaluateOperations(target, now) {
		if decision.Ready || decision.ReasonCode != "grant_revoked" {
			t.Fatalf("%s readiness = %+v, want shared grant_revoked refusal", decision.Operation, decision)
		}
	}
}

func TestSelectCanRequireAnOperationWithoutBlockingHeadlessTargetsOnGUI(t *testing.T) {
	target := Target{
		ID: "mac", Label: "Mac", OS: "darwin", DeviceKind: "bridge-node", Available: true,
		Transport: Transport{Kind: TransportBridge, Available: true}, BridgeTrust: &BridgeTrust{Registered: true, Online: true},
		Scopes: []string{"vrooli-bridge:write"}, Readiness: []ReadinessCheck{
			ReadinessCheckFor(ReadinessRegistry, true, ""), ReadinessCheckFor(ReadinessHeartbeat, true, ""),
			ReadinessCheckFor(ReadinessChannel, true, ""), ReadinessCheckFor(ReadinessProtocol, true, ""),
			ReadinessCheckFor(ReadinessDispatch, true, ""), ReadinessCheckFor(ReadinessBridgeScope, true, ""),
		},
	}
	headless := Select(Inventory{Targets: []Target{target}}, SelectionRequest{OS: "darwin", Operation: OperationHeadlessExecution})
	if !headless.Available {
		t.Fatalf("headless selection = %+v, want available", headless)
	}
	visual := Select(Inventory{Targets: []Target{target}}, SelectionRequest{OS: "darwin", Operation: OperationVisualValidation})
	if visual.Available || visual.Reason != "visual validation requires an active GUI session" {
		t.Fatalf("visual selection = %+v, want explicit GUI refusal", visual)
	}
}

func TestSSHTransportIsAValidTargetThatCannotHostSessions(t *testing.T) {
	target := Target{ID: "host:203.0.113.10", Platform: "vps", OS: "linux", DeviceKind: "host", Available: true, Transport: Transport{Kind: TransportSSH, ID: "host:203.0.113.10", Available: true}}
	if err := target.Validate(); err != nil {
		t.Fatalf("ssh target must validate: %v", err)
	}
	if ok, reason := CanHostSession(target); ok || reason == "" {
		t.Fatalf("ssh target must not host sessions: ok=%v reason=%q", ok, reason)
	}
	bad := target
	bad.Transport.Kind = TransportKind("telnet")
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown transport must be refused")
	}
}
