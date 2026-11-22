package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleAgentRunContinue proxies POST /api/v1/agent/runs/{id}/continue to agent-manager.
func (s *Server) handleAgentRunContinue(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 120*time.Second)
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
		apiRun := agentRunToAPI(wireResp.Run)
		resp.Run = &apiRun
	}

	hctx.Resp.OK(resp)
}

// handleAgentRunApprove proxies POST /api/v1/agent/runs/{id}/approve to agent-manager.
func (s *Server) handleAgentRunApprove(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 30*time.Second)
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
		resp.FilesApplied = int(wireResp.Result.FilesApplied)
		resp.CommitHash = wireResp.Result.CommitHash
		resp.Message = wireResp.Result.Message
	}

	hctx.Resp.OK(resp)
}

// handleAgentRunReject proxies POST /api/v1/agent/runs/{id}/reject to agent-manager.
func (s *Server) handleAgentRunReject(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 30*time.Second)
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

	wireResp, err := s.agentManagerClient.StopRun(hctx.Ctx, id)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(AgentStopResponse{
		Status: wireResp.Status,
	})
}
