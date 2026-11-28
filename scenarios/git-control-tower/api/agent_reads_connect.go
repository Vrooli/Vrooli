package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	agentreadsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/agent_reads"
	agentreadsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/agent_reads/agent_reads_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ agentreadsconnect.AgentReadServiceHandler = (*agentReadConnectServer)(nil)

type agentReadConnectServer struct{ server *Server }

func (s agentReadConnectServer) ListProfiles(ctx context.Context, req *connect.Request[agentreadsv1.ListProfilesRequest]) (*connect.Response[agentreadsv1.ListProfilesResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("profile list request is required")
	}
	ctx, cancel, _, err := s.server.agentReadContext(ctx, req.Msg.GetRepositoryId(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	result, err := s.server.agentManagerClient.ListProfiles(ctx)
	if err != nil {
		return nil, agentManagerReadError(ctx, err)
	}
	response := &agentreadsv1.ListProfilesResponse{Profiles: make([]*agentreadsv1.AgentProfile, 0, len(result.GetProfiles())), Total: result.GetTotal()}
	for _, profile := range result.GetProfiles() {
		response.Profiles = append(response.Profiles, profileToProto(agentProfileToAPI(profile)))
	}
	return connect.NewResponse(response), nil
}

func (s agentReadConnectServer) ListRuns(ctx context.Context, req *connect.Request[agentreadsv1.ListRunsRequest]) (*connect.Response[agentreadsv1.ListRunsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("run list request is required")
	}
	slug, err := requiredScenarioSlug(req.Msg.GetScenarioSlug())
	if err != nil {
		return nil, err
	}
	limit := int(req.Msg.GetLimit())
	if limit < 0 {
		return nil, invalidConnectRequest("limit must not be negative")
	}
	if limit == 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}
	ctx, cancel, repo, err := s.server.agentReadContext(ctx, req.Msg.GetRepositoryId(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	result, err := s.server.agentManagerClient.ListRuns(ctx, "gct-"+slug, 50)
	if err != nil {
		return nil, agentManagerReadError(ctx, err)
	}
	expectedScope, scopeErr := resolveScopePath(repo.Path, "scenario", slug)
	response := &agentreadsv1.ListRunsResponse{Runs: make([]*agentreadsv1.AgentRun, 0, min(limit, len(result.GetRuns())))}
	for _, run := range result.GetRuns() {
		matches, filterErr := s.server.agentRunMatchesListFilter(ctx, run, "gct-"+slug, expectedScope, scopeErr)
		if filterErr != nil {
			return nil, filterErr
		}
		if !matches {
			continue
		}
		response.Runs = append(response.Runs, runToProto(agentRunToAPI(run)))
		if len(response.Runs) == limit {
			break
		}
	}
	response.Total = int32(len(response.Runs))
	return connect.NewResponse(response), nil
}

func (s *Server) agentRunMatchesListFilter(ctx context.Context, run *ampb.Run, tag, expectedScope string, scopeErr error) (bool, error) {
	if run == nil || run.GetTag() != tag || run.GetTaskId() == "" {
		return false, nil
	}
	return s.agentRunTaskMatchesScope(ctx, run.GetTaskId(), expectedScope, scopeErr)
}

func (s *Server) agentRunTaskMatchesScope(ctx context.Context, taskID, expectedScope string, scopeErr error) (bool, error) {
	task, err := s.agentManagerClient.GetTask(ctx, taskID)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return false, nil
		}
		return false, agentManagerReadError(ctx, err)
	}
	return scopeErr == nil && task.GetTask().GetScopePath() == expectedScope, nil
}

func (s agentReadConnectServer) GetRun(ctx context.Context, req *connect.Request[agentreadsv1.GetRunRequest]) (*connect.Response[agentreadsv1.AgentRun], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("run request is required")
	}
	runID, err := requiredRunID(req.Msg.GetRunId())
	if err != nil {
		return nil, err
	}
	slug, err := requiredScenarioSlug(req.Msg.GetScenarioSlug())
	if err != nil {
		return nil, err
	}
	ctx, cancel, repo, err := s.server.agentReadContext(ctx, req.Msg.GetRepositoryId(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	result, err := s.server.getGCTRun(ctx, runID, slug, repo.Path)
	if err != nil {
		return nil, agentManagerReadError(ctx, err)
	}
	return connect.NewResponse(runToProto(agentRunToAPI(result))), nil
}

func (s agentReadConnectServer) GetRunEvents(ctx context.Context, req *connect.Request[agentreadsv1.GetRunEventsRequest]) (*connect.Response[agentreadsv1.GetRunEventsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("run events request is required")
	}
	runID, err := requiredRunID(req.Msg.GetRunId())
	if err != nil {
		return nil, err
	}
	slug, err := requiredScenarioSlug(req.Msg.GetScenarioSlug())
	if err != nil {
		return nil, err
	}
	after := int64(-1)
	if req.Msg.AfterSequence != nil {
		after = req.Msg.GetAfterSequence()
	}
	if after < -1 {
		return nil, invalidConnectRequest("after_sequence must be at least -1")
	}
	limit := int(req.Msg.GetLimit())
	if limit < 0 {
		return nil, invalidConnectRequest("limit must not be negative")
	}
	if limit == 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	ctx, cancel, repo, err := s.server.agentReadContext(ctx, req.Msg.GetRepositoryId(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if _, err := s.server.getGCTRun(ctx, runID, slug, repo.Path); err != nil {
		return nil, agentManagerReadError(ctx, err)
	}
	result, err := s.server.agentManagerClient.GetRunEvents(ctx, runID, int(after), limit)
	if err != nil {
		return nil, agentManagerReadError(ctx, err)
	}
	response := &agentreadsv1.GetRunEventsResponse{Events: make([]*agentreadsv1.AgentRunEvent, 0, len(result.GetEvents()))}
	for _, event := range result.GetEvents() {
		response.Events = append(response.Events, runEventToProto(agentRunEventToAPI(event)))
	}
	return connect.NewResponse(response), nil
}

func (s agentReadConnectServer) GetRunDiff(ctx context.Context, req *connect.Request[agentreadsv1.GetRunDiffRequest]) (*connect.Response[agentreadsv1.AgentRunDiff], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("run diff request is required")
	}
	runID, err := requiredRunID(req.Msg.GetRunId())
	if err != nil {
		return nil, err
	}
	slug, err := requiredScenarioSlug(req.Msg.GetScenarioSlug())
	if err != nil {
		return nil, err
	}
	ctx, cancel, repo, err := s.server.agentReadContext(ctx, req.Msg.GetRepositoryId(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if _, err := s.server.getGCTRun(ctx, runID, slug, repo.Path); err != nil {
		return nil, agentManagerReadError(ctx, err)
	}
	result, err := s.server.agentManagerClient.GetRunDiff(ctx, runID)
	if err != nil {
		return nil, agentManagerReadError(ctx, err)
	}
	response := &agentreadsv1.AgentRunDiff{RunId: runID, Files: []*agentreadsv1.AgentRunDiffFile{}}
	if result.GetDiff() != nil {
		response.Content = result.GetDiff().GetContent()
		for _, file := range result.GetDiff().GetFiles() {
			mapped := agentFileDiffToAPI(file)
			response.Files = append(response.Files, &agentreadsv1.AgentRunDiffFile{Path: mapped.Path, ChangeType: mapped.ChangeType, Additions: int32(mapped.Additions), Deletions: int32(mapped.Deletions), IsBinary: mapped.IsBinary, Patch: mapped.Patch})
		}
	}
	return connect.NewResponse(response), nil
}

func (s *Server) getGCTRun(ctx context.Context, runID, scenarioSlug, repoPath string) (*ampb.Run, error) {
	result, err := s.agentManagerClient.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	run := result.GetRun()
	if run == nil || run.GetTag() != "gct-"+scenarioSlug || run.GetTaskId() == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("run is not bound to the requested GCT scenario"))
	}
	ownerTask, err := s.agentManagerClient.GetTask(ctx, run.GetTaskId())
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("run task is no longer available"))
		}
		return nil, err
	}
	expectedScope, err := resolveScopePath(repoPath, "scenario", scenarioSlug)
	if err != nil || ownerTask.GetTask().GetScopePath() != expectedScope {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("run task is not bound to the requested repository and scenario"))
	}
	return run, nil
}

func (s *Server) agentReadContext(parent context.Context, repoID string, timeout time.Duration) (context.Context, context.CancelFunc, ResolvedRepo, error) {
	return s.ownerReadContext(parent, repoID, timeout, "agent-manager", s.agentManagerClient != nil)
}

func requiredRunID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" || len(id) > 255 {
		return "", invalidConnectRequest("run_id is required and must be at most 255 characters")
	}
	return id, nil
}

func requiredScenarioSlug(raw string) (string, error) {
	slug := strings.TrimSpace(raw)
	if slug == "" || len(slug) > 255 {
		return "", invalidConnectRequest("scenario_slug is required and must be at most 255 characters")
	}
	return slug, nil
}

func agentManagerReadError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
		}
		return connect.NewError(connect.CodeCanceled, ctx.Err())
	}
	code := connect.CodeOf(err)
	if code == connect.CodeUnknown {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("agent-manager read failed: %w", err))
	}
	return connect.NewError(code, err)
}

func profileToProto(profile AgentProfile) *agentreadsv1.AgentProfile {
	return &agentreadsv1.AgentProfile{Id: profile.ID, Key: profile.Key, Name: profile.Name, Description: profile.Description, RoleRef: profile.RoleRef}
}

func runToProto(run AgentRun) *agentreadsv1.AgentRun {
	result := &agentreadsv1.AgentRun{
		Id: run.ID, TaskId: run.TaskID, SessionId: run.SessionID, Status: run.Status, Phase: run.Phase,
		ProgressPercent: int32(run.ProgressPercent), ErrorMsg: run.ErrorMsg, ApprovalState: run.ApprovalState,
		PromptPreview: run.PromptPreview, SandboxId: run.SandboxID, CommitHash: run.CommitHash,
		CreatedAt: run.CreatedAt, StartedAt: run.StartedAt, EndedAt: run.EndedAt,
	}
	if summary := run.Summary; summary != nil {
		result.Summary = &agentreadsv1.AgentRunSummary{FilesModified: summary.FilesModified, FilesCreated: summary.FilesCreated, FilesDeleted: summary.FilesDeleted, TokensUsed: summary.TokensUsed, TurnsUsed: int32(summary.TurnsUsed), CostEstimate: summary.CostEstimate}
	}
	if actions := run.Actions; actions != nil {
		result.Actions = &agentreadsv1.AgentRunActions{CanInvestigate: actions.CanInvestigate, CanApplyInvestigation: actions.CanApplyInvestigation, CanDelete: actions.CanDelete, CanStop: actions.CanStop, CanRetry: actions.CanRetry, CanContinue: actions.CanContinue, CanApprove: actions.CanApprove, CanReject: actions.CanReject, CanReview: actions.CanReview, CanExtractRecommendations: actions.CanExtractRecommendations, CanRegenerateRecommendations: actions.CanRegenerateRecommendations}
	}
	return result
}

func runEventToProto(event AgentRunEvent) *agentreadsv1.AgentRunEvent {
	result := &agentreadsv1.AgentRunEvent{Id: event.ID, RunId: event.RunID, Sequence: event.Sequence, EventType: event.EventType, Timestamp: event.Timestamp}
	if data, ok := event.Data.(map[string]interface{}); ok && len(data) > 0 {
		if structured, err := structpb.NewStruct(data); err == nil {
			result.Data = structured
		}
	}
	return result
}
