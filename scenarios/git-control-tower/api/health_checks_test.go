package main

import (
	"context"
	"fmt"
	"testing"
)

// [REQ:GCT-OT-P0-001] Health check endpoint

func TestHealthChecks_Run_UnhealthyWhenRepoMissing(t *testing.T) {
	fakeGit := NewFakeGitRunner()

	h := NewHealthChecks(HealthCheckDeps{
		DB:      nil,
		Git:     fakeGit,
		RepoDir: "",
	})

	result := h.Run(context.Background())
	if result.Readiness {
		t.Fatalf("expected readiness=false, got true")
	}
	if result.Status != "unhealthy" {
		t.Fatalf("expected status=unhealthy, got %q", result.Status)
	}
	if dep, ok := result.Dependencies["repository"]; !ok || dep.Status == "" {
		t.Fatalf("expected repository dependency to be set")
	}
}

func TestHealthChecks_CheckRepository_WithFakeRepo(t *testing.T) {
	fakeGit := NewFakeGitRunner()

	h := NewHealthChecks(HealthCheckDeps{
		DB:      nil,
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	deps := map[string]HealthDependency{}
	errs := map[string]string{}

	ok := h.checkRepository(context.Background(), deps, errs)
	if !ok {
		t.Fatalf("expected repository check to pass, got deps=%v errs=%v", deps, errs)
	}
	if deps["repository"].Status != "accessible" || !deps["repository"].Connected {
		t.Fatalf("expected deps[repository]=accessible+connected, got %+v", deps["repository"])
	}
	if !fakeGit.AssertCalled("RevParse") {
		t.Fatalf("expected RevParse to be called")
	}
}

func TestHealthChecks_CheckRepository_NotARepo(t *testing.T) {
	fakeGit := NewFakeGitRunner().WithNotARepository()
	fakeGit.RevParseError = fmt.Errorf("fatal: not a git repository")

	h := NewHealthChecks(HealthCheckDeps{
		DB:      nil,
		Git:     fakeGit,
		RepoDir: "/not/a/repo",
	})
	deps := map[string]HealthDependency{}
	errs := map[string]string{}

	ok := h.checkRepository(context.Background(), deps, errs)
	if ok {
		t.Fatalf("expected repository check to fail")
	}
	if deps["repository"].Status != "unavailable" {
		t.Fatalf("expected status=unavailable, got %q", deps["repository"].Status)
	}
}

func TestHealthChecks_CheckGitBinary_Available(t *testing.T) {
	fakeGit := NewFakeGitRunner()

	h := NewHealthChecks(HealthCheckDeps{
		DB:      nil,
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	deps := map[string]HealthDependency{}
	errs := map[string]string{}

	ok := h.checkGitBinary(deps, errs)
	if !ok {
		t.Fatalf("expected git binary check to pass")
	}
	if deps["git"].Status != "available" {
		t.Fatalf("expected status=available, got %q", deps["git"].Status)
	}
}

func TestHealthChecks_CheckGitBinary_Missing(t *testing.T) {
	fakeGit := NewFakeGitRunner().WithGitUnavailable()

	h := NewHealthChecks(HealthCheckDeps{
		DB:      nil,
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	deps := map[string]HealthDependency{}
	errs := map[string]string{}

	ok := h.checkGitBinary(deps, errs)
	if ok {
		t.Fatalf("expected git binary check to fail")
	}
	if deps["git"].Status != "missing" {
		t.Fatalf("expected status=missing, got %q", deps["git"].Status)
	}
}

func TestHealthChecks_Run_Healthy(t *testing.T) {
	fakeGit := NewFakeGitRunner()

	// Note: DB is nil so database check will fail, but git and repo should pass
	h := NewHealthChecks(HealthCheckDeps{
		DB:      nil,
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})

	result := h.Run(context.Background())
	// Readiness is false because DB is nil
	if result.Readiness {
		t.Fatalf("expected readiness=false (DB nil), got true")
	}

	// But git and repository should be healthy
	if result.Dependencies["git"].Status != "available" {
		t.Fatalf("expected git=available, got %q", result.Dependencies["git"].Status)
	}
	if result.Dependencies["repository"].Status != "accessible" {
		t.Fatalf("expected repository=accessible, got %q", result.Dependencies["repository"].Status)
	}
}
