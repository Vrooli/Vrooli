package corpus

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
	_ "modernc.org/sqlite"
)

func TestCollectionPrivacyInheritanceAndFederationCeiling(t *testing.T) { // [REQ:DOC-P0-018] [REQ:DOC-P0-024] [REQ:DOC-P0-025]
	db, err := sql.Open("sqlite", "file:corpus-test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.CreateCollection(ctx, Collection{ID: "c1", Name: "matter", DefaultPrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, Federated: true}))
	require.Error(t, repo.AddDocument(ctx, Membership{CollectionID: "c1", DocumentHash: "x", PrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL}))
	require.NoError(t, repo.AddDocument(ctx, Membership{CollectionID: "c1", DocumentHash: "x", PrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_SECRET}))
	allowed, err := repo.CanRead(ctx, "c1", documentpb.PrivacyClass_PRIVACY_CLASS_SECRET)
	require.NoError(t, err)
	require.False(t, allowed)
}
