package vroolicli

import "testing"

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
