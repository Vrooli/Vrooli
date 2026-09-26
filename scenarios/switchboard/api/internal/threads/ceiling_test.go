package threads

import (
	"testing"

	"github.com/stretchr/testify/require"
	"switchboard/internal/trust"
)

// [REQ:SWBD-P1-001]
func TestRoomCeilingNarrows(t *testing.T) {
	require.Equal(t, trust.Trusted, RoomCeiling([]trust.Tier{trust.Owner, trust.Trusted}))
	require.Equal(t, trust.Known, RoomCeiling([]trust.Tier{trust.Trusted, trust.Known}))
}
