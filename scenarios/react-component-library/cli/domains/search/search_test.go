package search

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search/search_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestSearchManifestCoverage(t *testing.T) {
	raw := cliapptest.ReadManifest(t)
	cliapp.RequireProtoServiceCoverage(t, raw, searchv1.File_react_component_library_v1_search_search_proto, "SearchService")
}

func TestSearchGroupRegistersEveryManifestCommand(t *testing.T) {
	raw := cliapptest.ReadManifest(t)
	group, err := Register(&cliapp.ScenarioApp{}, raw)
	require.NoError(t, err)

	manifest, err := cliapp.ParseManifest(raw)
	require.NoError(t, err)
	declared := manifest.FindGroup(GroupName)
	require.NotNil(t, declared)
	require.Len(t, group.Subcommands, len(declared.Commands))

	for _, manifestCommand := range declared.Commands {
		command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, manifestCommand.Name)
		require.NotNilf(t, command.RunCtx, "command %s/%s has no bound handler", GroupName, manifestCommand.Name)
		require.NotNilf(t, manifestCommand.Architecture, "command %s/%s declares no architecture", GroupName, manifestCommand.Name)
		require.Equalf(t, manifestCommand.Architecture.Primitive, string(command.PrimitiveEvidence()),
			"command %s/%s primitive evidence does not match its manifest declaration", GroupName, manifestCommand.Name)
	}
}

func TestSearchQueryForwardsAskKindsAndLimit(t *testing.T) {
	var got *searchv1.SearchRequest
	mux := http.NewServeMux()
	mux.Handle(searchconnect.SearchServiceSearchProcedure, connect.NewUnaryHandlerSimple(
		searchconnect.SearchServiceSearchProcedure,
		func(_ context.Context, req *searchv1.SearchRequest) (*searchv1.SearchResponse, error) {
			got = req
			return &searchv1.SearchResponse{
				Query:  "dashboard",
				Total:  1,
				TookMs: 4,
				Results: []*searchv1.SearchResult{{
					Id:    "asset-1",
					Kind:  searchv1.SearchKind_SEARCH_KIND_ASSET,
					Title: "Dashboard shell",
					Href:  "/library/assets/asset-1",
				}},
				Status: &searchv1.SearchStatus{
					Indexed:       true,
					DocumentCount: 42,
					SemanticLeg:   searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED,
				},
			}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "query")

	out, err := cliapptest.RunCommand(t, command, core, "dashboard", "--ask", "--kind", "asset", "--kind", "workspace", "--limit", "5")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "dashboard", got.GetQuery())
	require.True(t, got.GetAsk())
	require.Equal(t, []searchv1.SearchKind{
		searchv1.SearchKind_SEARCH_KIND_ASSET,
		searchv1.SearchKind_SEARCH_KIND_WORKSPACE,
	}, got.GetKinds())
	require.Equal(t, int32(5), got.GetLimit())

	require.Contains(t, out, "Ranked results")
	require.Contains(t, out, "[asset] Dashboard shell")
	require.Contains(t, out, "semantic-leg=disabled")
	require.Contains(t, out, "Semantic leg: disabled — results are lexical only.")
}

func TestSearchStatusNeverClaimsSemanticSearchIsActive(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(searchconnect.SearchServiceGetSearchStatusProcedure, connect.NewUnaryHandlerSimple(
		searchconnect.SearchServiceGetSearchStatusProcedure,
		func(_ context.Context, _ *searchv1.GetSearchStatusRequest) (*searchv1.SearchStatus, error) {
			return &searchv1.SearchStatus{
				Indexed:       true,
				DocumentCount: 42,
				IndexedAt:     "2026-11-19T10:00:00Z",
				SemanticLeg:   searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED,
			}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "status")

	out, err := cliapptest.RunCommand(t, command, core)
	require.NoError(t, err)
	require.Contains(t, out, "semantic_leg: disabled")
	require.Contains(t, out, "results are lexical only")
	require.NotContains(t, out, "semantic search is active")
}
