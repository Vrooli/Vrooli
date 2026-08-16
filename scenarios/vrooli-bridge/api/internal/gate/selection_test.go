package gate

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
)

type selectionPresence struct {
	online       map[string]bool
	dispatchable map[string]bool
}

func (p selectionPresence) IsOnline(nodeID string) bool { return p.online[nodeID] }

func (p selectionPresence) Dispatchable(nodeID string) bool { return p.dispatchable[nodeID] }

func TestEligibleByOSAgreesWithSharedSelector(t *testing.T) {
	healthy := targetmodel.Target{
		ID: "node-linux", NodeID: "node-linux", Platform: "desktop", OS: "linux",
		Transport: targetmodel.Transport{Kind: targetmodel.TransportBridge}, Available: true,
		Reason:      targetmodel.ReasonBridgeAuthorizedDesktop,
		BridgeTrust: &targetmodel.BridgeTrust{Registered: true, DispatchAuthorized: true},
	}
	offline := targetmodel.Target{
		ID: "node-darwin", NodeID: "node-darwin", Platform: "desktop", OS: "darwin",
		Transport: targetmodel.Transport{Kind: targetmodel.TransportBridge}, Available: true,
		Reason:      targetmodel.ReasonBridgeAuthorizedDesktop,
		BridgeTrust: &targetmodel.BridgeTrust{Registered: true, DispatchAuthorized: true},
	}

	presence := selectionPresence{
		online:       map[string]bool{"node-linux": true, "node-darwin": false},
		dispatchable: map[string]bool{"node-linux": true, "node-darwin": false},
	}
	svc := &service{presence: presence}
	got := svc.eligibleByOS([]NodeRef{
		{ID: healthy.ID, OS: healthy.OS, Target: healthy},
		{ID: offline.ID, OS: offline.OS, Target: offline},
	})

	expectedOffline := offline
	expectedOffline.Available = false
	expectedOffline.Reason = targetmodel.ReasonBridgeOffline
	expectedOffline.MissingCapability = "bridge dispatch reachability"
	expectedOffline.NextAction = "restore the node channel and protocol compatibility"
	expected := targetmodel.SelectByOS(targetmodel.Inventory{Targets: []targetmodel.Target{healthy, expectedOffline}}, []string{"linux", "darwin"}, targetmodel.SelectionRequest{TransportKinds: []targetmodel.TransportKind{targetmodel.TransportBridge}})

	require.Equal(t, expected["linux"].Target.ID, got["linux"].Target.ID)
	require.Equal(t, expected["linux"].Reason, got["linux"].Reason)
	require.Equal(t, expected["darwin"].Target.ID, got["darwin"].Target.ID)
	require.Equal(t, expected["darwin"].Reason, got["darwin"].Reason)
	require.Equal(t, expected["darwin"].NextAction, got["darwin"].NextAction)
}
