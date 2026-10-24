package routing

import (
	"testing"

	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestCollapseDocumentHitsKeepsBestPassageAndReportsMergedCount(t *testing.T) {
	groups := []*routingv1.ProviderResultGroup{{
		ProviderId: "docs",
		Hits: []*routingv1.SearchHit{
			{Id: "chunk-1", Path: "docs/flows.md", Snippet: "weaker passage", Score: 0.20},
			{Id: "chunk-2", Path: "docs/flows.md", Snippet: "best passage", Score: 0.80},
			{Id: "chunk-3", Path: "docs/other.md", Snippet: "other document", Score: 0.70},
		},
	}}

	got := collapseDocumentHits(groups)
	require.Len(t, got, 1)
	require.Len(t, got[0].GetHits(), 2)
	require.Equal(t, "best passage", got[0].GetHits()[0].GetSnippet())
	require.EqualValues(t, 2, got[0].GetHits()[0].GetMergedCount())
	require.EqualValues(t, 1, got[0].GetHits()[1].GetMergedCount())
}

func TestCollapseDocumentHitsDoesNotCrossProviderGroups(t *testing.T) {
	groups := []*routingv1.ProviderResultGroup{
		{ProviderId: "one", Hits: []*routingv1.SearchHit{{Id: "a", Path: "same.md", Score: 0.4}}},
		{ProviderId: "two", Hits: []*routingv1.SearchHit{{Id: "b", Path: "same.md", Score: 0.9}}},
	}

	got := collapseDocumentHits(groups)
	require.Len(t, got[0].GetHits(), 1)
	require.Len(t, got[1].GetHits(), 1)
	require.EqualValues(t, 1, got[0].GetHits()[0].GetMergedCount())
	require.EqualValues(t, 1, got[1].GetHits()[0].GetMergedCount())
}
