package baseline

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	baselinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
)

type incompleteCollectionServer struct {
	baselinesconnect.UnimplementedBaselinesServiceHandler
}

func (*incompleteCollectionServer) GetCollectionStatus(context.Context, *connect.Request[baselinesv1.GetCollectionStatusRequest]) (*connect.Response[baselinesv1.GetCollectionStatusResponse], error) {
	return connect.NewResponse(&baselinesv1.GetCollectionStatusResponse{
		Collection: &baselinesv1.BaselineCollection{Name: "before", Coverage: &baselinesv1.CollectionCoverage{Required: 2, Ready: 1, Pending: 1}},
		Standing:   &commonv1.OperationStanding{Lifecycle: "executing", Directive: "wait"},
	}), nil
}

func (*incompleteCollectionServer) GetCollectionDiffStatus(context.Context, *connect.Request[baselinesv1.GetCollectionDiffStatusRequest]) (*connect.Response[baselinesv1.GetCollectionDiffStatusResponse], error) {
	return connect.NewResponse(&baselinesv1.GetCollectionDiffStatusResponse{
		Collection: &baselinesv1.BaselineCollection{Name: "before"}, OperationId: "op-1", Classification: "not-ready",
		Standing: &commonv1.OperationStanding{Lifecycle: "executing", Directive: "wait", ReattachCommand: "git-control-tower baseline collection diff wait --name before --operation-id op-1 --json"},
	}), nil
}

func (*incompleteCollectionServer) WaitCollectionDiff(context.Context, *connect.Request[baselinesv1.WaitCollectionDiffRequest]) (*connect.Response[baselinesv1.WaitCollectionDiffResponse], error) {
	return connect.NewResponse(&baselinesv1.WaitCollectionDiffResponse{Collection: &baselinesv1.BaselineCollection{Name: "before"}, OperationId: "op-1", Classification: "not-ready", Detached: true, Standing: &commonv1.OperationStanding{Lifecycle: "executing", Directive: "wait"}}), nil
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

func TestCollectionWaitCommandsReturnRenderedDetachOutcome(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	withIncompleteCollectionServer(t)
	err := runCollectionDiffWait(nil, []string{"--name", "before", "--operation-id", "op-1", "--json"})
	var outcome renderedExitError
	if !errors.As(err, &outcome) || outcome.code != 124 {
		t.Fatalf("wait outcome = %v", err)
	}
}

func TestCollectionDiffWaitExitPreservesTerminalVerdict(t *testing.T) {
	tests := []struct {
		name           string
		lifecycle      string
		classification string
		expectedExit   int
	}{
		{name: "complete regression", lifecycle: "terminal", classification: "regression", expectedExit: exitRegression},
		{name: "complete not comparable", lifecycle: "terminal", classification: "not-comparable", expectedExit: exitNotComparable},
		{name: "complete clean", lifecycle: "terminal", classification: "clean", expectedExit: exitOK},
		{name: "detached", lifecycle: "executing", classification: "clean", expectedExit: 124},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := renderedExitForStanding(&commonv1.OperationStanding{Lifecycle: tc.lifecycle}, tc.classification)
			if tc.expectedExit == exitOK && err != nil {
				t.Fatalf("clean returned %v", err)
			}
			if tc.expectedExit != exitOK {
				var outcome renderedExitError
				if !errors.As(err, &outcome) || outcome.code != tc.expectedExit {
					t.Fatalf("exit = %v, want %d", err, tc.expectedExit)
				}
			}
		})
	}
}
