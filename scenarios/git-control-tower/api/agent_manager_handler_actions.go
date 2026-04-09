package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func (s *Server) handleAgentRunEvents(w http.ResponseWriter, r *http.Request) {
	hctx := agentRunOperation(w, r, s)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	id := requireRunID(hctx, r)
	if id == "" {
		return
	}

	afterSequence, ok := parseOptionalIntParam(hctx, r, "afterSequence", -1, -1)
	if !ok {
		return
	}
	limit, ok := parseOptionalIntParam(hctx, r, "limit", 100, 1)
	if !ok {
		return
	}
	if limit > 500 {
		limit = 500
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

	hctx.Resp.OK(AgentRunEventsResponse{Events: events})
}

// agentRunOperation initializes a handler context with agent-manager availability check.
func agentRunOperation(w http.ResponseWriter, r *http.Request, s *Server) *HandlerContext {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return nil
	}
	if !s.capabilities.IsAvailable(hctx.Ctx, "agent-manager") {
		hctx.Resp.ServiceUnavailable("agent-manager is not available")
		hctx.Cancel()
		return nil
	}
	return hctx
}

// requireRunID extracts and validates the run ID path parameter.
func requireRunID(hctx *HandlerContext, r *http.Request) string {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		hctx.Resp.BadRequest("run ID is required")
	}
	return id
}

// parseOptionalIntParam parses an optional integer query parameter with validation.
func parseOptionalIntParam(hctx *HandlerContext, r *http.Request, name string, defaultVal, minVal int) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return defaultVal, true
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < minVal {
		hctx.Resp.BadRequest(name + " must be a valid integer >= " + strconv.Itoa(minVal))
		return 0, false
	}
	return parsed, true
}

// handleAgentRunDiff proxies GET /api/v1/agent/runs/{id}/diff to agent-manager.
func (s *Server) handleAgentRunDiff(w http.ResponseWriter, r *http.Request) {
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
		apiRun := wireRunToAPI(wireResp.Run)
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
		resp.FilesApplied = wireResp.Result.FilesApplied
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
