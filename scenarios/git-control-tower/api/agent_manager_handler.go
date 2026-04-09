package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleAttachmentUpload proxies POST /api/v1/agent/attachments/upload to agent-manager.
func (s *Server) handleAttachmentUpload(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 60*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		return
	}

	wireResp, err := s.agentManagerClient.UploadAttachment(hctx.Ctx, r.Body, r.Header.Get("Content-Type"))
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(AttachmentUploadResponse{
		ID:          wireResp.ID,
		FileName:    wireResp.FileName,
		ContentType: wireResp.ContentType,
		FileSize:    wireResp.FileSize,
	})
}

// handleAgentProfiles proxies GET /api/v1/agent/profiles to agent-manager.
func (s *Server) handleAgentProfiles(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
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
	hctx := RepoRead(w, r, s.git, s.repos, 120*time.Second)
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

	taskResp, err := s.agentManagerClient.CreateTask(hctx.Ctx, agentTaskCreateRequest{
		Task: buildAgentTaskData(req),
	})
	if err != nil {
		hctx.Resp.InternalError(fmt.Sprintf("create task: %s", err.Error()))
		return
	}

	runResp, err := s.agentManagerClient.CreateRun(hctx.Ctx, buildAgentRunRequest(taskResp.Task.ID, req))
	if err != nil {
		hctx.Resp.InternalError(fmt.Sprintf("create run: %s", err.Error()))
		return
	}

	hctx.Resp.OK(AgentRunCreateResponse{
		RunID:  runResp.Run.ID,
		TaskID: taskResp.Task.ID,
	})
}

func buildAgentTaskData(req AgentRunRequest) agentTaskData {
	taskData := agentTaskData{
		Title:       fmt.Sprintf("GCT review: %s", req.ScenarioSlug),
		Description: req.Prompt,
		ScopePath:   fmt.Sprintf("scenarios/%s/", req.ScenarioSlug),
	}
	for _, aid := range req.AttachmentIDs {
		taskData.ContextAttachments = append(taskData.ContextAttachments, agentContextAttachment{
			Type:         "image",
			AttachmentID: aid,
			Label:        "Uploaded image",
		})
	}
	return taskData
}

func buildAgentRunRequest(taskID string, req AgentRunRequest) agentRunCreateInternalRequest {
	runReq := agentRunCreateInternalRequest{
		TaskID:  taskID,
		RunMode: 1, // RUN_MODE_SANDBOXED
		Tag:     fmt.Sprintf("gct-%s", req.ScenarioSlug),
	}
	if req.ProfileID != "" {
		runReq.AgentProfileID = req.ProfileID
	} else if req.ProfileKey != "" {
		runReq.ProfileRef = &agentProfileRef{ProfileKey: req.ProfileKey}
	}
	return runReq
}

// handleAgentRunList proxies GET /api/v1/agent/runs to agent-manager.
func (s *Server) handleAgentRunList(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
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
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
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
