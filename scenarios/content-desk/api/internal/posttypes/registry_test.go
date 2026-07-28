package posttypes_test

import (
	"context"
	"testing"

	localdb "content-desk/internal/database"
	"content-desk/internal/posttypes"
	"content-desk/internal/testutil/db"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
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
	require.Len(t, types, 1)
	require.Equal(t, []string{"credential_claim", "disclosure"}, types[0].FailureModes)
	require.ErrorContains(t, r.Upsert(context.Background(), posttypes.PostType{ID: "dev-log", Status: posttypes.StatusV0, FailureModes: []string{"disclosure", "disclosure"}}), "duplicate failure mode")
	types, err = r.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"credential_claim", "disclosure"}, types[0].FailureModes, "invalid registration must not erase the existing declaration")
}
