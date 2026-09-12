//go:build linux

package collectors

import (
	"os"
	"testing"
)

func TestParseBuddyInfoFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/buddyinfo_linux.txt")
	if err != nil {
		t.Fatal(err)
	}
	got := parseBuddyInfo(string(raw))
	if got.minOrder != 6 {
		t.Fatalf("minimum highest free order = %d, want 6", got.minOrder)
	}
	if got.values["fragmentation_max_free_order"] != 6 {
		t.Fatalf("max order payload = %#v", got.values)
	}
	if _, ok := got.values["buddyinfo_Normal"]; !ok {
		t.Fatalf("Normal zone missing: %#v", got.values)
	}
	if share, ok := got.values["fragmentation_low_order_share"].(float64); !ok || share < 0 || share > 1 {
		t.Fatalf("low-order share = %#v", got.values["fragmentation_low_order_share"])
	}
}

func TestParseBuddyInfoFullyFragmentedZone(t *testing.T) {
	got := parseBuddyInfo("Node 0, zone Normal 7 0 0 0 0 0 0 0 0 0 0\n")
	if got.minOrder != 0 {
		t.Fatalf("fully fragmented index = %d, want 0", got.minOrder)
	}
}
