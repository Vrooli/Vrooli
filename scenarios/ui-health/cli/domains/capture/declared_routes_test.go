package capture

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	repocontract "github.com/vrooli/repo-contract-go"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/search/search_v1connect"
)

func routeFixture(t *testing.T, app string) string {
	t.Helper()
	repo, err := repocontract.ResolveRepoRoot()
	require.NoError(t, err)
	root := t.TempDir()
	ui := filepath.Join(root, "scenarios", "sample", "ui", "src")
	require.NoError(t, os.MkdirAll(ui, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "routes.ts"), []byte(`export const appRoutes = { coverage: "/coverage", detail: "/assets/:id", capabilities: "/capabilities" } as const;`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "App.tsx"), []byte(app), 0644))
	compilerDir := filepath.Join(root, "scenarios", "ui-health", "ui", "node_modules")
	require.NoError(t, os.MkdirAll(compilerDir, 0755))
	require.NoError(t, os.Symlink(filepath.Join(repo, "scenarios", "ui-health", "ui", "node_modules", "typescript"), filepath.Join(compilerDir, "typescript")))
	return root
}

func TestDeclaredRouteUsesRouterDeclarations(t *testing.T) {
	root := routeFixture(t, `
import { Route, Routes } from "react-router-dom";
import { appRoutes as urls } from "./routes";
export default function App() { return <Routes><Route element={<Shell />}>
  {/* <Route path="/wrong" element={<CoveragePage />} /> */}
  <Route path={urls.coverage} element={<Page><CoveragePage /></Page>} />
  <Route path={urls.capabilities} element={<CapabilitiesPage />} />
  <Route path={urls.detail} element={<DetailPage />} />
  <Route path="/first" element={<AmbiguousPage />} />
  <Route path="/second" element={<AmbiguousPage />} />
  <Route path={unknown} element={<DynamicPage />} />
  <Route path="/parent"><Route path="/child" element={<NestedPage />} /></Route>
  <Route path="*" element={<NotFoundPage />} />
</Route></Routes> }`)
	for _, tc := range []struct{ name, want string }{
		{"CoveragePage", "/coverage"}, {"ui/src/pages/CoveragePage.tsx", "/coverage"},
		{"CapabilitiesPage", "/capabilities"}, {"DetailPage", ""}, {"DynamicPage", ""},
		{"MissingPage", ""}, {"NestedPage", ""}, {"NotFoundPage", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			route, err := declaredRoute(context.Background(), root, "sample", tc.name)
			require.NoError(t, err)
			require.Equal(t, tc.want, route)
		})
	}
	_, err := declaredRoute(context.Background(), root, "sample", "AmbiguousPage")
	require.ErrorContains(t, err, "multiple declared routes")
}

type routeSearchFake struct {
	searchconnect.SearchServiceClient
}

func (routeSearchFake) Search(_ context.Context, req *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	return connect.NewResponse(&searchv1.SearchResponse{Results: []*searchv1.SearchResult{{Scenario: "missing-scenario", FilePath: "ui/src/pages/UnknownPage.tsx", Score: 0.95}}}), nil
}

type routeAIFake struct{ aiconnect.AIServiceClient }

func (routeAIFake) GetDOMTree(context.Context, *connect.Request[aiv1.GetDOMTreeRequest]) (*connect.Response[aiv1.GetDOMTreeResponse], error) {
	return connect.NewResponse(&aiv1.GetDOMTreeResponse{}), nil
}

func TestResolvedSurfaceWithoutRouteCannotCaptureRoot(t *testing.T) {
	h := &handlers{search: routeSearchFake{}, ai: routeAIFake{}}
	resolved, err := h.resolve(context.Background(), "missing-scenario", "UnknownPage", "http://localhost:1234")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.Empty(t, resolved.Route)
	for _, rung := range []string{"selector", "declared-route", "observed-route", "corpus"} {
		require.Contains(t, resolved.FollowUp, rung)
	}
	_, _, err = h.captureOne(context.Background(), nil, "http://localhost:1234", resolved, "desktop", "light", false)
	require.ErrorContains(t, err, "surface route unresolved")
}

func TestConcreteRoutesPreserveExplicitRootAndQuery(t *testing.T) {
	for _, route := range []string{"/", "/coverage", "/assets/button?story=default"} {
		require.True(t, concreteRoute(route), route)
	}
	for _, route := range []string{"", "//example.com", "/assets/:id", "*", "https://example.com/coverage"} {
		require.False(t, concreteRoute(route), route)
	}
}
