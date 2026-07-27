package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"scenario-to-desktop/cli/internal/support"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeTelemetryRPC struct {
	ingestRequest *domainv1.IngestTelemetryRequest
	tailRequest   *domainv1.TelemetryTailRequest
	deleted       string
	err           error
}

func (f *fakeTelemetryRPC) IngestTelemetry(_ context.Context, request *connect.Request[domainv1.IngestTelemetryRequest]) (*connect.Response[domainv1.IngestTelemetryResponse], error) {
	f.ingestRequest = request.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&domainv1.IngestTelemetryResponse{EventsIngested: int32(len(request.Msg.Events))}), nil
}

func (f *fakeTelemetryRPC) GetTelemetrySummary(_ context.Context, _ *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&domainv1.TelemetryPayloadResponse{Payload: mustStruct(map[string]any{"summary": "ok"})}), nil
}

func (f *fakeTelemetryRPC) GetTelemetryInsights(_ context.Context, _ *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&domainv1.TelemetryPayloadResponse{Payload: mustStruct(map[string]any{"insights": []any{}})}), nil
}

func (f *fakeTelemetryRPC) GetTelemetryTail(_ context.Context, request *connect.Request[domainv1.TelemetryTailRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error) {
	f.tailRequest = request.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&domainv1.TelemetryPayloadResponse{Payload: mustStruct(map[string]any{"entries": []any{}})}), nil
}

func (f *fakeTelemetryRPC) DeleteTelemetry(_ context.Context, request *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryDeleteResponse], error) {
	f.deleted = request.Msg.ScenarioName
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&domainv1.TelemetryDeleteResponse{ScenarioName: request.Msg.ScenarioName, Deleted: true}), nil
}

func mustStruct(value map[string]any) *structpb.Struct {
	result, err := structpb.NewStruct(value)
	if err != nil {
		panic(err)
	}
	return result
}

func newTestCommands(rpc telemetryRPC) *Commands {
	return &Commands{rpc: rpc}
}

func newDownloadCommands(t *testing.T, handler http.Handler) (*Commands, *cliapp.ScenarioApp) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             "scenario-to-desktop-test",
		Version:          "test",
		Description:      "test",
		DefaultAPIBase:   server.URL,
		AllowAnonymous:   true,
		CommandGroups:    func(*cliapp.ScenarioApp) []cliapp.CommandGroup { return nil },
		SubcommandGroups: func(*cliapp.ScenarioApp) []cliapp.SubcommandGroup { return nil },
	})
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	return &Commands{deps: support.Dependencies{Core: func() *cliapp.ScenarioApp { return app }}, rpc: &fakeTelemetryRPC{}}, app
}

func assertModesSucceed(t *testing.T, modes cliapptest.PrimitiveModes) {
	t.Helper()
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if !strings.Contains(modes.Human, "Result:") && !strings.Contains(modes.Human, "Summary:") {
		t.Fatalf("human output missing primitive report: %q", modes.Human)
	}
	var document any
	if err := json.Unmarshal([]byte(modes.JSON), &document); err != nil {
		t.Fatalf("JSON output invalid: %v (%q)", err, modes.JSON)
	}
}

func TestIngestPrimitiveUsesTypedConnectRequest(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "events.jsonl")
	if err := os.WriteFile(path, []byte("{\"event\":\"startup\"}\nnot-json\n{\"event\":\"shutdown\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rpc := &fakeTelemetryRPC{}
	commands := newTestCommands(rpc)
	assertModesSucceed(t, cliapptest.RunPrimitiveHandlerModes(t, commands.ingestPrimitive(), telemetryScenarioArgs("file", "source"), []string{"demo", "--file", path, "--source", "runtime"}, nil))
	if rpc.ingestRequest.GetScenarioName() != "demo" || rpc.ingestRequest.GetSource() != "runtime" {
		t.Fatalf("unexpected request: %#v", rpc.ingestRequest)
	}
	if len(rpc.ingestRequest.GetEvents()) != 2 {
		t.Fatalf("events = %d, want 2", len(rpc.ingestRequest.GetEvents()))
	}
}

func TestTelemetryCommandsValidateArguments(t *testing.T) {
	for name, schema := range map[string]cliapp.ArgSchema{
		"ingest":   telemetryScenarioArgs("file", "source"),
		"summary":  telemetryScenarioArgs(),
		"insights": telemetryScenarioArgs(),
		"tail":     telemetryScenarioArgs("limit"),
		"download": telemetryScenarioArgs("output"),
		"delete":   telemetryScenarioArgs(),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := cliapptest.NewTestRunContextFromArgs(schema, nil, nil, nil, nil); err == nil {
				t.Fatal("missing scenario should fail production argument parsing")
			}
		})
	}
}

func TestTelemetryPayloadCommandsUseConnect(t *testing.T) {
	rpc := &fakeTelemetryRPC{}
	commands := newTestCommands(rpc)
	assertModesSucceed(t, cliapptest.RunPrimitiveHandlerModes(t, commands.summaryPrimitive(), telemetryScenarioArgs(), []string{"demo"}, nil))
	assertModesSucceed(t, cliapptest.RunPrimitiveHandlerModes(t, commands.insightsPrimitive(), telemetryScenarioArgs(), []string{"demo"}, nil))
	assertModesSucceed(t, cliapptest.RunPrimitiveHandlerModes(t, commands.tailPrimitive(), telemetryScenarioArgs("limit"), []string{"demo", "--limit", "50"}, nil))
	if rpc.tailRequest.GetScenarioName() != "demo" || rpc.tailRequest.GetLimit() != 50 {
		t.Fatalf("unexpected tail request: %#v", rpc.tailRequest)
	}
}

func TestDeleteUsesTypedResponse(t *testing.T) {
	rpc := &fakeTelemetryRPC{}
	commands := newTestCommands(rpc)
	assertModesSucceed(t, cliapptest.RunPrimitiveHandlerModes(t, commands.deletePrimitive(), telemetryScenarioArgs(), []string{"demo"}, nil))
	if rpc.deleted != "demo" {
		t.Fatalf("deleted scenario = %q, want demo", rpc.deleted)
	}
}

func TestConnectErrorsAreReturned(t *testing.T) {
	commands := newTestCommands(&fakeTelemetryRPC{err: errors.New("unavailable")})
	modes := cliapptest.RunPrimitiveHandlerModes(t, commands.summaryPrimitive(), telemetryScenarioArgs(), []string{"demo"}, nil)
	if modes.HumanErr == nil || modes.JSONErr == nil || !strings.Contains(modes.HumanErr.Error(), "telemetry summary") || !strings.Contains(modes.JSONErr.Error(), "telemetry summary") {
		t.Fatalf("errors = human=%v json=%v, want wrapped API errors", modes.HumanErr, modes.JSONErr)
	}
}

func TestDownloadPrimitiveRetainsFileStreamingHTTPException(t *testing.T) {
	commands, app := newDownloadCommands(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/deployment/telemetry/demo/download" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte("telemetry-data"))
	}))
	output := filepath.Join(t.TempDir(), "telemetry.jsonl")
	assertModesSucceed(t, cliapptest.RunPrimitiveHandlerModes(t, commands.downloadPrimitive(), telemetryScenarioArgs("output"), []string{"demo", "--output", output}, app))
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "telemetry-data" {
		t.Fatalf("downloaded data = %q, error = %v", data, err)
	}
}
