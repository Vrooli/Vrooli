package graph

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph/graph_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client graphconnect.GraphServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: graphconnect.NewGraphServiceClient(httpClient, baseURL)}
}

func (h *handlers) describe(ctx cliapp.RunContext) error {
	resp, err := h.client.DescribeTechTree(context.Background(), connect.NewRequest(&graphv1.DescribeTechTreeRequest{
		ScenarioFilter:  splitCSV(ctx.Flag("scenarios")),
		StabilityFilter: ctx.Flag("stability"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("describe graph", err, nil)
	}
	return renderGraph(ctx, "Scenario interface graph", resp.Msg)
}

func (h *handlers) neighbors(ctx cliapp.RunContext) error {
	depth, err := parseOptionalInt32(ctx.Flag("depth"))
	if err != nil {
		return err
	}
	resp, err := h.client.GetNeighborhood(context.Background(), connect.NewRequest(&graphv1.GetNeighborhoodRequest{
		Scenario:       ctx.Positional("scenario"),
		Depth:          depth,
		ScenarioFilter: splitCSV(ctx.Flag("scenarios")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get graph neighborhood", err, nil)
	}
	return renderGraph(ctx, fmt.Sprintf("Neighborhood for %s", ctx.Positional("scenario")), resp.Msg)
}

func (h *handlers) path(ctx cliapp.RunContext) error {
	resp, err := h.client.FindPath(context.Background(), connect.NewRequest(&graphv1.FindPathRequest{
		FromScenario:   ctx.Positional("from"),
		ToScenario:     ctx.Positional("to"),
		ScenarioFilter: splitCSV(ctx.Flag("scenarios")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("find graph path", err, nil)
	}
	return renderGraph(ctx, fmt.Sprintf("Path from %s to %s", ctx.Positional("from"), ctx.Positional("to")), resp.Msg)
}

func (h *handlers) ancestors(ctx cliapp.RunContext) error {
	resp, err := h.client.ListAncestors(context.Background(), connect.NewRequest(&graphv1.ListAncestorsRequest{
		Scenario:       ctx.Positional("scenario"),
		ScenarioFilter: splitCSV(ctx.Flag("scenarios")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list graph ancestors", err, nil)
	}
	return renderGraph(ctx, fmt.Sprintf("Ancestors for %s", ctx.Positional("scenario")), resp.Msg)
}

func (h *handlers) export(ctx cliapp.RunContext) error {
	format, err := exportFormat(ctx.Flag("format"))
	if err != nil {
		return err
	}
	resp, err := h.client.ExportTechTree(context.Background(), connect.NewRequest(&graphv1.ExportTechTreeRequest{
		Format:          format,
		ScenarioFilter:  splitCSV(ctx.Flag("scenarios")),
		StabilityFilter: ctx.Flag("stability"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("export graph", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	_, err = fmt.Fprint(ctx.Stdout(), resp.Msg.GetContent())
	return err
}

func renderGraph(ctx cliapp.RunContext, title string, resp *graphv1.DescribeTechTreeResponse) error {
	if resp == nil || resp.Graph == nil {
		return fmt.Errorf("server returned no graph")
	}
	graph := resp.Graph
	results := make([]string, 0, len(graph.GetEdges()))
	for _, edge := range graph.GetEdges() {
		results = append(results, fmt.Sprintf("%s -> %s", edge.GetFromScenario(), edge.GetToScenario()))
	}
	if len(results) == 0 {
		results = append(results, "(no edges)")
	}
	for _, graphErr := range graph.GetErrors() {
		results = append(results, fmt.Sprintf("error: %s %s: %s", graphErr.GetSource(), graphErr.GetScenario(), graphErr.GetMessage()))
	}
	return cliapp.RenderProtoList(ctx, resp, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: %d node(s), %d edge(s).", title, len(graph.GetNodes()), len(graph.GetEdges()))},
		ResultsHeading: "Edges",
		Results:        results,
		RetrievalHints: []string{
			"`graph neighbors <scenario>` — inspect local dependencies",
			"`graph export --format dot` — export for Graphviz",
		},
	})
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseOptionalInt32(value string) (int32, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("depth must be an integer: %w", err)
	}
	return int32(parsed), nil
}

func exportFormat(value string) (graphv1.ExportFormat, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "text":
		return graphv1.ExportFormat_EXPORT_FORMAT_TEXT, nil
	case "dot":
		return graphv1.ExportFormat_EXPORT_FORMAT_DOT, nil
	case "json":
		return graphv1.ExportFormat_EXPORT_FORMAT_JSON, nil
	default:
		return graphv1.ExportFormat_EXPORT_FORMAT_UNSPECIFIED, fmt.Errorf("unknown export format %q (use text, dot, or json)", value)
	}
}
