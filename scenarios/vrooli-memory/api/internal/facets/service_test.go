package facets

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	localdb "vrooli-memory/internal/database"
	"vrooli-memory/internal/inference"
	"vrooli-memory/internal/journal"
	"vrooli-memory/internal/testutil/mocks"
)

func newService(t *testing.T) (*Service, *journal.SQLiteRepository) {
	t.Helper()
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:facets?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db.Primary())
	require.NoError(t, repo.Seed(context.Background()))
	return NewService(repo), journal.NewSQLiteRepository(db.Primary())
}

func TestSeedHasExactlySixStableFacets(t *testing.T) {
	s, _ := newService(t)
	require.NoError(t, s.Seed(context.Background()))
	items, err := s.List(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 6)
}

func TestUnknownFacetIsHardError(t *testing.T) {
	s, _ := newService(t)
	require.ErrorAs(t, s.Validate(context.Background(), "invented"), new(ErrUnknownFacet))
	require.NoError(t, s.Validate(context.Background(), UnclassifiedFacet))
}

func TestJournalWriteRejectsUnknownExplicitFacet(t *testing.T) {
	s, j := newService(t)
	_, err := journal.NewService(j, &mocks.FakeInference{ClassifyOut: "episode", EmbedOut: []float64{1}}, s).
		Append(context.Background(), journal.Entry{Body: "do not silently route this", FacetID: "invented"})
	var unknown ErrUnknownFacet
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "invented", unknown.ID)
}

func TestOnlyEpisodeIsCompactionEligibleAndPinExemptsIt(t *testing.T) { // [REQ:VMEM-P0-005]
	s, j := newService(t)
	ctx := context.Background()
	episode, err := j.Append(ctx, journal.Entry{Body: "episode", FacetID: "episode"}, nil)
	require.NoError(t, err)
	_, err = s.ReFacet(ctx, Assignment{EntryID: episode.ID, FacetID: "episode"})
	require.NoError(t, err)
	eligible, err := s.CompactionEligible(ctx, episode.ID)
	require.NoError(t, err)
	require.True(t, eligible)
	require.NoError(t, s.SetPin(ctx, episode.ID, true))
	eligible, err = s.CompactionEligible(ctx, episode.ID)
	require.NoError(t, err)
	require.False(t, eligible)
	fact, err := j.Append(ctx, journal.Entry{Body: "fact", FacetID: "environment-fact"}, nil)
	require.NoError(t, err)
	_, err = s.ReFacet(ctx, Assignment{EntryID: fact.ID, FacetID: "environment-fact"})
	require.NoError(t, err)
	eligible, err = s.CompactionEligible(ctx, fact.ID)
	require.NoError(t, err)
	require.False(t, eligible)
}

func TestRefacetRetainsHistory(t *testing.T) {
	s, j := newService(t)
	ctx := context.Background()
	entry, err := j.Append(ctx, journal.Entry{Body: "rule", FacetID: "episode"}, nil)
	require.NoError(t, err)
	_, err = s.ReFacet(ctx, Assignment{EntryID: entry.ID, FacetID: "episode"})
	require.NoError(t, err)
	_, err = s.ReFacet(ctx, Assignment{EntryID: entry.ID, FacetID: "standing-rule"})
	require.NoError(t, err)
	history, err := s.repo.Assignments(ctx, entry.ID)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, "episode", history[0].FacetID)
	require.Equal(t, "standing-rule", history[1].FacetID)
}

func TestRefacetCorpusAppendsRuleAndClassifierDecisions(t *testing.T) {
	s, j := newService(t)
	ctx := context.Background()
	rule, err := s.CreateRule(ctx, Rule{ID: "imported-episode", Priority: 10, FacetID: "episode", SourceRuntime: "importer"})
	require.NoError(t, err)
	_, err = j.Append(ctx, journal.Entry{Body: "imported completion", Attribution: journal.Attribution{SourceRuntime: "importer"}}, nil)
	require.NoError(t, err)
	_, err = j.Append(ctx, journal.Entry{Body: "unmatched memory"}, nil)
	require.NoError(t, err)
	_, err = s.DryRunRule(ctx, rule.ID)
	require.NoError(t, err)
	require.NoError(t, s.EnableRule(ctx, rule.ID))

	classifier := &mocks.FakeInference{ClassifyOut: "gotcha"}
	refacet := NewService(s.repo, classifier)
	result, err := refacet.RefacetCorpus(ctx, "agent-memory")
	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
	require.Equal(t, 2, result.Assigned)
	require.Equal(t, 1, result.RuleAssigned)
	require.Equal(t, 1, result.Classified)
	require.Zero(t, result.Failed)
}

type contextualRefacetClassifier struct{ kind string }

func (c *contextualRefacetClassifier) Classify(context.Context, string) (string, error) {
	return "thread", nil
}

func (c *contextualRefacetClassifier) ClassifyEntry(_ context.Context, _ string, kind string) (string, error) {
	c.kind = kind
	return "episode", nil
}

func (*contextualRefacetClassifier) Embed(context.Context, string, inference.EmbeddingTask) ([]float64, error) {
	return []float64{1}, nil
}

func (*contextualRefacetClassifier) Summarize(context.Context, string) (string, error) {
	return "", nil
}

func TestRefacetCorpusPassesEntryKindToContextualClassifier(t *testing.T) {
	s, _ := newService(t)
	repo := s.repo.(*SQLiteRepository)
	db := repo.db
	_, err := db.Exec(`INSERT INTO entries(id,scope,body,facet_id,source_runtime,kind,created_at) VALUES('contextual-entry','agent-memory','Trigger: request\nApproach: work\nEvidence: tests\nOutcome: done','unclassified','','work-record','2026-08-05T00:00:00Z')`)
	require.NoError(t, err)
	classifier := &contextualRefacetClassifier{}
	result, err := NewService(repo, classifier).RefacetCorpus(context.Background(), "agent-memory")
	require.NoError(t, err)
	require.Equal(t, 1, result.Classified)
	require.Equal(t, "work-record", classifier.kind)
	assignments, err := repo.Assignments(context.Background(), "contextual-entry")
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	require.Equal(t, "episode", assignments[0].FacetID)
}

func TestSupersessionLeavesOriginalEntryRetrievable(t *testing.T) { // [REQ:VMEM-P1-004]
	s, j := newService(t)
	ctx := context.Background()
	original, err := j.Append(ctx, journal.Entry{Body: "old rule", FacetID: "standing-rule"}, nil)
	require.NoError(t, err)
	replacement, err := j.Append(ctx, journal.Entry{Body: "new rule", FacetID: "standing-rule"}, nil)
	require.NoError(t, err)
	require.NoError(t, s.MarkSuperseded(ctx, original.ID, replacement.ID))

	stored, err := j.Get(ctx, original.ID)
	require.NoError(t, err)
	require.Equal(t, original.Body, stored.Body)
}

func TestStandingRulePinBudgetCreatesTradeoffAndRenewalDoesNotDuplicate(t *testing.T) { // [REQ:VMEM-P1-010]
	s, j := newService(t)
	ctx := context.Background()
	ids := make([]string, 0, 9)
	for i := 0; i < 9; i++ {
		entry, err := j.Append(ctx, journal.Entry{Body: "standing rule " + time.Now().Format("150405.000000") + string(rune('a'+i)), FacetID: "standing-rule"}, nil)
		require.NoError(t, err)
		_, err = s.ReFacet(ctx, Assignment{EntryID: entry.ID, FacetID: "standing-rule"})
		require.NoError(t, err)
		ids = append(ids, entry.ID)
	}
	for _, id := range ids[:8] {
		require.NoError(t, s.SetPin(ctx, id, true))
	}
	var budget ErrPinBudgetExceeded
	require.ErrorAs(t, s.SetPin(ctx, ids[8], true), &budget)
	proposals, err := s.ListPinProposals(ctx)
	require.NoError(t, err)
	var found bool
	for _, proposal := range proposals {
		if proposal.ID == budget.ProposalID {
			found = true
			require.Contains(t, proposal.EntryIDs, ids[8])
		}
	}
	require.True(t, found)
	require.NoError(t, s.ResolvePinProposal(ctx, budget.ProposalID, true))
	var pinCount int
	require.NoError(t, s.repo.(*SQLiteRepository).db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pins`).Scan(&pinCount))
	require.Equal(t, 8, pinCount)
	require.True(t, mustPinned(t, s, ctx, ids[8]))
	_, err = s.repo.(*SQLiteRepository).db.ExecContext(ctx, `UPDATE pins SET review_at=? WHERE entry_id=?`, time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano), ids[1])
	require.NoError(t, err)
	require.False(t, mustPinned(t, s, ctx, ids[1]))
}

func mustPinned(t *testing.T, s *Service, ctx context.Context, id string) bool {
	t.Helper()
	ok, err := s.repo.(*SQLiteRepository).Pinned(ctx, id)
	require.NoError(t, err)
	return ok
}

// A thread is unresolved work. Once resolved it must leave ambient recall
// without being summarized: expire-on-resolution is how a facet leaves the
// frontier when its retention policy forbids compaction.
func TestResolvedThreadExcludedFromWakeAndCompaction(t *testing.T) { // [REQ:VMEM-P0-005]
	ctx := context.Background()
	s, journalRepo := newService(t)

	open, err := journalRepo.Append(ctx, journal.Entry{Body: "still investigating the flake", FacetID: "thread"}, nil)
	require.NoError(t, err)
	resolved, err := journalRepo.Append(ctx, journal.Entry{Body: "flake root-caused and fixed", FacetID: "thread"}, nil)
	require.NoError(t, err)
	require.NoError(t, s.Assign(ctx, open.ID, "thread", "test"))
	require.NoError(t, s.Assign(ctx, resolved.ID, "thread", "test"))

	policies, err := s.List(ctx)
	require.NoError(t, err)
	var threadPolicy string
	var threadCompactable bool
	for _, p := range policies {
		if p.ID == "thread" {
			threadPolicy, threadCompactable = p.RetentionPolicy, p.CompactionEligible
		}
	}
	require.Equal(t, "expire-on-resolution", threadPolicy)
	require.False(t, threadCompactable, "a thread must never be summarized; it expires instead")

	require.NoError(t, s.ResolveThread(ctx, resolved.ID))

	// The resolved thread is marked, the open one is not. Recall's node source
	// drops exactly the marked-and-expiring entries.
	var marked int
	require.NoError(t, s.repo.(*SQLiteRepository).db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM marks WHERE entry_id=? AND kind='resolved'`, resolved.ID).Scan(&marked))
	require.Equal(t, 1, marked)
	require.NoError(t, s.repo.(*SQLiteRepository).db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM marks WHERE entry_id=? AND kind='resolved'`, open.ID).Scan(&marked))
	require.Equal(t, 0, marked, "an unresolved thread stays in ambient recall")
}
