package ontology

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	ontologyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology"
	ontologyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology/ontology_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client ontologyconnect.OntologyServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: ontologyconnect.NewOntologyServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCapabilities(ctx cliapp.RunContext) error {
	kind, err := capabilityKind(ctx.Flag("kind"))
	if err != nil {
		return err
	}
	resp, err := h.client.ListCapabilities(context.Background(), connect.NewRequest(&ontologyv1.ListCapabilitiesRequest{
		ParentId:           ctx.Flag("parent"),
		Kind:               kind,
		IncludeDescendants: flagBool(ctx.Flag("descendants")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list capabilities", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetCapabilities()))
	for _, capability := range resp.Msg.GetCapabilities() {
		results = append(results, capabilityLine(capability))
	}
	if len(results) == 0 {
		results = append(results, "(no capabilities)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d capability node(s).", len(resp.Msg.GetCapabilities()))},
		ResultsHeading: "Capabilities",
		Results:        results,
	})
}

func (h *handlers) getCapability(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCapability(context.Background(), connect.NewRequest(&ontologyv1.GetCapabilityRequest{Slug: ctx.Positional("slug")}))
	if err != nil {
		return cliapp.WrapAPIError("get capability", err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{capabilityLine(resp.Msg)},
	})
}

func (h *handlers) upsertCapability(ctx cliapp.RunContext) error {
	kind, err := capabilityKind(ctx.Flag("kind"))
	if err != nil {
		return err
	}
	resp, err := h.client.UpsertCapability(context.Background(), connect.NewRequest(&ontologyv1.UpsertCapabilityRequest{
		Capability: &ontologyv1.Capability{
			Slug:        ctx.Positional("slug"),
			Name:        ctx.Flag("name"),
			Description: ctx.Flag("description"),
			Kind:        kind,
			ParentId:    ctx.Flag("parent"),
			Importance:  flagFloat(ctx.Flag("importance")),
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("upsert capability", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{capabilityLine(resp.Msg)},
		Changes: []string{fmt.Sprintf("Capability %s saved.", resp.Msg.GetSlug())},
	})
}

func (h *handlers) removeCapability(ctx cliapp.RunContext) error {
	resp, err := h.client.DeleteCapability(context.Background(), connect.NewRequest(&ontologyv1.DeleteCapabilityRequest{Slug: ctx.Positional("slug")}))
	if err != nil {
		return cliapp.WrapAPIError("delete capability", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{ctx.Positional("slug")},
		Changes: []string{fmt.Sprintf("Deleted: %t.", resp.Msg.GetDeleted())},
	})
}

func (h *handlers) addEdge(ctx cliapp.RunContext) error {
	edgeType, err := edgeType(ctx.Flag("type"))
	if err != nil {
		return err
	}
	resp, err := h.client.UpsertCapabilityEdge(context.Background(), connect.NewRequest(&ontologyv1.UpsertCapabilityEdgeRequest{
		Edge: &ontologyv1.CapabilityEdge{FromId: ctx.Positional("from"), ToId: ctx.Positional("to"), Type: edgeType},
	}))
	if err != nil {
		return cliapp.WrapAPIError("add capability edge", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{edgeLine(resp.Msg)},
		Changes: []string{"Capability edge saved."},
	})
}

func (h *handlers) removeEdge(ctx cliapp.RunContext) error {
	edgeType, err := edgeType(ctx.Flag("type"))
	if err != nil {
		return err
	}
	resp, err := h.client.DeleteCapabilityEdge(context.Background(), connect.NewRequest(&ontologyv1.DeleteCapabilityEdgeRequest{
		Edge: &ontologyv1.CapabilityEdge{FromId: ctx.Positional("from"), ToId: ctx.Positional("to"), Type: edgeType},
	}))
	if err != nil {
		return cliapp.WrapAPIError("delete capability edge", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s -> %s", ctx.Positional("from"), ctx.Positional("to"))},
		Changes: []string{fmt.Sprintf("Deleted: %t.", resp.Msg.GetDeleted())},
	})
}

func (h *handlers) importTopology(ctx cliapp.RunContext) error {
	body, err := readFile(ctx.Flag("from-file"))
	if err != nil {
		return err
	}
	resp, err := h.client.ImportTopology(context.Background(), connect.NewRequest(&ontologyv1.ImportTopologyRequest{Json: body}))
	if err != nil {
		return cliapp.WrapAPIError("import topology", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("sectors=%d capabilities=%d edges=%d", resp.Msg.GetSectorsTotal(), resp.Msg.GetCapabilitiesTotal(), resp.Msg.GetEdgesTotal()),
		},
		Changes: []string{
			fmt.Sprintf("Imported sectors=%d capabilities=%d edges=%d.", resp.Msg.GetSectorsImported(), resp.Msg.GetCapabilitiesImported(), resp.Msg.GetEdgesImported()),
		},
	})
}

func (h *handlers) fulfill(ctx cliapp.RunContext) error {
	resp, err := h.client.LinkFulfillment(context.Background(), connect.NewRequest(&ontologyv1.LinkFulfillmentRequest{
		Fulfillment: &ontologyv1.Fulfillment{
			CapabilityId: ctx.Positional("capability"),
			ScenarioSlug: ctx.Positional("scenario"),
			Note:         ctx.Flag("note"),
		},
	}))
	if err != nil {
		return cliapp.WrapAPIError("link fulfillment", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fulfillmentLine(resp.Msg)},
		Changes: []string{"Fulfillment linked."},
	})
}

func (h *handlers) unfulfill(ctx cliapp.RunContext) error {
	resp, err := h.client.UnlinkFulfillment(context.Background(), connect.NewRequest(&ontologyv1.UnlinkFulfillmentRequest{
		CapabilityId: ctx.Positional("capability"),
		ScenarioSlug: ctx.Positional("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("unlink fulfillment", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s <- %s", ctx.Positional("capability"), ctx.Positional("scenario"))},
		Changes: []string{fmt.Sprintf("Deleted: %t.", resp.Msg.GetDeleted())},
	})
}

func (h *handlers) listFulfillments(ctx cliapp.RunContext) error {
	resp, err := h.client.ListFulfillments(context.Background(), connect.NewRequest(&ontologyv1.ListFulfillmentsRequest{
		CapabilityId: ctx.Flag("capability"),
		ScenarioSlug: ctx.Flag("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list fulfillments", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetFulfillments()))
	for _, fulfillment := range resp.Msg.GetFulfillments() {
		results = append(results, fulfillmentLine(fulfillment))
	}
	if len(results) == 0 {
		results = append(results, "(no fulfillments)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d fulfillment link(s).", len(resp.Msg.GetFulfillments()))},
		ResultsHeading: "Fulfillments",
		Results:        results,
	})
}

func (h *handlers) coverage(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCoverage(context.Background(), connect.NewRequest(&ontologyv1.GetCoverageRequest{IncludeSubtreeRollup: flagBool(ctx.Flag("subtree"))}))
	if err != nil {
		return cliapp.WrapAPIError("get coverage", err, nil)
	}
	results := []string{
		fmt.Sprintf("built=%d inflight=%d gap=%d unmapped=%d", resp.Msg.GetBuiltCapabilities(), resp.Msg.GetInflightCapabilities(), resp.Msg.GetGapCapabilities(), resp.Msg.GetUnmappedScenarios()),
		fmt.Sprintf("ontology=%.2f implementation=%.2f", resp.Msg.GetOntologyCompleteness(), resp.Msg.GetImplementationSituatedness()),
	}
	for _, sector := range resp.Msg.GetSectors() {
		results = append(results, fmt.Sprintf("%s built=%d inflight=%d gap=%d total=%d", sector.GetSectorSlug(), sector.GetBuiltCapabilities(), sector.GetInflightCapabilities(), sector.GetGapCapabilities(), sector.GetTotalCapabilities()))
	}
	if resp.Msg.GetGraphError() != "" {
		results = append(results, "graph error: "+resp.Msg.GetGraphError())
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d capability node(s), %d scenario node(s).", resp.Msg.GetTotalCapabilities(), resp.Msg.GetTotalScenarios())},
		ResultsHeading: "Coverage",
		Results:        results,
	})
}

func (h *handlers) focus(ctx cliapp.RunContext) error {
	limit, err := flagInt32(ctx.Flag("limit"))
	if err != nil {
		return err
	}
	resp, err := h.client.ListFocus(context.Background(), connect.NewRequest(&ontologyv1.ListFocusRequest{Limit: limit}))
	if err != nil {
		return cliapp.WrapAPIError("list focus", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetItems()))
	for _, item := range resp.Msg.GetItems() {
		target := item.GetCapabilitySlug()
		if target == "" {
			target = strings.Join(item.GetRelatedScenarios(), ",")
		}
		results = append(results, fmt.Sprintf("%s %s score=%.2f downstream=%d", item.GetReason().String(), target, item.GetScore(), item.GetDownstreamDependents()))
	}
	if len(results) == 0 {
		results = append(results, "(no focus items)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d focus item(s).", len(resp.Msg.GetItems()))},
		ResultsHeading: "Focus",
		Results:        results,
	})
}

func (h *handlers) capabilityScenarios(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCapabilityScenarios(context.Background(), connect.NewRequest(&ontologyv1.GetCapabilityScenariosRequest{
		CapabilitySlug:     ctx.Positional("slug"),
		IncludeDescendants: flagBool(ctx.Flag("descendants")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get capability scenarios", err, nil)
	}
	results := []string{
		"built: " + strings.Join(resp.Msg.GetBuiltScenarios(), ","),
		"planned: " + strings.Join(resp.Msg.GetPlannedScenarios(), ","),
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s has %d fulfillment link(s).", resp.Msg.GetCapabilitySlug(), len(resp.Msg.GetFulfillments()))},
		ResultsHeading: "Scenarios",
		Results:        results,
	})
}

func (h *handlers) scenarioCapabilities(ctx cliapp.RunContext) error {
	resp, err := h.client.GetScenarioCapabilities(context.Background(), connect.NewRequest(&ontologyv1.GetScenarioCapabilitiesRequest{ScenarioSlug: ctx.Positional("slug")}))
	if err != nil {
		return cliapp.WrapAPIError("get scenario capabilities", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetCapabilities()))
	for _, capability := range resp.Msg.GetCapabilities() {
		results = append(results, capabilityLine(capability))
	}
	if len(results) == 0 {
		results = append(results, "(no capabilities)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s fulfills %d capability node(s).", resp.Msg.GetScenarioSlug(), len(resp.Msg.GetCapabilities()))},
		ResultsHeading: "Capabilities",
		Results:        results,
	})
}

func (h *handlers) overlay(ctx cliapp.RunContext) error {
	resp, err := h.client.DescribeOverlayGraph(context.Background(), connect.NewRequest(&ontologyv1.DescribeOverlayGraphRequest{
		IncludeImplementation: flagBool(ctx.Flag("implementation")),
		IncludeOntology:       flagBool(ctx.Flag("ontology")),
		IncludeFulfillment:    flagBool(ctx.Flag("fulfillment")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("describe overlay graph", err, nil)
	}
	graph := resp.Msg.GetGraph()
	results := make([]string, 0, len(graph.GetEdges()))
	for _, edge := range graph.GetEdges() {
		results = append(results, fmt.Sprintf("%s -> %s", edge.GetFromScenario(), edge.GetToScenario()))
	}
	if len(results) == 0 {
		results = append(results, "(no edges)")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Overlay graph: %d node(s), %d edge(s).", len(graph.GetNodes()), len(graph.GetEdges()))},
		ResultsHeading: "Edges",
		Results:        results,
	})
}

func capabilityLine(capability *ontologyv1.Capability) string {
	parent := capability.GetParentId()
	if parent == "" {
		parent = "root"
	}
	return fmt.Sprintf("%s [%s parent=%s]", capability.GetSlug(), capability.GetKind().String(), parent)
}

func edgeLine(edge *ontologyv1.CapabilityEdge) string {
	return fmt.Sprintf("%s -> %s [%s]", edge.GetFromId(), edge.GetToId(), edge.GetType().String())
}

func fulfillmentLine(fulfillment *ontologyv1.Fulfillment) string {
	return fmt.Sprintf("%s <- %s", fulfillment.GetCapabilityId(), fulfillment.GetScenarioSlug())
}

func capabilityKind(value string) (ontologyv1.CapabilityKind, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_UNSPECIFIED, nil
	case "sector":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR, nil
	case "capability":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPABILITY, nil
	case "component":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_COMPONENT, nil
	case "capstone":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPSTONE, nil
	case "simulation":
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_SIMULATION, nil
	default:
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_UNSPECIFIED, fmt.Errorf("unknown capability kind %q", value)
	}
}

func edgeType(value string) (ontologyv1.CapabilityEdgeType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "progression":
		return ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_PROGRESSION, nil
	case "decomposes":
		return ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_DECOMPOSES, nil
	case "requires":
		return ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_REQUIRES, nil
	default:
		return ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_UNSPECIFIED, fmt.Errorf("unknown edge type %q", value)
	}
}

func readFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("--from-file is required")
	}
	// #nosec G304 -- this CLI command intentionally reads the operator-supplied
	// topology file path from --from-file.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

func flagBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func flagInt32(value string) (int32, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("limit must be a 32-bit integer: %w", err)
	}
	return int32(parsed), nil
}

func flagFloat(value string) float64 {
	var out float64
	_, _ = fmt.Sscanf(strings.TrimSpace(value), "%f", &out)
	return out
}
