package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleAuditorRunCheck handles POST /api/v1/repo/rules-run
func (s *Server) handleAuditorRunCheck(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "scenario-auditor") {
		hctx.Resp.ServiceUnavailable("scenario-auditor is not available")
		return
	}

	var proxyReq AuditorRunCheckProxyRequest
	if !ParseJSONBody(w, r, &proxyReq) {
		return
	}

	scenarioName := strings.TrimSpace(proxyReq.ScenarioName)
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenario_name is required")
		return
	}

	checkType := strings.TrimSpace(proxyReq.CheckType)
	if checkType == "" {
		checkType = "full"
	}

	result, err := s.auditorClient.StartCheck(hctx.Ctx, scenarioName, checkType)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAuditorJobStatus handles GET /api/v1/repo/rules-job/{jobId}
func (s *Server) handleAuditorJobStatus(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "scenario-auditor") {
		hctx.Resp.ServiceUnavailable("scenario-auditor is not available")
		return
	}

	jobID := strings.TrimSpace(mux.Vars(r)["jobId"])
	if jobID == "" {
		hctx.Resp.BadRequest("jobId is required")
		return
	}

	result, err := s.auditorClient.GetJobStatus(hctx.Ctx, jobID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAuditorRules handles GET /api/v1/repo/rules
func (s *Server) handleAuditorRules(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "scenario-auditor") {
		hctx.Resp.ServiceUnavailable("scenario-auditor is not available")
		return
	}

	result, err := s.auditorClient.ListRules(hctx.Ctx)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAuditorFix handles POST /api/v1/repo/rules-fix
func (s *Server) handleAuditorFix(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 120*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "scenario-auditor") {
		hctx.Resp.ServiceUnavailable("scenario-auditor is not available")
		return
	}

	var req AuditorFixRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if len(req.ScenarioNames) == 0 {
		hctx.Resp.BadRequest("scenario_names is required")
		return
	}
	if len(req.RuleIDs) == 0 {
		hctx.Resp.BadRequest("rule_ids is required")
		return
	}

	result, err := s.auditorClient.ApplyFix(hctx.Ctx, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAuditorViolations handles GET /api/v1/repo/rules-violations?scenarioName=X
func (s *Server) handleAuditorViolations(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "scenario-auditor") {
		hctx.Resp.ServiceUnavailable("scenario-auditor is not available")
		return
	}

	scenarioName := strings.TrimSpace(r.URL.Query().Get("scenarioName"))
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenarioName query parameter is required")
		return
	}

	result, err := s.auditorClient.GetViolations(hctx.Ctx, scenarioName)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}
