package conversationsearch

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"agent-manager/internal/conversationpolicy"
)

// The source fixture seeds run-1 as harness=claude-code, model=model-actual,
// with message events event-1 and event-3. Naming these makes the policy
// assertions read against intent rather than bare literals.
const (
	fixtureRunID   = "run-1"
	fixtureModel   = "model-actual"
	fixtureEventID = "event-3"
)

// defaultFixtureNormalizer wraps the recipe-v1/512/32 triple repeated verbatim
// across the search-seam tests so a recipe change is one edit.
func defaultFixtureNormalizer(t *testing.T) *Normalizer {
	t.Helper()
	return fixtureNormalizer(t, "recipe-v1", 512, 32)
}

// searchExclude builds a search-seam model exclude rule.
func searchExclude(model string) conversationpolicy.Rule {
	return conversationpolicy.Rule{Action: conversationpolicy.ActionExclude, EnforceAt: conversationpolicy.EnforceSearch, Model: model}
}

func mustSearchPolicy(t *testing.T, rules ...conversationpolicy.Rule) *conversationpolicy.Policy {
	t.Helper()
	policy, err := conversationpolicy.New(rules)
	require.NoError(t, err)
	return policy
}

// seededDB opens a fresh projection DB and seeds the source fixture (run-1,
// harness=claude-code, model=model-actual, events event-1/event-3).
func seededDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := openProjectionTestDB(t)
	createSourceFixtureSchema(t, db)
	seedSourceFixture(t, db)
	return db
}

// sourceOn builds a source over an already-seeded DB with the given policy
// (nil leaves source loading unchanged).
func sourceOn(t *testing.T, db *sqlx.DB, policy *conversationpolicy.Policy) *SQLiteSource {
	t.Helper()
	opts := []SQLiteSourceOption{}
	if policy != nil {
		opts = append(opts, WithVisibilityPolicy(policy))
	}
	source, err := NewSQLiteSource(db, defaultFixtureNormalizer(t), opts...)
	require.NoError(t, err)
	return source
}

// TestSQLiteSourceSearchPolicyHidesMatchingRun proves the search seam filters
// documents whose run matches a search-enforced exclude rule across all three
// read paths (page, run, single-event), while the canonical run_events rows
// remain untouched (reversible hide), and the hide is counted once per load.
func TestSQLiteSourceSearchPolicyHidesMatchingRun(t *testing.T) {
	t.Parallel()
	db := seededDB(t)
	// Baseline: without a policy the fixture yields its messages.
	base, err := sourceOn(t, db, nil).LoadSourcePage(context.Background(), nil, 10)
	require.NoError(t, err)
	require.NotEmpty(t, base.Documents, "fixture should produce documents without a policy")

	policy := mustSearchPolicy(t, searchExclude(fixtureModel))
	hidden := sourceOn(t, db, policy)

	page, err := hidden.LoadSourcePage(context.Background(), nil, 10)
	require.NoError(t, err)
	require.Empty(t, page.Documents, "search-excluded run must be absent from the projection")

	runDocs, err := hidden.LoadRunDocuments(context.Background(), fixtureRunID)
	require.NoError(t, err)
	require.Empty(t, runDocs, "search-excluded run must be absent from incremental projection")

	// LoadEventDocuments shares the same loadDocuments filter; a single-event
	// request for an excluded run must also come back empty.
	eventDocs, err := hidden.LoadEventDocuments(context.Background(), fixtureRunID, fixtureEventID)
	require.NoError(t, err)
	require.Empty(t, eventDocs, "search-excluded run must be absent from the single-event path")

	// The hide is counted once per conversation per load pass (not once per
	// event), so it is observable rather than silent — three load calls, three
	// counts of the one hidden conversation.
	require.Equal(t, int64(3), policy.Snapshot().SearchHidden)

	// The canonical rows are untouched: the hide is reversible.
	var remaining int
	require.NoError(t, db.Get(&remaining, `SELECT COUNT(*) FROM run_events WHERE run_id = ? AND event_type = 'message'`, fixtureRunID))
	require.Positive(t, remaining, "policy hide must not delete canonical events")
}

// TestSQLiteSourceSearchPolicyGlobHidesModelFamily pins the A-P1-3 glob
// contract at the search seam: a `model:"model-*"` rule hides a run whose
// resolved model carries a suffix, which bare exact-match would miss.
func TestSQLiteSourceSearchPolicyGlobHidesModelFamily(t *testing.T) {
	t.Parallel()
	policy := mustSearchPolicy(t, searchExclude("model-*"))
	page, err := sourceOn(t, seededDB(t), policy).LoadSourcePage(context.Background(), nil, 10)
	require.NoError(t, err)
	require.Empty(t, page.Documents, "model glob must hide the run whose model is model-actual")
	require.Equal(t, int64(1), policy.Snapshot().SearchHidden)
}

// TestSQLiteSourceImportEnforcedRuleDoesNotHideAtSearch confirms an
// import-enforced rule never affects the search seam (enforcement points are
// independent).
func TestSQLiteSourceImportEnforcedRuleDoesNotHideAtSearch(t *testing.T) {
	t.Parallel()
	policy := mustSearchPolicy(t, conversationpolicy.Rule{Action: conversationpolicy.ActionExclude, EnforceAt: conversationpolicy.EnforceImport, Model: fixtureModel})
	page, err := sourceOn(t, seededDB(t), policy).LoadSourcePage(context.Background(), nil, 10)
	require.NoError(t, err)
	require.NotEmpty(t, page.Documents, "an import-only rule must not hide at the search seam")
	require.Equal(t, int64(0), policy.Snapshot().SearchHidden)
}
