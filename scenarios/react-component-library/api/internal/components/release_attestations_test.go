package components

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// The committed registry is consulted only to tell a stale index apart from a
// real rewrite. It must vouch for the exact bytes on disk, at the exact path,
// or it grants nothing.
func TestAttestsReleaseOnlyAcceptsExactAttestedBytes(t *testing.T) {
	attestations := map[string]string{
		"components/Portal/versions/1.1.1/Portal.tsx": "abc123",
	}

	require.True(t, attestsRelease(attestations, "components/Portal/versions/1.1.1/Portal.tsx", "abc123"),
		"bytes the registry names at this path are attested")
	require.True(t, attestsRelease(attestations, "components/Portal/versions/1.1.1/./Portal.tsx", "abc123"),
		"path comparison is normalised")

	require.False(t, attestsRelease(attestations, "components/Portal/versions/1.1.1/Portal.tsx", "different"),
		"a rewrite the registry does not name is not attested")
	require.False(t, attestsRelease(attestations, "components/Other/versions/1.0.0/Other.tsx", "abc123"),
		"an attestation for one path must not vouch for another")
	require.False(t, attestsRelease(nil, "components/Portal/versions/1.1.1/Portal.tsx", "abc123"),
		"with no registry the index stays the sole authority")
	require.False(t, attestsRelease(attestations, "components/Portal/versions/1.1.1/Portal.tsx", ""),
		"empty incoming bytes never match")
}

func TestLoadReleaseAttestationsToleratesAMissingRegistry(t *testing.T) {
	got, err := loadReleaseAttestations(fstest.MapFS{})
	require.NoError(t, err)
	require.Empty(t, got, "a library with no registry reconciles nothing rather than failing")
}

func TestLoadReleaseAttestationsRejectsAMalformedRegistry(t *testing.T) {
	_, err := loadReleaseAttestations(fstest.MapFS{
		"released-version-hashes.json": {Data: []byte("{ not json")},
	})
	require.Error(t, err, "an unreadable registry must not silently become an empty one")
}

func TestLoadReleaseAttestationsSkipsIncompleteEntries(t *testing.T) {
	got, err := loadReleaseAttestations(fstest.MapFS{
		"released-version-hashes.json": {Data: []byte(`{"schemaVersion":1,"entries":[
			{"path":"components/A/versions/1.0.0/A.tsx","sha256":"aaa"},
			{"path":"","sha256":"bbb"},
			{"path":"components/C/versions/1.0.0/C.tsx","sha256":""}
		]}`)},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"components/A/versions/1.0.0/A.tsx": "aaa"}, got)
}
