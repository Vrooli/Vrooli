//go:build !windows

package privsep

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapGroupsKeepsThePrimaryGroupWithinTheLimit(t *testing.T) {
	groups := []uint32{20, 12, 61, 79, 80, 81, 702, 33, 98, 100, 204, 250, 395, 398, 399, 701, 33333}
	capped := capGroups(groups, 20, 16)
	require.Len(t, capped, 16)
	require.Equal(t, uint32(20), capped[0], "the primary group is kept first")
	require.NotContains(t, capped, uint32(33333))
	require.Equal(t, groups, capGroups(groups, 20, 0), "no limit keeps every group")
	short := []uint32{20, 12}
	require.Equal(t, short, capGroups(short, 20, 16))
}
