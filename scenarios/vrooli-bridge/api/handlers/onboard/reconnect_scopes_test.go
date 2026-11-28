package onboard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/registry"
)

type pairingScopeReader struct {
	node registry.Node
	err  error
}

func (r pairingScopeReader) Get(context.Context, string) (registry.Node, error) {
	return r.node, r.err
}

func TestExistingPairingScopesPreservesCurrentApprovedGrants(t *testing.T) {
	h := NewConnectHandler(Deps{Registry: pairingScopeReader{node: registry.Node{
		ID: "node-old", Scopes: []string{"vrooli-bridge:read", "scenario-to-cloud:read"},
	}}})
	scopes, set, err := h.existingPairingScopes(context.Background(), machines.Machine{
		Lineage: []machines.NodeLineage{{NodeID: "node-old", Current: true}},
	})
	require.NoError(t, err)
	require.True(t, set)
	require.Equal(t, []string{"vrooli-bridge:read", "scenario-to-cloud:read"}, scopes)
}

func TestExistingPairingScopesDoesNotRestoreRevokedGrants(t *testing.T) {
	h := NewConnectHandler(Deps{Registry: pairingScopeReader{node: registry.Node{
		ID: "node-old", Scopes: []string{"vrooli-bridge:write"}, RevokedAt: time.Now(),
	}}})
	scopes, set, err := h.existingPairingScopes(context.Background(), machines.Machine{
		Lineage: []machines.NodeLineage{{NodeID: "node-old", Current: true}},
	})
	require.NoError(t, err)
	require.True(t, set)
	require.Empty(t, scopes)
}

func TestExistingPairingScopesRejectsRegistryIdentityMismatch(t *testing.T) {
	h := NewConnectHandler(Deps{Registry: pairingScopeReader{node: registry.Node{
		ID: "different-node", Scopes: []string{"unrelated:write"},
	}}})
	_, _, err := h.existingPairingScopes(context.Background(), machines.Machine{
		ID: "machine-1", Lineage: []machines.NodeLineage{{NodeID: "current-node", Current: true}},
	})
	require.ErrorContains(t, err, "registry returned node")
}

func TestExistingPairingScopesFailsClosedWhenRegistryUnavailable(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, _, err := h.existingPairingScopes(context.Background(), machines.Machine{
		ID: "machine-1", Lineage: []machines.NodeLineage{{NodeID: "current-node", Current: true}},
	})
	require.ErrorContains(t, err, "registry unavailable")
}

func TestExistingPairingScopesRejectsAmbiguousCurrentLineage(t *testing.T) {
	h := NewConnectHandler(Deps{Registry: pairingScopeReader{node: registry.Node{ID: "node-1"}}})
	_, _, err := h.existingPairingScopes(context.Background(), machines.Machine{
		ID: "machine-1", Lineage: []machines.NodeLineage{
			{NodeID: "node-1", Current: true}, {NodeID: "node-2", Current: true},
		},
	})
	require.ErrorContains(t, err, "multiple current node identities")
}
