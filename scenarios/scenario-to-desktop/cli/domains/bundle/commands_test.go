package bundle

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
)

type fakePipelineRPC struct {
	request  *pipelinev1.BundleCleanRequest
	response *pipelinev1.BundleCleanResponse
	err      error
}

func (f *fakePipelineRPC) CleanBundle(_ context.Context, request *connect.Request[pipelinev1.BundleCleanRequest]) (*connect.Response[pipelinev1.BundleCleanResponse], error) {
	f.request = request.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.response), nil
}

var cleanSchema = cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "location-mode", Default: "proper", Values: []string{"proper", "staging", "temp"}}, {Name: "pipeline-id"}}}

func runClean(t *testing.T, command *Commands, args []string) cliapptest.PrimitiveModes {
	t.Helper()
	return cliapptest.RunPrimitiveHandlerModes(t, command.cleanPrimitive(), cleanSchema, args, nil)
}

func assertCleanSuccess(t *testing.T, modes cliapptest.PrimitiveModes) {
	t.Helper()
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("clean errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	var value any
	if err := json.Unmarshal([]byte(modes.JSON), &value); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestCleanPrimitiveUsesProductionParserAndTypedRequest(t *testing.T) {
	rpc := &fakePipelineRPC{response: &pipelinev1.BundleCleanResponse{Path: "/tmp/bundle", Removed: true}}
	assertCleanSuccess(t, runClean(t, &Commands{rpc: rpc}, []string{"demo", "--location-mode", "staging", "--pipeline-id", "pipe-1"}))
	if rpc.request.GetScenarioName() != "demo" || rpc.request.GetLocationMode() != "staging" || rpc.request.GetPipelineId() != "pipe-1" {
		t.Fatalf("request = %#v", rpc.request)
	}
	if _, err := cliapptest.NewTestRunContextFromArgs(cleanSchema, nil, nil, nil, nil); err == nil {
		t.Fatal("missing scenario accepted")
	}
	if _, err := cliapptest.NewTestRunContextFromArgs(cleanSchema, []string{"demo", "--framework", "electron"}, nil, nil, nil); err == nil {
		t.Fatal("removed framework flag accepted")
	}
}

func TestCleanPrimitiveRejectsIncompleteOrFailedOperations(t *testing.T) {
	modes := runClean(t, &Commands{rpc: &fakePipelineRPC{response: &pipelinev1.BundleCleanResponse{Path: "/tmp/bundle"}}}, []string{"demo", "--location-mode", "temp"})
	if modes.HumanErr == nil || modes.JSONErr == nil {
		t.Fatal("missing pipeline id accepted")
	}
	modes = runClean(t, &Commands{rpc: &fakePipelineRPC{err: errors.New("unavailable")}}, []string{"demo"})
	if modes.HumanErr == nil || modes.JSONErr == nil {
		t.Fatal("API error was lost")
	}
}
