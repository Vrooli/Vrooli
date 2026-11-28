package dispatch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"switchboard/internal/channels"
	"switchboard/internal/threads"
)

func TestAgentManagerRunnerCreatesTaskAndRun(t *testing.T) {
	var authorization []string
	var runRequest *apipb.CreateRunRequest
	mux := http.NewServeMux()
	mux.Handle(apiconnect.AgentManagerServiceCreateTaskProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateTaskProcedure, func(_ context.Context, req *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
		authorization = append(authorization, req.Header().Get("Authorization"))
		require.Equal(t, "Switchboard conversation", req.Msg.GetTask().GetTitle())
		return connect.NewResponse(&apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}}), nil
	}))
	mux.Handle(apiconnect.AgentManagerServiceCreateRunProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateRunProcedure, func(_ context.Context, req *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
		authorization = append(authorization, req.Header().Get("Authorization"))
		runRequest = req.Msg
		return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}}), nil
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	run, err := (AgentManagerRunner{BaseURL: server.URL, Identity: fixtureSwitcherOwner{}}).Run(WithAuthorization(context.Background(), "Bearer fixture-owner"), "agent-1", []string{"read"}, "hello")
	require.NoError(t, err)
	require.Equal(t, "agent-manager run accepted: run-1", run)
	require.Equal(t, []string{"Bearer fixture-owner", "Bearer fixture-owner"}, authorization)
	require.Equal(t, "task-1", runRequest.GetTaskId())
	require.Equal(t, "agent-1", runRequest.GetProfileRef().GetProfileKey())
	require.Equal(t, []string{"read"}, runRequest.GetInlineConfig().GetAllowedTools())
	require.True(t, runRequest.GetInlineConfig().GetClearAllowedTools())
}

func TestAgentManagerRunnerForwardsRequestBearerWithoutPersistingIt(t *testing.T) {
	var authorization []string
	mux := http.NewServeMux()
	mux.Handle(apiconnect.AgentManagerServiceCreateTaskProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateTaskProcedure, func(_ context.Context, req *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
		authorization = append(authorization, req.Header().Get("Authorization"))
		return connect.NewResponse(&apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}}), nil
	}))
	mux.Handle(apiconnect.AgentManagerServiceCreateRunProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateRunProcedure, func(_ context.Context, req *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
		authorization = append(authorization, req.Header().Get("Authorization"))
		return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}}), nil
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	ctx := WithAuthorization(context.Background(), "Bearer owner-token")
	_, err := (AgentManagerRunner{BaseURL: server.URL, Identity: fixtureSwitcherOwner{}}).Run(ctx, "agent-1", []string{"read"}, "hello")
	require.NoError(t, err)
	require.Equal(t, []string{"Bearer owner-token", "Bearer owner-token"}, authorization)
}

func TestAgentManagerRunnerFailsClosedOnMalformedResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(apiconnect.AgentManagerServiceCreateTaskProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateTaskProcedure, func(context.Context, *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
		return connect.NewResponse(&apipb.CreateTaskResponse{}), nil
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	_, err := (AgentManagerRunner{BaseURL: server.URL, Identity: fixtureSwitcherOwner{}}).Run(WithAuthorization(context.Background(), "Bearer fixture-owner"), "agent-1", nil, "hello")
	require.ErrorContains(t, err, "no id")
}

func TestAgentManagerRunnerContinuesDurableThreadRun(t *testing.T) {
	var methods []string
	mux := http.NewServeMux()
	mux.Handle(apiconnect.AgentManagerServiceCreateTaskProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateTaskProcedure, func(context.Context, *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
		methods = append(methods, "CreateTask")
		return connect.NewResponse(&apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}}), nil
	}))
	mux.Handle(apiconnect.AgentManagerServiceCreateRunProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateRunProcedure, func(context.Context, *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
		methods = append(methods, "CreateRun")
		return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}}), nil
	}))
	mux.Handle(apiconnect.AgentManagerServiceGetRunEventsProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceGetRunEventsProcedure, func(_ context.Context, req *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
		methods = append(methods, "GetRunEvents")
		require.Equal(t, "run-1", req.Msg.GetRunId())
		return connect.NewResponse(&apipb.GetRunEventsResponse{}), nil
	}))
	mux.Handle(apiconnect.AgentManagerServiceContinueRunProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceContinueRunProcedure, func(_ context.Context, req *connect.Request[domainpb.ContinueRunRequest]) (*connect.Response[domainpb.ContinueRunResponse], error) {
		methods = append(methods, "ContinueRun")
		require.Equal(t, "second", req.Msg.GetMessage())
		return connect.NewResponse(&domainpb.ContinueRunResponse{Success: true}), nil
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(threads.Schema)))
	store := threads.NewStore(database)
	runner := AgentManagerRunner{BaseURL: server.URL, Identity: fixtureSwitcherOwner{}, Threads: store}
	require.NoError(t, func() error {
		_, err := runner.RunConversation(WithAuthorization(context.Background(), "Bearer fixture-owner"), "agent-1", []string{"read"}, "telegram", "chat-1", "first")
		return err
	}())
	require.NoError(t, func() error {
		_, err := runner.RunConversation(WithAuthorization(context.Background(), "Bearer fixture-owner"), "agent-1", []string{"read"}, "telegram", "chat-1", "second")
		return err
	}())
	require.Equal(t, []string{"CreateTask", "CreateRun", "GetRunEvents", "ContinueRun"}, methods)
}

func TestAgentManagerRunnerDeliversAssistantEventOnSameThread(t *testing.T) {
	replies := make(chan channels.Outbound, 1)
	mux := http.NewServeMux()
	mux.Handle(apiconnect.AgentManagerServiceCreateTaskProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateTaskProcedure, func(context.Context, *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
		return connect.NewResponse(&apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}}), nil
	}))
	mux.Handle(apiconnect.AgentManagerServiceCreateRunProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateRunProcedure, func(context.Context, *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
		return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}}), nil
	}))
	mux.Handle(apiconnect.AgentManagerServiceGetRunEventsProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceGetRunEventsProcedure, func(context.Context, *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
		return connect.NewResponse(&apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{{Sequence: 1, Data: &domainpb.RunEvent_Message{Message: &domainpb.MessageEventData{Role: "assistant", Content: "hello from the agent"}}}}}), nil
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	runner := AgentManagerRunner{BaseURL: server.URL, Identity: fixtureSwitcherOwner{}, Wait: time.Second, Send: func(ctx context.Context, out channels.Outbound) error {
		replies <- out
		return nil
	}}
	reply, err := runner.RunConversation(WithAuthorization(context.Background(), "Bearer fixture-owner"), "agent-1", []string{"read"}, "telegram", "chat-1", "hello")
	require.NoError(t, err)
	require.Empty(t, reply)
	select {
	case out := <-replies:
		require.Equal(t, "telegram", out.ChannelID)
		require.Equal(t, "chat-1", out.ThreadKey)
		require.Equal(t, "hello from the agent", out.Text)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for assistant reply")
	}
}

func TestAgentManagerRunnerMapsApprovalPendingAsDefinitive(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(apiconnect.AgentManagerServiceCreateTaskProcedure, connect.NewUnaryHandler(apiconnect.AgentManagerServiceCreateTaskProcedure, func(context.Context, *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
		detail, err := connect.NewErrorDetail(&apipb.AuthorizationApprovalPending{Subject: "switchboard", Detail: "operator approval required"})
		require.NoError(t, err)
		connectErr := connect.NewError(connect.CodeFailedPrecondition, errors.New("approval pending"))
		connectErr.AddDetail(detail)
		return nil, connectErr
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	_, err := (AgentManagerRunner{BaseURL: server.URL, Identity: fixtureSwitcherOwner{}}).Run(WithAuthorization(context.Background(), "Bearer fixture-owner"), "agent-1", nil, "hello")
	require.ErrorIs(t, err, ErrAuthorizationApprovalPending)
}

func TestAgentManagerRunnerBlocksAuthorizationRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("authorization-bearing Connect request followed a redirect")
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == apiconnect.AgentManagerServiceCreateTaskProcedure {
			http.Redirect(w, r, target.URL+apiconnect.AgentManagerServiceCreateTaskProcedure, http.StatusTemporaryRedirect)
			return
		}
		http.NotFound(w, r)
	}))
	defer redirect.Close()
	_, err := (AgentManagerRunner{BaseURL: redirect.URL, Identity: fixtureSwitcherOwner{}}).Run(WithAuthorization(context.Background(), "Bearer fixture-owner"), "agent-1", nil, "hello")
	require.Error(t, err)
}
