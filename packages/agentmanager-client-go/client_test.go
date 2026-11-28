package agentmanagerclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type testResolver string

func (r testResolver) ResolveBaseURL(context.Context) (string, error) {
	return string(r), nil
}

type testManager struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
}

func (testManager) Health(context.Context, *connect.Request[apipb.HealthRequest]) (*connect.Response[apipb.HealthResponse], error) {
	return connect.NewResponse(&apipb.HealthResponse{Readiness: true}), nil
}

func (testManager) CreateTask(_ context.Context, req *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
	task := req.Msg.Task
	task.Id = "task-1"
	return connect.NewResponse(&apipb.CreateTaskResponse{Task: task}), nil
}

func (testManager) CreateRun(_ context.Context, req *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
	return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1", Tag: req.Msg.GetTag()}}), nil
}

func (testManager) GetRun(_ context.Context, req *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	return connect.NewResponse(&apipb.GetRunResponse{Run: &domainpb.Run{Id: req.Msg.RunId}}), nil
}

func TestClientConnectsAndRunsSharedProcedures(t *testing.T) {
	path, handler := apiconnect.NewAgentManagerServiceHandler(testManager{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := NewWithResolver(time.Second, testResolver(server.URL+"/"))
	service, err := client.Connect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ready, err := service.Health(context.Background())
	if err != nil || !ready {
		t.Fatalf("Health() = %v, %v", ready, err)
	}
	task, err := service.CreateTask(context.Background(), &domainpb.Task{Title: "task"})
	if err != nil || task.Id != "task-1" {
		t.Fatalf("CreateTask() = %+v, %v", task, err)
	}
	run, err := service.CreateRun(context.Background(), &apipb.CreateRunRequest{TaskId: task.Id, Tag: stringPointer("content-desk:draft:draft")})
	if err != nil || run.Id != "run-1" || run.Tag != "content-desk:draft:draft" {
		t.Fatalf("CreateRun() = %+v, %v", run, err)
	}
	gotRun, err := service.GetRun(context.Background(), run.Id)
	if err != nil || gotRun.Id != run.Id {
		t.Fatalf("GetRun() = %+v, %v", gotRun, err)
	}
}

func stringPointer(value string) *string { return &value }
