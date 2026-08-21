package agents

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	agentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents"
	agentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/agents/agents_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/agents"
)

type connectHandler struct {
	agentsconnect.UnimplementedAgentsServiceHandler
	legacy *domain.Handlers
}

func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return agentsconnect.NewAgentsServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) ListAgents(ctx context.Context, req *connect.Request[agentsv1.ListAgentsRequest]) (*connect.Response[agentsv1.ListAgentsResponse], error) {
	return transportbridge.InvokeWrappedJSON(ctx, req.Header(), h.legacy.List, http.MethodGet, "/agents", nil, nil, "agents", &agentsv1.ListAgentsResponse{})
}

func (h *connectHandler) GetAgent(ctx context.Context, req *connect.Request[agentsv1.GetAgentRequest]) (*connect.Response[agentsv1.Agent], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.Get, http.MethodGet, "/agents/"+url.PathEscape(req.Msg.GetId()), nil, map[string]string{"id": req.Msg.GetId()}, &agentsv1.Agent{})
}

func (h *connectHandler) CreateAgent(ctx context.Context, req *connect.Request[agentsv1.CreateAgentRequest]) (*connect.Response[agentsv1.Agent], error) {
	body, err := transportbridge.ProtoBody(req.Msg.GetAgent())
	if err != nil {
		return nil, err
	}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.Create, http.MethodPost, "/agents", body, nil, &agentsv1.Agent{})
}

func (h *connectHandler) UpdateAgent(ctx context.Context, req *connect.Request[agentsv1.UpdateAgentRequest]) (*connect.Response[agentsv1.Agent], error) {
	body, err := transportbridge.MaskedBody(req.Msg.GetAgent(), req.Msg.GetUpdateMask().GetPaths())
	if err != nil {
		return nil, err
	}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.Update, http.MethodPut, "/agents/"+url.PathEscape(req.Msg.GetId()), body, map[string]string{"id": req.Msg.GetId()}, &agentsv1.Agent{})
}

func (h *connectHandler) DeleteAgent(ctx context.Context, req *connect.Request[agentsv1.DeleteAgentRequest]) (*connect.Response[agentsv1.DeleteAgentResponse], error) {
	_, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.Delete, http.MethodDelete, "/agents/"+url.PathEscape(req.Msg.GetId()), nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&agentsv1.DeleteAgentResponse{}), nil
}

func (h *connectHandler) ListAgentTeams(ctx context.Context, req *connect.Request[agentsv1.ListAgentTeamsRequest]) (*connect.Response[agentsv1.ListAgentTeamsResponse], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.ListTeams, http.MethodGet, "/agents/"+url.PathEscape(req.Msg.GetId())+"/teams", nil, map[string]string{"id": req.Msg.GetId()}, &agentsv1.ListAgentTeamsResponse{})
}

func (h *connectHandler) GetSoul(ctx context.Context, req *connect.Request[agentsv1.GetSoulRequest]) (*connect.Response[agentsv1.Soul], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.GetSoul, http.MethodGet, "/agents/"+url.PathEscape(req.Msg.GetId())+"/soul", nil, map[string]string{"id": req.Msg.GetId()}, &agentsv1.Soul{})
}

func (h *connectHandler) SetSoul(ctx context.Context, req *connect.Request[agentsv1.SetSoulRequest]) (*connect.Response[agentsv1.Soul], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.SetSoul, http.MethodPut, "/agents/"+url.PathEscape(req.Msg.GetId())+"/soul", map[string]string{"content": req.Msg.GetContent()}, map[string]string{"id": req.Msg.GetId()}, &agentsv1.Soul{})
}

func (h *connectHandler) ManageSoul(ctx context.Context, req *connect.Request[agentsv1.ManageSoulRequest]) (*connect.Response[agentsv1.Soul], error) {
	if req.Msg.Content == nil {
		return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.GetSoul, http.MethodGet, "/agents/"+url.PathEscape(req.Msg.GetId())+"/soul", nil, map[string]string{"id": req.Msg.GetId()}, &agentsv1.Soul{})
	}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.SetSoul, http.MethodPut, "/agents/"+url.PathEscape(req.Msg.GetId())+"/soul", map[string]string{"content": req.Msg.GetContent()}, map[string]string{"id": req.Msg.GetId()}, &agentsv1.Soul{})
}

func (h *connectHandler) ListFiles(ctx context.Context, req *connect.Request[agentsv1.ListFilesRequest]) (*connect.Response[agentsv1.ListFilesResponse], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.ListFiles, http.MethodGet, "/agents/"+url.PathEscape(req.Msg.GetId())+"/files", nil, map[string]string{"id": req.Msg.GetId()}, &agentsv1.ListFilesResponse{})
}

func (h *connectHandler) GetFile(ctx context.Context, req *connect.Request[agentsv1.GetFileRequest]) (*connect.Response[agentsv1.FileContent], error) {
	target := "/agents/" + url.PathEscape(req.Msg.GetId()) + "/files/content?path=" + url.QueryEscape(req.Msg.GetPath())
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.GetFile, http.MethodGet, target, nil, map[string]string{"id": req.Msg.GetId()}, &agentsv1.FileContent{})
}

func (h *connectHandler) SetFile(ctx context.Context, req *connect.Request[agentsv1.SetFileRequest]) (*connect.Response[agentsv1.FileContent], error) {
	target := "/agents/" + url.PathEscape(req.Msg.GetId()) + "/files/content?path=" + url.QueryEscape(req.Msg.GetPath())
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.SetFile, http.MethodPut, target, map[string]string{"content": req.Msg.GetContent()}, map[string]string{"id": req.Msg.GetId()}, &agentsv1.FileContent{})
}

func (h *connectHandler) CreateFile(ctx context.Context, req *connect.Request[agentsv1.CreateFileRequest]) (*connect.Response[agentsv1.FileContent], error) {
	body := map[string]any{"path": req.Msg.GetPath(), "content": req.Msg.GetContent(), "isDir": req.Msg.GetIsDir()}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.CreateFile, http.MethodPost, "/agents/"+url.PathEscape(req.Msg.GetId())+"/files", body, map[string]string{"id": req.Msg.GetId()}, &agentsv1.FileContent{})
}

func (h *connectHandler) RenameFile(ctx context.Context, req *connect.Request[agentsv1.RenameFileRequest]) (*connect.Response[agentsv1.FileContent], error) {
	body := map[string]string{"from": req.Msg.GetFrom(), "to": req.Msg.GetTo()}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.RenameFile, http.MethodPost, "/agents/"+url.PathEscape(req.Msg.GetId())+"/files/rename", body, map[string]string{"id": req.Msg.GetId()}, &agentsv1.FileContent{})
}

func (h *connectHandler) DeleteFile(ctx context.Context, req *connect.Request[agentsv1.DeleteFileRequest]) (*connect.Response[agentsv1.DeleteFileResponse], error) {
	target := "/agents/" + url.PathEscape(req.Msg.GetId()) + "/files?path=" + url.QueryEscape(req.Msg.GetPath())
	_, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.DeleteFile, http.MethodDelete, target, nil, map[string]string{"id": req.Msg.GetId()})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&agentsv1.DeleteFileResponse{}), nil
}
