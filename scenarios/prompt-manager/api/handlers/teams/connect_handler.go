package teams

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	teamsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams"
	teamsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/teams/teams_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"prompt-manager/handlers/transportbridge"
	workspace "prompt-manager/internal/effortworkspace"
	domain "prompt-manager/internal/teams"
)

type connectHandler struct {
	teamsconnect.UnimplementedTeamsServiceHandler
	legacy     *domain.Handlers
	knowledge  knowledgeHandlers
	workspaces *workspace.Store
}

type knowledgeHandlers interface {
	AddKnowledge(http.ResponseWriter, *http.Request)
	ListTeamCorpus(http.ResponseWriter, *http.Request)
	UpdateTeamCorpusHandler(http.ResponseWriter, *http.Request)
	DeleteTeamCorpusHandler(http.ResponseWriter, *http.Request)
}

func NewConnectMount(legacy *domain.Handlers, extras ...any) (string, http.Handler) {
	var kh knowledgeHandlers
	var ws *workspace.Store
	for _, extra := range extras {
		switch value := extra.(type) {
		case knowledgeHandlers:
			kh = value
		case *workspace.Store:
			ws = value
		}
	}
	return teamsconnect.NewTeamsServiceHandler(&connectHandler{legacy: legacy, knowledge: kh, workspaces: ws})
}

func (h *connectHandler) ListEffortWorkspaces(ctx context.Context, req *connect.Request[teamsv1.ListEffortWorkspacesRequest]) (*connect.Response[teamsv1.EffortWorkspaceListResponse], error) {
	if h.workspaces == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("effort workspace store not configured"))
	}
	result, err := h.workspaces.List(ctx, req.Msg.GetTeamId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &teamsv1.EffortWorkspaceListResponse{TeamId: result.TeamID}
	for _, item := range result.Workspaces {
		workspace := &teamsv1.EffortWorkspace{EffortRef: item.EffortRef, Slug: item.Slug, Stage: item.Stage}
		for _, file := range item.Files {
			workspace.Files = append(workspace.Files, &teamsv1.EffortWorkspaceFile{Path: file.Path, IsDir: file.IsDir, Size: file.Size})
		}
		out.Workspaces = append(out.Workspaces, workspace)
	}
	for _, item := range result.Unavailable {
		out.Unavailable = append(out.Unavailable, &teamsv1.EffortWorkspaceUnavailable{EffortRef: item.EffortRef, Reason: item.Reason})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetEffortWorkspaceContent(ctx context.Context, req *connect.Request[teamsv1.GetEffortWorkspaceContentRequest]) (*connect.Response[teamsv1.EffortWorkspaceContent], error) {
	if h.workspaces == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("effort workspace store not configured"))
	}
	result, err := h.workspaces.Read(ctx, req.Msg.GetEffortRef(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	contentType := mime.TypeByExtension(filepath.Ext(result.Path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	content := result.Content
	var contentBytes []byte
	if domain.IsBinaryPreviewType(contentType) {
		content = ""
		contentBytes = result.ContentBytes
	}
	previewDataURL := ""
	if len(contentBytes) > 0 {
		previewDataURL = "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(contentBytes)
	}
	return connect.NewResponse(&teamsv1.EffortWorkspaceContent{
		EffortRef: result.EffortRef, Path: result.Path, Content: content,
		ContentBytes: contentBytes, ContentType: contentType,
		ContentTruncated: result.ContentTruncated,
		PreviewDataUrl:   previewDataURL,
	}), nil
}

func (h *connectHandler) ListTeams(ctx context.Context, req *connect.Request[teamsv1.ListTeamsRequest]) (*connect.Response[teamsv1.ListTeamsResponse], error) {
	result, err := h.legacy.ListTeams(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &teamsv1.ListTeamsResponse{}
	payload, err := json.Marshal(map[string]any{"teams": result})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetTeam(ctx context.Context, req *connect.Request[teamsv1.GetTeamRequest]) (*connect.Response[teamsv1.TeamDetails], error) {
	result, err := h.legacy.GetTeam(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &teamsv1.TeamDetails{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateTeam(ctx context.Context, req *connect.Request[teamsv1.CreateTeamRequest]) (*connect.Response[teamsv1.TeamDetails], error) {
	if req.Msg.GetTeam() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("team is required"))
	}
	raw, err := protojson.Marshal(req.Msg.GetTeam())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var input domain.CreateRequest
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := h.legacy.CreateTeam(ctx, input)
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "displayName is required" {
			code = connect.CodeInvalidArgument
		}
		if strings.Contains(err.Error(), "already exists") {
			code = connect.CodeAlreadyExists
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.TeamDetails{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) UpdateTeam(ctx context.Context, req *connect.Request[teamsv1.UpdateTeamRequest]) (*connect.Response[teamsv1.TeamDetails], error) {
	body, err := transportbridge.MaskedBody(req.Msg.GetTeam(), req.Msg.GetUpdateMask().GetPaths())
	if err != nil {
		return nil, err
	}
	return teamCall(ctx, req.Header(), h.legacy.Update, http.MethodPut, req.Msg.GetId(), "", body, nil, &teamsv1.TeamDetails{})
}

func (h *connectHandler) DeleteTeam(ctx context.Context, req *connect.Request[teamsv1.DeleteTeamRequest]) (*connect.Response[teamsv1.DeleteTeamResponse], error) {
	if err := h.legacy.DeleteTeam(ctx, req.Msg.GetId()); err != nil {
		code := connect.CodeInternal
		if strings.Contains(err.Error(), "not found") {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&teamsv1.DeleteTeamResponse{}), nil
}

func (h *connectHandler) GetExclusiveMembers(ctx context.Context, req *connect.Request[teamsv1.GetExclusiveMembersRequest]) (*connect.Response[teamsv1.ExclusiveMembersResponse], error) {
	result, err := h.legacy.ListExclusiveMembers(ctx, req.Msg.GetTeamId())
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "Team not found" {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.ExclusiveMembersResponse{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) AddMember(ctx context.Context, req *connect.Request[teamsv1.AddMemberRequest]) (*connect.Response[teamsv1.Member], error) {
	result, err := h.legacy.AddTeamMember(ctx, req.Msg.GetTeamId(), domain.AddMemberRequest{AgentID: req.Msg.GetAgentId(), Roles: req.Msg.GetRoles()})
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "agentId is required" {
			code = connect.CodeInvalidArgument
		}
		if err.Error() == "Team not found" || err.Error() == "Agent not found" {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.Member{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) UpdateMember(ctx context.Context, req *connect.Request[teamsv1.UpdateMemberRequest]) (*connect.Response[teamsv1.Member], error) {
	result, err := h.legacy.UpdateTeamMember(ctx, req.Msg.GetTeamId(), req.Msg.GetAgentId(), domain.UpdateMemberRequest{Roles: req.Msg.GetRoles(), Status: req.Msg.Status})
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "Membership not found" || err.Error() == "Team not found" {
			code = connect.CodeNotFound
		}
		if strings.HasPrefix(err.Error(), "cannot deactivate") {
			code = connect.CodeAborted
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.Member{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RemoveMember(ctx context.Context, req *connect.Request[teamsv1.RemoveMemberRequest]) (*connect.Response[teamsv1.RemoveMemberResponse], error) {
	if err := h.legacy.RemoveTeamMember(ctx, req.Msg.GetTeamId(), req.Msg.GetAgentId()); err != nil {
		code := connect.CodeInternal
		if err.Error() == "Team not found" || strings.Contains(err.Error(), "not found") {
			code = connect.CodeNotFound
		}
		if strings.HasPrefix(err.Error(), "cannot remove") {
			code = connect.CodeAborted
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&teamsv1.RemoveMemberResponse{}), nil
}

func (h *connectHandler) GetRoles(ctx context.Context, req *connect.Request[teamsv1.GetRolesRequest]) (*connect.Response[teamsv1.GetRolesResponse], error) {
	roles, err := h.legacy.ListTeamRoles(ctx, req.Msg.GetTeamId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &teamsv1.GetRolesResponse{}
	for _, role := range roles {
		out.Roles = append(out.Roles, &teamsv1.Role{Id: role.ID, Name: role.Name, Description: role.Description})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SetRoles(ctx context.Context, req *connect.Request[teamsv1.SetRolesRequest]) (*connect.Response[teamsv1.GetRolesResponse], error) {
	roles := make([]domain.RoleDTO, 0, len(req.Msg.GetRoles()))
	for _, role := range req.Msg.GetRoles() {
		roles = append(roles, domain.RoleDTO{ID: role.GetId(), Name: role.GetName(), Description: role.GetDescription()})
	}
	result, err := h.legacy.SetTeamRoles(ctx, req.Msg.GetTeamId(), domain.SetRolesRequest{Roles: roles})
	if err != nil {
		code := connect.CodeInternal
		if strings.Contains(err.Error(), "not found") {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.GetRolesResponse{}
	for _, role := range result {
		out.Roles = append(out.Roles, &teamsv1.Role{Id: role.ID, Name: role.Name, Description: role.Description})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListSharedFiles(ctx context.Context, req *connect.Request[teamsv1.ListSharedFilesRequest]) (*connect.Response[teamsv1.ListSharedFilesResponse], error) {
	result, err := h.legacy.ListTeamSharedFiles(ctx, req.Msg.GetTeamId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &teamsv1.ListSharedFilesResponse{TeamId: result.TeamID}
	for _, file := range result.Files {
		out.Files = append(out.Files, &teamsv1.SharedFileEntry{Path: file.Path, IsDir: file.IsDir, Size: file.Size})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetSharedFile(ctx context.Context, req *connect.Request[teamsv1.GetSharedFileRequest]) (*connect.Response[teamsv1.SharedFileContent], error) {
	result, err := h.legacy.ReadTeamSharedFile(ctx, req.Msg.GetTeamId(), req.Msg.GetPath())
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "path is required" || strings.Contains(err.Error(), "invalid path") {
			code = connect.CodeInvalidArgument
		}
		if err.Error() == "Team not found" || err.Error() == "File not found" {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&teamsv1.SharedFileContent{
		TeamId: result.TeamID, Path: result.Path, Content: result.Content,
		ContentBytes: result.ContentBytes, ContentType: result.ContentType,
		ContentTruncated: result.ContentTruncated,
		PreviewDataUrl:   previewDataURL(result.ContentType, result.ContentBytes),
	}), nil
}

func previewDataURL(contentType string, content []byte) string {
	if len(content) == 0 {
		return ""
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(content)
}

func (h *connectHandler) SetSharedFile(ctx context.Context, req *connect.Request[teamsv1.SetSharedFileRequest]) (*connect.Response[teamsv1.SharedFileContent], error) {
	if err := h.legacy.WriteTeamSharedFile(ctx, req.Msg.GetTeamId(), req.Msg.GetPath(), req.Msg.GetContent()); err != nil {
		code := connect.CodeInvalidArgument
		if err.Error() == "Team not found" {
			code = connect.CodeNotFound
		}
		if strings.Contains(err.Error(), "not supported") {
			code = connect.CodeInternal
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&teamsv1.SharedFileContent{TeamId: req.Msg.GetTeamId(), Path: req.Msg.GetPath(), Content: req.Msg.GetContent()}), nil
}

func (h *connectHandler) CreateSharedFile(ctx context.Context, req *connect.Request[teamsv1.CreateSharedFileRequest]) (*connect.Response[teamsv1.SharedFileContent], error) {
	input := domain.TeamSharedFileCreateRequest{Path: req.Msg.GetPath(), Content: req.Msg.GetContent(), IsDir: req.Msg.GetIsDir()}
	if err := h.legacy.CreateTeamSharedFile(ctx, req.Msg.GetTeamId(), input); err != nil {
		code := connect.CodeInvalidArgument
		if err.Error() == "Team not found" {
			code = connect.CodeNotFound
		}
		if strings.Contains(err.Error(), "not supported") {
			code = connect.CodeInternal
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&teamsv1.SharedFileContent{TeamId: req.Msg.GetTeamId(), Path: req.Msg.GetPath(), Content: req.Msg.GetContent()}), nil
}

func (h *connectHandler) RenameSharedFile(ctx context.Context, req *connect.Request[teamsv1.RenameSharedFileRequest]) (*connect.Response[teamsv1.SharedFileContent], error) {
	input := domain.TeamSharedFileRenameRequest{From: req.Msg.GetFrom(), To: req.Msg.GetTo()}
	if err := h.legacy.RenameTeamSharedFile(ctx, req.Msg.GetTeamId(), input); err != nil {
		code := connect.CodeInvalidArgument
		if err.Error() == "Team not found" {
			code = connect.CodeNotFound
		}
		if strings.Contains(err.Error(), "not supported") {
			code = connect.CodeInternal
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&teamsv1.SharedFileContent{TeamId: req.Msg.GetTeamId(), Path: req.Msg.GetTo()}), nil
}

func (h *connectHandler) DeleteSharedFile(ctx context.Context, req *connect.Request[teamsv1.DeleteSharedFileRequest]) (*connect.Response[teamsv1.DeleteSharedFileResponse], error) {
	if err := h.legacy.DeleteTeamSharedFile(ctx, req.Msg.GetTeamId(), req.Msg.GetPath()); err != nil {
		code := connect.CodeInvalidArgument
		if err.Error() == "Team not found" || strings.Contains(err.Error(), "not found") {
			code = connect.CodeNotFound
		}
		if strings.Contains(err.Error(), "not supported") {
			code = connect.CodeInternal
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&teamsv1.DeleteSharedFileResponse{}), nil
}

func (h *connectHandler) GetOrgChart(ctx context.Context, req *connect.Request[teamsv1.GetOrgChartRequest]) (*connect.Response[teamsv1.OrgChart], error) {
	result, err := h.legacy.ReadOrgChart(ctx, req.Msg.GetTeamId())
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "Team not found" {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.OrgChart{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SetOrgChart(ctx context.Context, req *connect.Request[teamsv1.SetOrgChartRequest]) (*connect.Response[teamsv1.OrgChart], error) {
	edges := make([]domain.OrgEdgeDTO, 0, len(req.Msg.GetEdges()))
	for _, edge := range req.Msg.GetEdges() {
		edges = append(edges, domain.OrgEdgeDTO{ManagerAgentID: edge.GetManagerAgentId(), ReportAgentID: edge.GetReportAgentId()})
	}
	managed := make([]domain.ManagedTeamEdgeDTO, 0, len(req.Msg.GetManagedTeamEdges()))
	for _, edge := range req.Msg.GetManagedTeamEdges() {
		managed = append(managed, domain.ManagedTeamEdgeDTO{ManagerTeamID: edge.GetManagerTeamId(), ManagedTeamID: edge.GetManagedTeamId(), Relationship: edge.GetRelationship(), AuthorityRef: edge.GetAuthorityRef(), Status: edge.GetStatus()})
	}
	result, err := h.legacy.WriteOrgChart(ctx, req.Msg.GetTeamId(), domain.SetOrgChartRequest{Edges: edges, ManagedTeamEdges: managed})
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "Team not found" {
			code = connect.CodeNotFound
		}
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "cycle") || strings.Contains(err.Error(), "manager") || strings.Contains(err.Error(), "report") {
			code = connect.CodeInvalidArgument
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.OrgChart{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) UpdateOrgChartEdge(ctx context.Context, req *connect.Request[teamsv1.UpdateOrgChartEdgeRequest]) (*connect.Response[teamsv1.OrgChart], error) {
	result, err := h.legacy.UpdateOrgEdge(ctx, req.Msg.GetTeamId(), req.Msg.GetReportAgentId(), domain.UpdateOrgEdgeRequest{ManagerAgentID: req.Msg.GetManagerAgentId()})
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "Team not found" {
			code = connect.CodeNotFound
		}
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "cycle") {
			code = connect.CodeInvalidArgument
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.OrgChart{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) DeleteOrgChartEdge(ctx context.Context, req *connect.Request[teamsv1.DeleteOrgChartEdgeRequest]) (*connect.Response[teamsv1.OrgChart], error) {
	result, err := h.legacy.DeleteOrgEdge(ctx, req.Msg.GetTeamId(), req.Msg.GetReportAgentId())
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "Team not found" || strings.Contains(err.Error(), "not found") {
			code = connect.CodeNotFound
		}
		if strings.Contains(err.Error(), "required") {
			code = connect.CodeInvalidArgument
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.OrgChart{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListMessages(ctx context.Context, req *connect.Request[teamsv1.ListMessagesRequest]) (*connect.Response[teamsv1.Inbox], error) {
	result, err := h.legacy.ReadTeamMessages(ctx, req.Msg.GetTeamId(), req.Msg.GetAgentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &teamsv1.Inbox{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SendMessage(ctx context.Context, req *connect.Request[teamsv1.SendMessageRequest]) (*connect.Response[teamsv1.Message], error) {
	result, err := h.legacy.SendTeamMessageDirect(ctx, req.Msg.GetTeamId(), req.Msg.GetAgentId(), domain.SendTeamMessageRequest{FromAgentID: req.Msg.GetFromAgentId(), Content: req.Msg.GetContent()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&teamsv1.Message{Id: result.ID, TeamId: result.TeamID, FromAgentId: result.FromAgentID, ToAgentId: result.ToAgentID, Content: result.Content, CreatedAt: result.CreatedAt}), nil
}

func (h *connectHandler) ClearMessages(ctx context.Context, req *connect.Request[teamsv1.ClearMessagesRequest]) (*connect.Response[teamsv1.ClearMessagesResponse], error) {
	if err := h.legacy.ClearTeamMessagesDirect(ctx, req.Msg.GetTeamId(), req.Msg.GetAgentId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&teamsv1.ClearMessagesResponse{}), nil
}

func (h *connectHandler) DeleteMessage(ctx context.Context, req *connect.Request[teamsv1.DeleteMessageRequest]) (*connect.Response[teamsv1.DeleteMessageResponse], error) {
	if err := h.legacy.DeleteTeamMessageDirect(ctx, req.Msg.GetTeamId(), req.Msg.GetAgentId(), req.Msg.GetMessageId()); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&teamsv1.DeleteMessageResponse{}), nil
}

func (h *connectHandler) ListAvailableClaudeCodeTeams(ctx context.Context, req *connect.Request[teamsv1.ListAvailableClaudeCodeTeamsRequest]) (*connect.Response[teamsv1.ListAvailableClaudeCodeTeamsResponse], error) {
	items, err := h.legacy.ListAvailableClaudeCodeTeams(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &teamsv1.ListAvailableClaudeCodeTeamsResponse{}
	for _, item := range items {
		out.Teams = append(out.Teams, &teamsv1.AvailableClaudeCodeTeam{Name: item.Name, MemberCount: int32(item.MemberCount)})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ImportClaudeCodeTeam(ctx context.Context, req *connect.Request[teamsv1.ImportClaudeCodeTeamRequest]) (*connect.Response[teamsv1.TeamDetails], error) {
	result, err := h.legacy.ImportClaudeCodeTeam(ctx, req.Msg.GetTeamName())
	if err != nil {
		code := connect.CodeInvalidArgument
		if strings.Contains(err.Error(), "not found") {
			code = connect.CodeNotFound
		}
		if strings.Contains(err.Error(), "already exists") {
			code = connect.CodeAlreadyExists
		}
		if strings.Contains(err.Error(), "failed to read") {
			code = connect.CodeInternal
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.TeamDetails{}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ExportClaudeCodeTeam(ctx context.Context, req *connect.Request[teamsv1.ExportClaudeCodeTeamRequest]) (*connect.Response[teamsv1.ExportClaudeCodeTeamResponse], error) {
	result, err := h.legacy.ExportClaudeCodeTeam(ctx, req.Msg.GetTeamId())
	if err != nil {
		code := connect.CodeInternal
		if err.Error() == "Team not found" {
			code = connect.CodeNotFound
		}
		if strings.Contains(err.Error(), "only supported") {
			code = connect.CodeInvalidArgument
		}
		return nil, connect.NewError(code, err)
	}
	out := &teamsv1.ExportClaudeCodeTeamResponse{TeamId: req.Msg.GetTeamId()}
	payload, err := json.Marshal(map[string]any{"teamId": req.Msg.GetTeamId(), "export": result})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := protojson.Unmarshal(payload, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListKnowledge(ctx context.Context, req *connect.Request[teamsv1.ListKnowledgeRequest]) (*connect.Response[teamsv1.ListKnowledgeResponse], error) {
	if h.knowledge == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("knowledge handler is not configured"))
	}
	suffix := "/knowledge"
	query := url.Values{}
	if req.Msg.GetTopic() != "" {
		query.Set("topic", req.Msg.GetTopic())
	}
	if req.Msg.GetTopicPrefix() != "" {
		query.Set("topic_prefix", req.Msg.GetTopicPrefix())
	}
	if req.Msg.GetLast() != 0 {
		query.Set("last", fmt.Sprintf("%d", req.Msg.GetLast()))
	}
	if encoded := query.Encode(); encoded != "" {
		suffix += "?" + encoded
	}
	result, err := invokeTeam(ctx, req.Header(), h.knowledge.ListTeamCorpus, http.MethodGet, req.Msg.GetTeamId(), suffix, nil, nil)
	if err != nil {
		return nil, err
	}
	out := &teamsv1.ListKnowledgeResponse{}
	if err := transportbridge.Decode(result.Body, out); err != nil {
		return nil, err
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) AddKnowledge(ctx context.Context, req *connect.Request[teamsv1.AddKnowledgeRequest]) (*connect.Response[teamsv1.KnowledgeEntry], error) {
	if h.knowledge == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("knowledge handler is not configured"))
	}
	body := map[string]string{
		"topic":       req.Msg.GetTopic(),
		"content":     req.Msg.GetContent(),
		"caller_note": req.Msg.GetCallerNote(),
		"source":      req.Msg.GetSource(),
		"supersedes":  req.Msg.GetSupersedes(),
	}
	return knowledgeCall(ctx, req.Header(), h.knowledge.AddKnowledge, http.MethodPost, req.Msg.GetTeamId(), "/knowledge", body, nil, &teamsv1.KnowledgeEntry{})
}

func (h *connectHandler) UpdateKnowledge(ctx context.Context, req *connect.Request[teamsv1.UpdateKnowledgeRequest]) (*connect.Response[teamsv1.KnowledgeEntry], error) {
	if h.knowledge == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("knowledge handler is not configured"))
	}
	body := map[string]string{}
	if req.Msg.Topic != nil {
		body["topic"] = req.Msg.GetTopic()
	}
	if req.Msg.Content != nil {
		body["content"] = req.Msg.GetContent()
	}
	if req.Msg.Source != nil {
		body["source"] = req.Msg.GetSource()
	}
	if req.Msg.Supersedes != nil {
		body["supersedes"] = req.Msg.GetSupersedes()
	}
	vars := map[string]string{"knowledgeId": req.Msg.GetKnowledgeId()}
	return knowledgeCall(ctx, req.Header(), h.knowledge.UpdateTeamCorpusHandler, http.MethodPut, req.Msg.GetTeamId(), "/knowledge/"+url.PathEscape(req.Msg.GetKnowledgeId()), body, vars, &teamsv1.KnowledgeEntry{})
}

func (h *connectHandler) DeleteKnowledge(ctx context.Context, req *connect.Request[teamsv1.DeleteKnowledgeRequest]) (*connect.Response[teamsv1.DeleteKnowledgeResponse], error) {
	if h.knowledge == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("knowledge handler is not configured"))
	}
	vars := map[string]string{"knowledgeId": req.Msg.GetKnowledgeId()}
	_, err := invokeKnowledge(ctx, req.Header(), h.knowledge.DeleteTeamCorpusHandler, http.MethodDelete, req.Msg.GetTeamId(), "/knowledge/"+url.PathEscape(req.Msg.GetKnowledgeId()), nil, vars)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&teamsv1.DeleteKnowledgeResponse{}), nil
}

func knowledgeCall[T any](ctx context.Context, headers http.Header, handler http.HandlerFunc, method, teamID, suffix string, body any, vars map[string]string, out *T) (*connect.Response[T], error) {
	result, err := invokeKnowledge(ctx, headers, handler, method, teamID, suffix, body, vars)
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

func invokeKnowledge(ctx context.Context, headers http.Header, handler http.HandlerFunc, method, teamID, suffix string, body any, vars map[string]string) (transportbridge.Result, error) {
	if vars == nil {
		vars = map[string]string{}
	}
	vars["id"] = teamID
	target := "/teams/" + url.PathEscape(teamID) + suffix
	return transportbridge.Invoke(ctx, headers, handler, method, target, body, vars)
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
