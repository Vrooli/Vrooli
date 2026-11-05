package components

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"
	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
	componenttestsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests/componenttests_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type componentTestService struct {
	componenttestsconnect.UnimplementedComponentTestsServiceHandler
	response *componenttestsv1.RunComponentTestResponse
	err      error
}

func (s *componentTestService) RunComponentTest(context.Context, *connect.Request[componenttestsv1.RunComponentTestRequest]) (*connect.Response[componenttestsv1.RunComponentTestResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.response), nil
}

func componentTestCommandHandler(t *testing.T, svc *componentTestService) (cliapp.Command, *cliapp.ScenarioApp) {
	t.Helper()
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	core := cliapptest.NewTestApp(t, componentTestMux(t, svc))
	group, err := Register(core, manifest)
	require.NoError(t, err)
	for _, command := range group.Subcommands {
		if command.Name == "test" {
			return command, core
		}
	}
	t.Fatal("components test command missing from manifest")
	return cliapp.Command{}, nil
}

func componentTestMux(t *testing.T, svc *componentTestService) http.Handler {
	t.Helper()
	componentsPath, componentsHandler := componentsconnect.NewComponentsServiceHandler(&componentsService{})
	testsPath, testsHandler := componenttestsconnect.NewComponentTestsServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(componentsPath, componentsHandler)
	mux.Handle(testsPath, testsHandler)
	return mux
}

func componentTestResponse(verdict string) *componenttestsv1.RunComponentTestResponse {
	if verdict == "missing" {
		return &componenttestsv1.RunComponentTestResponse{}
	}
	return &componenttestsv1.RunComponentTestResponse{
		Report: &componenttestsv1.ComponentTestReport{
			Id: "report-123", RootLibraryId: "templates.collection-page", RootVersion: "1.7.3",
			Verdict: verdict, SourceRevision: "source-sha", Results: []*componenttestsv1.ComponentTestResult{{
				Stage: "experience_evidence", AssetLibraryId: "templates.collection-page", Version: "1.7.3",
				Verdict: verdict, Message: "story evidence retained", Remediation: "inspect the story result",
			}}, Artifacts: []*componenttestsv1.ComponentTestArtifact{{
				Kind: "screenshot", Label: "desktop", StoryId: "service-catalog", AssetLibraryId: "templates.collection-page", Version: "1.7.3", Reference: "artifact://report-123/desktop.png",
			}},
		}, Reused: true, SourceRevision: "source-sha",
	}
}

func runComponentTestCommand(t *testing.T, svc *componentTestService, jsonMode bool) (string, error) {
	t.Helper()
	command, core := componentTestCommandHandler(t, svc)
	require.NotNil(t, command.RunCtx)
	ctx, out := cliapptest.NewCapturedRunContext(core, command.Args, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "templates.collection-page"},
		Flags:       map[string]string{"version": "1.7.3"}, JSON: jsonMode,
	})
	err := command.RunCtx(ctx)
	return out.String(), err
}

func TestComponentsTestCommandExitFollowsDurableReportAfterRendering(t *testing.T) {
	for _, tc := range []struct {
		verdict       string
		wantErr       string
		missingReport bool
	}{
		{verdict: "passed"},
		{verdict: "failed", wantErr: "component test failed: report report-123"},
		{verdict: "blocked", wantErr: "component test blocked: report report-123"},
		{verdict: "unknown", wantErr: `unrecognized verdict "unknown"`},
		{verdict: "missing", wantErr: "server returned no component test report", missingReport: true},
	} {
		t.Run(tc.verdict+"/json", func(t *testing.T) {
			out, err := runComponentTestCommand(t, &componentTestService{response: componentTestResponse(tc.verdict)}, true)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			var decoded componenttestsv1.RunComponentTestResponse
			require.NoError(t, protojson.Unmarshal([]byte(out), &decoded))
			if tc.missingReport {
				require.Nil(t, decoded.GetReport())
				return
			}
			require.Equal(t, tc.verdict, decoded.GetReport().GetVerdict())
			require.Equal(t, "source-sha", decoded.GetSourceRevision())
			require.True(t, decoded.GetReused())
			require.Len(t, decoded.GetReport().GetResults(), 1)
			require.Len(t, decoded.GetReport().GetArtifacts(), 1)
		})

		t.Run(tc.verdict+"/human", func(t *testing.T) {
			out, err := runComponentTestCommand(t, &componentTestService{response: componentTestResponse(tc.verdict)}, false)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tc.missingReport {
				require.Contains(t, out, "Component test response contained no report.")
				require.NotContains(t, out, "components test-show")
				return
			}
			require.Contains(t, out, "Component test reused revision source-s since report report-123 ("+tc.verdict+").")
			require.Contains(t, out, "experience_evidence")
			require.Contains(t, out, "templates.collection-page@1.7.3")
		})
	}
}

func TestComponentsTestCommandPreservesTransportError(t *testing.T) {
	transportErr := connect.NewError(connect.CodeUnavailable, errors.New("capture service unavailable"))
	out, err := runComponentTestCommand(t, &componentTestService{err: transportErr}, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "capture service unavailable")
	require.Empty(t, out, "no report should be emitted when the RPC itself failed")
}
