package preference_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	"music-library/internal/preference"
)

func TestStorePersistsVersionsAndAppendOnlyJudgments(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(preference.Schema)))
	store := preference.NewStore(db)
	ctx := context.Background()
	profile, err := store.Get(ctx, "")
	require.NoError(t, err)
	require.Equal(t, "default", profile.Context)
	require.Equal(t, int64(0), profile.Revision)

	profile, err = store.SetControl(ctx, "default", preference.Control{Attribute: "tag:syncopated", Mode: preference.ControlBoost, Value: 0.7})
	require.NoError(t, err)
	require.Equal(t, int64(1), profile.Revision)
	leftID, rightID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	judgment := preference.Judgment{
		LeftID: leftID, RightID: rightID, Choice: preference.ChooseLeft,
		Reason: "the hook catches me", AttributeScope: "tag:syncopated",
		LeftEvidence:  []preference.Feature{{Attribute: "tag:syncopated", Value: 1, Confidence: 1, Source: "catalog:" + leftID}},
		RightEvidence: []preference.Feature{{Attribute: "tag:syncopated", Value: -1, Confidence: 1, Source: "catalog:" + rightID}},
	}
	profile, err = store.RecordJudgment(ctx, "default", judgment)
	require.NoError(t, err)
	require.Equal(t, int64(2), profile.Revision)
	require.Len(t, profile.Judgments, 1)
	require.Greater(t, profile.Weights["tag:syncopated"].Mean, 0.0)
	require.Equal(t, "operator", profile.Judgments[0].Source)
	judgmentID := profile.Judgments[0].ID

	profile, err = store.RecordJudgment(ctx, "default", preference.Judgment{
		LeftID: leftID, RightID: rightID, Choice: preference.ChooseRight, Reverses: judgmentID,
		LeftEvidence: judgment.LeftEvidence, RightEvidence: judgment.RightEvidence,
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), profile.Revision)
	require.Len(t, profile.Judgments, 2)
	require.Less(t, profile.Weights["tag:syncopated"].Mean, 0.0, "a reversal appends a corrected judgment and supersedes the earlier label in replay")

	reloaded, err := store.Get(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, profile.Revision, reloaded.Revision)
	require.Equal(t, profile.Weights, reloaded.Weights)
	require.Len(t, reloaded.Judgments, 2)
	require.Equal(t, judgmentID, reloaded.Judgments[0].ID)
	var versionCount, eventCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM music_taste_profile_versions WHERE context='default'").Scan(&versionCount))
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM music_taste_judgments WHERE context='default'").Scan(&eventCount))
	require.Equal(t, 3, versionCount)
	require.Equal(t, 2, eventCount)
}

func TestStoreRejectsUnknownReversalAndUnboundEvidence(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(preference.Schema)))
	store := preference.NewStore(db)
	ctx := context.Background()
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	_, err := store.RecordJudgment(ctx, "focus", preference.Judgment{LeftID: a, RightID: b, Choice: preference.ChooseTie, Reverses: "missing"})
	require.ErrorIs(t, err, preference.ErrJudgmentNotFound)
	_, err = store.RecordJudgment(ctx, "focus", preference.Judgment{
		LeftID: a, RightID: b, Choice: preference.ChooseLeft,
		LeftEvidence: []preference.Feature{{Attribute: "tag:pop", Value: 1, Confidence: 1, Source: "unverified"}},
	})
	require.ErrorIs(t, err, preference.ErrInvalid)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM music_taste_judgments").Scan(&count))
	require.Zero(t, count, "invalid and unbound labels must not be persisted")
}
