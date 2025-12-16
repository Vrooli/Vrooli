package main

import (
	"context"
	"strings"
	"time"
)

// HealthCheckDeps contains dependencies for health checks.
// SEAM: Uses GitRunner interface for git operations, enabling testing without real git.
// SEAM: Uses DBChecker interface for database operations, enabling testing without real database.
type HealthCheckDeps struct {
	DB      DBChecker
	Git     GitRunner
	RepoDir string
}

// HealthCheckResult represents the response from the health check endpoint.
type HealthCheckResult struct {
	Status       string                      `json:"status"`
	Service      string                      `json:"service"`
	Version      string                      `json:"version"`
	Readiness    bool                        `json:"readiness"`
	Timestamp    string                      `json:"timestamp"`
	Dependencies map[string]HealthDependency `json:"dependencies"`
	Errors       map[string]string           `json:"errors,omitempty"`
}

// HealthChecks performs dependency health validation.
type HealthChecks struct {
	deps HealthCheckDeps
}

// HealthDependency represents the status of a single dependency.
type HealthDependency struct {
	Connected bool   `json:"connected"`
	Status    string `json:"status,omitempty"`
}

// NewHealthChecks creates a new HealthChecks instance.
// If deps.Git is nil, a default ExecGitRunner is used.
func NewHealthChecks(deps HealthCheckDeps) *HealthChecks {
	if deps.Git == nil {
		deps.Git = &ExecGitRunner{GitPath: "git"}
	}
	deps.RepoDir = strings.TrimSpace(deps.RepoDir)
	return &HealthChecks{deps: deps}
}

// Run executes all health checks and returns the combined result.
func (h *HealthChecks) Run(ctx context.Context) HealthCheckResult {
	dependencies := make(map[string]HealthDependency, 3)
	errors := make(map[string]string, 3)

	dbOK := h.checkDatabase(ctx, dependencies, errors)
	gitOK := h.checkGitBinary(dependencies, errors)
	repoOK := h.checkRepository(ctx, dependencies, errors)

	readiness := dbOK && gitOK && repoOK
	status := "healthy"
	if !readiness {
		status = "unhealthy"
	}

	result := HealthCheckResult{
		Status:       status,
		Service:      "git-control-tower-api",
		Version:      "1.0.0",
		Readiness:    readiness,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Dependencies: dependencies,
	}
	if len(errors) > 0 {
		result.Errors = errors
	}
	return result
}

func (h *HealthChecks) checkDatabase(ctx context.Context, deps map[string]HealthDependency, errs map[string]string) bool {
	if h.deps.DB == nil || !h.deps.DB.IsConfigured() {
		deps["database"] = HealthDependency{Connected: false, Status: "unconfigured"}
		errs["database"] = "database handle not initialized"
		return false
	}
	if err := h.deps.DB.Ping(ctx); err != nil {
		deps["database"] = HealthDependency{Connected: false, Status: "disconnected"}
		errs["database"] = err.Error()
		return false
	}
	deps["database"] = HealthDependency{Connected: true, Status: "connected"}
	return true
}

func (h *HealthChecks) checkGitBinary(deps map[string]HealthDependency, errs map[string]string) bool {
	path, err := h.deps.Git.LookPath()
	if err != nil || strings.TrimSpace(path) == "" {
		deps["git"] = HealthDependency{Connected: false, Status: "missing"}
		if err != nil {
			errs["git"] = err.Error()
		} else {
			errs["git"] = "git binary not found in PATH"
		}
		return false
	}
	deps["git"] = HealthDependency{Connected: true, Status: "available"}
	return true
}

func (h *HealthChecks) checkRepository(ctx context.Context, deps map[string]HealthDependency, errs map[string]string) bool {
	if h.deps.RepoDir == "" {
		deps["repository"] = HealthDependency{Connected: false, Status: "unknown"}
		errs["repository"] = "could not resolve repository root"
		return false
	}

	out, err := h.deps.Git.RevParse(ctx, h.deps.RepoDir, "--is-inside-work-tree")
	if err != nil {
		deps["repository"] = HealthDependency{Connected: false, Status: "unavailable"}
		errs["repository"] = err.Error()
		return false
	}
	if strings.TrimSpace(string(out)) != "true" {
		deps["repository"] = HealthDependency{Connected: false, Status: "unavailable"}
		errs["repository"] = "not inside a git work tree"
		return false
	}
	deps["repository"] = HealthDependency{Connected: true, Status: "accessible"}
	return true
}
