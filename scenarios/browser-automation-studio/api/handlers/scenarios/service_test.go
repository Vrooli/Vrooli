package scenarios

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/scenarios"
	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/scenarios/scenariosconnect"
)

type fakeDiscovery struct {
	listResult []scenarioport.ScenarioMetadata
	listErr    error

	resolveURL  string
	resolveInfo *scenarioport.PortInfo
	resolveErr  error

	statusResult string
	lastName     string
}

func (f *fakeDiscovery) List(context.Context) ([]scenarioport.ScenarioMetadata, error) {
	return f.listResult, f.listErr
}

func (f *fakeDiscovery) ResolveURL(_ context.Context, name string) (string, *scenarioport.PortInfo, error) {
	f.lastName = name
	return f.resolveURL, f.resolveInfo, f.resolveErr
}

func (f *fakeDiscovery) Status(_ context.Context, _ string) (string, error) {
	return f.statusResult, nil
}

func newTestClient(t *testing.T, d *fakeDiscovery) scenariosconnect.ScenariosServiceClient {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	mount := Module(Deps{Discovery: d, Logger: logger})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return scenariosconnect.NewScenariosServiceClient(srv.Client(), srv.URL)
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

func TestList_HappyPath(t *testing.T) {
	d := &fakeDiscovery{listResult: []scenarioport.ScenarioMetadata{
		{Name: "alpha", Description: "A", Status: "running"},
		{Name: "beta", Description: "", Status: "stopped"},
	}}
	client := newTestClient(t, d)

	resp, err := client.List(context.Background(), connect.NewRequest(&scenariosv1.ListScenariosRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Scenarios, 2)
	require.Equal(t, "alpha", resp.Msg.Scenarios[0].Name)
	require.Equal(t, "running", resp.Msg.Scenarios[0].Status)
	require.Equal(t, "beta", resp.Msg.Scenarios[1].Name)
}

func TestList_ErrorMapsToInternal(t *testing.T) {
	d := &fakeDiscovery{listErr: errors.New("boom")}
	client := newTestClient(t, d)

	_, err := client.List(context.Background(), connect.NewRequest(&scenariosv1.ListScenariosRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestGetPort_HappyPath(t *testing.T) {
	d := &fakeDiscovery{
		resolveURL:   "http://localhost:5123",
		resolveInfo:  &scenarioport.PortInfo{Name: "api", Port: 5123},
		statusResult: "running",
	}
	client := newTestClient(t, d)

	resp, err := client.GetPort(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioPortRequest{Name: "alpha"}))
	require.NoError(t, err)
	require.Equal(t, int32(5123), resp.Msg.Port)
	require.Equal(t, "running", resp.Msg.Status)
	require.Equal(t, "http://localhost:5123", resp.Msg.Url)
	require.Equal(t, "alpha", d.lastName)
}

func TestGetPort_EmptyNameRejected(t *testing.T) {
	d := &fakeDiscovery{}
	client := newTestClient(t, d)

	_, err := client.GetPort(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioPortRequest{Name: "  "}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGetPort_ResolveErrorMapsToInternal(t *testing.T) {
	d := &fakeDiscovery{resolveErr: errors.New("nope")}
	client := newTestClient(t, d)

	_, err := client.GetPort(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioPortRequest{Name: "alpha"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestModule_RequiresLogger(t *testing.T) {
	require.Panics(t, func() { Module(Deps{}) })
}
