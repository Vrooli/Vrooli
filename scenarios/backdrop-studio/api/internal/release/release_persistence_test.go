package release

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func persistenceRequest(id string) Request {
	return Request{CandidateID: id, StyleID: "s", Strategy: "procedural", SurfaceID: "web.hero", Placement: "full_bleed", AltText: "A durable ambient backdrop"}
}

func persistenceCandidate(t *testing.T, id string) CandidateEvidence {
	t.Helper()
	candidate := authoritativeCandidate(t)
	candidate.ID = id
	return candidate
}

func persistenceSources(t *testing.T, id string) (ProvenanceSource, CandidateSource) {
	t.Helper()
	return fakeProvenance{id: {Strategy: "procedural", ModelBacked: false}}, fakeCandidateSource{id: persistenceCandidate(t, id)}
}

func TestReleasedBackdropSurvivesStoreRestart(t *testing.T) {
	dataDir := t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: dataDir})
	provenance, candidates := persistenceSources(t, "restart")
	first := NewStoreWithPublisherAndRoots(nil, provenance, roots, candidates)

	released, err := first.Release(persistenceRequest("restart"))
	require.NoError(t, err)

	second := NewStoreWithPublisherAndRoots(nil, nil, roots)
	loaded, err := second.Get(released.ID)
	require.NoError(t, err)
	require.Equal(t, released.ID, loaded.ID)
	require.Equal(t, released.JobID, loaded.JobID)
	require.Equal(t, released.ImagePNG, loaded.ImagePNG)
	require.Equal(t, released.ContentHash, loaded.ContentHash)
	require.Equal(t, released.Regions, loaded.Regions)
}

func TestReleasedBackdropRefusesTamperedBytesAndMetadata(t *testing.T) {
	dataDir := t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: dataDir})
	provenance, candidates := persistenceSources(t, "tamper")
	store := NewStoreWithPublisherAndRoots(nil, provenance, roots, candidates)
	released, err := store.Release(persistenceRequest("tamper"))
	require.NoError(t, err)

	assetPath := filepath.Join(dataDir, releasedBackdropDir, released.ID, "asset.png")
	require.NoError(t, os.WriteFile(assetPath, []byte("tampered"), 0600))
	_, err = store.Get(released.ID)
	require.ErrorContains(t, err, "integrity")

	// Restore the bytes, then prove the metadata checksum is independently
	// checked rather than trusting a readable JSON record.
	require.NoError(t, os.WriteFile(assetPath, released.ImagePNG, 0600))
	metadataPath := filepath.Join(dataDir, releasedBackdropDir, released.ID, "metadata.json")
	metadata, err := os.ReadFile(metadataPath)
	require.NoError(t, err)
	metadata[len(metadata)-2] ^= 1
	require.NoError(t, os.WriteFile(metadataPath, metadata, 0600))
	_, err = store.Get(released.ID)
	require.ErrorContains(t, err, "metadata integrity")
}

func TestReleasedBackdropsUseIsolatedLeasedRoots(t *testing.T) {
	primary := storage.Paths{ConfigDir: t.TempDir(), DataDir: t.TempDir(), CacheDir: t.TempDir(), LogsDir: t.TempDir(), StateDir: t.TempDir()}
	roots := filerouting.New(primary)
	provenance, candidates := persistenceSources(t, "isolated")
	store := NewStoreWithPublisherAndRoots(nil, provenance, roots, candidates)
	testCtx := database.WithTestMode(context.Background())

	_, err := store.ReleaseContext(testCtx, persistenceRequest("isolated"))
	require.ErrorContains(t, err, "active leased root")
	_, err = roots.InstallLeasedTestRoots("release-persistence-test", time.Minute, true)
	require.NoError(t, err)
	defer func() { require.NoError(t, roots.ClearTestRoots("release-persistence-test")) }()

	released, err := store.ReleaseContext(testCtx, persistenceRequest("isolated"))
	require.NoError(t, err)
	_, err = store.Get(released.ID)
	require.ErrorContains(t, err, "not found")
	loaded, err := store.GetContext(testCtx, released.ID)
	require.NoError(t, err)
	require.Equal(t, released.ImagePNG, loaded.ImagePNG)
	require.NotZero(t, roots.LeaseStats().TestRootWrites)
	require.Zero(t, roots.LeaseStats().PrimaryWritesDuringTestMode)
}

func TestConcurrentReleaseCallsCommitOneImmutableArtifact(t *testing.T) {
	dataDir := t.TempDir()
	roots := filerouting.New(storage.Paths{DataDir: dataDir})
	provenance, candidates := persistenceSources(t, "concurrent")
	const writers = 16
	results := make(chan Backdrop, writers)
	errors := make(chan error, writers)
	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Independent store instances model concurrent API workers or
			// processes sharing the same routed data root.
			store := NewStoreWithPublisherAndRoots(nil, provenance, roots, candidates)
			backdrop, err := store.Release(persistenceRequest("concurrent"))
			if err != nil {
				errors <- err
				return
			}
			results <- backdrop
		}()
	}
	wg.Wait()
	close(results)
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	var first Backdrop
	for backdrop := range results {
		if first.ID == "" {
			first = backdrop
		}
		require.Equal(t, first.ContentHash, backdrop.ContentHash)
	}
	require.NotEmpty(t, first.ID)
	restarted := NewStoreWithPublisherAndRoots(nil, nil, roots)
	loaded, err := restarted.Get(first.ID)
	require.NoError(t, err)
	require.Equal(t, first.ContentHash, loaded.ContentHash)
}
