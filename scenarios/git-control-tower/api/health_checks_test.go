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
	if result.Dependencies["repository"] == "" {
		t.Fatalf("expected repository dependency status to be set")
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
	deps := map[string]string{}
	errs := map[string]string{}

	ok := h.checkRepository(ctx, deps, errs)
	if !ok {
		t.Fatalf("expected repository check to pass, got deps=%v errs=%v", deps, errs)
	}
	if deps["repository"] != "accessible" {
		t.Fatalf("expected deps[repository]=accessible, got %q", deps["repository"])
	}
}

