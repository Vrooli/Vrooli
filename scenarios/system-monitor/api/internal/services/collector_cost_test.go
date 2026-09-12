package services

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestCollectionProfileForHost(t *testing.T) {
	tests := []struct {
		name     string
		snapshot hostinventory.Snapshot
		want     CollectionProfile
	}{
		{name: "raspberry-pi-class-cpu", snapshot: hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 2}}, want: CollectionProfileLowPower},
		{name: "small-memory-host", snapshot: hostinventory.Snapshot{Memory: hostinventory.Memory{TotalBytes: 1 * 1024 * 1024 * 1024}}, want: CollectionProfileLowPower},
		{name: "four-core-boundary", snapshot: hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 4}}, want: CollectionProfileStandard},
		{name: "two-gib-boundary", snapshot: hostinventory.Snapshot{Memory: hostinventory.Memory{TotalBytes: 2 * 1024 * 1024 * 1024}}, want: CollectionProfileStandard},
		{name: "missing-inventory-is-not-low-power", snapshot: hostinventory.Snapshot{}, want: CollectionProfileStandard},
		{name: "workstation", snapshot: hostinventory.Snapshot{CPU: hostinventory.CPU{Cores: 8}, Memory: hostinventory.Memory{TotalBytes: 16 * 1024 * 1024 * 1024}}, want: CollectionProfileStandard},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CollectionProfileForHost(tc.snapshot); got != tc.want {
				t.Fatalf("profile = %q, want %q", got, tc.want)
			}
		})
	}
}
