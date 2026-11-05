package trust

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// [REQ:SWBD-P0-009] [REQ:SWBD-P0-010] [REQ:SWBD-P0-011] [REQ:SWBD-P1-007]
func TestResolveNeverWidens(t *testing.T) {
	r := Resolve(Known, Owner, Grant{Scopes: []string{"read", "owner"}})
	require.NotContains(t, r.Scopes, "owner")
	require.Contains(t, r.Scopes, "read")
}

func TestResolveEmptyFailsClosed(t *testing.T) {
	r := Resolve(Stranger, Stranger, Grant{Scopes: []string{"owner"}})
	require.Empty(t, r.Scopes)
}
