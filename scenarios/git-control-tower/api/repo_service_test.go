package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestRepoService_OpenSetsActive(t *testing.T) {
	store := newTestRepoStore(t)
	git := NewFakeGitRunner()
	service := NewRepoService(store, git)

	repo, err := service.Open(context.Background(), "/demo")
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	if repo == nil || repo.Path != "/demo" {
		t.Fatalf("unexpected repo: %#v", repo)
	}

	active, err := store.GetActive(context.Background())
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if active == nil || active.ID != repo.ID {
		t.Fatalf("expected active repo id %d, got %#v", repo.ID, active)
	}
}

func TestRepoService_ResolveWithRepoID(t *testing.T) {
	store := newTestRepoStore(t)
	git := NewFakeGitRunner()
	service := NewRepoService(store, git)

	repo, err := service.Open(context.Background(), "/demo")
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/repo/status", nil)
	req.Header.Set(repoIDHeader, fmt.Sprintf("%d", repo.ID))

	resolved, err := service.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("resolve repo: %v", err)
	}
	if resolved.ID != repo.ID {
		t.Fatalf("expected repo id %d, got %d", repo.ID, resolved.ID)
	}
	if resolved.Path != repo.Path {
		t.Fatalf("expected repo path %q, got %q", repo.Path, resolved.Path)
	}
}

func TestRepoService_ResolveFallbackRegistersRepo(t *testing.T) {
	store := newTestRepoStore(t)
	git := NewFakeGitRunner()
	git.RepoRoot = "/fallback"
	service := NewRepoService(store, git)

	req := httptest.NewRequest("GET", "/api/v1/repo/status", nil)
	resolved, err := service.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("resolve repo: %v", err)
	}
	if resolved.Path != "/fallback" {
		t.Fatalf("expected fallback path, got %q", resolved.Path)
	}

	active, err := store.GetActive(context.Background())
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if active == nil || active.Path != "/fallback" {
		t.Fatalf("expected active fallback repo, got %#v", active)
	}
}

func TestRepoService_CloneCallsGitClone(t *testing.T) {
	store := newTestRepoStore(t)
	git := NewFakeGitRunner()
	service := NewRepoService(store, git)

	repo, err := service.Clone(context.Background(), "https://example.com/repo.git", "/clone")
	if err != nil {
		t.Fatalf("clone repo: %v", err)
	}
	if repo == nil || repo.Path != "/clone" {
		t.Fatalf("unexpected repo: %#v", repo)
	}

	found := false
	for _, call := range git.Calls {
		if call.Method == "Clone" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected git clone to be called")
	}
}
