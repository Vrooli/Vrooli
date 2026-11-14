package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/analysis_operations"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/analysis_operations/analysis_operations_v1connect"
	"scenario-dependency-analyzer/cli/domains"
)

type fleetFixture struct {
	rpc.UnimplementedAnalysisOperationsServiceHandler
	starts, waits atomic.Int32
	kind          pb.AnalysisOperationKind
	apply         bool
	state         pb.AnalysisOperationState
}

func (f *fleetFixture) StartOperation(ctx context.Context, req *connect.Request[pb.StartOperationRequest]) (*connect.Response[pb.StartOperationResponse], error) {
	f.starts.Add(1)
	if req.Msg.GetClientOperationId() != "replay-key" {
		return nil, connect.NewError(connect.CodeInvalidArgument, context.Canceled)
	}
	f.kind, f.apply = req.Msg.GetKind(), req.Msg.GetApplyGraph()
	return connect.NewResponse(&pb.StartOperationResponse{Operation: &pb.AnalysisOperation{Id: "receipt", State: pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_QUEUED}}), nil
}

func (f *fleetFixture) WaitOperation(ctx context.Context, req *connect.Request[pb.WaitOperationRequest]) (*connect.Response[pb.WaitOperationResponse], error) {
	f.waits.Add(1)
	return connect.NewResponse(&pb.WaitOperationResponse{Operation: &pb.AnalysisOperation{Id: req.Msg.GetOperationId(), State: f.state, Error: "fixture outcome"}}), nil
}

func TestFleetCommandsUseTypedAdmissionAndOneWait(t *testing.T) {
	for _, test := range []struct {
		name              string
		args              []string
		kind              pb.AnalysisOperationKind
		apply, wait, fail bool
	}{
		{"analyze", []string{"analyze", "all", "--client-operation-id", "replay-key", "--json"}, pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_FLEET_ANALYSIS, false, true, false},
		{"graph", []string{"graph", "rebuild", "--apply", "--client-operation-id", "replay-key", "--json"}, pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_GRAPH_REBUILD, true, true, false},
		{"start-only", []string{"analysis", "start", "--client-operation-id", "replay-key", "--json"}, pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_FLEET_ANALYSIS, false, false, false},
		{"partial", []string{"analyze", "all", "--client-operation-id", "replay-key", "--json"}, pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_FLEET_ANALYSIS, false, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := &fleetFixture{state: pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_SUCCEEDED}
			if test.fail {
				fixture.state = pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_PARTIAL
			}
			path, handler := rpc.NewAnalysisOperationsServiceHandler(fixture)
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			var legacy atomic.Int32
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"status":"healthy"}`))
					return
				}
				legacy.Add(1)
				http.NotFound(w, r)
			})
			server := httptest.NewServer(mux)
			defer server.Close()
			t.Setenv("SDA_FLEET_TEST_CONFIG_DIR", t.TempDir())
			core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "sda-fleet-fixture", DefaultAPIBase: server.URL, ConfigDirEnvVars: []string{"SDA_FLEET_TEST_CONFIG_DIR"}, AllowAnonymous: true})
			if err != nil {
				t.Fatal(err)
			}
			core.SetCommandsWithSubgroups(domains.CommandGroups(core), domains.SubcommandGroups(core))
			err = (&App{core: core}).Run(test.args)
			wantWait := int32(0)
			if test.wait {
				wantWait = 1
			}
			if (err != nil) != test.fail || fixture.starts.Load() != 1 || fixture.waits.Load() != wantWait || legacy.Load() != 0 || fixture.kind != test.kind || fixture.apply != test.apply {
				t.Fatalf("err=%v starts=%d waits=%d legacy=%d kind=%s apply=%t", err, fixture.starts.Load(), fixture.waits.Load(), legacy.Load(), fixture.kind, fixture.apply)
			}
		})
	}
}
