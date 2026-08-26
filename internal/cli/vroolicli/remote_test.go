package vroolicli

import "testing"

func TestRelayTimeoutArgMirrorsForwardedLifecycleCeiling(t *testing.T) {
	if got := relayTimeoutArg([]string{"--best-effort", "--timeout", "600", "--json"}); got != "600" {
		t.Fatalf("relayTimeoutArg = %q, want 600", got)
	}
}

func TestRelayTimeoutArgIgnoresInvalidOrMissingValues(t *testing.T) {
	for _, args := range [][]string{
		{"--timeout"},
		{"--timeout", "0"},
		{"--timeout", "-1"},
		{"--timeout", "slow"},
	} {
		if got := relayTimeoutArg(args); got != "" {
			t.Fatalf("relayTimeoutArg(%v) = %q, want empty", args, got)
		}
	}
}

func TestSelectBridgeNodePrefersTheOnlyReadyRecord(t *testing.T) {
	id, err := selectBridgeNode([]bridgeNodeSummary{
		{ID: "old", Name: "minimouse", Online: false},
		{ID: "ready", Name: "minimouse", Online: true, Dispatchable: true},
	}, "minimouse")
	if err != nil {
		t.Fatalf("selectBridgeNode: %v", err)
	}
	if id != "ready" {
		t.Fatalf("selected %q, want ready", id)
	}
}

func TestSelectBridgeNodeUsesNewestOfflineRecordForAuthoritativeReason(t *testing.T) {
	id, err := selectBridgeNode([]bridgeNodeSummary{
		{ID: "one", Name: "minimouse"},
		{ID: "two", Name: "minimouse"},
	}, "minimouse")
	if err != nil {
		t.Fatalf("selectBridgeNode: %v", err)
	}
	if id != "one" {
		t.Fatalf("selected %q, want newest record one", id)
	}
}
