package components

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
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

func TestReleaseRepairCoversTheExactPublishedDigest(t *testing.T) {
	fsys := fstest.MapFS{
		"released-version-hashes.json": {Data: []byte(`{"schemaVersion":1,"entries":[
			{"path":"components/A/versions/1.0.0/A.tsx","sha256":"published"},
			{"path":"components/B/versions/1.0.0/B.tsx","sha256":"other"}
		]}`)},
		"release-repairs.json": {Data: []byte(`{"schemaVersion":1,"repairs":[
			{"path":"components/./A/versions/1.0.0/A.tsx","publishedSha256":"published","repairedSha256":"repaired","reason":"later commit edited the release"}
		]}`)},
	}
	got, err := loadReleaseAttestations(fsys)
	require.NoError(t, err)
	require.Equal(t, "repaired", got["components/A/versions/1.0.0/A.tsx"])
	require.True(t, attestsRelease(got, "components/A/versions/1.0.0/A.tsx", "repaired"))
	require.False(t, attestsRelease(got, "components/A/versions/1.0.0/A.tsx", "published"),
		"a repaired path must not keep accepting the superseded bytes")
	require.Equal(t, "other", got["components/B/versions/1.0.0/B.tsx"])
}

func TestReleaseRepairMustMatchThePublishedDigestAndStateAReason(t *testing.T) {
	registry := `{"schemaVersion":1,"entries":[{"path":"components/A/versions/1.0.0/A.tsx","sha256":"published"}]}`
	cases := map[string]string{
		"wrong published digest": `{"repairedSha256":"repaired","reason":"why"}`,
		"missing reason":         `{"publishedSha256":"published","repairedSha256":"repaired"}`,
		"missing repaired bytes": `{"publishedSha256":"published","reason":"why"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := loadReleaseAttestations(fstest.MapFS{
				"released-version-hashes.json": {Data: []byte(registry)},
				"release-repairs.json":         {Data: []byte(`{"schemaVersion":1,"repairs":[{"path":"components/A/versions/1.0.0/A.tsx",` + body + `}]}`)},
			})
			require.Error(t, err)
		})
	}
}

// The library's own registries must close every published-byte drift. This is
// the executable version of the startup check: a path the indexer refused must
// either match its published digest or be covered by a reviewed repair.
func TestLibraryRegistriesCoverEveryPublishedRelease(t *testing.T) {
	library := os.DirFS(filepath.Join("..", "..", "..", "library"))
	attestations, err := loadReleaseAttestations(library)
	require.NoError(t, err)
	require.NotEmpty(t, attestations)

	for _, relative := range []string{
		"components/AppShell/versions/2.1.5/AppShell.tsx",
		"components/AppShell/versions/2.1.5/story.tsx",
		"components/BottomNav/versions/1.5.7/BottomNav.tsx",
		"components/CollectionList/versions/1.3.6/CollectionList.tsx",
		"components/OverlayActionBar/versions/1.0.0/OverlayActionBar.tsx",
		"components/OverlayActionBar/versions/1.0.1/OverlayActionBar.tsx",
		"components/OverlayActionBar/versions/1.0.1/story.tsx",
	} {
		bytes, readErr := fs.ReadFile(library, relative)
		require.NoError(t, readErr, relative)
		digest := sha256.Sum256(bytes)
		require.True(t, attestsRelease(attestations, relative, hex.EncodeToString(digest[:])),
			"%s must be attested by its published digest or a repair", relative)
	}
}
