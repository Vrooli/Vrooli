package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleTestExecution handles POST /api/v1/repo/test-execution
func (s *Server) handleTestExecution(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 600*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "test-genie") {
		hctx.Resp.ServiceUnavailable("test-genie is not available")
		return
	}

	var req TestExecutionRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.ScenarioName) == "" {
		hctx.Resp.BadRequest("scenarioName is required")
		return
	}

	result, err := s.testGenieClient.ExecuteSuite(hctx.Ctx, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleTestExecutionList handles GET /api/v1/repo/test-executions?scenarioName=...&limit=...
func (s *Server) handleTestExecutionList(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "test-genie") {
		hctx.Resp.ServiceUnavailable("test-genie is not available")
		return
	}

	scenarioName := strings.TrimSpace(r.URL.Query().Get("scenarioName"))
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenarioName query parameter is required")
		return
	}

	limit := 10
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			hctx.Resp.BadRequest("limit must be a positive integer")
			return
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}

	result, err := s.testGenieClient.ListExecutions(hctx.Ctx, scenarioName, limit)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleTestExecutionDetail handles GET /api/v1/repo/test-executions/{id}
func (s *Server) handleTestExecutionDetail(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "test-genie") {
		hctx.Resp.ServiceUnavailable("test-genie is not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		hctx.Resp.BadRequest("execution ID is required")
		return
	}

	result, err := s.testGenieClient.GetExecution(hctx.Ctx, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}
