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
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	wireResp, err := s.agentManagerClient.ListProfiles(hctx.Ctx)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	profiles := make([]AgentProfile, 0, len(wireResp.Profiles))
	for i := range wireResp.Profiles {
		profiles = append(profiles, wireProfileToAPI(&wireResp.Profiles[i]))
	}

	hctx.Resp.OK(AgentProfileListResponse{
		Profiles: profiles,
		Total:    wireResp.Total,
	})
}

// handleAgentRunCreate handles POST /api/v1/agent/run.
// Composite endpoint: creates a Task then a Run in agent-manager.
func (s *Server) handleAgentRunCreate(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 120*time.Second)
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
		Task: agentTaskData{
			Title:       fmt.Sprintf("GCT review: %s", req.ScenarioSlug),
			Description: req.Prompt,
			ScopePath:   fmt.Sprintf("scenarios/%s/", req.ScenarioSlug),
		},
	})
	if err != nil {
		hctx.Resp.InternalError(fmt.Sprintf("create task: %s", err.Error()))
		return
	}

	// Step 2: Create a Run (with tag for filtering)
	runReq := agentRunCreateInternalRequest{
		TaskID:  taskResp.Task.ID,
		RunMode: 1, // RUN_MODE_SANDBOXED
		Tag:     fmt.Sprintf("gct-%s", req.ScenarioSlug),
	}
	if req.ProfileID != "" {
		runReq.AgentProfileID = req.ProfileID
	} else if req.ProfileKey != "" {
		runReq.ProfileRef = &agentProfileRef{ProfileKey: req.ProfileKey}
	}

	runResp, err := s.agentManagerClient.CreateRun(hctx.Ctx, runReq)
	if err != nil {
		hctx.Resp.InternalError(fmt.Sprintf("create run: %s", err.Error()))
		return
	}

	hctx.Resp.OK(AgentRunCreateResponse{
		RunID:  runResp.Run.ID,
		TaskID: taskResp.Task.ID,
	})
}

// handleAgentRunList proxies GET /api/v1/agent/runs to agent-manager.
func (s *Server) handleAgentRunList(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
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

	tagPrefix := fmt.Sprintf("gct-%s", scenarioSlug)
	wireResp, err := s.agentManagerClient.ListRuns(hctx.Ctx, tagPrefix, limit)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	runs := make([]AgentRun, 0, len(wireResp.Runs))
	for i := range wireResp.Runs {
		runs = append(runs, wireRunToAPI(&wireResp.Runs[i]))
	}

	hctx.Resp.OK(AgentRunListResponse{
		Runs:  runs,
		Total: wireResp.Total,
	})
}

// handleAgentRunDetail proxies GET /api/v1/agent/runs/{id} to agent-manager.
func (s *Server) handleAgentRunDetail(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
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

	wireResp, err := s.agentManagerClient.GetRun(hctx.Ctx, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	apiRun := wireRunToAPI(&wireResp.Run)
	hctx.Resp.OK(apiRun)
}

// handleAgentRunEvents proxies GET /api/v1/agent/runs/{id}/events to agent-manager.
func (s *Server) handleAgentRunEvents(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
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

	wireResp, err := s.agentManagerClient.GetRunEvents(hctx.Ctx, id, afterSequence, limit)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	events := make([]AgentRunEvent, 0, len(wireResp.Events))
	for i := range wireResp.Events {
		events = append(events, wireRunEventToAPI(&wireResp.Events[i]))
	}

	hctx.Resp.OK(AgentRunEventsResponse{
		Events: events,
	})
}

// handleAgentRunDiff proxies GET /api/v1/agent/runs/{id}/diff to agent-manager.
func (s *Server) handleAgentRunDiff(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
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

	wireResp, err := s.agentManagerClient.GetRunDiff(hctx.Ctx, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	resp := AgentRunDiffResponse{RunID: id}
	if wireResp.Diff != nil {
		resp.Content = wireResp.Diff.Content
		files := make([]AgentRunDiffFile, 0, len(wireResp.Diff.Files))
		for i := range wireResp.Diff.Files {
			files = append(files, wireFileDiffToAPI(&wireResp.Diff.Files[i]))
		}
		resp.Files = files
	}

	hctx.Resp.OK(resp)
}

// handleAgentRunContinue proxies POST /api/v1/agent/runs/{id}/continue to agent-manager.
func (s *Server) handleAgentRunContinue(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 120*time.Second)
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

	wireResp, err := s.agentManagerClient.ContinueRun(hctx.Ctx, id, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	resp := AgentContinueResponse{
		Success: wireResp.Success,
	}
	if wireResp.Run != nil {
		apiRun := wireRunToAPI(wireResp.Run)
		resp.Run = &apiRun
	}

	hctx.Resp.OK(resp)
}

// handleAgentRunApprove proxies POST /api/v1/agent/runs/{id}/approve to agent-manager.
func (s *Server) handleAgentRunApprove(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 30*time.Second)
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

	wireResp, err := s.agentManagerClient.ApproveRun(hctx.Ctx, id, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	resp := AgentApproveResponse{}
	if wireResp.Result != nil {
		resp.Success = wireResp.Result.Success
		resp.FilesApplied = wireResp.Result.FilesApplied
		resp.CommitHash = wireResp.Result.CommitHash
		resp.Message = wireResp.Result.Message
	}

	hctx.Resp.OK(resp)
}

// handleAgentRunReject proxies POST /api/v1/agent/runs/{id}/reject to agent-manager.
func (s *Server) handleAgentRunReject(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 30*time.Second)
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

	wireResp, err := s.agentManagerClient.RejectRun(hctx.Ctx, id, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(AgentRejectResponse{
		Status: wireResp.Status,
	})
}

// handleAgentRunStop proxies POST /api/v1/agent/runs/{id}/stop to agent-manager.
func (s *Server) handleAgentRunStop(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
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

	wireResp, err := s.agentManagerClient.StopRun(hctx.Ctx, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(AgentStopResponse{
		Status: wireResp.Status,
	})
}
