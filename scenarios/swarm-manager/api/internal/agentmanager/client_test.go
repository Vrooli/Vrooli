package agentmanager

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type getRunHandler struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
}

func (getRunHandler) GetRun(context.Context, *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	return connect.NewResponse(&apipb.GetRunResponse{Run: &domainpb.Run{Id: "run-1"}}), nil
}

type createRunHandler struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	authorization, identity string
	err                     error
}

func (h *createRunHandler) CreateRun(_ context.Context, r *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
	h.authorization = r.Header().Get("Authorization")
	h.identity = r.Header().Get("X-Agent-Identity-Token")
	if h.err != nil {
		return nil, h.err
	}
	return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}}), nil
}

func TestAgentManagerClientUsesGeneratedConnectProcedure(t *testing.T) {
	path, handler := apiconnect.NewAgentManagerServiceHandler(getRunHandler{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	run, err := client.GetRun(context.Background(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if run.GetId() != "run-1" {
		t.Fatalf("run id = %q", run.GetId())
	}
}

func TestCreateRunPreservesTypedAuthorizationApprovalDetail(t *testing.T) {
	detail := &apipb.AuthorizationApprovalPending{Subject: "swarm-manager", Detail: "standing approval required"}
	connectDetail, err := connect.NewErrorDetail(detail)
	if err != nil {
		t.Fatal(err)
	}
	connectErr := connect.NewError(connect.CodeFailedPrecondition, errors.New("approval pending"))
	connectErr.AddDetail(connectDetail)
	h := &createRunHandler{err: connectErr}
	path, handler := apiconnect.NewAgentManagerServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	_, err = client.CreateRun(auth01FixtureCaller(t, context.Background()), &apipb.CreateRunRequest{TaskId: "fixture-task"})
	var pending *ApprovalPendingError
	if !errors.As(err, &pending) || pending.Detail.GetSubject() != detail.Subject || pending.Detail.GetDetail() != detail.Detail {
		t.Fatalf("typed approval error = %#v (%v)", pending, err)
	}
}

func TestCreateRunUsesGeneratedConnectAndCarriesOnlyCallerProofHeaders(t *testing.T) {
	h := &createRunHandler{}
	path, handler := apiconnect.NewAgentManagerServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	run, err := client.CreateRun(auth01FixtureCaller(t, context.Background()), &apipb.CreateRunRequest{TaskId: "fixture-task"})
	if err != nil {
		t.Fatal(err)
	}
	if run.GetId() != "run-1" {
		t.Fatalf("run id = %q", run.GetId())
	}
	if h.authorization != "Bearer fixture-owner" || h.identity != "" {
		t.Fatalf("proof headers authorization=%q identity=%q", h.authorization, h.identity)
	}
}

func TestAgentManagerConnectErrorsAreClassified(t *testing.T) {
	err := connectError(connect.NewError(connect.CodeUnavailable, context.DeadlineExceeded))
	if err == nil || !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("unavailable error = %v", err)
	}
	err = connectError(connect.NewError(connect.CodeInvalidArgument, context.Canceled))
	if err == nil || !errors.Is(err, ErrRequestFailed) {
		t.Fatalf("request error = %v", err)
	}
}

func TestContinueRunWithIdempotencyKeyRequiresKey(t *testing.T) {
	client := NewHTTPClientWithResolver(func(context.Context) (string, error) { return "http://unused", nil }, nil)
	err := client.ContinueRunWithIdempotencyKey(context.Background(), "run-1", "continue", " ")
	if err == nil || !errors.Is(err, ErrRequestFailed) {
		t.Fatalf("error = %v", err)
	}
}
