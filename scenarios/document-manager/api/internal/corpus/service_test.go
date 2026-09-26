package corpus

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
	_ "modernc.org/sqlite"
)

func TestExportImportRoundTripPreservesOpenArchiveAndAnchors(t *testing.T) { // [REQ:DOC-P0-017]
	ctx := context.Background()
	open := func(label string) *SQLiteRepository {
		db, err := sql.Open("sqlite", "file:corpus-roundtrip-"+label+"-"+t.Name()+"?mode=memory&cache=shared")
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		_, err = db.ExecContext(ctx, Schema())
		require.NoError(t, err)
		return NewSQLiteRepository(db)
	}

	source := NewService(open("source"))
	c, err := source.CreateCollection(ctx, "portable", documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL, false)
	require.NoError(t, err)
	_, err = source.AddDocument(ctx, c.ID, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL)
	require.NoError(t, err)
	require.NoError(t, source.Repo.AddAnchor(ctx, c.ID, "vrooli-anchor:1/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/1/logical/1/0-4"))
	archive, err := source.Export(ctx, c.ID)
	require.NoError(t, err)
	decoded, err := DecodeArchive(archive)
	require.NoError(t, err)

	destination := NewService(open("destination"))
	imported, count, err := destination.Import(ctx, archive)
	require.NoError(t, err)
	require.Equal(t, decoded.Collection.ID, imported.ID)
	require.Equal(t, 1, count)
	anchors, err := destination.Repo.ListAnchors(ctx, imported.ID)
	require.NoError(t, err)
	require.Equal(t, decoded.AnchorURIs, anchors)
	got, err := destination.ListDocuments(ctx, imported.ID, 0)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, decoded.Documents[0].DocumentHash, got[0].DocumentHash)
}
