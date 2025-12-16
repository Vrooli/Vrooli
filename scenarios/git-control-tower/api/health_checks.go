package main

import (
	"context"
	"database/sql"
	"os/exec"
	"strings"
	"time"
)

type HealthCheckDeps struct {
	DB      *sql.DB
	GitPath string
	RepoDir string
}

type HealthCheckResult struct {
	Status       string                     `json:"status"`
	Service      string                     `json:"service"`
	Version      string                     `json:"version"`
	Readiness    bool                       `json:"readiness"`
	Timestamp    string                     `json:"timestamp"`
	Dependencies map[string]HealthDependency `json:"dependencies"`
	Errors       map[string]string          `json:"errors,omitempty"`
}

type HealthChecks struct {
	deps HealthCheckDeps
}

type HealthDependency struct {
	Connected bool   `json:"connected"`
	Status    string `json:"status,omitempty"`
}

func NewHealthChecks(deps HealthCheckDeps) *HealthChecks {
	deps.GitPath = strings.TrimSpace(deps.GitPath)
	if deps.GitPath == "" {
		deps.GitPath = "git"
	}
	deps.RepoDir = strings.TrimSpace(deps.RepoDir)
	return &HealthChecks{deps: deps}
}

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
	if h.deps.DB == nil {
		deps["database"] = HealthDependency{Connected: false, Status: "unconfigured"}
		errs["database"] = "database handle not initialized"
		return false
	}
	if err := h.deps.DB.PingContext(ctx); err != nil {
		deps["database"] = HealthDependency{Connected: false, Status: "disconnected"}
		errs["database"] = err.Error()
		return false
	}
	deps["database"] = HealthDependency{Connected: true, Status: "connected"}
	return true
}

func (h *HealthChecks) checkGitBinary(deps map[string]HealthDependency, errs map[string]string) bool {
	path, err := exec.LookPath(h.deps.GitPath)
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
	cmd := exec.CommandContext(ctx, h.deps.GitPath, "-C", h.deps.RepoDir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.CombinedOutput()
	if err != nil {
		deps["repository"] = HealthDependency{Connected: false, Status: "unavailable"}
		errs["repository"] = strings.TrimSpace(string(out))
		if errs["repository"] == "" {
			errs["repository"] = err.Error()
		}
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
