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

func TestSearchEnumLabelsCoverSupportedAndUnknownValues(t *testing.T) {
	for value, want := range map[searchv1.SemanticLegState]string{
		searchv1.SemanticLegState_SEMANTIC_LEG_STATE_READY:    "ready",
		searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED: "degraded",
		searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED: "disabled",
		searchv1.SemanticLegState(99):                         "unspecified",
	} {
		if got := semanticLegLabel(value); got != want {
			t.Errorf("semanticLegLabel(%v)=%q, want %q", value, got, want)
		}
	}
	for value, want := range map[searchv1.SearchKind]string{
		searchv1.SearchKind_SEARCH_KIND_WORKSPACE: "workspace", searchv1.SearchKind_SEARCH_KIND_PAGE: "page",
		searchv1.SearchKind_SEARCH_KIND_DISPLAY: "display", searchv1.SearchKind_SEARCH_KIND_STATE: "state",
		searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE: "page-type", searchv1.SearchKind_SEARCH_KIND_ASSET: "asset",
		searchv1.SearchKind_SEARCH_KIND_COMMAND: "command", searchv1.SearchKind_SEARCH_KIND_EVIDENCE: "evidence",
		searchv1.SearchKind(99): "unspecified",
	} {
		if got := searchKindLabel(value); got != want {
			t.Errorf("searchKindLabel(%v)=%q, want %q", value, got, want)
		}
	}
	for value, want := range map[searchv1.SearchAvailability]string{
		searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE:   "available",
		searchv1.SearchAvailability_SEARCH_AVAILABILITY_DRAFT:       "draft",
		searchv1.SearchAvailability_SEARCH_AVAILABILITY_PLANNED:     "planned",
		searchv1.SearchAvailability_SEARCH_AVAILABILITY_UNAVAILABLE: "unavailable",
		searchv1.SearchAvailability(99):                             "",
	} {
		if got := availabilityLabel(value); got != want {
			t.Errorf("availabilityLabel(%v)=%q, want %q", value, got, want)
		}
	}
}

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

func TestSearchSuggestAndReindexCommands(t *testing.T) {
	var gotSuggest *searchv1.SuggestRequest
	var gotReindex *searchv1.ReindexRequest
	mux := http.NewServeMux()
	mux.Handle(searchconnect.SearchServiceSuggestProcedure, connect.NewUnaryHandlerSimple(
		searchconnect.SearchServiceSuggestProcedure,
		func(_ context.Context, req *searchv1.SuggestRequest) (*searchv1.SuggestResponse, error) {
			gotSuggest = req
			return &searchv1.SuggestResponse{Suggestions: []string{"dashboard shell", "dashboard card"}}, nil
		},
	))
	mux.Handle(searchconnect.SearchServiceReindexProcedure, connect.NewUnaryHandlerSimple(
		searchconnect.SearchServiceReindexProcedure,
		func(_ context.Context, req *searchv1.ReindexRequest) (*searchv1.ReindexResponse, error) {
			gotReindex = req
			return &searchv1.ReindexResponse{DocumentCount: 12, IndexedAt: "2026-11-20T10:00:00Z"}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)

	suggest := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "suggest")
	out, err := cliapptest.RunCommand(t, suggest, core, "dash", "--limit", "2")
	require.NoError(t, err)
	require.Equal(t, "dash", gotSuggest.GetQuery())
	require.Equal(t, int32(2), gotSuggest.GetLimit())
	require.Contains(t, out, "dashboard shell")
	require.Contains(t, out, "search query <term>")

	reindex := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "reindex")
	out, err = cliapptest.RunCommand(t, reindex, core, "--force")
	require.NoError(t, err)
	require.True(t, gotReindex.GetForce())
	require.Contains(t, out, "Reindexed 12 document(s)")
	require.Contains(t, out, "force=true")
}

func TestSearchKindNormalizationAndLimitBoundaries(t *testing.T) {
	kind, err := parseSearchKind(" SEARCH_KIND_PAGE_TYPE ")
	require.NoError(t, err)
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, kind)
	kind, err = parseSearchKind("")
	require.NoError(t, err)
	require.Equal(t, searchv1.SearchKind_SEARCH_KIND_UNSPECIFIED, kind)
	_, err = parseSearchKind("not-a-kind")
	require.ErrorContains(t, err, "--kind must be one of")

	limit, err := parseLimit("2147483647")
	require.NoError(t, err)
	require.Equal(t, int32(2147483647), limit)
	for _, raw := range []string{"-1", "2147483648", "many"} {
		_, err = parseLimit(raw)
		require.ErrorContains(t, err, "--limit must be a non-negative integer")
	}
}

func TestSearchReportRendersResultMetadataAndDegradedStatus(t *testing.T) {
	report := (&handlers{}).queryReport(nil, &searchv1.SearchResponse{
		Query: "ui", Results: []*searchv1.SearchResult{
			nil,
			{
				Id: "fallback-title", Kind: searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, Subtitle: "built-in", Score: 0.875,
				Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_DRAFT, Href: " /pages/type ",
			},
		},
		Status: &searchv1.SearchStatus{Stale: true, LastError: " vector timeout ", SemanticLeg: searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED},
	})
	require.Contains(t, report.Results[0], "[page-type] fallback-title")
	require.Contains(t, report.Results[0], "— built-in")
	require.Contains(t, report.Results[0], "score 0.875")
	require.Contains(t, report.Results[0], "[draft]")
	require.Contains(t, report.Results[0], "→ /pages/type")
	require.Contains(t, report.Results[1], "Semantic leg: degraded")
	require.Contains(t, report.Summary[1], "stale=true")
	require.Contains(t, report.Summary[1], "last-error=vector timeout")
}
