package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation/validation_v1connect"

	"quality-health/internal/surfaces"
)

type planRecorder struct {
	validationconnect.UnimplementedValidationServiceHandler
	request *validationv1.ValidateScenarioRequest
}

func (p *planRecorder) ValidateScenario(_ context.Context, req *connect.Request[validationv1.ValidateScenarioRequest]) (*connect.Response[validationv1.ValidateScenarioResponse], error) {
	p.request = req.Msg
	return connect.NewResponse(&validationv1.ValidateScenarioResponse{Plan: &validationv1.ExecutionPlan{Commands: []*validationv1.PlannedCommand{
		{WorkspaceId: "ui", TestKind: "typecheck"},
		{WorkspaceId: "api", TestKind: "unit"},
	}}}), nil
}

func TestTypecheckCoverageQueriesTheResolvedScenarioRoot(t *testing.T) {
	recorder := &planRecorder{}
	path, handler := validationconnect.NewValidationServiceHandler(recorder)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	inv := surfaces.Inventory{Scenario: "demo", RootPath: "/tmp/isolated/scenarios/demo"}
	covered, err := typecheckCoverage(context.Background(), validationconnect.NewValidationServiceClient(server.Client(), server.URL), inv)
	if err != nil {
		t.Fatalf("typecheckCoverage: %v", err)
	}
	if recorder.request.GetPath() != inv.RootPath || recorder.request.GetScenario() != "demo" {
		t.Fatalf("plan request = scenario %q path %q, want demo at %q", recorder.request.GetScenario(), recorder.request.GetPath(), inv.RootPath)
	}
	if !covered["ui"] || covered["api"] {
		t.Fatalf("covered = %v, want only the typecheck workspace", covered)
	}
}
