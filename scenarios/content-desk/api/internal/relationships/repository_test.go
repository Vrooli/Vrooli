package relationships_test

import (
	"context"
	"testing"

	"content-desk/internal/artifacts"
	"content-desk/internal/projects"
	"content-desk/internal/relationships"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestPinnedRelationsPrimaryFamilyAndCollectionAreNonOwning(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(relationships.Schema)))
	_, err := db.ExecContext(ctx, `INSERT INTO projects(id,name,status,created_at) VALUES ('p','Project','active','now')`)
	require.NoError(t, err)
	for _, id := range []string{"a", "b", "c"} {
		_, err = db.ExecContext(ctx, `INSERT INTO drafts(id,campaign_id,post_type_id,status,created_at,updated_at) VALUES (?, '', 'x','drafted','now','now')`, id)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO draft_project_refs(draft_id,project_id) VALUES (?, 'p')`, id)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO draft_revisions(id,draft_id,body,actor_kind,capacity,created_at) VALUES (?, ?, ?, 'operator','author','now')`, "rev-"+id, id, id)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO draft_slots(draft_id,campaign_id,channel,format) VALUES (?, '', 'x','text')`, id)
		require.NoError(t, err)
	}
	repo := relationships.NewSQLiteRepository(db)
	first, err := repo.CreateRelation(ctx, relationships.Relation{SourceDraftID: "a", SourceRevisionID: "rev-a", TargetDraftID: "b", TargetRevisionID: "rev-b", ProjectID: "p", Kind: "family", Primary: true})
	require.NoError(t, err)
	_, err = repo.CreateRelation(ctx, relationships.Relation{SourceDraftID: "a", SourceRevisionID: "rev-a", TargetDraftID: "c", TargetRevisionID: "rev-c", ProjectID: "p", Kind: "family", Primary: true})
	require.ErrorIs(t, err, relationships.ErrConflict)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM content_relationships`).Scan(&count))
	require.Equal(t, 1, count, "failed primary-family insert must roll back")
	got, err := repo.ListRelations(ctx, "a")
	require.NoError(t, err)
	require.Equal(t, []relationships.Relation{first}, got)
	c, err := repo.CreateCollection(ctx, "p", "Reading list")
	require.NoError(t, err)
	collections, err := repo.ListCollections(ctx, "p")
	require.NoError(t, err)
	require.Equal(t, []relationships.Collection{c}, collections)
	_, err = db.ExecContext(ctx, `INSERT INTO saved_collections(id,project_id,name,created_at) VALUES ('c2','p','Archive','now')`)
	require.NoError(t, err)
	collections, err = repo.ListCollections(ctx, "p")
	require.NoError(t, err)
	require.Equal(t, []string{"Archive", "Reading list"}, []string{collections[0].Name, collections[1].Name})
	err = repo.AddCollectionItem(ctx, relationships.CollectionItem{CollectionID: c.ID, DraftID: "a", RevisionID: "rev-a"})
	require.NoError(t, err)
	items, err := repo.ListCollectionItems(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, []relationships.CollectionItem{{CollectionID: c.ID, DraftID: "a", RevisionID: "rev-a"}}, items)
	require.NoError(t, repo.RemoveCollectionItem(ctx, relationships.CollectionItem{CollectionID: c.ID, DraftID: "a", RevisionID: "rev-a"}))
	items, err = repo.ListCollectionItems(ctx, c.ID)
	require.Empty(t, items, "removing a saved reference must not remove its draft")
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM drafts WHERE id = 'a'`).Scan(&count))
	require.Equal(t, 1, count)
	require.ErrorIs(t, repo.RemoveCollectionItem(ctx, relationships.CollectionItem{CollectionID: c.ID, DraftID: "a", RevisionID: "rev-a"}), relationships.ErrNotFound)
	require.NoError(t, repo.AddCollectionItem(ctx, relationships.CollectionItem{CollectionID: c.ID, DraftID: "a", RevisionID: "rev-a"}))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM draft_slots WHERE draft_id = 'a'`).Scan(&count))
	require.Equal(t, 1, count, "saved membership must not allocate an artifact slot")
	impact, err := repo.PreviewImpact(ctx, "a", "move")
	require.NoError(t, err)
	require.Equal(t, int64(1), impact.Relationships)
	require.Equal(t, int64(1), impact.CollectionMemberships)
	require.True(t, impact.HasDraftSlot)
	// Preview leaves all authoritative rows intact.
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM content_relationships`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM saved_collection_items`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestRejectsUnpinnedAndCrossProjectReferences(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(relationships.Schema)))
	_, err := db.ExecContext(ctx, `INSERT INTO projects(id,name,status,created_at) VALUES ('p','Project','active','now'),('q','Other','active','now')`)
	require.NoError(t, err)
	for _, entry := range []struct{ id, project string }{{"a", "p"}, {"b", "q"}} {
		_, err = db.ExecContext(ctx, `INSERT INTO drafts(id,campaign_id,post_type_id,status,created_at,updated_at) VALUES (?, '', 'x','drafted','now','now')`, entry.id)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO draft_project_refs(draft_id,project_id) VALUES (?,?)`, entry.id, entry.project)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO draft_revisions(id,draft_id,body,actor_kind,capacity,created_at) VALUES (?, ?, '', 'operator','author','now')`, "rev-"+entry.id, entry.id)
		require.NoError(t, err)
	}
	repo := relationships.NewSQLiteRepository(db)
	_, err = repo.CreateRelation(ctx, relationships.Relation{SourceDraftID: "a", SourceRevisionID: "missing", TargetDraftID: "b", TargetRevisionID: "rev-b", ProjectID: "p", Kind: "related"})
	require.ErrorIs(t, err, relationships.ErrNotFound)
	_, err = repo.CreateRelation(ctx, relationships.Relation{SourceDraftID: "a", SourceRevisionID: "rev-a", TargetDraftID: "b", TargetRevisionID: "rev-b", ProjectID: "p", Kind: "related"})
	require.ErrorIs(t, err, relationships.ErrInvalid)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM content_relationships`).Scan(&count))
	require.Zero(t, count)
}
