package planning

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"tech-tree-designer/internal/testutil/db"
)

func TestSQLiteRepositoryScenarioAndFileRoundTrip(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, d, apidb.SchemaProviderFunc(Schema)))

	repo := NewSQLiteRepository(d)
	created, err := repo.CreateScenario(ctx, CreateInput{
		Slug:        "planned-demo",
		DisplayName: "Planned Demo",
		Sector:      "engineering",
		Tier:        "foundation",
	})
	require.NoError(t, err)
	require.Equal(t, "planned-demo", created.Slug)
	require.Equal(t, DefaultTargetStability, created.TargetStability)

	file, err := repo.PutFile(ctx, PutFileInput{
		Slug: "planned-demo",
		Path: "planned-demo/v1/api/service.proto",
		Text: validProtoText(),
	})
	require.NoError(t, err)
	require.Equal(t, "planned-demo/v1/api/service.proto", file.Path)

	got, err := repo.GetScenario(ctx, "planned-demo")
	require.NoError(t, err)
	require.Len(t, got.Files, 1)
	require.Equal(t, validProtoText(), got.Files[0].Text)

	listed, err := repo.ListScenarios(ctx, ListFilter{Sector: "engineering"})
	require.NoError(t, err)
	require.Len(t, listed, 1)

	deleted, err := repo.DeleteFile(ctx, "planned-demo", "planned-demo/v1/api/service.proto")
	require.NoError(t, err)
	require.True(t, deleted)
}

func TestSQLiteRepositoryRejectsUnsafePath(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, d, apidb.SchemaProviderFunc(Schema)))

	repo := NewSQLiteRepository(d)
	_, err := repo.CreateScenario(ctx, CreateInput{Slug: "planned-demo"})
	require.NoError(t, err)

	_, err = repo.PutFile(ctx, PutFileInput{Slug: "planned-demo", Path: "../escape.proto", Text: "syntax = \"proto3\";"})
	var invalid ErrInvalidArgument
	require.ErrorAs(t, err, &invalid)
}
