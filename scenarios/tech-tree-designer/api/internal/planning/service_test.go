package planning

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeRepository struct {
	scenarios []Scenario
}

func (f *fakeRepository) CreateScenario(context.Context, CreateInput) (Scenario, error) {
	panic("not implemented")
}

func (f *fakeRepository) ListScenarios(context.Context, ListFilter) ([]Scenario, error) {
	return f.scenarios, nil
}

func (f *fakeRepository) GetScenario(_ context.Context, slug string) (Scenario, error) {
	for _, scenario := range f.scenarios {
		if scenario.Slug == slug {
			return scenario, nil
		}
	}
	return Scenario{}, ErrScenarioNotFound{Slug: slug}
}

func (f *fakeRepository) PutFile(context.Context, PutFileInput) (ProtoFile, error) {
	panic("not implemented")
}

func (f *fakeRepository) DeleteFile(context.Context, string, string) (bool, error) {
	panic("not implemented")
}

func TestServicePlannedGraphDerivesNodesAndEdgesFromImports(t *testing.T) {
	svc := NewService(&fakeRepository{scenarios: []Scenario{{
		Slug:        "planned-demo",
		DisplayName: "Planned Demo",
		Sector:      "engineering",
		Tier:        "foundation",
		Files: []ProtoFile{{
			Path: "planned-demo/v1/api/service.proto",
			Text: `syntax = "proto3";
import "proto-health/v1/shared/surface.proto";
import "planned-demo/v1/internal/self.proto";
`,
		}},
	}}}, nil, nil)

	graph, err := svc.PlannedGraph(context.Background())
	require.NoError(t, err)
	require.Len(t, graph.GetNodes(), 1)
	require.Equal(t, "planned-demo", graph.GetNodes()[0].GetScenario())
	require.Equal(t, "none", graph.GetNodes()[0].GetTransportWorld())
	require.Equal(t, []string{"experimental"}, graph.GetNodes()[0].GetStability())
	require.Len(t, graph.GetEdges(), 1)
	require.Equal(t, "planned-demo", graph.GetEdges()[0].GetFromScenario())
	require.Equal(t, "proto-health", graph.GetEdges()[0].GetToScenario())
	require.Equal(t, "proto-health/v1/shared/surface.proto", graph.GetEdges()[0].GetEvidence()[0].GetImportPath())
}
