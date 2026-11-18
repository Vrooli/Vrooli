package handlers

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/proto"
)

type attachedRunConnectStub struct {
	orchestration.RunService
	attachRequest orchestration.AttachRunRequest
	detachID      uuid.UUID
	detachReason  string
	attached      *orchestration.AttachRunResult
	detached      *domain.Run
}

func (s *attachedRunConnectStub) AttachRun(_ context.Context, req orchestration.AttachRunRequest) (*orchestration.AttachRunResult, error) {
	s.attachRequest = req
	return s.attached, nil
}

func (s *attachedRunConnectStub) DetachRun(_ context.Context, id uuid.UUID, reason string) (*domain.Run, error) {
	s.detachID = id
	s.detachReason = reason
	return s.detached, nil
}

func TestAttachedRunConnectPreservesOneTimeIdentityAndDelegationHeaders(t *testing.T) {
	runID := uuid.New()
	taskID := uuid.New()
	expiresAt := time.Date(2026, 11, 16, 21, 0, 0, 0, time.UTC)
	service := &attachedRunConnectStub{attached: &orchestration.AttachRunResult{
		Run:       &domain.Run{ID: runID, HarnessKind: "codex", HarnessSessionID: "session-42"},
		Token:     "one-time-identity-secret",
		ExpiresAt: expiresAt,
	}}
	owner := New(orchestration.HandlerServices{RunService: service})
	connectHandler := NewAgentManagerConnectHandler(owner)
	_, handler := apiconnect.NewAgentManagerServiceHandler(connectHandler)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)

	request := connect.NewRequest(&apipb.AttachRunRequest{
		TaskId:           proto.String(taskID.String()),
		HarnessKind:      "codex",
		HarnessSessionId: "session-42",
		ProcessId:        proto.Int32(4242),
		HarnessTitle:     proto.String("coding session"),
	})
	request.Header().Set(cliutil.HumanOwnerDelegationHeader, "verified-owner-delegation")
	request.Header().Set(cliutil.WebConsoleSessionIDHeader, "web-console-session")
	response, err := client.AttachRun(t.Context(), request)
	if err != nil {
		t.Fatalf("AttachRun: %v", err)
	}
	if response.Msg.GetRun().GetId() != runID.String() || response.Msg.GetIdentityToken() != "one-time-identity-secret" || !response.Msg.GetExpiresAt().AsTime().Equal(expiresAt) {
		t.Fatalf("AttachRun response = %v", response.Msg)
	}
	if service.attachRequest.TaskID == nil || *service.attachRequest.TaskID != taskID || service.attachRequest.ProcessID != 4242 || service.attachRequest.HarnessTitle != "coding session" {
		t.Fatalf("AttachRun metadata = %+v", service.attachRequest)
	}
	if service.attachRequest.HumanOwnerDelegation != "verified-owner-delegation" || service.attachRequest.WebConsoleSessionID != "web-console-session" {
		t.Fatalf("AttachRun owner metadata = %+v", service.attachRequest)
	}
}

func TestDetachRunConnectUsesPathTargetAndReason(t *testing.T) {
	runID := uuid.New()
	service := &attachedRunConnectStub{detached: &domain.Run{ID: runID, Status: domain.RunStatusComplete}}
	owner := New(orchestration.HandlerServices{RunService: service})
	connectHandler := NewAgentManagerConnectHandler(owner)
	_, handler := apiconnect.NewAgentManagerServiceHandler(connectHandler)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)

	response, err := client.DetachRun(t.Context(), connect.NewRequest(&apipb.DetachRunRequest{RunId: runID.String(), Reason: proto.String("coding agent exited")}))
	if err != nil {
		t.Fatalf("DetachRun: %v", err)
	}
	if response.Msg.GetRun().GetId() != runID.String() || service.detachID != runID || service.detachReason != "coding agent exited" {
		t.Fatalf("DetachRun response=%v service target=%s reason=%q", response.Msg, service.detachID, service.detachReason)
	}
}
