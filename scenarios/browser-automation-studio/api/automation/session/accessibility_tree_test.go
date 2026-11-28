package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

type accessibilityAuthorizer struct {
	requests []sessionpermission.ActionRequest
	err      error
}

func (a *accessibilityAuthorizer) Authorize(_ context.Context, request sessionpermission.ActionRequest) error {
	a.requests = append(a.requests, request)
	return a.err
}

type accessibilityCaller struct{}

func (accessibilityCaller) BrowserCallerID(context.Context) (string, error) {
	return "caller:tree", nil
}

func TestObserveAccessibilityTreeAuthorizesCurrentPageOnceUnderLease(t *testing.T) {
	var treeReads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session/tree-session/record/navigation-state":
			_, _ = w.Write([]byte(`{"session_id":"tree-session","url":"https://allowed.example/current","title":"Current"}`))
		case "/session/tree-session/accessibility-tree":
			treeReads++
			require.Equal(t, "exec-tree", r.Header.Get(driver.SessionExecutionHeader))
			require.Equal(t, "lease-tree", r.Header.Get(driver.SessionLeaseHeader))
			require.Equal(t, "page-tree", r.URL.Query().Get("expected_page_id"))
			_ = json.NewEncoder(w).Encode(driver.AccessibilityTreeResponse{SessionID: "tree-session", PageID: "page-tree", Tree: &driver.AccessibilityTree{Contract: "bas-accessibility-tree-observation/v1", URL: "https://allowed.example", NodeCount: 1, Root: driver.AccessibilityTreeNode{Role: "RootWebArea", Children: []driver.AccessibilityTreeNode{}}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	authorizer := &accessibilityAuthorizer{}
	owner := &Session{
		id: "tree-session", executionID: "exec-tree", leaseID: "lease-tree",
		initialDriverPageID: "page-tree", activeDriverPageID: "page-tree", client: client,
		authorizer: authorizer, callerResolver: accessibilityCaller{}, expectedCallerID: "caller:tree",
	}
	response, err := owner.ObserveAccessibilityTree(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, "RootWebArea", response.Tree.Root.Role)
	require.Equal(t, 1, treeReads)
	require.Len(t, authorizer.requests, 1)
	require.Equal(t, "observe", authorizer.requests[0].Kind)
	require.Equal(t, "https://allowed.example/current", authorizer.requests[0].CurrentURL)
	require.Equal(t, "caller:tree", authorizer.requests[0].CallerID)
	require.Equal(t, "lease-tree", authorizer.requests[0].LeaseID)
}

func TestObserveAccessibilityTreeDoesNotReadAfterOwnerDenies(t *testing.T) {
	treeReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/tree-session/record/navigation-state" {
			_, _ = w.Write([]byte(`{"session_id":"tree-session","url":"https://outside.example/private","title":"Outside"}`))
			return
		}
		if r.URL.Path == "/session/tree-session/accessibility-tree" {
			treeReads++
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	authorizer := &accessibilityAuthorizer{err: sessionpermission.ErrDenied}
	owner := &Session{id: "tree-session", executionID: "exec-tree", leaseID: "lease-tree", initialDriverPageID: "page-tree", activeDriverPageID: "page-tree", client: client, authorizer: authorizer, callerResolver: accessibilityCaller{}, expectedCallerID: "caller:tree"}
	_, err = owner.ObserveAccessibilityTree(context.Background(), "page-tree")
	require.ErrorIs(t, err, sessionpermission.ErrDenied)
	require.Equal(t, 0, treeReads)
	require.Len(t, authorizer.requests, 1)
	require.Equal(t, "https://outside.example/private", authorizer.requests[0].CurrentURL)
}
