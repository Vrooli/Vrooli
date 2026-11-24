package agentmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type connectFixture struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	start *apipb.StartWorkflowExecutionRequest
	result *apipb.GetWorkflowExecutionResultRequest
}
func(f *connectFixture)StartWorkflowExecution(_ context.Context,r *connect.Request[apipb.StartWorkflowExecutionRequest])(*connect.Response[apipb.WorkflowExecutionResponse],error){f.start=r.Msg;return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution:&domainpb.WorkflowExecution{Id:"execution-1"}}),nil}
func(f *connectFixture)GetWorkflowExecutionResult(_ context.Context,r *connect.Request[apipb.GetWorkflowExecutionResultRequest])(*connect.Response[apipb.WorkflowExecutionResponse],error){f.result=r.Msg;return connect.NewResponse(&apipb.WorkflowExecutionResponse{Execution:&domainpb.WorkflowExecution{Id:r.Msg.ExecutionId}}),nil}

func TestHTTPClientUsesGeneratedConnectAndExplicitResultAuthorization(t *testing.T){
	f:=&connectFixture{};path,handler:=apiconnect.NewAgentManagerServiceHandler(f);mux:=http.NewServeMux();mux.Handle(path,handler);server:=httptest.NewServer(mux);defer server.Close()
	c:=&HTTPClient{Resolver:func(context.Context)(string,error){return server.URL,nil},HTTP:server.Client()}
	started,err:=c.Start(t.Context(),&apipb.StartWorkflowExecutionRequest{Owner:"web-search",WorkflowKey:"web-search/research",IdempotencyKey:"key"});if err!=nil{t.Fatal(err)}
	if started.GetId()!="execution-1"||f.start.GetOwner()!="web-search"||f.start.GetWorkflowKey()!="web-search/research"{t.Fatalf("start=%+v request=%+v",started,f.start)}
	result,err:=c.Result(t.Context(),"execution-1");if err!=nil{t.Fatal(err)}
	if result.GetId()!="execution-1"||!f.result.GetExplicitlyAuthorized(){t.Fatalf("result=%+v request=%+v",result,f.result)}
}
