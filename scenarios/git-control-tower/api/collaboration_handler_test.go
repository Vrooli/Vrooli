package main

import (
	"net/http/httptest"
	"testing"
)

func TestCollaborationStatusSeparatesUnconfiguredFromHostUnavailable(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/collaboration/status", nil)
	host, err := collaborationHostFromRequest(req)
	if err == nil || host.RepositoryID != "" {
		t.Fatalf("unexpected configured host: %+v err=%v", host, err)
	}
	req = httptest.NewRequest("GET", "/api/v1/collaboration/status?kind=github&instance=https%3A%2F%2Fgithub.example&repository=org%2Frepo", nil)
	host, err = collaborationHostFromRequest(req)
	if err != nil || host.RepositoryID != "org/repo" {
		t.Fatalf("host identity not preserved: %+v err=%v", host, err)
	}
}
