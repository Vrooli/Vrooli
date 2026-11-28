package control

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// The Off request names the complete enrolled wiring. A new backend count on
// another addressable header cannot authorize a controller-wide mode mutation.
func TestAdversarialBOffScopeRejectsPeerWiringDrift(t *testing.T) {
	expected := map[string]uint32{"strip|0": 24, "peer|0": 12}
	c := SDKController{Zones: []SDKZone{{Name: "Strip", LEDCount: 24, LEDMax: 500, Flags: 2}, {Name: "Peer", LEDCount: 12, LEDMax: 500, Flags: 2}}}
	require.True(t, completeControllerOffScope(c, expected))
	c.Zones[1].LEDCount = 500
	require.False(t, completeControllerOffScope(c, expected), "peer wiring changed on the mutation connection")
	c.Zones[1].LEDCount = 0
	require.False(t, completeControllerOffScope(c, expected))
	c.Zones[1].LEDCount = 12
	c.Zones = append(c.Zones, SDKZone{Name: "Unexpected", LEDCount: 1})
	require.False(t, completeControllerOffScope(c, expected))
}

func TestAdversarialBOffScopeCannotSpendOnePeerWitnessTwice(t *testing.T) {
	c := SDKController{Zones: []SDKZone{{Name: "Strip", LEDCount: 24, LEDMax: 500, Flags: 2}, {Name: "Strip", LEDCount: 24, LEDMax: 500, Flags: 2}}}
	require.False(t, completeControllerOffScope(c, map[string]uint32{"strip|0": 24, "peer|0": 24}))
}
