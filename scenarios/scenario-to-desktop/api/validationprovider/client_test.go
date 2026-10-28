package validationprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestValidateIsolationEvidence(t *testing.T) {
	valid := nativeDetailForTest(t, map[string]any{
		"installed":                            true,
		"lease_id":                             "lease-1",
		"install_error":                        "",
		"heartbeat_error":                      "",
		"clear_error":                          "",
		"test_pool_requests":                   2,
		"primary_during_test_mode_requests":    0,
		"test_root_writes":                     1,
		"primary_root_writes_during_test_mode": 0,
	})
	if err := validateIsolationEvidence(valid); err != nil {
		t.Fatalf("valid isolation evidence rejected: %v", err)
	}

	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing lease", mutate: func(fields map[string]any) { fields["lease_id"] = "" }},
		{name: "install failure", mutate: func(fields map[string]any) { fields["install_error"] = "storage unavailable" }},
		{name: "primary request", mutate: func(fields map[string]any) { fields["primary_during_test_mode_requests"] = 1 }},
		{name: "primary write", mutate: func(fields map[string]any) { fields["primary_root_writes_during_test_mode"] = 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			fields := map[string]any{
				"installed":                            true,
				"lease_id":                             "lease-1",
				"install_error":                        "",
				"heartbeat_error":                      "",
				"clear_error":                          "",
				"primary_during_test_mode_requests":    0,
				"primary_root_writes_during_test_mode": 0,
			}
			test.mutate(fields)
			if err := validateIsolationEvidence(nativeDetailForTest(t, fields)); err == nil {
				t.Fatal("invalid isolation evidence was accepted")
			}
		})
	}
}

type fixedURLResolver string

func (r fixedURLResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return string(r), nil
}

type bindingCaptureProvider struct {
	binding *scenariovalidationv1.DesktopValidationBinding
}

func (p *bindingCaptureProvider) StartValidationRun(_ context.Context, request *connect.Request[scenariovalidationv1.StartValidationRunRequest]) (*connect.Response[scenariovalidationv1.StartValidationRunResponse], error) {
	p.binding = request.Msg.GetDesktopBinding()
	return connect.NewResponse(&scenariovalidationv1.StartValidationRunResponse{Run: &scenariovalidationv1.ValidationRun{RunId: "provider-run"}}), nil
}

func (p *bindingCaptureProvider) GetValidationRun(context.Context, *connect.Request[scenariovalidationv1.GetValidationRunRequest]) (*connect.Response[scenariovalidationv1.GetValidationRunResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.GetValidationRunResponse{Run: &scenariovalidationv1.ValidationRun{RunId: "provider-run"}}), nil
}

func (p *bindingCaptureProvider) WaitValidationRun(context.Context, *connect.Request[scenariovalidationv1.WaitValidationRunRequest]) (*connect.Response[scenariovalidationv1.WaitValidationRunResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.WaitValidationRunResponse{Run: &scenariovalidationv1.ValidationRun{RunId: "provider-run"}}), nil
}

func (p *bindingCaptureProvider) AbortValidationRun(context.Context, *connect.Request[scenariovalidationv1.AbortValidationRunRequest]) (*connect.Response[scenariovalidationv1.AbortValidationRunResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.AbortValidationRunResponse{}), nil
}

func TestExecuteCarriesBundledRuntimeInConnectBinding(t *testing.T) {
	provider := &bindingCaptureProvider{}
	path, handler := scenariovalidationconnect.NewDurableValidationRunServiceHandler(provider)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewClient(fixedURLResolver(server.URL), server.Client())
	client.Execute(context.Background(), Request{
		MatrixRunID: "matrix", CellID: "cell", ScenarioName: "demo", ScenarioPath: "/scenario",
		WorkflowPath: "bas/cases/journey.json", Target: &domainv1.AppTarget{TargetId: "target", CdpEndpoint: "http://127.0.0.1:9222", RendererId: "renderer", RendererUrl: "http://127.0.0.1:41235", ApiUrl: "http://127.0.0.1:41236", TestDatabaseRoot: "/private/session/test-database"},
		ProfileID: "normal", ContextID: "context", ArtifactDigest: "sha256:artifact", BundledRuntime: true,
		IsolationLeaseID: "matrix:cell", PrepareIsolationRestart: true,
	})
	if provider.binding == nil || !provider.binding.GetBundledRuntime() {
		t.Fatalf("provider binding = %+v, want bundled_runtime=true", provider.binding)
	}
	if provider.binding.GetApiUrl() != "http://127.0.0.1:41236" {
		t.Fatalf("provider API binding = %q, want owner-discovered target API endpoint", provider.binding.GetApiUrl())
	}
	if provider.binding.GetTestDatabaseRoot() != "/private/session/test-database" {
		t.Fatalf("provider test database root = %q, want target-owned private root", provider.binding.GetTestDatabaseRoot())
	}
	if provider.binding.GetIsolationLeaseId() != "matrix:cell" || !provider.binding.GetPrepareIsolationRestart() || provider.binding.GetResumeIsolationRestart() {
		t.Fatalf("provider restart binding = %+v, want prepare phase for stable lease", provider.binding)
	}
}

func TestExecutePersistsSelectedBASVariantInDurableBinding(t *testing.T) {
	provider := &bindingCaptureProvider{}
	path, handler := scenariovalidationconnect.NewDurableValidationRunServiceHandler(provider)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewClient(fixedURLResolver(server.URL), server.Client())
	client.Execute(context.Background(), Request{
		MatrixRunID: "matrix", CellID: "cell", ScenarioName: "demo", ScenarioPath: "/scenario",
		WorkflowPath: "bas/cases/journey.json", Target: &domainv1.AppTarget{TargetId: "target", CdpEndpoint: "http://127.0.0.1:9222", RendererId: "renderer", RendererUrl: "http://127.0.0.1:41235"},
		ProfileID: "normal", ContextID: "context", ArtifactDigest: "sha256:artifact", BrowserAutomationStudioVariant: "preview-2",
	})
	if provider.binding == nil || provider.binding.GetBrowserAutomationStudioVariant() != "preview-2" {
		t.Fatalf("durable binding = %+v, want selected BAS variant preview-2", provider.binding)
	}
}

func nativeDetailForTest(t *testing.T, isolation map[string]any) *anypb.Any {
	t.Helper()
	detail, err := structpb.NewStruct(map[string]any{
		"execution": map[string]any{"isolation": isolation},
	})
	if err != nil {
		t.Fatal(err)
	}
	native, err := anypb.New(detail)
	if err != nil {
		t.Fatal(err)
	}
	return native
}

func TestArtifactEvidenceUsesNormalizedWorkflowURI(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "workflow.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	reference, err := anypb.New(mustStruct(t, map[string]any{"reference": path}))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := artifactEvidence(&scenariovalidationv1.ValidationRun{RunId: "run-1", ArtifactReferences: []*anypb.Any{reference}}, root, Request{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(evidence[0].GetUri(), "run-1//") {
		t.Fatalf("workflow URI retained an absolute-path double slash: %q", evidence[0].GetUri())
	}
}

func mustStruct(t *testing.T, fields map[string]any) *structpb.Struct {
	t.Helper()
	value, err := structpb.NewStruct(fields)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
