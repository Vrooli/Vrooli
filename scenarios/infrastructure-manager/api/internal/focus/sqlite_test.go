package focus

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestSchemaStoresFindingsAndEfficacyWithoutActuation(t *testing.T) {
	schema := strings.ToLower(Schema())
	require.Contains(t, schema, "focus_findings")
	require.Contains(t, schema, "focus_efficacy")
	require.NotContains(t, schema, "restart")
	require.NotContains(t, schema, "shelve")

	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	require.NoError(t, repo.SaveFindings(context.Background(), []RankedFinding{{
		Finding: Finding{ID: "coverage/A1", CellRef: "coverage/A1", Title: "Missing", SensorRef: "coverage/A1", ExpectedReturn: "NOW"},
		Rank:    1, RankExplanation: "measurement improvement follows operational findings",
	}}))
	records, err := repo.Efficacy(context.Background(), "coverage/A1")
	require.NoError(t, err)
	require.Empty(t, records, "a finding without a completed work join is not fabricated as efficacy")
}

func TestServiceEfficacyNamesMissingWorkJoin(t *testing.T) {
	service := &Service{}
	records, err := service.Efficacy(context.Background(), "coverage/A1")
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, EfficacyUnmeasurable, records[0].Verdict)
	require.Contains(t, records[0].ObservedReturn, "not persisted")
}
