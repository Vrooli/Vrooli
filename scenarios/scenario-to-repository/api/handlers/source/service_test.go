package source

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	domain "scenario-to-repository/internal/source"

	"github.com/stretchr/testify/require"
)

func TestSQLiteStoreRetainsArtifactAndDistributionIdentity(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(domain.Schema())
	require.NoError(t, err)
	store := NewSQLiteStore(db)
	ctx := context.Background()
	artifact := domain.Artifact{ArtifactID: "artifact-1", ArchiveDigest: "sha256:archive", Status: "assembled"}
	require.NoError(t, store.saveArtifact(ctx, artifact))
	gotArtifact, ok, err := store.loadArtifact(ctx, artifact.ArtifactID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, artifact, gotArtifact)
	distribution := domain.Distribution{DistributionID: "distribution-1", ArtifactID: artifact.ArtifactID, UpdatedAt: time.Now().UTC()}
	require.NoError(t, store.saveDistribution(ctx, distribution))
	gotDistribution, ok, err := store.loadDistribution(ctx, distribution.DistributionID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, distribution, gotDistribution)
}
