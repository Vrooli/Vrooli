package ledger_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"content-desk/internal/artifacts"
	"content-desk/internal/claims"
	claimsmocks "content-desk/internal/claims/mocks"
	"content-desk/internal/ledger"
	"content-desk/internal/testutil/db"

	localdb "content-desk/internal/database"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
)

func newLedgerDB(t *testing.T) *sqlFixture {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), d,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(artifacts.Schema),
		database.SchemaProviderFunc(claims.Schema),
		database.SchemaProviderFunc(ledger.Schema),
	))
	return &sqlFixture{db: d, repo: ledger.NewSQLiteRepository(d), importer: ledger.NewImporter(d)}
}

// [REQ:CONTENTD-P1-002]
func TestContaminatedByClaimReturnsOnlyPublishedDraftsCitingClaim(t *testing.T) {
	t.Run("[CONTENTD-P1-002] contamination lists cited publishes", func(t *testing.T) {
		fixture := newLedgerDB(t)
		ctx := context.Background()
		_, err := fixture.db.ExecContext(ctx, `INSERT INTO claims (id, statement, kind, verification_status, created_at) VALUES ('claim-1', 'A claim', 'qualitative', 'verified', '2026-07-01T00:00:00Z')`)
		require.NoError(t, err)
		_, err = fixture.db.ExecContext(ctx, `INSERT INTO claim_citations (draft_id, claim_id, span_start, span_end) VALUES ('draft-cited', 'claim-1', 0, 7)`)
		require.NoError(t, err)
		_, err = fixture.repo.RecordPublish(ctx, ledger.PublishRecord{ID: "published-cited", DraftID: "draft-cited", Channel: "x", SourceKind: "gated", PublishedAt: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)})
		require.NoError(t, err)
		_, err = fixture.repo.RecordPublish(ctx, ledger.PublishRecord{ID: "published-unrelated", DraftID: "draft-unrelated", Channel: "x", SourceKind: "gated", PublishedAt: time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)})
		require.NoError(t, err)

		contaminated, err := fixture.repo.ContaminatedByClaim(ctx, "claim-1")
		require.NoError(t, err)
		require.Len(t, contaminated, 1)
		require.Equal(t, "published-cited", contaminated[0].ID)
	})
}

// [REQ:CONTENTD-P1-001] [REQ:CONTENTD-P1-002]
func TestChangedCheckStalesClaimAndSurfacesEveryCitingPublish(t *testing.T) {
	t.Run("[CONTENTD-P1-001] changed checks stale claims and contaminate publishes", func(t *testing.T) {
		fixture := newLedgerDB(t)
		ctx := context.Background()
		body := "The metric is current."
		library := claims.NewLibrary(fixture.db, &claimsmocks.FakeRunner{Result: claims.CheckResult{ActualResult: "current", Matches: true}})
		claim, err := library.Create(ctx, claims.Claim{Statement: "The metric is current", Kind: claims.KindQuantitative}, claims.Evidence{Kind: claims.EvidenceKindCheck, Command: "ignored", ExpectedResult: "current"})
		require.NoError(t, err)
		require.NoError(t, library.Cite(ctx, claims.Citation{DraftID: "draft-a", ClaimID: claim.ID, Start: 0, End: len(body)}, body))
		require.NoError(t, library.Cite(ctx, claims.Citation{DraftID: "draft-b", ClaimID: claim.ID, Start: 0, End: len(body)}, body))
		_, err = fixture.repo.RecordPublish(ctx, ledger.PublishRecord{ID: "published-a", DraftID: "draft-a", Channel: "x", PublishedAt: time.Now().UTC()})
		require.NoError(t, err)
		_, err = fixture.repo.RecordPublish(ctx, ledger.PublishRecord{ID: "published-b", DraftID: "draft-b", Channel: "x", PublishedAt: time.Now().UTC()})
		require.NoError(t, err)

		updated, err := library.Sweep(ctx)
		require.NoError(t, err)
		require.Len(t, updated, 1)
		require.Equal(t, claims.StateVerified, updated[0].VerificationStatus)

		library = claims.NewLibrary(fixture.db, &claimsmocks.FakeRunner{Result: claims.CheckResult{ActualResult: "changed", Matches: false}})
		updated, err = library.Sweep(ctx)
		require.NoError(t, err)
		require.Equal(t, claims.StateStale, updated[0].VerificationStatus)
		contaminated, err := fixture.repo.ContaminatedByClaim(ctx, claim.ID)
		require.NoError(t, err)
		require.Len(t, contaminated, 2)
		require.ElementsMatch(t, []string{"published-a", "published-b"}, []string{contaminated[0].ID, contaminated[1].ID})
	})
}

// [REQ:CONTENTD-P1-004]
func TestCoverageGroupsPersistedDimensionsAndMarksOldCellsStale(t *testing.T) {
	t.Run("[CONTENTD-P1-004] coverage groups campaign lane channel and sku", func(t *testing.T) {
		fixture := newLedgerDB(t)
		ctx := context.Background()
		_, err := fixture.db.ExecContext(ctx, `INSERT INTO drafts (id, campaign_id, post_type_id, lane, sku, body, status, created_at, updated_at) VALUES ('draft-fresh', 'campaign-a', 'education', 'launch', 'sku-1', '', 'published', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z'), ('draft-old', 'campaign-a', 'education', 'launch', 'sku-1', '', 'published', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z'), ('draft-stale', 'campaign-a', 'education', 'retention', 'sku-2', '', 'published', '2026-07-01T00:00:00Z', '2026-07-01T00:00:00Z')`)
		require.NoError(t, err)
		now := time.Now().UTC()
		_, err = fixture.repo.RecordPublish(ctx, ledger.PublishRecord{ID: "fresh", DraftID: "draft-fresh", Channel: "linkedin", PublishedAt: now})
		require.NoError(t, err)
		_, err = fixture.repo.RecordPublish(ctx, ledger.PublishRecord{ID: "old", DraftID: "draft-old", Channel: "linkedin", PublishedAt: now.Add(-48 * time.Hour)})
		require.NoError(t, err)
		_, err = fixture.repo.RecordPublish(ctx, ledger.PublishRecord{ID: "stale", DraftID: "draft-stale", Channel: "linkedin", PublishedAt: now.Add(-48 * time.Hour)})
		require.NoError(t, err)

		cells, err := fixture.repo.Coverage(ctx, 24*time.Hour)
		require.NoError(t, err)
		require.Len(t, cells, 2)
		require.Equal(t, "campaign-a", cells[0].CampaignID)
		require.Equal(t, "launch", cells[0].Lane)
		require.Equal(t, "linkedin", cells[0].Channel)
		require.Equal(t, "sku-1", cells[0].SKU)
		require.Equal(t, 2, cells[0].PublishCount)
		require.False(t, cells[0].Stale, "the newest publish keeps its coverage cell fresh")
		require.Equal(t, "retention", cells[1].Lane)
		require.Equal(t, "sku-2", cells[1].SKU)
		require.True(t, cells[1].Stale, "a dimension with no recent publish is stale")
	})
}

type sqlFixture struct {
	db       *sql.DB
	repo     ledger.Repository
	importer *ledger.Importer
}

// [REQ:CONTENTD-P0-011] [REQ:CONTENTD-P0-012]
func TestRecordPublishAllowsImportedHistoryWithoutDraftAndListsIt(t *testing.T) {
	t.Run("[CONTENTD-P0-011] imported history records familiarity and narration inputs", func(t *testing.T) {
		fixture := newLedgerDB(t)
		when := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
		recorded, err := fixture.repo.RecordPublish(context.Background(), ledger.PublishRecord{Channel: "x-twitter", Audience: "oss-contributor", PublishedURL: "https://example.test/post/1", PlatformPostID: "1", SourceKind: "imported", PublishedAt: when})
		require.NoError(t, err)
		require.Empty(t, recorded.DraftID)

		history, err := fixture.repo.ListPublishHistory(context.Background(), 10)
		require.NoError(t, err)
		require.Len(t, history, 1)
		require.Equal(t, recorded.ID, history[0].ID)
		require.Empty(t, history[0].DraftID)
		require.Equal(t, "imported", history[0].SourceKind)
	})
}

// [REQ:CONTENTD-P0-012]
func TestImportedNarrationIsQueryableByScenario(t *testing.T) {
	t.Run("[CONTENTD-P0-012] narration ledger is queryable by scenario", func(t *testing.T) {
		fixture := newLedgerDB(t)
		path := filepath.Join(t.TempDir(), "improvements.jsonl")
		require.NoError(t, os.WriteFile(path, []byte(`{"subject":"Claim coverage","scenario":"content-desk","at":"2026-07-28T00:00:00Z"}`+"\n"), 0o600))
		result, err := fixture.importer.Import(context.Background(), []ledger.ImportSource{{Name: "published-improvements-log", Path: path}})
		require.NoError(t, err)
		require.True(t, result.Complete)
		items, err := fixture.repo.NarratedForScenario(context.Background(), "content-desk")
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, "Claim coverage", items[0].Subject)
	})
}

// [REQ:CONTENTD-P0-013]
func TestImporterIsIdempotentByNormalizedContentKey(t *testing.T) {
	t.Run("[CONTENTD-P0-013] import is idempotent by normalized content key", func(t *testing.T) {
		fixture := newLedgerDB(t)
		path := filepath.Join(t.TempDir(), "mentions.jsonl")
		line := `{"audience":"oss-contributor","subject":"Vrooli","subject_kind":"concept","at":"2026-04-27T19:30:00Z","is_first_mention":true}` + "\n"
		require.NoError(t, os.WriteFile(path, []byte(line), 0o600))
		source := ledger.ImportSource{Name: "published-scenario-mentions", Path: path}

		first, err := fixture.importer.Import(context.Background(), []ledger.ImportSource{source})
		require.NoError(t, err)
		require.True(t, first.Complete)
		require.Equal(t, 1, first.Imported)

		second, err := fixture.importer.Import(context.Background(), []ledger.ImportSource{source})
		require.NoError(t, err)
		require.True(t, second.Complete)
		require.Zero(t, second.Imported)
		require.Equal(t, 1, second.Skipped)

		familiarity, err := fixture.repo.SubjectFamiliarity(context.Background(), "Vrooli", "oss-contributor")
		require.NoError(t, err)
		require.Equal(t, 1, familiarity.MentionCount)
		require.True(t, familiarity.FirstMention)
	})
}

func TestImporterReportsPerSourceFailureAndNeverClaimsComplete(t *testing.T) {
	fixture := newLedgerDB(t)
	goodPath := filepath.Join(t.TempDir(), "mentions.jsonl")
	require.NoError(t, os.WriteFile(goodPath, []byte(`{"subject":"Vrooli","audience":"oss-contributor"}`+"\n"), 0o600))
	missing := filepath.Join(t.TempDir(), "missing.jsonl")

	result, err := fixture.importer.Import(context.Background(), []ledger.ImportSource{
		{Name: "published-scenario-mentions", Path: goodPath},
		{Name: "publish-log", Path: missing},
	})
	require.NoError(t, err)
	require.False(t, result.Complete)
	require.Len(t, result.Failures, 1)
	require.Equal(t, "publish-log", result.Failures[0].Source)
	require.Equal(t, 1, result.Imported)
}
