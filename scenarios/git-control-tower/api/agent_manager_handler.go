package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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

	taskResp, err := s.agentManagerClient.CreateTask(hctx.Ctx, buildAgentTaskData(hctx.RepoDir, req))
	if err != nil {
		hctx.Resp.InternalError(fmt.Sprintf("create task: %s", err.Error()))
		return
	}

	runResp, err := s.agentManagerClient.CreateRun(hctx.Ctx, buildAgentRunRequest(taskResp.Task.GetId(), req))
	if err != nil {
		hctx.Resp.InternalError(fmt.Sprintf("create run: %s", err.Error()))
		return
	}

	hctx.Resp.OK(AgentRunCreateResponse{
		RunID:  runResp.Run.GetId(),
		TaskID: taskResp.Task.GetId(),
	})
}

func buildAgentTaskData(repoDir string, req AgentRunRequest) *ampb.Task {
	scopePath := ""
	if resolved, err := resolveScopePath(repoDir, "scenario", req.ScenarioSlug); err == nil {
		scopePath = resolved
	}
	taskData := &ampb.Task{
		Title:       fmt.Sprintf("GCT review: %s", req.ScenarioSlug),
		Description: req.Prompt,
		ScopePath:   scopePath,
	}
	for _, aid := range req.AttachmentIDs {
		taskData.ContextAttachments = append(taskData.ContextAttachments, &ampb.ContextAttachment{
			Type:         "image",
			AttachmentId: aid,
			Label:        "Uploaded image",
		})
	}
	return taskData
}

func buildAgentRunRequest(taskID string, req AgentRunRequest) *apipb.CreateRunRequest {
	mode := ampb.RunMode_RUN_MODE_SANDBOXED
	tag := fmt.Sprintf("gct-%s", req.ScenarioSlug)
	runReq := &apipb.CreateRunRequest{TaskId: taskID, RunMode: &mode, Tag: &tag}
	if req.ProfileID != "" {
		runReq.AgentProfileId = &req.ProfileID
	} else if req.ProfileKey != "" {
		runReq.ProfileRef = &apipb.ProfileRef{ProfileKey: req.ProfileKey}
	} else {
		runReq.ProfileRef = &apipb.ProfileRef{ProfileKey: "git-control-tower/reviewer"}
	}
	return runReq
}
