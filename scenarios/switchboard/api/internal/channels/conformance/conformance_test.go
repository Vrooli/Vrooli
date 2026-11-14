package conformance

import (
	"context"
	"github.com/stretchr/testify/require"
	"switchboard/internal/channels"
	"switchboard/internal/channels/adapters/fake"
	"testing"
)

// [REQ:SWBD-P0-003] [REQ:SWBD-P0-004] [REQ:SWBD-P0-014] [REQ:SWBD-P0-015] [REQ:SWBD-P0-016] [REQ:SWBD-P1-010] [REQ:SWBD-P1-011]
func TestEveryCasePassesWithoutAdapterBranches(t *testing.T) {
	a := fake.New("fixture")
	d := channels.Descriptor{Kind: "channel", SchemaVersion: 1, ID: "fixture", DisplayName: "Fixture", Transport: "fixture", Cost: "free", Limits: channels.Limits{MaxMediaBytes: 100}}
	for _, c := range Run(context.Background(), a, d) {
		require.True(t, c.Passed, "%s: %s", c.Name, c.Detail)
	}
}
