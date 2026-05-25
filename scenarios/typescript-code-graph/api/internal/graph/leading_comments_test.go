package graph_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/sidecar"
	sidecarmocks "typescript-code-graph/internal/sidecar/mocks"
)

// TestLeadingCommentsPassThroughVerbatim is the load-bearing rcl
// contract (REQ-P0-003): every declaration node carries its
// leading_comments[] verbatim from the sidecar — no whitespace trim,
// no JSDoc parsing, no normalization. react-component-library's
// migration off regex parsing depends on this byte-for-byte.
func TestLeadingCommentsPassThroughVerbatim(t *testing.T) {
	wantComments := []string{
		"/** @vrooliWidget kind=card */",
		"// inline",
	}

	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		ExtractFn: func(ctx context.Context, p string) (sidecar.ExtractResult, error) {
			return sidecar.ExtractResult{
				Graph: sidecar.RawGraph{
					Nodes: []sidecar.RawNode{{
						ID:              "ts_component:ts_module:root:WidgetCard",
						Kind:            201,
						Name:            "WidgetCard",
						Path:            "src/Widget.tsx",
						LeadingComments: wantComments,
					}},
				},
			}, nil
		},
	}

	svc := graph.NewService(fake, graph.NewPathMutex())
	out, err := svc.Extract(context.Background(), graph.ExtractInput{ScenarioPath: "/abs/proj"})
	require.NoError(t, err)
	require.Len(t, out.Graph.Nodes, 1)

	got := out.Graph.Nodes[0].LeadingComments
	require.Equal(t, wantComments, got,
		"leading_comments[] must survive Normalize byte-for-byte; "+
			"any change here breaks the react-component-library migration")
}

// TestLeadingCommentsHashStability proves the hash includes the
// comments — REQ-P1-005 cache-hit detection breaks if a JSDoc edit
// goes unnoticed.
func TestLeadingCommentsHashStability(t *testing.T) {
	mk := func(comments []string) sidecar.RawGraph {
		return sidecar.RawGraph{Nodes: []sidecar.RawNode{{
			ID: "n", Kind: 201, LeadingComments: comments,
		}}}
	}
	g1 := graph.Normalize(mk([]string{"/** a */"}))
	g2 := graph.Normalize(mk([]string{"/** b */"}))
	require.NotEqual(t, graph.GraphHash(g1), graph.GraphHash(g2))
}

// TestLeadingComments_RealSidecar_JsdocTags is the end-to-end load-bearing
// rcl contract test (REQ-P0-003): runs the real Node sidecar against the
// ts-jsdoc-tags fixture and asserts that JSDoc tags (`@vrooliWidget
// kind=card`, `@vrooliComponent name=Heading`) plus plain `//` and `/* */`
// comments survive verbatim in `leading_comments[]`. If this regresses,
// react-component-library's migration off regex parsing breaks.
func TestLeadingComments_RealSidecar_JsdocTags(t *testing.T) {
	sup := startRealSupervisor(t)
	svc := graph.NewService(sup, graph.NewPathMutex())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fixDir := fixtureAbsPath(t, "ts-jsdoc-tags")
	out, err := svc.Extract(ctx, graph.ExtractInput{ScenarioPath: fixDir})
	require.NoError(t, err)

	wantByID := map[string][]string{
		"ts_component:src/widget.tsx:FeaturedCard": {"/** @vrooliWidget kind=card */"},
		"ts_component:src/widget.tsx:Heading":      {"/** @vrooliComponent name=Heading */"},
		"ts_function:src/widget.tsx:plainHelper":   {"// single-line"},
		"ts_const:src/widget.tsx:CONFIG":           {"/* block */"},
	}

	gotByID := make(map[string][]string, len(out.Graph.Nodes))
	for _, n := range out.Graph.Nodes {
		gotByID[n.ID] = n.LeadingComments
	}
	for id, want := range wantByID {
		require.Contains(t, gotByID, id, "node %s missing from extracted graph", id)
		require.Equal(t, want, gotByID[id],
			"leading_comments verbatim contract broke for %s — "+
				"react-component-library migration depends on byte-for-byte fidelity", id)
	}
}
