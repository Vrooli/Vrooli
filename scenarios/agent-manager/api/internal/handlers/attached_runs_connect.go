package handlers

import (
	"context"
	"errors"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// AttachRun returns the one-time identity for a locally launched harness.
// Loopback admission remains enforced by AgentOperationMiddleware; the two
// attribution headers remain transport metadata and are verified by the owner.
func (h *AgentManagerConnectHandler) AttachRun(ctx context.Context, req *connect.Request[apipb.AttachRunRequest]) (*connect.Response[apipb.AttachRunResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	var taskID *uuid.UUID
	if req.Msg.TaskId != nil {
		parsed, err := uuid.Parse(req.Msg.GetTaskId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		taskID = &parsed
	}
	processID := 0
	if req.Msg.ProcessId != nil {
		processID = int(req.Msg.GetProcessId())
	}
	attached, err := h.h.svc.AttachRun(ctx, orchestration.AttachRunRequest{
		TaskID:               taskID,
		HarnessKind:          req.Msg.GetHarnessKind(),
		HarnessSession:       req.Msg.GetHarnessSessionId(),
		ProcessID:            processID,
		HarnessTitle:         req.Msg.GetHarnessTitle(),
		HumanOwnerDelegation: req.Header().Get(cliutil.HumanOwnerDelegationHeader),
		WebConsoleSessionID:  req.Header().Get(cliutil.WebConsoleSessionIDHeader),
	})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	if attached == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("attach service returned no identity"))
	}
	return connect.NewResponse(&apipb.AttachRunResponse{
		Run:           protoconv.RunToProto(attached.Run),
		IdentityToken: attached.Token,
		ExpiresAt:     protoconv.TimestampToProto(attached.ExpiresAt),
	}), nil
}

// DetachRun closes the caller-selected attached run. AgentOperationMiddleware
// admits verified agent identities only when the request run ID matches their
// subject; the orchestration owner independently verifies the retained token.
func (h *AgentManagerConnectHandler) DetachRun(ctx context.Context, req *connect.Request[apipb.DetachRunRequest]) (*connect.Response[apipb.DetachRunResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	run, err := h.h.svc.DetachRun(ctx, id, req.Msg.GetReason())
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&apipb.DetachRunResponse{Run: protoconv.RunToProto(run)}), nil
}
