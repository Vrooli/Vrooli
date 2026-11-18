package integrations

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"agent-inbox/resilience"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type testMultipartFile struct{ *bytes.Reader }

func (testMultipartFile) Close() error { return nil }

type inboxAgentManagerConnectStub struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	t *testing.T
}

func TestCallAgentManagerPreservesPreCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &AgentManagerClient{retryCfg: resilience.RetryConfig{MaxAttempts: 3}}
	_, err := callAgentManager(ctx, c, true, func(context.Context, apiconnect.AgentManagerServiceClient) (*string, error) {
		t.Fatal("cancelled operation invoked RPC")
		return nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled operation error = %v, want context.Canceled", err)
	}
}

func TestCallAgentManagerDoesNotRetryWritesWithoutIdempotency(t *testing.T) {
	calls := 0
	c := &AgentManagerClient{
		baseURL:    "http://127.0.0.1:1",
		httpClient: http.DefaultClient,
		retryCfg:   resilience.RetryConfig{MaxAttempts: 3},
	}
	_, err := callAgentManager(context.Background(), c, false, func(context.Context, apiconnect.AgentManagerServiceClient) (*string, error) {
		calls++
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("response lost"))
	})
	if err == nil || calls != 1 {
		t.Fatalf("write retry calls = %d, error = %v; want one attempt and an error", calls, err)
	}
}

func TestCallAgentManagerCountsDeadlineAsCircuitFailure(t *testing.T) {
	breaker := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{FailureThreshold: 1})
	c := &AgentManagerClient{
		baseURL:    "http://127.0.0.1:1",
		httpClient: http.DefaultClient,
		retryCfg:   resilience.RetryConfig{MaxAttempts: 1},
		cb:         breaker,
	}
	_, err := callAgentManager(context.Background(), c, true, func(context.Context, apiconnect.AgentManagerServiceClient) (*string, error) {
		return nil, connect.NewError(connect.CodeDeadlineExceeded, errors.New("peer timed out"))
	})
	if connect.CodeOf(err) != connect.CodeDeadlineExceeded {
		t.Fatalf("deadline error = %v", err)
	}
	if got := breaker.State(); got != resilience.StateOpen {
		t.Fatalf("breaker state = %s, want open after deadline", got)
	}
}

func TestCallAgentManagerZeroRetriesStillMakesInitialRead(t *testing.T) {
	calls := 0
	c := &AgentManagerClient{
		baseURL:    "http://127.0.0.1:1",
		httpClient: http.DefaultClient,
		retryCfg:   resilience.RetryConfig{MaxAttempts: 0},
	}
	_, err := callAgentManager(context.Background(), c, true, func(context.Context, apiconnect.AgentManagerServiceClient) (*string, error) {
		calls++
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("peer unavailable"))
	})
	if err == nil || calls != 1 {
		t.Fatalf("zero-retry read calls = %d, error = %v; want one initial call and an error", calls, err)
	}
}

func TestAgentManagerClientUsesFreshSharedPeerResolution(t *testing.T) {
	testServer := httptest.NewServer(http.NotFoundHandler())
	defer testServer.Close()
	server, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(server.Port())
	if err != nil {
		t.Fatal(err)
	}
	lookups := 0
	resolver := discovery.NewResolver(discovery.ResolverConfig{
		CacheTTL: -1,
		CommandRunner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			lookups++
			if name != "vrooli" || strings.Join(args, " ") != "scenario port agent-manager API_PORT" {
				t.Fatalf("unexpected shared discovery command: %s %v", name, args)
			}
			return []byte(strconv.Itoa(port)), nil
		},
	})
	c := &AgentManagerClient{resolver: resolver}
	for i := 0; i < 2; i++ {
		baseURL, err := c.getBaseURL(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if baseURL != "http://localhost:"+strconv.Itoa(port) {
			t.Fatalf("resolved URL = %q", baseURL)
		}
	}
	if lookups != 2 {
		t.Fatalf("discovery lookups = %d, want one fresh lookup per operation", lookups)
	}
}

func (s inboxAgentManagerConnectStub) CreateTask(_ context.Context, req *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
	if req.Msg.GetTask().GetTitle() != "Agent Chat Session" {
		s.t.Errorf("task title = %q", req.Msg.GetTask().GetTitle())
	}
	return connect.NewResponse(&apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-id"}}), nil
}
func (s inboxAgentManagerConnectStub) CreateRun(_ context.Context, req *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
	if req.Msg.GetTaskId() != "task-id" {
		s.t.Errorf("task id = %q", req.Msg.GetTaskId())
	}
	if req.Msg.RunMode != nil {
		s.t.Errorf("run mode override = %s, want profile default", req.Msg.GetRunMode())
	}
	if got := req.Msg.GetProfileRef().GetProfileKey(); got != "agent-inbox/default" {
		s.t.Errorf("profile key = %q", got)
	}
	return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-id", SessionId: "session-id"}}), nil
}
func (s inboxAgentManagerConnectStub) GetRun(_ context.Context, req *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	return connect.NewResponse(&apipb.GetRunResponse{Run: &domainpb.Run{Id: req.Msg.GetRunId(), Status: domainpb.RunStatus_RUN_STATUS_RUNNING}}), nil
}
func (s inboxAgentManagerConnectStub) GetRunEvents(_ context.Context, req *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
	if req.Msg.GetAfterSequence() != 7 {
		s.t.Errorf("after sequence = %d", req.Msg.GetAfterSequence())
	}
	return connect.NewResponse(&apipb.GetRunEventsResponse{}), nil
}
func (s inboxAgentManagerConnectStub) ListRuns(_ context.Context, req *connect.Request[apipb.ListRunsRequest]) (*connect.Response[apipb.ListRunsResponse], error) {
	if req.Msg.GetTagPrefix() != "inbox" {
		s.t.Errorf("tag prefix = %q", req.Msg.GetTagPrefix())
	}
	return connect.NewResponse(&apipb.ListRunsResponse{Total: 1, Runs: []*domainpb.Run{{Id: "run-id", Status: domainpb.RunStatus_RUN_STATUS_RUNNING}}}), nil
}
func (s inboxAgentManagerConnectStub) StopRun(_ context.Context, req *connect.Request[apipb.StopRunRequest]) (*connect.Response[apipb.StopRunResponse], error) {
	return connect.NewResponse(&apipb.StopRunResponse{Status: req.Msg.GetRunId()}), nil
}
func (s inboxAgentManagerConnectStub) ContinueRun(_ context.Context, req *connect.Request[domainpb.ContinueRunRequest]) (*connect.Response[domainpb.ContinueRunResponse], error) {
	if req.Msg.GetMessage() != "next" {
		s.t.Errorf("message = %q", req.Msg.GetMessage())
	}
	return connect.NewResponse(&domainpb.ContinueRunResponse{Success: true}), nil
}

func TestAgentManagerClientUsesGeneratedConnectForProtoProcedures(t *testing.T) {
	path, handler := apiconnect.NewAgentManagerServiceHandler(inboxAgentManagerConnectStub{t: t})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path || len(r.URL.Path) > len(path) {
			handler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := &AgentManagerClient{baseURL: server.URL, httpClient: server.Client(), retryCfg: resilience.RetryConfig{MaxAttempts: 1}}
	ctx := context.Background()
	chat, err := c.StartAgentChat(ctx, "prompt", AgentChatConfig{ProjectPath: "/repo"})
	if err != nil || chat.TaskID != "task-id" || chat.RunID != "run-id" || chat.SessionID != "session-id" {
		t.Fatalf("start chat = %+v, %v", chat, err)
	}
	if _, err := c.CheckAgentStatus(ctx, "run-id"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetEvents(ctx, "run-id", 7); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetRunStatus(ctx, "run-id"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ListRuns(ctx, ListRunsOptions{TagPrefix: "inbox", Limit: 10, Offset: 0}); err != nil {
		t.Fatal(err)
	}
	if err := c.StopRun(ctx, "run-id"); err != nil {
		t.Fatal(err)
	}
	if err := c.ContinueChat(ctx, "run-id", "next", nil); err != nil {
		t.Fatal(err)
	}
}

func TestAgentManagerAttachmentUploadRemainsMultipartREST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/attachments/upload" {
			t.Errorf("request route = %s %s", r.Method, r.URL.Path)
		}
		if _, _, err := r.FormFile("file"); err != nil {
			t.Errorf("multipart file part: %v", err)
		}
		_, _ = io.WriteString(w, `{"id":"attachment-1","file_name":"note.txt","content_type":"text/plain","file_size":4}`)
	}))
	defer server.Close()
	c := &AgentManagerClient{baseURL: server.URL, httpClient: server.Client()}
	got, err := c.UploadAttachment(context.Background(), testMultipartFile{bytes.NewReader([]byte("note"))}, &multipart.FileHeader{Filename: "note.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "attachment-1" || got.FileName != "note.txt" || got.FileSize != 4 {
		t.Fatalf("upload response = %+v", got)
	}
}
