package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAccessibilityTreeUsesLeaseHeadersAndExpectedPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "exec-1", r.Header.Get(SessionExecutionHeader))
		require.Equal(t, "lease-1", r.Header.Get(SessionLeaseHeader))
		require.Equal(t, "page-1", r.URL.Query().Get("expected_page_id"))
		_ = json.NewEncoder(w).Encode(AccessibilityTreeResponse{
			SessionID: "session-1", PageID: "page-1",
			Tree: &AccessibilityTree{Contract: "bas-accessibility-tree-observation/v1", Root: AccessibilityTreeNode{Role: "RootWebArea", Children: []AccessibilityTreeNode{}}},
		})
	}))
	defer server.Close()
	client, err := NewClientWithURL(server.URL, WithoutCircuitBreaker())
	require.NoError(t, err)
	response, err := client.GetAccessibilityTree(context.Background(), "session-1", "exec-1", "lease-1", "page-1")
	require.NoError(t, err)
	require.Equal(t, "RootWebArea", response.Tree.Root.Role)
}

func TestGetAccessibilityTreeRejectsWrongOwnerAndPageIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(AccessibilityTreeResponse{SessionID: "some-other-session", PageID: "page-2", Tree: &AccessibilityTree{Contract: "bas-accessibility-tree-observation/v1"}})
	}))
	defer server.Close()
	client, err := NewClientWithURL(server.URL, WithoutCircuitBreaker())
	require.NoError(t, err)
	_, err = client.GetAccessibilityTree(context.Background(), "session-1", "exec-1", "lease-1", "page-1")
	require.ErrorContains(t, err, "different session or page")
}
