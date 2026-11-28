package sketch

import (
	"context"
	"net/http"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	experienceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience/experience_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
)

// handlerPathServer records the wire-level requests made through registered
// commands, so these tests exercise manifest binding, Connect transport, and
// human rendering together.
type handlerPathServer struct {
	experienceconnect.UnimplementedExperienceServiceHandler
	get        *sketchv1.GetSketchRequest
	put        *sketchv1.PutSketchRequest
	verify     *sketchv1.VerifySketchRequest
	importPage *sketchv1.ImportPageRequest
}

func (s *handlerPathServer) GetSketch(_ context.Context, req *connect.Request[sketchv1.GetSketchRequest]) (*connect.Response[sketchv1.GetSketchResponse], error) {
	s.get = req.Msg
	return connect.NewResponse(&sketchv1.GetSketchResponse{
		Sketch:       &sharedv1.Sketch{Viewport: "mobile", Notes: []*sharedv1.SketchNote{{Text: "keep me"}}},
		DocumentPath: "design/shop/home.sketch.json", ContentHash: "revision-17",
	}), nil
}

func (s *handlerPathServer) PutSketch(_ context.Context, req *connect.Request[sketchv1.PutSketchRequest]) (*connect.Response[sketchv1.PutSketchResponse], error) {
	s.put = req.Msg
	return connect.NewResponse(&sketchv1.PutSketchResponse{Changed: true, DocumentPath: "design/shop/home.sketch.json", ContentHash: "revision-18"}), nil
}

func (s *handlerPathServer) VerifySketch(_ context.Context, req *connect.Request[sketchv1.VerifySketchRequest]) (*connect.Response[sketchv1.VerifySketchResponse], error) {
	s.verify = req.Msg
	return connect.NewResponse(&sketchv1.VerifySketchResponse{Passes: true, Coverage: &sketchv1.Coverage{Resolved: 3, Total: 4, ResolvedLocal: 1, BuiltPercent: 75, LibraryBacked: 2, Local: 2}}), nil
}

func (s *handlerPathServer) ImportPage(_ context.Context, req *connect.Request[sketchv1.ImportPageRequest]) (*connect.Response[sketchv1.ImportPageResponse], error) {
	s.importPage = req.Msg
	return connect.NewResponse(&sketchv1.ImportPageResponse{Items: []*sketchv1.DecompositionItem{{}, {}}, Templates: []*sketchv1.TemplateCandidate{{}}, Written: req.Msg.GetWrite()}), nil
}

func runRegisteredSketchCommand(t *testing.T, server *handlerPathServer, name string, args ...string) string {
	t.Helper()
	path, handler := experienceconnect.NewExperienceServiceHandler(server)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	core := cliapptest.NewTestApp(t, mux)
	manifest, err := os.ReadFile("../../manifest.json")
	require.NoError(t, err)
	group, err := Register(core, manifest)
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, "sketch", name)
	output, err := cliapptest.RunCommand(t, command, core, args...)
	require.NoError(t, err)
	return output
}

func TestGetRegisteredCommandCallsExperienceAndRendersDocument(t *testing.T) {
	server := &handlerPathServer{}
	output := runRegisteredSketchCommand(t, server, "get", "shop/home")
	require.Equal(t, "shop", server.get.GetTarget().GetScenario())
	require.Equal(t, "home", server.get.GetTarget().GetPage())
	require.Contains(t, output, "Sketch: shop/home")
	require.Contains(t, output, "Document: design/shop/home.sketch.json")
}

func TestSetRegisteredCommandPreservesExistingSketchAndUsesReadRevision(t *testing.T) {
	server := &handlerPathServer{}
	output := runRegisteredSketchCommand(t, server, "set", "shop/home", "--template", "layout.dashboard", "--version", "2.1.0", "--viewport", "tablet")
	require.Equal(t, "revision-17", server.put.GetExpectedContentHash())
	require.Equal(t, "layout.dashboard", server.put.GetSketch().GetTemplate().GetAsset())
	require.Equal(t, "2.1.0", server.put.GetSketch().GetTemplate().GetVersion())
	require.Equal(t, "tablet", server.put.GetSketch().GetViewport())
	require.Len(t, server.put.GetSketch().GetNotes(), 1, "set should preserve fields not selected by flags")
	require.Contains(t, output, "Sketch written: shop/home")
	require.Contains(t, output, "Document: design/shop/home.sketch.json")
}

func TestVerifyRegisteredCommandRendersCoverage(t *testing.T) {
	server := &handlerPathServer{}
	output := runRegisteredSketchCommand(t, server, "verify", "shop/home")
	require.Equal(t, "shop", server.verify.GetTarget().GetScenario())
	require.Contains(t, output, "Sketch verification: shop/home")
	require.Contains(t, output, "Proven source bindings: 3/4 (1 custom)")
	require.Contains(t, output, "Built coverage: 75.00%")
	require.Contains(t, output, "Passes: true")
}

func TestImportRegisteredCommandReadsRevisionAndRendersCounts(t *testing.T) {
	server := &handlerPathServer{}
	output := runRegisteredSketchCommand(t, server, "import", "shop/home", "--write")
	require.Equal(t, "revision-17", server.importPage.GetExpectedContentHash())
	require.Equal(t, "shop", server.importPage.GetTarget().GetScenario())
	require.True(t, server.importPage.GetWrite())
	require.Contains(t, output, "Imported: shop/home")
	require.Contains(t, output, "Items: 2; templates: 1; written: true")
}
