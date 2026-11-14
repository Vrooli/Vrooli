package posttypes_test

import (
	"context"
	"testing"

	localdb "content-desk/internal/database"
	"content-desk/internal/posttypes"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func TestActivationEvaluatesEveryCriterion(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(posttypes.Schema)))
	r := posttypes.NewRegistry(d)
	require.NoError(t, r.Upsert(context.Background(), posttypes.PostType{ID: "dev-log", Status: posttypes.StatusActive, PairedSkill: "x-dev-log", SkillExists: true, DocV1: true, ResponsibilitiesDeclared: false}))
	evaluation, err := r.Evaluate(context.Background(), "dev-log")
	require.NoError(t, err)
	require.False(t, evaluation.Active)
	require.Len(t, evaluation.Criteria, 4)
	require.False(t, evaluation.Criteria[3].Passed)
	require.NoError(t, r.Upsert(context.Background(), posttypes.PostType{ID: "dev-log", Status: posttypes.StatusActive, PairedSkill: "x-dev-log", SkillExists: true, DocV1: true, ResponsibilitiesDeclared: true}))
	evaluation, err = r.Evaluate(context.Background(), "dev-log")
	require.NoError(t, err)
	require.True(t, evaluation.Active)
}

func TestPostTypeFailureModesRoundTripAndRejectDuplicates(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(posttypes.Schema)))
	r := posttypes.NewRegistry(d)
	require.NoError(t, r.Upsert(context.Background(), posttypes.PostType{ID: "dev-log", Status: posttypes.StatusV0, FailureModes: []string{"credential_claim", "disclosure"}}))
	types, err := r.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"credential_claim", "disclosure"}, postTypeByID(t, types, "dev-log").FailureModes)
	require.ErrorContains(t, r.Upsert(context.Background(), posttypes.PostType{ID: "dev-log", Status: posttypes.StatusV0, FailureModes: []string{"disclosure", "disclosure"}}), "duplicate failure mode")
	types, err = r.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"credential_claim", "disclosure"}, postTypeByID(t, types, "dev-log").FailureModes, "invalid registration must not erase the existing declaration")
}

func postTypeByID(t *testing.T, types []posttypes.PostType, id string) posttypes.PostType {
	t.Helper()
	for _, postType := range types {
		if postType.ID == id {
			return postType
		}
	}
	t.Fatalf("post type %q not found", id)
	return posttypes.PostType{}
}

// [REQ:CONTENTD-P0-008]
func TestCanonicalPostTypeSeedsReflectTheCurrentCanon(t *testing.T) {
	t.Run("[CONTENTD-P0-008] canonical post type registry is seeded", func(t *testing.T) {
		d := db.NewSQLite(t)
		require.NoError(t, database.EnsureSchemas(context.Background(), d, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(posttypes.Schema)))
		types, err := posttypes.NewRegistry(d).List(context.Background())
		require.NoError(t, err)
		require.Len(t, types, 12)
		active := make(map[string]posttypes.PostType)
		for _, postType := range types {
			if postType.Status == posttypes.StatusActive {
				active[postType.ID] = postType
			}
			require.NotEmpty(t, postType.FailureModes, postType.ID)
		}
		require.Len(t, active, 2)
		require.Contains(t, active, "dev-log")
		require.Contains(t, active, "scenario-spotlight")
	})
}
