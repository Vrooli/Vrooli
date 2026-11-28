package sketch

import (
	"context"
	"net/http"
	"os"
	"sync"
	"testing"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	experienceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience/experience_v1connect"
)

type experienceCommandServer struct {
	experienceconnect.UnimplementedExperienceServiceHandler
	mu     sync.Mutex
	placed *sketchv1.PlaceRequest
}

func (s *experienceCommandServer) GetSketch(context.Context, *connect.Request[sketchv1.GetSketchRequest]) (*connect.Response[sketchv1.GetSketchResponse], error) {
	return connect.NewResponse(&sketchv1.GetSketchResponse{ContentHash: "current-revision"}), nil
}

func (s *experienceCommandServer) Place(_ context.Context, req *connect.Request[sketchv1.PlaceRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.placed = req.Msg
	return connect.NewResponse(&sketchv1.PutSketchResponse{Changed: true, ContentHash: "placed-revision"}), nil
}

func TestPlaceCommandPreservesArgumentsThroughExperienceService(t *testing.T) {
	server := &experienceCommandServer{}
	path, handler := experienceconnect.NewExperienceServiceHandler(server)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	core := cliapptest.NewTestApp(t, mux)
	manifest, err := os.ReadFile("../../manifest.json")
	require.NoError(t, err)
	group, err := Register(core, manifest)
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, "sketch", "place")

	output, err := cliapptest.RunCommand(t, command, core, "demo/home", "main", "buttons.primary", "--version", "1.2.0", "--note", "retain the primary action")
	require.NoError(t, err)
	require.Contains(t, output, "Placed buttons.primary in main")
	require.Equal(t, "demo", server.placed.GetTarget().GetScenario())
	require.Equal(t, "home", server.placed.GetTarget().GetPage())
	require.Equal(t, "main", server.placed.GetRegion())
	require.Equal(t, "buttons.primary", server.placed.GetAsset())
	require.Equal(t, "1.2.0", server.placed.GetVersion())
	require.Equal(t, "retain the primary action", server.placed.GetNote())
	require.Equal(t, "current-revision", server.placed.GetExpectedContentHash())
}
