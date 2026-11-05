package graph

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the API client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client graphconnect.GoCodeGraphServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: graphconnect.NewGoCodeGraphServiceClient(httpClient, baseURL),
	}
}

// extract calls the generated Connect-RPC GoCodeGraphService.Extract
// method. Output routing: human consumers see a ListReport summarizing
// node/edge counts and top-level packages; --json consumers see the
// proto-typed ExtractResponse wire shape.
func (h *handlers) extract(ctx cliapp.RunContext) error {
	path := ctx.Positional("path")
	includeVendor := ctx.BoolFlag("include-vendor")
	profileValue := ""
	if ctx.FlagDeclared("profile") {
		profileValue = ctx.Flag("profile")
	}
	profile := extractionProfile(profileValue)
	var packagePatterns []string
	if ctx.FlagDeclared("package-pattern") {
		packagePatterns = ctx.FlagValues("package-pattern")
	}

	resp, err := h.client.Extract(context.Background(), connect.NewRequest(&graphv1.ExtractRequest{
		ModulePath:      path,
		IncludeVendor:   includeVendor,
		Profile:         profile,
		PackagePatterns: append([]string(nil), packagePatterns...),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("extract %q", path), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no extract response")
	}

	nodes := 0
	edges := 0
	var packages []string
	if g := resp.Msg.GetGraph(); g != nil {
		nodes = len(g.GetNodes())
		edges = len(g.GetEdges())
		for _, n := range g.GetNodes() {
			if n.GetKind() == commonv1.NodeKind_NODE_KIND_PACKAGE {
				packages = append(packages, n.GetName())
			}
		}
	}
	sort.Strings(packages)
	if len(packages) > 10 {
		packages = packages[:10]
	}

	summary := []string{
		fmt.Sprintf(
			"Extracted %d node(s), %d edge(s) in %dms (hash=%s).",
			nodes, edges, resp.Msg.GetExtractionMs(), resp.Msg.GetGraphHash(),
		),
	}
	if w := len(resp.Msg.GetWarnings()); w > 0 {
		summary = append(summary, fmt.Sprintf("%d warning(s) returned.", w))
	}

	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Top-level packages",
		Results:        packages,
		RetrievalHints: []string{
			fmt.Sprintf("`rewrite plan <ops.json> --module-path %s` — plan a rewrite for this module", path),
		},
	})
}

func extractionProfile(value string) graphv1.ExtractionProfile {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "semantic":
		return graphv1.ExtractionProfile_EXTRACTION_PROFILE_SEMANTIC
	case "structural":
		return graphv1.ExtractionProfile_EXTRACTION_PROFILE_STRUCTURAL
	default:
		return graphv1.ExtractionProfile_EXTRACTION_PROFILE_FULL
	}
}

// listFixtures calls GoCodeGraphService.ListFixtures and renders the fixture
// directories the API can validate. Human consumers see one line per fixture
// with its expected-graph status; --json consumers see the wire shape.
func (h *handlers) listFixtures(ctx cliapp.RunContext) error {
	resp, err := h.client.ListFixtures(context.Background(), connect.NewRequest(&graphv1.ListFixturesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list fixtures", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no list-fixtures response")
	}

	var results []string
	for _, f := range resp.Msg.GetFixtures() {
		status := "no expected graph"
		if f.GetHasExpected() {
			status = "expected graph present"
		}
		results = append(results, fmt.Sprintf("%s (%s) — %s", f.GetName(), f.GetPath(), status))
	}
	sort.Strings(results)

	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d fixture(s) available.", len(resp.Msg.GetFixtures()))},
		ResultsHeading: "Fixtures",
		Results:        results,
		RetrievalHints: []string{
			"`graph validate-fixture <name>` — re-extract and byte-compare a fixture",
		},
	})
}

// validateFixture calls GoCodeGraphService.ValidateFixture and reports
// pass/fail plus byte counts and (on failure) the diff. A non-passing result
// returns a non-nil error so CI/scripts can gate on the exit code.
func (h *handlers) validateFixture(ctx cliapp.RunContext) error {
	name := ctx.Positional("name")

	resp, err := h.client.ValidateFixture(context.Background(), connect.NewRequest(&graphv1.ValidateFixtureRequest{
		Name: name,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate fixture %q", name), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validate-fixture response")
	}

	verdict := "PASS"
	if !resp.Msg.GetPassed() {
		verdict = "FAIL"
	}
	summary := []string{
		fmt.Sprintf(
			"%s — fixture %q (expected %d bytes, actual %d bytes, hash=%s)",
			verdict, name, resp.Msg.GetExpectedBytes(), resp.Msg.GetActualBytes(), resp.Msg.GetGraphHash(),
		),
	}
	var results []string
	if !resp.Msg.GetPassed() && resp.Msg.GetDiff() != "" {
		results = append(results, resp.Msg.GetDiff())
	}

	if renderErr := cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Diff (expected vs actual)",
		Results:        results,
	}); renderErr != nil {
		return renderErr
	}
	if !resp.Msg.GetPassed() {
		return fmt.Errorf("fixture %q does not match its expected graph", name)
	}
	return nil
}
