package vroolicli

import (
	"context"
	"testing"
)

// TestHostInventoryDecodesQuotedByteCounts is the regression guard for the
// app-monitor memory bug: the CLI emits 64-bit byte counts as JSON *strings*
// ("65435598848"), which a plain encoding/json uint64 cannot parse. Decoding
// through the proto contract must round-trip the quoted number into uint64.
func TestHostInventoryDecodesQuotedByteCounts(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{output: []byte(`{"memory":{"total_bytes":"65435598848","available_bytes":"43553144832"},"swap":{"total_bytes":"74088177664"},"cpu":{"cores":32}}`)},
		},
	}
	client := New(WithRunner(runner))

	resp, err := client.HostInventory(context.Background())
	if err != nil {
		t.Fatalf("HostInventory returned error: %v", err)
	}
	if got := resp.GetMemory().GetTotalBytes(); got != 65435598848 {
		t.Fatalf("memory.total_bytes = %d, want 65435598848", got)
	}
	if got := resp.GetMemory().GetAvailableBytes(); got != 43553144832 {
		t.Fatalf("memory.available_bytes = %d, want 43553144832", got)
	}
	if got := resp.GetSwap().GetTotalBytes(); got != 74088177664 {
		t.Fatalf("swap.total_bytes = %d, want 74088177664", got)
	}
	if want := []string{"--no-stale-check", "host", "inventory", "--json"}; !equalArgs(runner.calls[0].args, want) {
		t.Fatalf("call args = %v, want %v", runner.calls[0].args, want)
	}
}

// TestStatusDecodesSummary guards the unified-status summary decode that
// app-monitor's getQuickSystemStatus relies on (the counts live under
// status.summary, not at the top level).
func TestStatusDecodesSummary(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{output: []byte(`{"success":true,"status":{"summary":{"resources_enabled":30,"resources_total":32,"scenarios_total":106,"scenarios_running":41},"resources":[{"ignored":"by_discard_unknown"}]}}`)},
		},
	}
	client := New(WithRunner(runner))

	resp, err := client.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	summary := resp.GetStatus().GetSummary()
	if got := summary.GetScenariosTotal(); got != 106 {
		t.Fatalf("scenarios_total = %d, want 106", got)
	}
	if got := summary.GetResourcesEnabled(); got != 30 {
		t.Fatalf("resources_enabled = %d, want 30", got)
	}
}

func equalArgs(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
