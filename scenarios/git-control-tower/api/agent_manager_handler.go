package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleAgentProfiles proxies GET /api/v1/agent/profiles to agent-manager.
func (s *Server) handleAgentProfiles(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	result, err := s.agentManagerClient.ListProfiles(hctx.Ctx)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAgentRunCreate handles POST /api/v1/agent/run.
// Composite endpoint: creates a Task then a Run in agent-manager.
func (s *Server) handleAgentRunCreate(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 120*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	var req AgentRunRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.ScenarioSlug) == "" {
		hctx.Resp.BadRequest("scenarioSlug is required")
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		hctx.Resp.BadRequest("prompt is required")
		return
	}

	// Step 1: Create a Task
	taskResp, err := s.agentManagerClient.CreateTask(hctx.Ctx, agentTaskCreateRequest{
		Title:       fmt.Sprintf("GCT review: %s", req.ScenarioSlug),
		Description: req.Prompt,
		ScopePath:   fmt.Sprintf("scenarios/%s/", req.ScenarioSlug),
	})
	if err != nil {
		hctx.Resp.InternalError(fmt.Sprintf("create task: %s", err.Error()))
		return
	}

	// Step 2: Create a Run
	runReq := agentRunCreateInternalRequest{
		TaskID:  taskResp.ID,
		RunMode: "sandboxed",
	}
	if req.ProfileID != "" {
		runReq.AgentProfileID = req.ProfileID
	} else if req.ProfileKey != "" {
		runReq.ProfileRef = req.ProfileKey
	}

	runResp, err := s.agentManagerClient.CreateRun(hctx.Ctx, runReq)
	if err != nil {
		hctx.Resp.InternalError(fmt.Sprintf("create run: %s", err.Error()))
		return
	}

	hctx.Resp.OK(AgentRunCreateResponse{
		RunID:  runResp.ID,
		TaskID: taskResp.ID,
	})
}

// handleAgentRunList proxies GET /api/v1/agent/runs to agent-manager.
func (s *Server) handleAgentRunList(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	scenarioSlug := strings.TrimSpace(r.URL.Query().Get("scenarioSlug"))
	if scenarioSlug == "" {
		hctx.Resp.BadRequest("scenarioSlug query parameter is required")
		return
	}

	limit := 5
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			hctx.Resp.BadRequest("limit must be a positive integer")
			return
		}
		if parsed > 50 {
			parsed = 50
		}
		limit = parsed
	}

	scopePrefix := fmt.Sprintf("scenarios/%s/", scenarioSlug)
	result, err := s.agentManagerClient.ListRuns(hctx.Ctx, scopePrefix, limit)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAgentRunDetail proxies GET /api/v1/agent/runs/{id} to agent-manager.
func (s *Server) handleAgentRunDetail(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		hctx.Resp.BadRequest("run ID is required")
		return
	}

	result, err := s.agentManagerClient.GetRun(hctx.Ctx, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAgentRunEvents proxies GET /api/v1/agent/runs/{id}/events to agent-manager.
func (s *Server) handleAgentRunEvents(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		hctx.Resp.BadRequest("run ID is required")
		return
	}

	afterSequence := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("afterSequence")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			hctx.Resp.BadRequest("afterSequence must be a non-negative integer")
			return
		}
		afterSequence = parsed
	}

	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			hctx.Resp.BadRequest("limit must be a positive integer")
			return
		}
		if parsed > 500 {
			parsed = 500
		}
		limit = parsed
	}

	result, err := s.agentManagerClient.GetRunEvents(hctx.Ctx, id, afterSequence, limit)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAgentRunDiff proxies GET /api/v1/agent/runs/{id}/diff to agent-manager.
func (s *Server) handleAgentRunDiff(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		hctx.Resp.BadRequest("run ID is required")
		return
	}

	result, err := s.agentManagerClient.GetRunDiff(hctx.Ctx, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAgentRunContinue proxies POST /api/v1/agent/runs/{id}/continue to agent-manager.
func (s *Server) handleAgentRunContinue(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 120*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		hctx.Resp.BadRequest("run ID is required")
		return
	}

	var req AgentContinueRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := s.agentManagerClient.ContinueRun(hctx.Ctx, id, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAgentRunApprove proxies POST /api/v1/agent/runs/{id}/approve to agent-manager.
func (s *Server) handleAgentRunApprove(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		hctx.Resp.BadRequest("run ID is required")
		return
	}

	var req AgentApproveRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := s.agentManagerClient.ApproveRun(hctx.Ctx, id, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAgentRunReject proxies POST /api/v1/agent/runs/{id}/reject to agent-manager.
func (s *Server) handleAgentRunReject(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		hctx.Resp.BadRequest("run ID is required")
		return
	}

	var req AgentRejectRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := s.agentManagerClient.RejectRun(hctx.Ctx, id, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleAgentRunStop proxies POST /api/v1/agent/runs/{id}/stop to agent-manager.
func (s *Server) handleAgentRunStop(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		hctx.Resp.BadRequest("run ID is required")
		return
	}

	result, err := s.agentManagerClient.StopRun(hctx.Ctx, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}
