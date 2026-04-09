package main

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// handleTidinessScore handles GET /api/v1/repo/tidiness-score?scenarioName=X
func (s *Server) handleTidinessScore(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "tidiness-manager") {
		hctx.Resp.ServiceUnavailable("tidiness-manager is not available")
		return
	}

	scenarioName := strings.TrimSpace(r.URL.Query().Get("scenarioName"))
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenarioName query parameter is required")
		return
	}

	result, err := s.tidinessClient.GetTidinessScore(hctx.Ctx, scenarioName)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleTidinessIssues handles GET /api/v1/repo/tidiness-issues?scenarioName=X&file=Y&category=Z&limit=N
func (s *Server) handleTidinessIssues(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "tidiness-manager") {
		hctx.Resp.ServiceUnavailable("tidiness-manager is not available")
		return
	}

	scenarioName := strings.TrimSpace(r.URL.Query().Get("scenarioName"))
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenarioName query parameter is required")
		return
	}

	file := strings.TrimSpace(r.URL.Query().Get("file"))
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	severity := strings.TrimSpace(r.URL.Query().Get("severity"))

	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			hctx.Resp.BadRequest("limit must be a positive integer")
			return
		}
		if parsed > 1000 {
			parsed = 1000
		}
		limit = parsed
	}

	result, err := s.tidinessClient.GetIssues(hctx.Ctx, scenarioName, file, category, severity, limit)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleTidinessStaleness handles GET /api/v1/repo/tidiness-staleness?scenarioName=X
func (s *Server) handleTidinessStaleness(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "tidiness-manager") {
		hctx.Resp.ServiceUnavailable("tidiness-manager is not available")
		return
	}

	scenarioName := strings.TrimSpace(r.URL.Query().Get("scenarioName"))
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenarioName query parameter is required")
		return
	}

	result, err := s.tidinessClient.GetStaleness(hctx.Ctx, scenarioName)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleTidinessLightScan handles POST /api/v1/repo/tidiness-scan
func (s *Server) handleTidinessLightScan(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 120*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "tidiness-manager") {
		hctx.Resp.ServiceUnavailable("tidiness-manager is not available")
		return
	}

	var proxyReq TidinessLightScanProxyRequest
	if !ParseJSONBody(w, r, &proxyReq) {
		return
	}

	scenarioName := strings.TrimSpace(proxyReq.ScenarioName)
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenario_name is required")
		return
	}

	repoRoot := strings.TrimSpace(s.git.ResolveRepoRoot(hctx.Ctx))
	if repoRoot == "" {
		hctx.Resp.InternalError("could not resolve repository root")
		return
	}
	absPath := filepath.Join(repoRoot, "scenarios", scenarioName)

	req := TidinessLightScanRequest{
		ScenarioPath: absPath,
		TimeoutSec:   proxyReq.TimeoutSec,
		Incremental:  proxyReq.Incremental,
	}

	result, err := s.tidinessClient.TriggerLightScan(hctx.Ctx, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleTidinessScenarioDetail handles GET /api/v1/repo/tidiness-scenario?scenarioName=X
func (s *Server) handleTidinessScenarioDetail(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "tidiness-manager") {
		hctx.Resp.ServiceUnavailable("tidiness-manager is not available")
		return
	}

	scenarioName := strings.TrimSpace(r.URL.Query().Get("scenarioName"))
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenarioName query parameter is required")
		return
	}

	result, err := s.tidinessClient.GetScenarioDetail(hctx.Ctx, scenarioName)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}
