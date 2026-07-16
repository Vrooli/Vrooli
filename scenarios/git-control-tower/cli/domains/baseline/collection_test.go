package baseline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	baselinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
)

type incompleteCollectionServer struct {
	baselinesconnect.UnimplementedBaselinesServiceHandler
}

func (*incompleteCollectionServer) GetCollection(context.Context, *connect.Request[baselinesv1.GetCollectionRequest]) (*connect.Response[baselinesv1.GetCollectionResponse], error) {
	return connect.NewResponse(&baselinesv1.GetCollectionResponse{
		Collection:  &baselinesv1.BaselineCollection{Name: "before", Coverage: &baselinesv1.CollectionCoverage{Required: 2, Ready: 1, Pending: 1}},
		WaitOutcome: &baselinesv1.CollectionWaitOutcome{Kind: "incomplete", RecoveryCommands: []string{"git-control-tower baseline collection show --name before --wait"}},
	}), nil
}

func (*incompleteCollectionServer) GetCollectionDiff(context.Context, *connect.Request[baselinesv1.GetCollectionDiffRequest]) (*connect.Response[baselinesv1.GetCollectionDiffResponse], error) {
	return connect.NewResponse(&baselinesv1.GetCollectionDiffResponse{
		Collection: &baselinesv1.BaselineCollection{Name: "before"}, OperationId: "op-1", Classification: "not-ready",
		WaitOutcome: &baselinesv1.CollectionWaitOutcome{Kind: "incomplete", RecoveryCommands: []string{"git-control-tower baseline collection diff status --name before --operation-id op-1 --wait"}},
	}), nil
}

func withIncompleteCollectionServer(t *testing.T) {
	t.Helper()
	path, handler := baselinesconnect.NewBaselinesServiceHandler(&incompleteCollectionServer{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	previous := clientFactory
	clientFactory = func(*cliapp.ScenarioApp) baselinesconnect.BaselinesServiceClient {
		return baselinesconnect.NewBaselinesServiceClient(http.DefaultClient, server.URL)
	}
	t.Cleanup(func() { clientFactory = previous })
}

func TestParseCollectionMemberUsesCollectionNameAndAllowsOverride(t *testing.T) {
	member, err := parseCollectionMember("plan-manager", "before")
	if err != nil || member.GetScenario() != "plan-manager" || member.GetBaselineName() != "before" || !member.GetRequired() {
		t.Fatalf("default member = %#v err=%v", member, err)
	}
	member, err = parseCollectionMember("git-control-tower:separate", "before")
	if err != nil || member.GetBaselineName() != "separate" {
		t.Fatalf("override member = %#v err=%v", member, err)
	}
	if _, err := parseCollectionMember(":bad", "before"); err == nil {
		t.Fatal("empty scenario accepted")
	}
}

func TestCollectionFollowupArgsOmitEmptyBranch(t *testing.T) {
	got := collectionFollowupArgs("before", "", "--wait")
	want := []string{"--name", "before", "--wait"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("follow-up args = %#v, want %#v", got, want)
	}
}

func TestCollectionWaitCommandsExitNotReadyOnTypedIncomplete(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	withIncompleteCollectionServer(t)
	previousExit := collectionExit
	var exitCodes []int
	collectionExit = func(code int) { exitCodes = append(exitCodes, code) }
	t.Cleanup(func() { collectionExit = previousExit })

	if err := runCollectionShow(nil, []string{"--name", "before", "--wait", "--json"}); err != nil {
		t.Fatalf("collection show: %v", err)
	}
	if err := runCollectionDiffStatus(nil, []string{"--name", "before", "--operation-id", "op-1", "--wait", "--json"}); err != nil {
		t.Fatalf("collection diff status: %v", err)
	}
	if !reflect.DeepEqual(exitCodes, []int{exitNotReady, exitNotReady}) {
		t.Fatalf("exit codes = %#v", exitCodes)
	}
}

func TestCollectionDiffStatusExitCodePreservesTerminalVerdict(t *testing.T) {
	complete := &baselinesv1.CollectionWaitOutcome{Kind: "complete"}
	tests := []struct {
		name         string
		response     *baselinesv1.GetCollectionDiffResponse
		waited       bool
		expectedExit int
	}{
		{name: "complete regression", response: &baselinesv1.GetCollectionDiffResponse{Classification: "regression", WaitOutcome: complete}, waited: true, expectedExit: exitRegression},
		{name: "complete not comparable", response: &baselinesv1.GetCollectionDiffResponse{Classification: "not-comparable", WaitOutcome: complete}, waited: true, expectedExit: exitNotComparable},
		{name: "complete clean", response: &baselinesv1.GetCollectionDiffResponse{Classification: "clean", WaitOutcome: complete}, waited: true, expectedExit: exitOK},
		{name: "typed incomplete", response: &baselinesv1.GetCollectionDiffResponse{Classification: "not-ready", WaitOutcome: &baselinesv1.CollectionWaitOutcome{Kind: "incomplete"}}, waited: true, expectedExit: exitNotReady},
		{name: "missing wait outcome fails closed", response: &baselinesv1.GetCollectionDiffResponse{Classification: "clean"}, waited: true, expectedExit: exitNotReady},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := collectionDiffStatusExitCode(tc.response, tc.waited); got != tc.expectedExit {
				t.Fatalf("exit code = %d, want %d", got, tc.expectedExit)
			}
		})
	}
}
