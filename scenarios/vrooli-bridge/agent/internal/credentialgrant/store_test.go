package credentialgrant

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryStoreRefusesRevokedGrant(t *testing.T) {
	s := NewMemoryStore(Grant{LogicalID: "vrooli/test", Field: "api-key", Class: ClassUserPrompt, Retention: RetentionEphemeral, Generation: 1})
	require.NotNil(t, s)
	require.NoError(t, s.Revoke("vrooli/test", "api-key"))
	_, ok := s.Lookup("vrooli/test", "api-key")
	require.False(t, ok)
}

func TestFileStorePersistsMetadataOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grants.json")
	s, err := LoadFile(path)
	require.NoError(t, err)
	require.NoError(t, s.Put(Grant{ID: "g1", NodeID: "n1", LogicalID: "vrooli/test", Field: "api-key", Class: ClassUserPrompt, Retention: RetentionDurable, Generation: 2}))
	reloaded, err := LoadFile(path)
	require.NoError(t, err)
	got, ok := reloaded.Lookup("vrooli/test", "api-key")
	require.True(t, ok)
	require.Equal(t, int64(2), got.Generation)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(data), "value")
}

func TestMemoryStoreEnforcesGrantClassRetentionPolicy(t *testing.T) {
	s := NewMemoryStore()
	require.ErrorContains(t, s.Put(Grant{LogicalID: "vrooli/test", Field: "token", Class: ClassInfrastructure, Retention: RetentionDurable, Generation: 1}), "cannot be durable")
	require.ErrorContains(t, s.Put(Grant{LogicalID: "vrooli/test", Field: "token", Class: ClassPerInstallGenerated, Retention: RetentionEphemeral, Generation: 1}), "cannot be distributed")
	require.NoError(t, s.Put(Grant{LogicalID: "vrooli/test", Field: "token", Class: ClassInfrastructure, Retention: RetentionEphemeral, Generation: 1}))
}
