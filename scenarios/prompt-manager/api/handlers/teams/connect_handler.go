package teams

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/proto"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/teams"
)

type connectHandler struct {
	teamsconnect.UnimplementedTeamsServiceHandler
	legacy *domain.Handlers
}

func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return teamsconnect.NewTeamsServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) ListTeams(ctx context.Context, req *connect.Request[teamsv1.ListTeamsRequest]) (*connect.Response[teamsv1.ListTeamsResponse], error) {
	return transportbridge.InvokeWrappedJSON(ctx, req.Header(), h.legacy.List, http.MethodGet, "/teams", nil, nil, "teams", &teamsv1.ListTeamsResponse{})
}

func (h *connectHandler) GetTeam(ctx context.Context, req *connect.Request[teamsv1.GetTeamRequest]) (*connect.Response[teamsv1.TeamDetails], error) {
	return teamCall(ctx, req.Header(), h.legacy.Get, http.MethodGet, req.Msg.GetId(), "", nil, nil, &teamsv1.TeamDetails{})
}

func (h *connectHandler) CreateTeam(ctx context.Context, req *connect.Request[teamsv1.CreateTeamRequest]) (*connect.Response[teamsv1.TeamDetails], error) {
	body, err := transportbridge.ProtoBody(req.Msg.GetTeam())
	if err != nil {
		return nil, err
	}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.Create, http.MethodPost, "/teams", body, nil, &teamsv1.TeamDetails{})
}

func (h *connectHandler) UpdateTeam(ctx context.Context, req *connect.Request[teamsv1.UpdateTeamRequest]) (*connect.Response[teamsv1.TeamDetails], error) {
	body, err := transportbridge.MaskedBody(req.Msg.GetTeam(), req.Msg.GetUpdateMask().GetPaths())
	if err != nil {
		return nil, err
	}
	return teamCall(ctx, req.Header(), h.legacy.Update, http.MethodPut, req.Msg.GetId(), "", body, nil, &teamsv1.TeamDetails{})
}

func (h *connectHandler) DeleteTeam(ctx context.Context, req *connect.Request[teamsv1.DeleteTeamRequest]) (*connect.Response[teamsv1.DeleteTeamResponse], error) {
	_, err := invokeTeam(ctx, req.Header(), h.legacy.Delete, http.MethodDelete, req.Msg.GetId(), "", nil, nil)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&teamsv1.DeleteTeamResponse{}), nil
}

func (h *connectHandler) GetExclusiveMembers(ctx context.Context, req *connect.Request[teamsv1.GetExclusiveMembersRequest]) (*connect.Response[teamsv1.ExclusiveMembersResponse], error) {
	return teamCall(ctx, req.Header(), h.legacy.GetExclusiveMembers, http.MethodGet, req.Msg.GetTeamId(), "/exclusive-members", nil, nil, &teamsv1.ExclusiveMembersResponse{})
}

func (h *connectHandler) AddMember(ctx context.Context, req *connect.Request[teamsv1.AddMemberRequest]) (*connect.Response[teamsv1.Member], error) {
	body := map[string]any{"agentId": req.Msg.GetAgentId(), "roles": req.Msg.GetRoles()}
	return teamCall(ctx, req.Header(), h.legacy.AddMember, http.MethodPost, req.Msg.GetTeamId(), "/members", body, nil, &teamsv1.Member{})
}

func (h *connectHandler) UpdateMember(ctx context.Context, req *connect.Request[teamsv1.UpdateMemberRequest]) (*connect.Response[teamsv1.Member], error) {
	body := map[string]any{"roles": req.Msg.GetRoles()}
	if req.Msg.Status != nil {
		body["status"] = req.Msg.GetStatus()
	}
	vars := map[string]string{"agentId": req.Msg.GetAgentId()}
	return teamCall(ctx, req.Header(), h.legacy.UpdateMember, http.MethodPut, req.Msg.GetTeamId(), "/members/"+url.PathEscape(req.Msg.GetAgentId()), body, vars, &teamsv1.Member{})
}

func (h *connectHandler) RemoveMember(ctx context.Context, req *connect.Request[teamsv1.RemoveMemberRequest]) (*connect.Response[teamsv1.RemoveMemberResponse], error) {
	_, err := invokeTeam(ctx, req.Header(), h.legacy.RemoveMember, http.MethodDelete, req.Msg.GetTeamId(), "/members/"+url.PathEscape(req.Msg.GetAgentId()), nil, map[string]string{"agentId": req.Msg.GetAgentId()})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&teamsv1.RemoveMemberResponse{}), nil
}

func (h *connectHandler) GetRoles(ctx context.Context, req *connect.Request[teamsv1.GetRolesRequest]) (*connect.Response[teamsv1.GetRolesResponse], error) {
	result, err := invokeTeam(ctx, req.Header(), h.legacy.GetRoles, http.MethodGet, req.Msg.GetTeamId(), "/roles", nil, nil)
	if err != nil {
		return nil, err
	}
	out := &teamsv1.GetRolesResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "roles", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SetRoles(ctx context.Context, req *connect.Request[teamsv1.SetRolesRequest]) (*connect.Response[teamsv1.GetRolesResponse], error) {
	roles := make([]map[string]string, 0, len(req.Msg.GetRoles()))
	for _, role := range req.Msg.GetRoles() {
		roles = append(roles, map[string]string{"id": role.GetId(), "name": role.GetName(), "description": role.GetDescription()})
	}
	body := map[string]any{"roles": roles}
	result, err := invokeTeam(ctx, req.Header(), h.legacy.SetRoles, http.MethodPut, req.Msg.GetTeamId(), "/roles", body, nil)
	if err != nil {
		return nil, err
	}
	out := &teamsv1.GetRolesResponse{}
	if err := transportbridge.DecodeWrapped(result.Body, "roles", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListSharedFiles(ctx context.Context, req *connect.Request[teamsv1.ListSharedFilesRequest]) (*connect.Response[teamsv1.ListSharedFilesResponse], error) {
	return teamCall(ctx, req.Header(), h.legacy.ListSharedFiles, http.MethodGet, req.Msg.GetTeamId(), "/shared/files", nil, nil, &teamsv1.ListSharedFilesResponse{})
}

func (h *connectHandler) GetSharedFile(ctx context.Context, req *connect.Request[teamsv1.GetSharedFileRequest]) (*connect.Response[teamsv1.SharedFileContent], error) {
	return teamCall(ctx, req.Header(), h.legacy.GetSharedFile, http.MethodGet, req.Msg.GetTeamId(), "/shared/files/content?path="+url.QueryEscape(req.Msg.GetPath()), nil, nil, &teamsv1.SharedFileContent{})
}

func (h *connectHandler) SetSharedFile(ctx context.Context, req *connect.Request[teamsv1.SetSharedFileRequest]) (*connect.Response[teamsv1.SharedFileContent], error) {
	return teamCall(ctx, req.Header(), h.legacy.SetSharedFile, http.MethodPut, req.Msg.GetTeamId(), "/shared/files/content?path="+url.QueryEscape(req.Msg.GetPath()), map[string]string{"content": req.Msg.GetContent()}, nil, &teamsv1.SharedFileContent{})
}

func (h *connectHandler) CreateSharedFile(ctx context.Context, req *connect.Request[teamsv1.CreateSharedFileRequest]) (*connect.Response[teamsv1.SharedFileContent], error) {
	body := map[string]any{"path": req.Msg.GetPath(), "content": req.Msg.GetContent(), "isDir": req.Msg.GetIsDir()}
	return teamCall(ctx, req.Header(), h.legacy.CreateSharedFile, http.MethodPost, req.Msg.GetTeamId(), "/shared/files", body, nil, &teamsv1.SharedFileContent{})
}

func (h *connectHandler) RenameSharedFile(ctx context.Context, req *connect.Request[teamsv1.RenameSharedFileRequest]) (*connect.Response[teamsv1.SharedFileContent], error) {
	body := map[string]string{"from": req.Msg.GetFrom(), "to": req.Msg.GetTo()}
	return teamCall(ctx, req.Header(), h.legacy.RenameSharedFile, http.MethodPost, req.Msg.GetTeamId(), "/shared/files/rename", body, nil, &teamsv1.SharedFileContent{})
}

func (h *connectHandler) DeleteSharedFile(ctx context.Context, req *connect.Request[teamsv1.DeleteSharedFileRequest]) (*connect.Response[teamsv1.DeleteSharedFileResponse], error) {
	_, err := invokeTeam(ctx, req.Header(), h.legacy.DeleteSharedFile, http.MethodDelete, req.Msg.GetTeamId(), "/shared/files?path="+url.QueryEscape(req.Msg.GetPath()), nil, nil)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&teamsv1.DeleteSharedFileResponse{}), nil
}

func (h *connectHandler) GetOrgChart(ctx context.Context, req *connect.Request[teamsv1.GetOrgChartRequest]) (*connect.Response[teamsv1.OrgChart], error) {
	return teamCall(ctx, req.Header(), h.legacy.GetOrgChart, http.MethodGet, req.Msg.GetTeamId(), "/org", nil, nil, &teamsv1.OrgChart{})
}

func (h *connectHandler) SetOrgChart(ctx context.Context, req *connect.Request[teamsv1.SetOrgChartRequest]) (*connect.Response[teamsv1.OrgChart], error) {
	edges := make([]map[string]string, 0, len(req.Msg.GetEdges()))
	for _, edge := range req.Msg.GetEdges() {
		edges = append(edges, map[string]string{"managerAgentId": edge.GetManagerAgentId(), "reportAgentId": edge.GetReportAgentId()})
	}
	return teamCall(ctx, req.Header(), h.legacy.SetOrgChart, http.MethodPut, req.Msg.GetTeamId(), "/org", map[string]any{"edges": edges}, nil, &teamsv1.OrgChart{})
}

func (h *connectHandler) UpdateOrgChartEdge(ctx context.Context, req *connect.Request[teamsv1.UpdateOrgChartEdgeRequest]) (*connect.Response[teamsv1.OrgChart], error) {
	vars := map[string]string{"reportId": req.Msg.GetReportAgentId()}
	body := map[string]string{"managerAgentId": req.Msg.GetManagerAgentId()}
	return teamCall(ctx, req.Header(), h.legacy.UpdateOrgChartEdge, http.MethodPut, req.Msg.GetTeamId(), "/org/edges/"+url.PathEscape(req.Msg.GetReportAgentId()), body, vars, &teamsv1.OrgChart{})
}

func (h *connectHandler) DeleteOrgChartEdge(ctx context.Context, req *connect.Request[teamsv1.DeleteOrgChartEdgeRequest]) (*connect.Response[teamsv1.OrgChart], error) {
	vars := map[string]string{"reportId": req.Msg.GetReportAgentId()}
	return teamCall(ctx, req.Header(), h.legacy.DeleteOrgChartEdge, http.MethodDelete, req.Msg.GetTeamId(), "/org/edges/"+url.PathEscape(req.Msg.GetReportAgentId()), nil, vars, &teamsv1.OrgChart{})
}

func (h *connectHandler) ListMessages(ctx context.Context, req *connect.Request[teamsv1.ListMessagesRequest]) (*connect.Response[teamsv1.Inbox], error) {
	vars := map[string]string{"agentId": req.Msg.GetAgentId()}
	return teamCall(ctx, req.Header(), h.legacy.ListTeamMessages, http.MethodGet, req.Msg.GetTeamId(), "/members/"+url.PathEscape(req.Msg.GetAgentId())+"/messages", nil, vars, &teamsv1.Inbox{})
}

func (h *connectHandler) SendMessage(ctx context.Context, req *connect.Request[teamsv1.SendMessageRequest]) (*connect.Response[teamsv1.Message], error) {
	vars := map[string]string{"agentId": req.Msg.GetAgentId()}
	body := map[string]string{"fromAgentId": req.Msg.GetFromAgentId(), "content": req.Msg.GetContent()}
	return teamCall(ctx, req.Header(), h.legacy.SendTeamMessage, http.MethodPost, req.Msg.GetTeamId(), "/members/"+url.PathEscape(req.Msg.GetAgentId())+"/messages", body, vars, &teamsv1.Message{})
}

func (h *connectHandler) ClearMessages(ctx context.Context, req *connect.Request[teamsv1.ClearMessagesRequest]) (*connect.Response[teamsv1.ClearMessagesResponse], error) {
	vars := map[string]string{"agentId": req.Msg.GetAgentId()}
	_, err := invokeTeam(ctx, req.Header(), h.legacy.ClearTeamMessages, http.MethodDelete, req.Msg.GetTeamId(), "/members/"+url.PathEscape(req.Msg.GetAgentId())+"/messages", nil, vars)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&teamsv1.ClearMessagesResponse{}), nil
}

func (h *connectHandler) DeleteMessage(ctx context.Context, req *connect.Request[teamsv1.DeleteMessageRequest]) (*connect.Response[teamsv1.DeleteMessageResponse], error) {
	vars := map[string]string{"agentId": req.Msg.GetAgentId(), "messageId": req.Msg.GetMessageId()}
	suffix := "/members/" + url.PathEscape(req.Msg.GetAgentId()) + "/messages/" + url.PathEscape(req.Msg.GetMessageId())
	_, err := invokeTeam(ctx, req.Header(), h.legacy.DeleteTeamMessage, http.MethodDelete, req.Msg.GetTeamId(), suffix, nil, vars)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&teamsv1.DeleteMessageResponse{}), nil
}

func (h *connectHandler) ListAvailableClaudeCodeTeams(ctx context.Context, req *connect.Request[teamsv1.ListAvailableClaudeCodeTeamsRequest]) (*connect.Response[teamsv1.ListAvailableClaudeCodeTeamsResponse], error) {
	return transportbridge.InvokeWrappedJSON(ctx, req.Header(), h.legacy.ListAvailableCCTeams, http.MethodGet, "/teams/import/claude-code/available", nil, nil, "teams", &teamsv1.ListAvailableClaudeCodeTeamsResponse{})
}

func (h *connectHandler) ImportClaudeCodeTeam(ctx context.Context, req *connect.Request[teamsv1.ImportClaudeCodeTeamRequest]) (*connect.Response[teamsv1.TeamDetails], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.ImportClaudeCode, http.MethodPost, "/teams/import/claude-code", map[string]string{"teamName": req.Msg.GetTeamName()}, nil, &teamsv1.TeamDetails{})
}

func (h *connectHandler) ExportClaudeCodeTeam(ctx context.Context, req *connect.Request[teamsv1.ExportClaudeCodeTeamRequest]) (*connect.Response[teamsv1.ExportClaudeCodeTeamResponse], error) {
	result, err := invokeTeam(ctx, req.Header(), h.legacy.ExportClaudeCode, http.MethodGet, req.Msg.GetTeamId(), "/export/claude-code", nil, nil)
	if err != nil {
		return nil, err
	}
	out := &teamsv1.ExportClaudeCodeTeamResponse{TeamId: req.Msg.GetTeamId()}
	if err := transportbridge.DecodeWrapped(result.Body, "export", out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func teamCall[T any](ctx context.Context, headers http.Header, handler http.HandlerFunc, method, teamID, suffix string, body any, vars map[string]string, out *T) (*connect.Response[T], error) {
	result, err := invokeTeam(ctx, headers, handler, method, teamID, suffix, body, vars)
	if err != nil {
		return nil, err
	}
	message, ok := any(out).(proto.Message)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("generated response is not a protobuf message"))
	}
	if err := transportbridge.Decode(result.Body, message); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func invokeTeam(ctx context.Context, headers http.Header, handler http.HandlerFunc, method, teamID, suffix string, body any, vars map[string]string) (transportbridge.Result, error) {
	if vars == nil {
		vars = map[string]string{}
	}
	vars["id"] = teamID
	target := "/teams/" + url.PathEscape(teamID) + suffix
	return transportbridge.Invoke(ctx, headers, handler, method, target, body, vars)
}
