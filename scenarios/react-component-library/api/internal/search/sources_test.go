package search

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
)

func TestCatalogPageTypeLinksOpenTheMatchingLibraryFilter(t *testing.T) {
	name := "Landing Page"
	href := catalogHref(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, name)
	u, err := url.Parse(href)
	require.NoError(t, err)
	require.Equal(t, "/library", u.Path)
	require.Equal(t, "templates", u.Query().Get("domain"))
	require.Equal(t, "components", u.Query().Get("group"))
	require.Equal(t, name, u.Query().Get("q"))
}

func TestCommandDocumentsMatchImplementedWorkbenchShortcuts(t *testing.T) {
	docs := commandDocuments(time.Unix(0, 0))
	byID := make(map[string]Document, len(docs))
	for _, doc := range docs {
		require.Equal(t, searchv1.SearchKind_SEARCH_KIND_COMMAND, doc.Kind)
		require.NotContains(t, byID, doc.SubjectID, "command IDs must be unique")
		byID[doc.SubjectID] = doc
	}

	cases := []struct {
		id, title, shortcut, href, description string
	}{
		{"command.new-workspace", "New workspace", "N", "/design?new=1", "Start a new workspace in Design."},
		{"command.go-design", "Go to Design", "G D", "/design", "Open the Design destination."},
		{"command.go-stats", "Go to Stats", "G S", "/stats", "Open the Stats destination."},
	}
	for _, tc := range cases {
		doc, ok := byID[tc.id]
		require.True(t, ok, "missing command %s", tc.id)
		require.Equal(t, tc.title, doc.Title)
		require.Equal(t, tc.href, doc.Href)
		require.Contains(t, doc.Body, tc.description)
		require.Equal(t, []string{tc.shortcut}, doc.Badges)
		require.Contains(t, doc.Body, tc.shortcut, "the shortcut must participate in search")
	}
	require.NotContains(t, byID, "command.reindex-search", "Settings does not perform a reindex")
	require.NotContains(t, byID, "command.search", "⌘K opens Spotlight; navigating to Start does not execute that action")
}
