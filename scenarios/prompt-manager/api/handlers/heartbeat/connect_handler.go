package heartbeat

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	"google.golang.org/protobuf/types/known/structpb"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/heartbeat"
)

type connectHandler struct {
	heartbeatconnect.UnimplementedHeartbeatServiceHandler
	legacy *domain.Handlers
}

func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return heartbeatconnect.NewHeartbeatServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) ListHeartbeats(c context.Context, r *connect.Request[heartbeatv1.TeamRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.ListHeartbeats, http.MethodGet, r.Msg.GetTeamId(), "/heartbeats", nil, nil)
}

func (h *connectHandler) GetHeartbeat(c context.Context, r *connect.Request[heartbeatv1.MemberRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.member(c, r.Header(), h.legacy.GetHeartbeat, http.MethodGet, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "", nil, nil)
}

func (h *connectHandler) CreateHeartbeat(c context.Context, r *connect.Request[heartbeatv1.MemberMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.member(c, r.Header(), h.legacy.CreateHeartbeat, http.MethodPost, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) UpdateHeartbeat(c context.Context, r *connect.Request[heartbeatv1.MemberMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.member(c, r.Header(), h.legacy.UpdateHeartbeat, http.MethodPut, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) DeleteHeartbeat(c context.Context, r *connect.Request[heartbeatv1.MemberRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.member(c, r.Header(), h.legacy.DeleteHeartbeat, http.MethodDelete, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "", nil, nil)
}

func (h *connectHandler) TriggerHeartbeat(c context.Context, r *connect.Request[heartbeatv1.MemberMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.member(c, r.Header(), h.legacy.TriggerHeartbeat, http.MethodPost, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "/trigger", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) TriggerTeam(c context.Context, r *connect.Request[heartbeatv1.TeamMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.TriggerTeam, http.MethodPost, r.Msg.GetTeamId(), "/trigger", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) GetTeamExecutionStatus(c context.Context, r *connect.Request[heartbeatv1.TeamRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.GetTeamExecutionStatus, http.MethodGet, r.Msg.GetTeamId(), "/execution-status", nil, nil)
}

func (h *connectHandler) ClearTeamQueueRunning(c context.Context, r *connect.Request[heartbeatv1.MemberMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.teamAgent(c, r.Header(), h.legacy.ClearTeamQueueRunning, http.MethodDelete, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "/queue/running/", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) ListTeamLogs(c context.Context, r *connect.Request[heartbeatv1.TeamQueryRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.ListTeamLogs, http.MethodGet, r.Msg.GetTeamId(), "/heartbeats/logs", nil, r.Msg.GetQuery())
}

func (h *connectHandler) ListLogs(c context.Context, r *connect.Request[heartbeatv1.MemberQueryRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.member(c, r.Header(), h.legacy.ListLogs, http.MethodGet, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "/logs", nil, r.Msg.GetQuery())
}

func (h *connectHandler) GetLog(c context.Context, r *connect.Request[heartbeatv1.LogRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.memberLog(c, r.Header(), h.legacy.GetLog, r.Msg.GetTeamId(), r.Msg.GetAgentId(), r.Msg.GetLogId())
}

func (h *connectHandler) GetResponsibilities(c context.Context, r *connect.Request[heartbeatv1.MemberRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.memberDoc(c, r.Header(), h.legacy.GetResponsibilities, http.MethodGet, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "responsibilities", nil)
}

func (h *connectHandler) SetResponsibilities(c context.Context, r *connect.Request[heartbeatv1.MemberMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.memberDoc(c, r.Header(), h.legacy.SetResponsibilities, http.MethodPut, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "responsibilities", r.Msg.GetBody())
}

func (h *connectHandler) GetHeartbeatInstructions(c context.Context, r *connect.Request[heartbeatv1.MemberRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.memberDoc(c, r.Header(), h.legacy.GetHeartbeatInstructions, http.MethodGet, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "heartbeat-instructions", nil)
}

func (h *connectHandler) SetHeartbeatInstructions(c context.Context, r *connect.Request[heartbeatv1.MemberMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.memberDoc(c, r.Header(), h.legacy.SetHeartbeatInstructions, http.MethodPut, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "heartbeat-instructions", r.Msg.GetBody())
}

func (h *connectHandler) GetMemberContext(c context.Context, r *connect.Request[heartbeatv1.MemberRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.memberDoc(c, r.Header(), h.legacy.GetMemberContext, http.MethodGet, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "context", nil)
}

func (h *connectHandler) GetLastHandoff(c context.Context, r *connect.Request[heartbeatv1.MemberRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.memberDoc(c, r.Header(), h.legacy.GetLastHandoff, http.MethodGet, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "handoff", nil)
}

func (h *connectHandler) ClearLastHandoff(c context.Context, r *connect.Request[heartbeatv1.MemberMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.memberDoc(c, r.Header(), h.legacy.ClearLastHandoff, http.MethodDelete, r.Msg.GetTeamId(), r.Msg.GetAgentId(), "handoff", r.Msg.GetBody())
}

func (h *connectHandler) GetHandoffHistory(c context.Context, r *connect.Request[heartbeatv1.TeamQueryRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.GetHandoffHistory, http.MethodGet, r.Msg.GetTeamId(), "/handoff-history", nil, r.Msg.GetQuery())
}

func (h *connectHandler) ClearHandoffHistory(c context.Context, r *connect.Request[heartbeatv1.TeamMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.ClearHandoffHistory, http.MethodDelete, r.Msg.GetTeamId(), "/handoff-history", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) GetTaskBoard(c context.Context, r *connect.Request[heartbeatv1.TeamQueryRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.GetTaskBoard, http.MethodGet, r.Msg.GetTeamId(), "/tasks", nil, r.Msg.GetQuery())
}

func (h *connectHandler) AddTask(c context.Context, r *connect.Request[heartbeatv1.TeamMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.AddTask, http.MethodPost, r.Msg.GetTeamId(), "/tasks", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) UpdateTask(c context.Context, r *connect.Request[heartbeatv1.TaskMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.task(c, r.Header(), h.legacy.UpdateTaskHandler, http.MethodPut, r.Msg.GetTeamId(), r.Msg.GetTaskId(), r.Msg.GetBody())
}

func (h *connectHandler) DeleteTask(c context.Context, r *connect.Request[heartbeatv1.TaskMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.task(c, r.Header(), h.legacy.DeleteTaskHandler, http.MethodDelete, r.Msg.GetTeamId(), r.Msg.GetTaskId(), r.Msg.GetBody())
}

func (h *connectHandler) CaptureBug(c context.Context, r *connect.Request[heartbeatv1.TeamMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.CaptureBug, http.MethodPost, r.Msg.GetTeamId(), "/bugs/capture", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) RepairBug(c context.Context, r *connect.Request[heartbeatv1.BugMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.bug(c, r.Header(), h.legacy.RepairBugCapture, r.Msg.GetTeamId(), r.Msg.GetDraftId(), r.Msg.GetBody())
}

func (h *connectHandler) GetRetention(c context.Context, r *connect.Request[heartbeatv1.TeamRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.GetRetention, http.MethodGet, r.Msg.GetTeamId(), "/retention", nil, nil)
}

func (h *connectHandler) PruneSharedState(c context.Context, r *connect.Request[heartbeatv1.TeamMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.PruneSharedState, http.MethodPost, r.Msg.GetTeamId(), "/prune", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) CreateTask(c context.Context, r *connect.Request[heartbeatv1.JsonMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.CreateTask, http.MethodPost, "/tasks", r.Msg.GetBody(), nil)
}

func (h *connectHandler) CreateRun(c context.Context, r *connect.Request[heartbeatv1.JsonMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.CreateRun, http.MethodPost, "/runs", r.Msg.GetBody(), nil)
}

func (h *connectHandler) ListRuns(c context.Context, r *connect.Request[heartbeatv1.QueryRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.ListRuns, http.MethodGet, "/runs", nil, r.Msg.GetQuery())
}

func (h *connectHandler) ListHeartbeatAttempts(c context.Context, r *connect.Request[heartbeatv1.QueryRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.ListHeartbeatAttempts, http.MethodGet, "/heartbeat-attempts", nil, r.Msg.GetQuery())
}

func (h *connectHandler) CreateInvestigationRun(c context.Context, r *connect.Request[heartbeatv1.JsonMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.CreateInvestigationRun, http.MethodPost, "/runs/investigate", r.Msg.GetBody(), nil)
}

func (h *connectHandler) CreateInvestigationApplyRun(c context.Context, r *connect.Request[heartbeatv1.JsonMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.CreateInvestigationApplyRun, http.MethodPost, "/runs/investigation-apply", r.Msg.GetBody(), nil)
}

func (h *connectHandler) GetRun(c context.Context, r *connect.Request[heartbeatv1.RunRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.run(c, r.Header(), h.legacy.GetRun, http.MethodGet, r.Msg.GetRunId(), "", nil, nil)
}

func (h *connectHandler) RetryRun(c context.Context, r *connect.Request[heartbeatv1.RunMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.run(c, r.Header(), h.legacy.RetryRun, http.MethodPost, r.Msg.GetRunId(), "/retry", r.Msg.GetBody(), nil)
}

func (h *connectHandler) GetRunEvents(c context.Context, r *connect.Request[heartbeatv1.RunQueryRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.run(c, r.Header(), h.legacy.GetRunEvents, http.MethodGet, r.Msg.GetRunId(), "/events", nil, r.Msg.GetQuery())
}

func (h *connectHandler) ContinueRun(c context.Context, r *connect.Request[heartbeatv1.RunMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.run(c, r.Header(), h.legacy.ContinueRun, http.MethodPost, r.Msg.GetRunId(), "/continue", r.Msg.GetBody(), nil)
}

func (h *connectHandler) GetHeartbeatControl(c context.Context, r *connect.Request[heartbeatv1.EmptyRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.GetHeartbeatControl, http.MethodGet, "/heartbeats/control", nil, nil)
}

func (h *connectHandler) UpdateHeartbeatControlPolicy(c context.Context, r *connect.Request[heartbeatv1.JsonMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.UpdateHeartbeatControlPolicy, http.MethodPut, "/heartbeats/control/policy", r.Msg.GetBody(), nil)
}

func (h *connectHandler) PauseHeartbeatControl(c context.Context, r *connect.Request[heartbeatv1.JsonMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.PauseHeartbeatControl, http.MethodPost, "/heartbeats/control/pause", r.Msg.GetBody(), nil)
}

func (h *connectHandler) ResumeHeartbeatControl(c context.Context, r *connect.Request[heartbeatv1.JsonMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.ResumeHeartbeatControl, http.MethodPost, "/heartbeats/control/resume", r.Msg.GetBody(), nil)
}

func (h *connectHandler) GetTeamHeartbeatControl(c context.Context, r *connect.Request[heartbeatv1.TeamRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.GetTeamHeartbeatControl, http.MethodGet, r.Msg.GetTeamId(), "/heartbeats/control", nil, nil)
}

func (h *connectHandler) UpdateTeamHeartbeatControlPolicy(c context.Context, r *connect.Request[heartbeatv1.TeamMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.UpdateTeamHeartbeatControlPolicy, http.MethodPut, r.Msg.GetTeamId(), "/heartbeats/control/policy", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) PauseTeamHeartbeatControl(c context.Context, r *connect.Request[heartbeatv1.TeamMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.PauseTeamHeartbeatControl, http.MethodPost, r.Msg.GetTeamId(), "/heartbeats/control/pause", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) ResumeTeamHeartbeatControl(c context.Context, r *connect.Request[heartbeatv1.TeamMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.ResumeTeamHeartbeatControl, http.MethodPost, r.Msg.GetTeamId(), "/heartbeats/control/resume", r.Msg.GetBody(), r.Msg.GetQuery())
}

func (h *connectHandler) ListRunning(c context.Context, r *connect.Request[heartbeatv1.EmptyRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.ListRunning, http.MethodGet, "/heartbeats/running", nil, nil)
}

func (h *connectHandler) StopRunning(c context.Context, r *connect.Request[heartbeatv1.MemberMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, r.Header(), h.legacy.StopRunning, http.MethodPost, "/heartbeats/running/"+url.PathEscape(r.Msg.GetTeamId())+"/"+url.PathEscape(r.Msg.GetAgentId())+"/stop", r.Msg.GetBody(), r.Msg.GetQuery(), map[string]string{"teamId": r.Msg.GetTeamId(), "agentId": r.Msg.GetAgentId()})
}

func (h *connectHandler) PreviewPrompt(c context.Context, r *connect.Request[heartbeatv1.JsonMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.PreviewPrompt, http.MethodPost, "/prompt-preview", r.Msg.GetBody(), nil)
}

func (h *connectHandler) PreviewPromptStructured(c context.Context, r *connect.Request[heartbeatv1.JsonMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simple(c, r.Header(), h.legacy.PreviewPromptStructured, http.MethodPost, "/prompt-preview-structured", r.Msg.GetBody(), nil)
}

func (h *connectHandler) PreviewPromptMatrix(c context.Context, r *connect.Request[heartbeatv1.TeamQueryRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.team(c, r.Header(), h.legacy.PreviewPromptMatrix, http.MethodGet, r.Msg.GetTeamId(), "/prompt-matrix", nil, r.Msg.GetQuery())
}

func (h *connectHandler) simple(c context.Context, hd http.Header, fn http.HandlerFunc, method, path string, body any, q map[string]string) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, hd, fn, method, path, body, q, nil)
}

func (h *connectHandler) simpleVars(c context.Context, hd http.Header, fn http.HandlerFunc, method, path string, body any, q map[string]string, vars map[string]string) (*connect.Response[heartbeatv1.JsonResponse], error) {
	result, err := transportbridge.Invoke(c, hd, fn, method, withQuery(path, q), transportbridge.ValueBody(asValue(body)), vars)
	if err != nil {
		return nil, err
	}
	value, err := transportbridge.DecodeValue(result.Body)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&heartbeatv1.JsonResponse{Data: value}), nil
}

func (h *connectHandler) team(c context.Context, hd http.Header, fn http.HandlerFunc, method, team, suffix string, body any, q map[string]string) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, hd, fn, method, "/teams/"+url.PathEscape(team)+suffix, body, q, map[string]string{"id": team})
}

func (h *connectHandler) member(c context.Context, hd http.Header, fn http.HandlerFunc, method, team, agent, suffix string, body any, q map[string]string) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, hd, fn, method, "/teams/"+url.PathEscape(team)+"/heartbeats/"+url.PathEscape(agent)+suffix, body, q, map[string]string{"id": team, "agentId": agent})
}

func (h *connectHandler) teamAgent(c context.Context, hd http.Header, fn http.HandlerFunc, method, team, agent, prefix string, body any, q map[string]string) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, hd, fn, method, "/teams/"+url.PathEscape(team)+prefix+url.PathEscape(agent), body, q, map[string]string{"id": team, "agentId": agent})
}

func (h *connectHandler) memberDoc(c context.Context, hd http.Header, fn http.HandlerFunc, method, team, agent, doc string, body any) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, hd, fn, method, "/teams/"+url.PathEscape(team)+"/members/"+url.PathEscape(agent)+"/"+doc, body, nil, map[string]string{"id": team, "agentId": agent})
}

func (h *connectHandler) memberLog(c context.Context, hd http.Header, fn http.HandlerFunc, team, agent, logID string) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, hd, fn, http.MethodGet, "/teams/"+url.PathEscape(team)+"/heartbeats/"+url.PathEscape(agent)+"/logs/"+url.PathEscape(logID), nil, nil, map[string]string{"id": team, "agentId": agent, "logId": logID})
}

func (h *connectHandler) task(c context.Context, hd http.Header, fn http.HandlerFunc, method, team, taskID string, body any) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, hd, fn, method, "/teams/"+url.PathEscape(team)+"/tasks/"+url.PathEscape(taskID), body, nil, map[string]string{"id": team, "taskId": taskID})
}

func (h *connectHandler) bug(c context.Context, hd http.Header, fn http.HandlerFunc, team, draftID string, body any) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, hd, fn, http.MethodPut, "/teams/"+url.PathEscape(team)+"/bugs/"+url.PathEscape(draftID)+"/capture", body, nil, map[string]string{"id": team, "draftId": draftID})
}

func (h *connectHandler) run(c context.Context, hd http.Header, fn http.HandlerFunc, method, runID, suffix string, body any, q map[string]string) (*connect.Response[heartbeatv1.JsonResponse], error) {
	return h.simpleVars(c, hd, fn, method, "/runs/"+url.PathEscape(runID)+suffix, body, q, map[string]string{"runId": runID})
}

func withQuery(path string, q map[string]string) string {
	if len(q) == 0 {
		return path
	}
	values := url.Values{}
	for k, v := range q {
		values.Set(k, v)
	}
	return path + "?" + values.Encode()
}

func asValue(v any) *structpb.Value {
	if value, ok := v.(*structpb.Value); ok {
		return value
	}
	return nil
}
