package capabilities

import (
	"context"
	"database/sql"
	"testing"

	"content-desk/internal/artifacts"
	internalcapabilities "content-desk/internal/capabilities"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"
)

func newArtifactOwner(t *testing.T) *sql.DB {
	t.Helper()
	db := databasetest.NewSQLite(t)
	_, err := db.ExecContext(context.Background(), artifacts.Schema())
	require.NoError(t, err)
	return db
}

func insertDraft(t *testing.T, db *sql.DB, id, status string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO drafts (id, campaign_id, post_type_id, status, created_at, updated_at) VALUES (?, 'campaign-1', 'post-type-1', ?, '2026-09-12T00:00:00Z', '2026-09-12T00:00:00Z')`,
		id, status)
	require.NoError(t, err)
}

func TestArtifactOutputResolverMapsCurrentLifecycle(t *testing.T) {
	db := newArtifactOwner(t)
	for _, tc := range []struct {
		id       string
		status   string
		accepted bool
	}{
		{id: "draft-drafted", status: "drafted", accepted: false},
		{id: "draft-reviewed", status: "reviewed", accepted: true},
		{id: "draft-approved", status: "approved", accepted: true},
		{id: "draft-published", status: "published", accepted: true},
		{id: "draft-abandoned", status: "abandoned", accepted: false},
	} {
		insertDraft(t, db, tc.id, tc.status)
	}

	resolver := newArtifactOutputResolver(db)
	for _, tc := range []struct {
		id       string
		accepted bool
	}{
		{id: "draft-drafted", accepted: false},
		{id: "draft-reviewed", accepted: true},
		{id: "draft-approved", accepted: true},
		{id: "draft-published", accepted: true},
		{id: "draft-abandoned", accepted: false},
	} {
		state, err := resolver.CurrentOutputState(context.Background(), tc.id)
		require.NoError(t, err, tc.id)
		require.True(t, state.Resolvable, tc.id)
		require.Equal(t, tc.accepted, state.Accepted, tc.id)
	}
}

func TestArtifactOutputResolverUnknownIdIsNotArtifact(t *testing.T) {
	resolver := newArtifactOutputResolver(newArtifactOwner(t))
	state, err := resolver.CurrentOutputState(context.Background(), "claim-1")
	require.NoError(t, err)
	require.False(t, state.Resolvable)
	require.False(t, state.Accepted)
}

func TestRehydrateUsesResolvedOwnerLifecycle(t *testing.T) {
	db := newArtifactOwner(t)
	insertDraft(t, db, "draft-approved", "approved")
	resolver := newArtifactOutputResolver(db)

	projection := internalcapabilities.RehydrateOutputQuality(context.Background(), internalcapabilities.QualityUnassessed, []internalcapabilities.CapabilityLink{
		{CapabilityID: "cap", Relation: internalcapabilities.RelationEvidence, TargetID: "draft-approved"},
	}, resolver)
	require.Equal(t, internalcapabilities.QualityAccepted, projection.Quality)
	require.Equal(t, "draft-approved", projection.Source)
}
