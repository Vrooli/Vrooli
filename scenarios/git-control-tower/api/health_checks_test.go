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

// --- Database seam tests (FakeDBChecker) ---

func TestHealthChecks_Run_AllHealthy(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeDB := NewFakeDBChecker()

	h := NewHealthChecks(HealthCheckDeps{
		DB:      fakeDB,
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})

	result := h.Run(context.Background())
	if !result.Readiness {
		t.Fatalf("expected readiness=true, got false; errors=%v", result.Errors)
	}
	if result.Status != "healthy" {
		t.Fatalf("expected status=healthy, got %q", result.Status)
	}
	if result.Dependencies["database"].Status != "connected" {
		t.Fatalf("expected database=connected, got %q", result.Dependencies["database"].Status)
	}
	if fakeDB.PingCalls != 1 {
		t.Fatalf("expected 1 ping call, got %d", fakeDB.PingCalls)
	}
}

func TestHealthChecks_CheckDatabase_Unconfigured(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeDB := NewFakeDBChecker().WithUnconfigured()

	h := NewHealthChecks(HealthCheckDeps{
		DB:      fakeDB,
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	deps := map[string]HealthDependency{}
	errs := map[string]string{}

	ok := h.checkDatabase(context.Background(), deps, errs)
	if ok {
		t.Fatalf("expected database check to fail for unconfigured")
	}
	if deps["database"].Status != "unconfigured" {
		t.Fatalf("expected status=unconfigured, got %q", deps["database"].Status)
	}
}

func TestHealthChecks_CheckDatabase_Disconnected(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeDB := NewFakeDBChecker().WithDisconnected()

	h := NewHealthChecks(HealthCheckDeps{
		DB:      fakeDB,
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	deps := map[string]HealthDependency{}
	errs := map[string]string{}

	ok := h.checkDatabase(context.Background(), deps, errs)
	if ok {
		t.Fatalf("expected database check to fail for disconnected")
	}
	if deps["database"].Status != "disconnected" {
		t.Fatalf("expected status=disconnected, got %q", deps["database"].Status)
	}
	if errs["database"] == "" {
		t.Fatalf("expected error message for disconnected database")
	}
}

func TestHealthChecks_CheckDatabase_PingError(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeDB := NewFakeDBChecker().WithPingError(fmt.Errorf("connection timeout"))

	h := NewHealthChecks(HealthCheckDeps{
		DB:      fakeDB,
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	deps := map[string]HealthDependency{}
	errs := map[string]string{}

	ok := h.checkDatabase(context.Background(), deps, errs)
	if ok {
		t.Fatalf("expected database check to fail for ping error")
	}
	if deps["database"].Status != "disconnected" {
		t.Fatalf("expected status=disconnected, got %q", deps["database"].Status)
	}
	if errs["database"] != "connection timeout" {
		t.Fatalf("expected error='connection timeout', got %q", errs["database"])
	}
}

func TestHealthChecks_CheckDatabase_Connected(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeDB := NewFakeDBChecker()

	h := NewHealthChecks(HealthCheckDeps{
		DB:      fakeDB,
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	deps := map[string]HealthDependency{}
	errs := map[string]string{}

	ok := h.checkDatabase(context.Background(), deps, errs)
	if !ok {
		t.Fatalf("expected database check to pass, got errs=%v", errs)
	}
	if deps["database"].Status != "connected" {
		t.Fatalf("expected status=connected, got %q", deps["database"].Status)
	}
	if !deps["database"].Connected {
		t.Fatalf("expected Connected=true")
	}
}
