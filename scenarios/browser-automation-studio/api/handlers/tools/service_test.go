package tools

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/internal/toolexecution"
	agentinboxpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	toolsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/tools"
	toolsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/tools/toolsconnect"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeRegistry struct {
	manifest *agentinboxpb.ToolManifest
	tools    map[string]*agentinboxpb.ToolDefinition
}

func (f *fakeRegistry) GetManifest(context.Context) *agentinboxpb.ToolManifest {
	return f.manifest
}

func (f *fakeRegistry) GetTool(_ context.Context, name string) *agentinboxpb.ToolDefinition {
	return f.tools[name]
}

type fakeExecutor struct {
	lastTool string
	lastArgs map[string]interface{}
	result   *toolexecution.ExecutionResult
	err      error
}

func (f *fakeExecutor) Execute(_ context.Context, toolName string, args map[string]interface{}) (*toolexecution.ExecutionResult, error) {
	f.lastTool = toolName
	f.lastArgs = args
	return f.result, f.err
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

func newTestClient(t *testing.T, reg *fakeRegistry, exec *fakeExecutor) toolsconnect.ToolsServiceClient {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	mount := Module(Deps{Registry: reg, Executor: exec, Logger: logger})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return toolsconnect.NewToolsServiceClient(srv.Client(), srv.URL)
}

func TestList_ReturnsManifest(t *testing.T) {
	reg := &fakeRegistry{manifest: &agentinboxpb.ToolManifest{
		ProtocolVersion: "1.0",
		Scenario:        &agentinboxpb.ScenarioInfo{Name: "browser-automation-studio"},
		Tools:           []*agentinboxpb.ToolDefinition{{Name: "execute_workflow"}},
	}}
	client := newTestClient(t, reg, &fakeExecutor{})

	resp, err := client.List(context.Background(), connect.NewRequest(&toolsv1.ListToolsRequest{}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.Manifest)
	require.Equal(t, "1.0", resp.Msg.Manifest.ProtocolVersion)
	require.Len(t, resp.Msg.Manifest.Tools, 1)
	require.Equal(t, "execute_workflow", resp.Msg.Manifest.Tools[0].Name)
}

func TestGet_HappyPath(t *testing.T) {
	reg := &fakeRegistry{tools: map[string]*agentinboxpb.ToolDefinition{
		"execute_workflow": {Name: "execute_workflow", Description: "Run a workflow"},
	}}
	client := newTestClient(t, reg, &fakeExecutor{})

	resp, err := client.Get(context.Background(), connect.NewRequest(&toolsv1.GetToolRequest{Name: "execute_workflow"}))
	require.NoError(t, err)
	require.Equal(t, "execute_workflow", resp.Msg.Tool.Name)
	require.Equal(t, "Run a workflow", resp.Msg.Tool.Description)
}

func TestGet_EmptyNameRejected(t *testing.T) {
	client := newTestClient(t, &fakeRegistry{}, &fakeExecutor{})
	_, err := client.Get(context.Background(), connect.NewRequest(&toolsv1.GetToolRequest{Name: "   "}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGet_NotFound(t *testing.T) {
	client := newTestClient(t, &fakeRegistry{tools: map[string]*agentinboxpb.ToolDefinition{}}, &fakeExecutor{})
	_, err := client.Get(context.Background(), connect.NewRequest(&toolsv1.GetToolRequest{Name: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestExecute_HappyPath(t *testing.T) {
	exec := &fakeExecutor{result: toolexecution.SuccessResult(map[string]interface{}{
		"execution_id": "exec-1",
		"status":       "completed",
	})}
	client := newTestClient(t, &fakeRegistry{}, exec)

	args, _ := structpb.NewStruct(map[string]interface{}{"workflow_id": "w-1"})
	resp, err := client.Execute(context.Background(), connect.NewRequest(&toolsv1.ExecuteToolRequest{
		ToolName:  "execute_workflow",
		Arguments: args,
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Success)
	require.False(t, resp.Msg.IsAsync)
	require.NotNil(t, resp.Msg.Result)
	require.Equal(t, "exec-1", resp.Msg.Result.Fields["execution_id"].GetStringValue())
	require.Equal(t, "execute_workflow", exec.lastTool)
	require.Equal(t, "w-1", exec.lastArgs["workflow_id"])
}

func TestExecute_AsyncResultRoundTrips(t *testing.T) {
	exec := &fakeExecutor{result: toolexecution.AsyncResult(
		map[string]interface{}{"execution_id": "exec-2"},
		"exec-2",
	)}
	client := newTestClient(t, &fakeRegistry{}, exec)

	resp, err := client.Execute(context.Background(), connect.NewRequest(&toolsv1.ExecuteToolRequest{
		ToolName: "execute_workflow",
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Success)
	require.True(t, resp.Msg.IsAsync)
	require.Equal(t, "exec-2", resp.Msg.RunId)
	require.Equal(t, toolexecution.StatusPending, resp.Msg.Status)
}

func TestExecute_ToolErrorRoundTrips(t *testing.T) {
	exec := &fakeExecutor{result: toolexecution.ErrorResult("workflow_id is required", toolexecution.CodeInvalidArgs)}
	client := newTestClient(t, &fakeRegistry{}, exec)

	resp, err := client.Execute(context.Background(), connect.NewRequest(&toolsv1.ExecuteToolRequest{
		ToolName: "execute_workflow",
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.Success)
	require.Equal(t, "workflow_id is required", resp.Msg.Error)
	require.Equal(t, toolexecution.CodeInvalidArgs, resp.Msg.Code)
}

func TestExecute_ExecutorErrorMapsToInternal(t *testing.T) {
	exec := &fakeExecutor{err: errors.New("dispatch failed")}
	client := newTestClient(t, &fakeRegistry{}, exec)

	_, err := client.Execute(context.Background(), connect.NewRequest(&toolsv1.ExecuteToolRequest{
		ToolName: "execute_workflow",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestExecute_EmptyToolNameRejected(t *testing.T) {
	client := newTestClient(t, &fakeRegistry{}, &fakeExecutor{})
	_, err := client.Execute(context.Background(), connect.NewRequest(&toolsv1.ExecuteToolRequest{ToolName: "  "}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestModule_RequiresDeps(t *testing.T) {
	require.Panics(t, func() { Module(Deps{}) })
	require.Panics(t, func() { Module(Deps{Logger: logrus.New()}) })
	require.Panics(t, func() { Module(Deps{Logger: logrus.New(), Registry: &fakeRegistry{}}) })
}
