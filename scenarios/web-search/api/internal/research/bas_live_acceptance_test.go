package research_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"

	"web-search/internal/evidence"
	"web-search/internal/research"
	"web-search/internal/research/fetch"
)

type liveThinHTTPFetcher struct{}

func (liveThinHTTPFetcher) Fetch(context.Context, string) (string, error) {
	return "Please enable JavaScript to continue.", nil
}

// TestLiveBASFetcherRetainsExecutionIdentity is an attended integration check.
// It is opt-in because it launches a real BAS browser capture and is not a
// hermetic unit test.
func TestLiveBASFetcherRetainsExecutionIdentity(t *testing.T) {
	if os.Getenv("WEB_SEARCH_RUN_BAS_LIVE") != "1" {
		t.Skip("set WEB_SEARCH_RUN_BAS_LIVE=1 for the attended BAS acceptance check")
	}
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(evidence.Schema)))
	repo := evidence.NewSQLiteRepository(db, func() time.Time { return time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC) })
	url := "https://example.com/"
	svc := research.NewService(research.Deps{
		Searcher: &fakeSearcher{candidates: []research.Candidate{{URL: url, Title: "Example Domain"}}},
		Fetcher:  &fetch.EscalatingFetcher{HTTP: liveThinHTTPFetcher{}, Browser: fetch.NewBASFetcher()},
		Synthesizer: &fakeSynthesizer{result: research.Synthesis{
			Text:      "Example Domain is available.",
			Citations: []research.Citation{{ResultIndex: 0, URL: url, Title: "Example Domain"}},
		}},
		ReceiptStore: repo,
	})
	out, err := svc.RunL2(context.Background(), "read the rendered example page", 1, false)
	require.NoError(t, err)
	require.Len(t, out.EvidenceReceiptIDs, 1)
	receipt, err := repo.GetReceipt(context.Background(), out.EvidenceReceiptIDs[0])
	require.NoError(t, err)
	require.NotEmpty(t, receipt.ProducerExecutionID)
	require.Equal(t, "bas-readable-text-v1", receipt.ExtractionRevision)
	t.Logf("BAS execution=%s evidence_receipt=%s", receipt.ProducerExecutionID, receipt.ReceiptID)
	passage, err := repo.CreatePassage(context.Background(), receipt.ReceiptID, 0, len([]byte("Example Domain")))
	require.NoError(t, err)
	require.NotEmpty(t, passage.PassageID)
}
