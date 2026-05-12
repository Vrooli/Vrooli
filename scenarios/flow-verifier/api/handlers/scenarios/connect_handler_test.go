package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	scenariosH "flow-verifier/handlers/scenarios"
	"flow-verifier/internal/artifacts"
	"flow-verifier/internal/flows"
	"flow-verifier/internal/scenarios"

	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios/scenarios_v1connect"
)

func newClient(t *testing.T, scenariosSvc scenariosH.Service, artifactsSvc scenariosH.ArtifactsService) scenariosconnect.ScenariosServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	handler := scenariosH.NewStreamHandler(scenariosH.StreamDeps{
		Scenarios: scenariosSvc,
		Artifacts: artifactsSvc,
		Logger:    logger,
	})
	path, h := scenariosconnect.NewScenariosServiceHandler(handler)
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return scenariosconnect.NewScenariosServiceClient(server.Client(), server.URL)
}

func TestListScenariosHappyPath(t *testing.T) {
	scenariosSvc := &fakeScenarios{
		root: "/tmp/v",
		list: []scenarios.Summary{
			{ID: "alpha", DisplayName: "Alpha", Path: "/tmp/v/scenarios/alpha", FlowCount: 2},
		},
	}
	client := newClient(t, scenariosSvc, nil)

	resp, err := client.ListScenarios(context.Background(), connect.NewRequest(&scenariosv1.ListScenariosRequest{}))
	require.NoError(t, err)
	require.Equal(t, "/tmp/v", resp.Msg.VrooliRoot)
	require.Len(t, resp.Msg.Scenarios, 1)
	require.Equal(t, "alpha", resp.Msg.Scenarios[0].Id)
	require.Equal(t, int32(2), resp.Msg.Scenarios[0].FlowCount)
}

func TestGetScenarioNotFound(t *testing.T) {
	client := newClient(t, &fakeScenarios{detailErr: scenarios.ErrScenarioNotFound}, nil)
	_, err := client.GetScenario(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioRequest{Id: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGenerateScenarioArtifactsStreams(t *testing.T) {
	scenariosSvc := &fakeScenarios{
		detail: scenarios.Detail{
			Summary: scenarios.Summary{ID: "s1", Path: "/tmp/v/s1"},
			Flows:   []flows.Summary{{FlowID: "fa"}, {FlowID: "fb"}},
		},
	}
	artifactsSvc := &fakeArtifacts{
		progress: []progressItem{
			{flowID: "fa", report: artifacts.Report{FlowID: "fa", Status: artifacts.StatusFresh}},
			{flowID: "fb", err: errors.New("boom")},
		},
	}
	client := newClient(t, scenariosSvc, artifactsSvc)

	stream, err := client.GenerateScenarioArtifacts(context.Background(), connect.NewRequest(&scenariosv1.GenerateScenarioArtifactsRequest{ScenarioId: "s1"}))
	require.NoError(t, err)
	var received []*scenariosv1.GenerateScenarioArtifactsResponse
	for stream.Receive() {
		received = append(received, stream.Msg())
	}
	require.NoError(t, stream.Err())
	require.Len(t, received, 2)
	require.Equal(t, "fa", received[0].FlowId)
	require.NotNil(t, received[0].Report)
	require.Equal(t, "fb", received[1].FlowId)
	require.Equal(t, "boom", received[1].ErrorMessage)
}

// --- fakes ---

type fakeScenarios struct {
	root      string
	list      []scenarios.Summary
	detail    scenarios.Detail
	detailErr error
}

func (f *fakeScenarios) List() ([]scenarios.Summary, error) { return f.list, nil }
func (f *fakeScenarios) Detail(_ string) (scenarios.Detail, error) {
	if f.detailErr != nil {
		return scenarios.Detail{}, f.detailErr
	}
	return f.detail, nil
}
func (f *fakeScenarios) Root() string { return f.root }

type progressItem struct {
	flowID string
	report artifacts.Report
	err    error
}

type fakeArtifacts struct {
	progress []progressItem
}

func (f *fakeArtifacts) GenerateForScenarioStream(_ context.Context, _ string, onProgress func(string, artifacts.Report, error) error) error {
	for _, p := range f.progress {
		if err := onProgress(p.flowID, p.report, p.err); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeArtifacts) ClearForScenario(_ string) ([]artifacts.ClearResult, error) {
	return nil, nil
}
