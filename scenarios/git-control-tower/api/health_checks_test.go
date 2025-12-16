package main

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// [REQ:GCT-OT-P0-001] Health check endpoint

func TestHealthChecks_Run_UnhealthyWhenRepoMissing(t *testing.T) {
	h := NewHealthChecks(HealthCheckDeps{
		DB:      nil,
		GitPath: "git",
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

func TestHealthChecks_CheckRepository_WithGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	cmd := exec.Command("git", "-C", repoDir, "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, string(out))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h := NewHealthChecks(HealthCheckDeps{
		DB:      nil,
		GitPath: "git",
		RepoDir: repoDir,
	})
	deps := map[string]HealthDependency{}
	errs := map[string]string{}

	ok := h.checkRepository(ctx, deps, errs)
	if !ok {
		t.Fatalf("expected repository check to pass, got deps=%v errs=%v", deps, errs)
	}
	if deps["repository"].Status != "accessible" || !deps["repository"].Connected {
		t.Fatalf("expected deps[repository]=accessible+connected, got %+v", deps["repository"])
	}
}
