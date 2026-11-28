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

func TestMemoryStoreDoesNotDowngradeGenerationOnReplay(t *testing.T) {
	s := NewMemoryStore()
	current := Grant{ID: "g-current", NodeID: "n1", LogicalID: "vrooli/test", Field: "api-key", Class: ClassUserPrompt, Retention: RetentionEphemeral, Generation: 3}
	stale := current
	stale.ID = "g-stale"
	stale.Generation = 2
	require.NoError(t, s.Put(current))
	require.NoError(t, s.Put(stale))

	got, ok := s.Lookup("vrooli/test", "api-key")
	require.True(t, ok)
	require.Equal(t, current.ID, got.ID)
	require.Equal(t, current.Generation, got.Generation)
}

func TestGenerationBoundRevocationTombstoneSurvivesReplayAndAllowsNewerGrant(t *testing.T) {
	s := NewMemoryStore()
	revoked := Grant{ID: "g1", NodeID: "n1", LogicalID: "service/api", Field: "token", Generation: 4}
	require.NoError(t, s.RevokeGrant(revoked))
	got := s.List()
	require.Len(t, got, 1)
	require.True(t, got[0].Revoked)
	require.False(t, func() bool { _, ok := s.Lookup("service/api", "token"); return ok }())

	// Delayed metadata for the revoked generation must not resurrect consent.
	require.NoError(t, s.Put(Grant{ID: "g1", NodeID: "n1", LogicalID: "service/api", Field: "token", Class: ClassUserPrompt, Retention: RetentionEphemeral, Generation: 4}))
	got = s.List()
	require.Len(t, got, 1)
	require.True(t, got[0].Revoked)

	// An older revoke cannot cancel a newer grant.
	require.NoError(t, s.Put(Grant{ID: "g2", NodeID: "n1", LogicalID: "service/api", Field: "token", Class: ClassUserPrompt, Retention: RetentionEphemeral, Generation: 5}))
	require.NoError(t, s.RevokeGrant(revoked))
	active, ok := s.Lookup("service/api", "token")
	require.True(t, ok)
	require.Equal(t, "g2", active.ID)
	require.Equal(t, int64(5), active.Generation)
}

func TestFileStorePersistsRevocationTombstoneAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grants.json")
	first, err := LoadFile(path)
	require.NoError(t, err)
	require.NoError(t, first.RevokeGrant(Grant{ID: "g1", NodeID: "n1", LogicalID: "service/api", Field: "token", Generation: 4}))

	restarted, err := LoadFile(path)
	require.NoError(t, err)
	grants := restarted.List()
	require.Len(t, grants, 1)
	require.True(t, grants[0].Revoked)
	require.Equal(t, int64(4), grants[0].Generation)
	require.False(t, func() bool { _, ok := restarted.Lookup("service/api", "token"); return ok }())
	// Replay after restart leaves the tombstone intact.
	require.NoError(t, restarted.Put(Grant{ID: "g1", NodeID: "n1", LogicalID: "service/api", Field: "token", Class: ClassUserPrompt, Retention: RetentionEphemeral, Generation: 4}))
	replayed := restarted.List()
	require.Len(t, replayed, 1)
	require.True(t, replayed[0].Revoked)
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
