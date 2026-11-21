package handlers

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

// The compatibility procedures deliberately reuse the established REST
// handlers for authorization, validation, and orchestration until those
// operations have owner-specific typed adapters. This keeps the legacy route
// and Connect entrypoint behavior aligned during caller migration.

func (h *AgentManagerConnectHandler) EnsureProfile(ctx context.Context, req *connect.Request[apipb.EnsureProfileRequest]) (*connect.Response[apipb.EnsureProfileResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.EnsureProfileResponse{}
	if err := callRESTHandler(ctx, req.Header(), req.Peer().Addr, h.h.EnsureProfile, req.Msg, response); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) GetTask(ctx context.Context, req *connect.Request[apipb.GetTaskRequest]) (*connect.Response[apipb.GetTaskResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.GetTaskResponse{}
	if err := callRESTHandlerWithVars(ctx, req.Header(), req.Peer().Addr, h.h.GetTask, req.Msg, response, map[string]string{"id": req.Msg.GetTaskId()}); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) GetRunByTag(ctx context.Context, req *connect.Request[apipb.GetRunByTagRequest]) (*connect.Response[apipb.GetRunByTagResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.GetRunByTagResponse{}
	if err := callRESTHandlerWithVars(ctx, req.Header(), req.Peer().Addr, h.h.GetRunByTag, req.Msg, response, map[string]string{"tag": req.Msg.GetTag()}); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) GetRunDiff(ctx context.Context, req *connect.Request[apipb.GetRunDiffRequest]) (*connect.Response[apipb.GetRunDiffResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.GetRunDiffResponse{}
	if err := callRESTHandlerWithVars(ctx, req.Header(), req.Peer().Addr, h.h.GetRunDiff, req.Msg, response, map[string]string{"id": req.Msg.GetRunId()}); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) ApproveRun(ctx context.Context, req *connect.Request[apipb.ApproveRunRequest]) (*connect.Response[apipb.ApproveRunResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.ApproveRunResponse{}
	if err := callRESTHandlerWithVars(ctx, req.Header(), req.Peer().Addr, h.h.ApproveRun, req.Msg, response, map[string]string{"id": req.Msg.GetRunId()}); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) RejectRun(ctx context.Context, req *connect.Request[apipb.RejectRunRequest]) (*connect.Response[apipb.RejectRunResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.RejectRunResponse{}
	if err := callRESTHandlerWithVars(ctx, req.Header(), req.Peer().Addr, h.h.RejectRun, req.Msg, response, map[string]string{"id": req.Msg.GetRunId()}); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) StopRun(ctx context.Context, req *connect.Request[apipb.StopRunRequest]) (*connect.Response[apipb.StopRunResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.StopRunResponse{}
	if err := callRESTHandlerWithVars(ctx, req.Header(), req.Peer().Addr, h.h.StopRun, req.Msg, response, map[string]string{"id": req.Msg.GetRunId()}); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) StopRunByTag(ctx context.Context, req *connect.Request[apipb.StopRunByTagRequest]) (*connect.Response[apipb.StopRunByTagResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.StopRunByTagResponse{}
	if err := callRESTHandlerWithVars(ctx, req.Header(), req.Peer().Addr, h.h.StopRunByTag, req.Msg, response, map[string]string{"tag": req.Msg.GetTag()}); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) SyncRunFromSandbox(ctx context.Context, req *connect.Request[apipb.SyncRunFromSandboxRequest]) (*connect.Response[apipb.SyncRunFromSandboxResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.SyncRunFromSandboxResponse{}
	if err := callRESTHandlerWithVars(ctx, req.Header(), req.Peer().Addr, h.h.SyncRunFromSandbox, req.Msg, response, map[string]string{"id": req.Msg.GetRunId()}); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) ContinueRun(ctx context.Context, req *connect.Request[domainpb.ContinueRunRequest]) (*connect.Response[domainpb.ContinueRunResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &domainpb.ContinueRunResponse{}
	if err := callRESTHandlerWithVars(ctx, req.Header(), req.Peer().Addr, h.h.ContinueRun, req.Msg, response, map[string]string{"id": req.Msg.GetRunId()}); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) StopAllRuns(ctx context.Context, req *connect.Request[apipb.StopAllRunsRequest]) (*connect.Response[apipb.StopAllRunsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &apipb.StopAllRunsResponse{}
	if err := callRESTHandler(ctx, req.Header(), req.Peer().Addr, h.h.StopAllRuns, req.Msg, response); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) GetRunnerStatus(ctx context.Context, req *connect.Request[apipb.GetRunnerStatusRequest]) (*connect.Response[apipb.GetRunnerStatusResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	response := &apipb.GetRunnerStatusResponse{}
	if err := callRESTHandler(ctx, req.Header(), req.Peer().Addr, h.h.GetRunnerStatus, req.Msg, response); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func callRESTHandlerWithVars(ctx context.Context, header http.Header, peer string, handler http.HandlerFunc, request, response proto.Message, vars map[string]string) error {
	wrapped := func(w http.ResponseWriter, r *http.Request) {
		handler(w, mux.SetURLVars(r, vars))
	}
	return callRESTHandler(ctx, header, peer, wrapped, request, response)
}
