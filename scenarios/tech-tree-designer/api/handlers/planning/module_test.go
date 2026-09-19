package planning

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	planningdomain "tech-tree-designer/internal/planning"

	planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning"
	planningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning/planning_v1connect"
)

type fakeRepository struct{}

func (fakeRepository) CreateScenario(_ context.Context, in planningdomain.CreateInput) (planningdomain.Scenario, error) {
	return planningdomain.Scenario{Slug: in.Slug, DisplayName: in.DisplayName, TargetStability: in.TargetStability}, nil
}

func (fakeRepository) ListScenarios(context.Context, planningdomain.ListFilter) ([]planningdomain.Scenario, error) {
	return nil, nil
}

func (fakeRepository) GetScenario(context.Context, string) (planningdomain.Scenario, error) {
	return planningdomain.Scenario{}, nil
}

func (fakeRepository) PutFile(context.Context, planningdomain.PutFileInput) (planningdomain.ProtoFile, error) {
	return planningdomain.ProtoFile{}, nil
}

func (fakeRepository) DeleteFile(context.Context, string, string) (bool, error) {
	return false, nil
}

type fakeValidator struct{}

func (fakeValidator) Validate(context.Context, planningdomain.Scenario) ([]planningdomain.PlanFinding, error) {
	return nil, nil
}

type fakeMaterializer struct{}

func (fakeMaterializer) Materialize(context.Context, planningdomain.Scenario) (planningdomain.MaterializeResult, error) {
	return planningdomain.MaterializeResult{}, nil
}

func TestModuleRegistersPlanningConnectHandler(t *testing.T) {
	service := planningdomain.NewService(fakeRepository{}, fakeValidator{}, fakeMaterializer{})
	mod := Module(service)
	require.Equal(t, "planning", mod.Name)
	require.Len(t, mod.Endpoints, 7)

	router := http.NewServeMux()
	path, handler := planningconnect.NewPlanningServiceHandler(NewHandler(service))
	router.Handle(path, handler)

	client := planningconnect.NewPlanningServiceClient(&http.Client{
		Transport: localRoundTripper{handler: router},
		Timeout:   5 * time.Second,
	}, "http://tech-tree-designer.test")
	resp, err := client.CreatePlannedScenario(context.Background(), connect.NewRequest(&planningv1.CreatePlannedScenarioRequest{
		Slug:            "planned-demo",
		DisplayName:     "Planned Demo",
		TargetStability: "experimental",
	}))

	require.NoError(t, err)
	require.Equal(t, "planned-demo", resp.Msg.GetSlug())
	require.Equal(t, "Planned Demo", resp.Msg.GetDisplayName())
}

type localRoundTripper struct {
	handler http.Handler
}

func (rt localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rt.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}
