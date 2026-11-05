package catalog_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	"music-library/internal/catalog"
)

func TestCatalogKeepsApprovalAndUsageIndependent(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(catalog.Schema)))
	s := catalog.NewStore(db)
	ctx := context.Background()
	id := strings.Repeat("a", 64)
	track := catalog.Track{SHA256: id, Title: "Example", SourcePath: "/owned/example.wav", TakeID: "take-1", Tags: []string{"upbeat", "instrumental"}, Approval: "liked", Feedback: "Good"}
	_, err := s.Put(ctx, track)
	require.NoError(t, err)
	_, err = s.RecordUse(ctx, id, catalog.Use{Project: "project-x", Revision: "cut-1", Status: "draft"})
	require.NoError(t, err)
	unused, err := s.List(ctx, catalog.Filter{Tag: "upbeat", UnusedOnly: true})
	require.NoError(t, err)
	require.Len(t, unused, 1)
	require.Equal(t, "liked", unused[0].Approval)
	_, err = s.RecordUse(ctx, id, catalog.Use{Project: "project-x", Revision: "cut-1", Status: "accepted"})
	require.NoError(t, err)
	_, err = s.RecordUse(ctx, id, catalog.Use{Project: "project-x", Revision: "cut-1", Status: "draft"})
	require.NoError(t, err)
	unused, err = s.List(ctx, catalog.Filter{UnusedOnly: true})
	require.NoError(t, err)
	require.Empty(t, unused)
	related, err := s.List(ctx, catalog.Filter{Project: "project-x"})
	require.NoError(t, err)
	require.Len(t, related, 1)
	require.Len(t, related[0].Uses, 1)
	require.Equal(t, "accepted", related[0].Uses[0].Status)
	_, err = s.Put(ctx, track)
	require.NoError(t, err)
	related, err = s.List(ctx, catalog.Filter{Project: "project-x"})
	require.NoError(t, err)
	require.Len(t, related[0].Uses, 1)
}

func TestCatalogRejectsMalformedOrUnknownUse(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(catalog.Schema)))
	s := catalog.NewStore(db)
	ctx := context.Background()
	_, err := s.Put(ctx, catalog.Track{SHA256: "bad", Title: "x", SourcePath: "x", Approval: "liked"})
	require.ErrorIs(t, err, catalog.ErrInvalid)
	_, err = s.RecordUse(ctx, strings.Repeat("b", 64), catalog.Use{Project: "x", Revision: "y", Status: "accepted"})
	require.ErrorIs(t, err, catalog.ErrNotFound)
}
