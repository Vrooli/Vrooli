package eval_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"

	localdb "search-hub/internal/database"
	"search-hub/internal/eval"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

// newStore returns a SQLite-backed eval Store with the production schema applied
// — the canonical compose pattern (db.NewSQLite + apidb.EnsureSchemas over the
// system + eval providers), so tests exercise the same shape main.go ships.
func newStore(t *testing.T) (eval.Store, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(eval.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC))
	return eval.NewSQLiteStore(d, clk), clk
}

func validSuite() *evalv1.EvalSuite {
	return &evalv1.EvalSuite{
		SuiteId:    "cli-health.commands.primary",
		ProviderId: "cli-health.commands",
		Name:       "CLI command discovery — primary",
		Minimum: &evalv1.EvalMinimum{
			ReviewedPositive: 2,
			Negative:         1,
			RequiredTags:     []string{"strong"},
		},
		Coverage: &evalv1.EvalCoverageConfig{
			RequiredTags: []string{"lifecycle"},
			RequiredTagGroups: []*evalv1.EvalCoverageTagGroup{{
				Id:                  "operations",
				Description:         "operator workflows",
				Tags:                []string{"lifecycle", "testing"},
				MinReviewedPositive: 2,
			}},
		},
		Cases: []*evalv1.EvalCase{
			{CaseId: "restart", Query: "how do I restart a scenario", Tags: []string{"strong"}, ExpectIds: []string{"restart"}, ExpectWithinTopK: 5, ExpectMinScore: 0.65},
			{CaseId: "gibberish-1", Query: "asdfqwer zxcvbnm", Tags: []string{"gibberish"}, ExpectNoStrongHit: true, ExpectMaxScore: 0.4},
		},
	}
}

func TestUpsertSuiteInsertThenUpdate(t *testing.T) {
	store, clk := newStore(t)
	ctx := context.Background()

	created, err := store.UpsertSuite(ctx, validSuite())
	require.NoError(t, err)
	require.True(t, created, "first upsert inserts")

	got, err := store.GetSuite(ctx, "cli-health.commands.primary")
	require.NoError(t, err)
	require.Equal(t, "active", got.GetState(), "state defaults to active")
	require.NotEmpty(t, got.GetCreatedAt())
	require.Equal(t, int32(2), got.GetMinimum().GetReviewedPositive(), "registration minimum survives protojson persistence")
	require.Equal(t, int32(1), got.GetMinimum().GetNegative())
	require.Equal(t, []string{"strong"}, got.GetMinimum().GetRequiredTags())
	require.Equal(t, []string{"lifecycle"}, got.GetCoverage().GetRequiredTags())
	require.Equal(t, "operations", got.GetCoverage().GetRequiredTagGroups()[0].GetId())
	require.Equal(t, []string{"lifecycle", "testing"}, got.GetCoverage().GetRequiredTagGroups()[0].GetTags())
	firstCreated := got.GetCreatedAt()

	clk.Advance(time.Hour)
	s2 := validSuite()
	s2.Name = "Renamed"
	created, err = store.UpsertSuite(ctx, s2)
	require.NoError(t, err)
	require.False(t, created, "second upsert updates")

	got, err = store.GetSuite(ctx, "cli-health.commands.primary")
	require.NoError(t, err)
	require.Equal(t, "Renamed", got.GetName())
	require.Equal(t, firstCreated, got.GetCreatedAt(), "created_at is immutable across updates")
	require.NotEqual(t, firstCreated, got.GetUpdatedAt(), "updated_at advances")

	all, err := store.ListSuites(ctx, eval.ListSuitesFilter{})
	require.NoError(t, err)
	require.Len(t, all, 1, "upsert must not duplicate the suite")
}

func TestUpsertSuiteRejectsInvalid(t *testing.T) {
	store, _ := newStore(t)
	bad := validSuite()
	bad.Cases = nil
	_, err := store.UpsertSuite(context.Background(), bad)
	require.Error(t, err)
	var invalid eval.ErrInvalidSuite
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "cases", invalid.Field)
}

func TestGetSuiteNotFound(t *testing.T) {
	store, _ := newStore(t)
	_, err := store.GetSuite(context.Background(), "nope")
	var nf eval.ErrSuiteNotFound
	require.ErrorAs(t, err, &nf)
}

func TestCorpusValidationReceiptPersistsExactSuiteDenominatorAndOutcomes(t *testing.T) {
	store, _ := newStore(t)
	type validationStore interface {
		eval.CorpusValidationReader
		AppendCorpusValidation(context.Context, string, *evalv1.ValidateCorpusResponse, time.Time) error
	}
	validation, ok := store.(validationStore)
	require.True(t, ok, "production store exposes durable corpus validation evidence")

	ctx := context.Background()
	observedAt := time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC)
	result := &evalv1.ValidateCorpusResponse{
		SuiteId: "cli-health.commands.primary", ProviderId: "cli-health.commands",
		SuiteSha256: strings.Repeat("a", 64), DenominatorSha256: strings.Repeat("b", 64),
		DenominatorCaseIds: []string{"restart", "search"},
		Cases: []*evalv1.CorpusValidationCase{
			{CaseId: "restart", Referential: evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_LIVE},
			{CaseId: "search", Referential: evalv1.ReferentialOutcome_REFERENTIAL_OUTCOME_HARD},
		},
		Rollup: &evalv1.CorpusValidationRollup{Positives: 2, Live: 1, Hard: 1},
	}
	require.NoError(t, validation.AppendCorpusValidation(ctx, result.GetSuiteId(), result, observedAt))

	stored, err := validation.LatestCorpusValidation(ctx, result.GetSuiteId())
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.Equal(t, observedAt, stored.CreatedAt)
	require.Equal(t, result.GetSuiteSha256(), stored.Result.GetSuiteSha256())
	require.Equal(t, result.GetDenominatorSha256(), stored.Result.GetDenominatorSha256())
	require.Equal(t, result.GetDenominatorCaseIds(), stored.Result.GetDenominatorCaseIds())
	require.Equal(t, result.GetCases(), stored.Result.GetCases())
	require.Equal(t, result.GetRollup(), stored.Result.GetRollup())
}

func TestDeleteSuiteRemovesRunsAndSuite(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()
	require.NoError(t, mustUpsert(t, store, validSuite()))
	require.NoError(t, store.AppendRun(ctx, &evalv1.EvalRun{RunId: "delete-run", SuiteId: "cli-health.commands.primary"}))

	require.NoError(t, store.DeleteSuite(ctx, "cli-health.commands.primary"))
	_, err := store.GetSuite(ctx, "cli-health.commands.primary")
	var suiteNotFound eval.ErrSuiteNotFound
	require.ErrorAs(t, err, &suiteNotFound)
	runs, err := store.ListRuns(ctx, eval.ListRunsFilter{SuiteID: "cli-health.commands.primary"})
	require.NoError(t, err)
	require.Empty(t, runs)

	err = store.DeleteSuite(ctx, "cli-health.commands.primary")
	require.ErrorAs(t, err, &suiteNotFound)
}

func TestListSuitesFilterByProvider(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()
	a := validSuite()
	require.NoError(t, mustUpsert(t, store, a))
	b := validSuite()
	b.SuiteId = "ui-health.surfaces.primary"
	b.ProviderId = "ui-health.surfaces"
	require.NoError(t, mustUpsert(t, store, b))

	got, err := store.ListSuites(ctx, eval.ListSuitesFilter{ProviderID: "ui-health.surfaces"})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "ui-health.surfaces.primary", got[0].GetSuiteId())

	all, err := store.ListSuites(ctx, eval.ListSuitesFilter{})
	require.NoError(t, err)
	require.Len(t, all, 2)
}

func TestAppendAndListRuns(t *testing.T) {
	store, clk := newStore(t)
	ctx := context.Background()
	require.NoError(t, mustUpsert(t, store, validSuite()))

	mkRun := func(id, tag string) *evalv1.EvalRun {
		return &evalv1.EvalRun{
			RunId:   id,
			SuiteId: "cli-health.commands.primary",
			Tag:     tag,
			Config:  &evalv1.ConfigSnapshot{RerankerLeg: "none"},
			Results: []*evalv1.CaseResult{{CaseId: "restart", Outcome: "met", ObservedTopScore: 0.7}},
			Aggregate: &evalv1.EvalAggregate{
				Cases: 2, Met: 1, MeanStrongTop1: 0.7, MaxGibberishScore: 0.3,
			},
		}
	}

	require.NoError(t, store.AppendRun(ctx, mkRun("run-1", "rerank-off")))
	clk.Advance(time.Minute)
	require.NoError(t, store.AppendRun(ctx, mkRun("run-2", "cross-encoder")))
	clk.Advance(time.Minute)
	require.NoError(t, store.AppendRun(ctx, mkRun("run-3", "cross-encoder")))

	// Newest first.
	all, err := store.ListRuns(ctx, eval.ListRunsFilter{SuiteID: "cli-health.commands.primary"})
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, "run-3", all[0].GetRunId())
	require.Equal(t, "run-1", all[2].GetRunId())
	// Blob round-trip preserved the nested aggregate + config.
	require.Equal(t, "none", all[0].GetConfig().GetRerankerLeg())
	require.InDelta(t, 0.7, all[0].GetAggregate().GetMeanStrongTop1(), 1e-9)

	// Tag filter.
	ce, err := store.ListRuns(ctx, eval.ListRunsFilter{SuiteID: "cli-health.commands.primary", Tag: "cross-encoder"})
	require.NoError(t, err)
	require.Len(t, ce, 2)

	// Limit.
	one, err := store.ListRuns(ctx, eval.ListRunsFilter{SuiteID: "cli-health.commands.primary", Limit: 1})
	require.NoError(t, err)
	require.Len(t, one, 1)
	require.Equal(t, "run-3", one[0].GetRunId())

	// Get by id.
	got, err := store.GetRun(ctx, "run-2")
	require.NoError(t, err)
	require.Equal(t, "cross-encoder", got.GetTag())

	_, err = store.GetRun(ctx, "missing")
	var nf eval.ErrRunNotFound
	require.ErrorAs(t, err, &nf)
}

func mustUpsert(t *testing.T, store eval.Store, s *evalv1.EvalSuite) error {
	t.Helper()
	_, err := store.UpsertSuite(context.Background(), s)
	return err
}
