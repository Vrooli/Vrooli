package hostreq

import (
	"testing"
	"time"
)

func TestObservedSafeguardsExposeTypedMetadata(t *testing.T) {
	items, err := ListObservedSafeguards("/home/matthalloran8/Vrooli", func() time.Time { return time.Date(2026, 8, 22, 13, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 || items[0].Name == "" || items[0].Capability == "" || items[0].CapabilityRole == "" {
		t.Fatalf("expected typed safeguard metadata, got %#v", items[:minObserved(1, len(items))])
	}
}

func minObserved(want, got int) int {
	if got < want {
		return got
	}
	return want
}
