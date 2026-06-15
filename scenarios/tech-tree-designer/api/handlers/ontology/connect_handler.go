package ontology

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	ontologydomain "tech-tree-designer/internal/ontology"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	ontologyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology"
	ontologyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology/ontology_v1connect"
)

type Handler struct {
	ontologyconnect.UnimplementedOntologyServiceHandler
	service *ontologydomain.Service
}

func NewHandler(service *ontologydomain.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListCapabilities(ctx context.Context, req *connect.Request[ontologyv1.ListCapabilitiesRequest]) (*connect.Response[ontologyv1.ListCapabilitiesResponse], error) {
	capabilities, err := h.service.ListCapabilities(ctx, ontologydomain.CapabilityFilter{
		ParentID:           req.Msg.GetParentId(),
		Kind:               req.Msg.GetKind(),
		IncludeDescendants: req.Msg.GetIncludeDescendants(),
	})
	if err != nil {
		return nil, toConnectError(err)
	}
	resp := &ontologyv1.ListCapabilitiesResponse{Capabilities: make([]*ontologyv1.Capability, 0, len(capabilities))}
	for _, capability := range capabilities {
		resp.Capabilities = append(resp.Capabilities, capabilityToProto(capability))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) GetCapability(ctx context.Context, req *connect.Request[ontologyv1.GetCapabilityRequest]) (*connect.Response[ontologyv1.Capability], error) {
	capability, err := h.service.GetCapability(ctx, ontologydomain.CapabilityRef{ID: req.Msg.GetId(), Slug: req.Msg.GetSlug()})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(capabilityToProto(capability)), nil
}

func (h *Handler) UpsertCapability(ctx context.Context, req *connect.Request[ontologyv1.UpsertCapabilityRequest]) (*connect.Response[ontologyv1.Capability], error) {
	capability, err := h.service.UpsertCapability(ctx, capabilityFromProto(req.Msg.GetCapability()))
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(capabilityToProto(capability)), nil
}

func (h *Handler) DeleteCapability(ctx context.Context, req *connect.Request[ontologyv1.DeleteCapabilityRequest]) (*connect.Response[ontologyv1.DeleteCapabilityResponse], error) {
	deleted, err := h.service.DeleteCapability(ctx, ontologydomain.CapabilityRef{ID: req.Msg.GetId(), Slug: req.Msg.GetSlug()})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&ontologyv1.DeleteCapabilityResponse{Deleted: deleted}), nil
}

func (h *Handler) UpsertCapabilityEdge(ctx context.Context, req *connect.Request[ontologyv1.UpsertCapabilityEdgeRequest]) (*connect.Response[ontologyv1.CapabilityEdge], error) {
	edge, err := h.service.UpsertCapabilityEdge(ctx, edgeFromProto(req.Msg.GetEdge()))
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(edgeToProto(edge)), nil
}

func (h *Handler) DeleteCapabilityEdge(ctx context.Context, req *connect.Request[ontologyv1.DeleteCapabilityEdgeRequest]) (*connect.Response[ontologyv1.DeleteCapabilityEdgeResponse], error) {
	deleted, err := h.service.DeleteCapabilityEdge(ctx, edgeFromProto(req.Msg.GetEdge()))
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&ontologyv1.DeleteCapabilityEdgeResponse{Deleted: deleted}), nil
}

func (h *Handler) ImportTopology(ctx context.Context, req *connect.Request[ontologyv1.ImportTopologyRequest]) (*connect.Response[ontologyv1.ImportTopologyResponse], error) {
	result, err := h.service.ImportTopology(ctx, []byte(req.Msg.GetJson()))
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&ontologyv1.ImportTopologyResponse{
		SectorsImported:      result.SectorsImported,
		CapabilitiesImported: result.CapabilitiesImported,
		EdgesImported:        result.EdgesImported,
		SectorsTotal:         result.SectorsTotal,
		CapabilitiesTotal:    result.CapabilitiesTotal,
		EdgesTotal:           result.EdgesTotal,
	}), nil
}

func (h *Handler) LinkFulfillment(ctx context.Context, req *connect.Request[ontologyv1.LinkFulfillmentRequest]) (*connect.Response[ontologyv1.Fulfillment], error) {
	fulfillment, err := h.service.LinkFulfillment(ctx, fulfillmentFromProto(req.Msg.GetFulfillment()))
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(fulfillmentToProto(fulfillment)), nil
}

func (h *Handler) UnlinkFulfillment(ctx context.Context, req *connect.Request[ontologyv1.UnlinkFulfillmentRequest]) (*connect.Response[ontologyv1.UnlinkFulfillmentResponse], error) {
	deleted, err := h.service.UnlinkFulfillment(ctx, req.Msg.GetCapabilityId(), req.Msg.GetScenarioSlug())
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&ontologyv1.UnlinkFulfillmentResponse{Deleted: deleted}), nil
}

func (h *Handler) ListFulfillments(ctx context.Context, req *connect.Request[ontologyv1.ListFulfillmentsRequest]) (*connect.Response[ontologyv1.ListFulfillmentsResponse], error) {
	fulfillments, err := h.service.ListFulfillments(ctx, ontologydomain.FulfillmentFilter{
		CapabilityID: req.Msg.GetCapabilityId(),
		ScenarioSlug: req.Msg.GetScenarioSlug(),
	})
	if err != nil {
		return nil, toConnectError(err)
	}
	resp := &ontologyv1.ListFulfillmentsResponse{Fulfillments: make([]*ontologyv1.Fulfillment, 0, len(fulfillments))}
	for _, fulfillment := range fulfillments {
		resp.Fulfillments = append(resp.Fulfillments, fulfillmentToProto(fulfillment))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) GetCoverage(ctx context.Context, req *connect.Request[ontologyv1.GetCoverageRequest]) (*connect.Response[ontologyv1.CoverageSummary], error) {
	coverage, err := h.service.GetCoverage(ctx, ontologydomain.CoverageRequest{IncludeSubtreeRollup: req.Msg.GetIncludeSubtreeRollup()})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(coverageToProto(coverage)), nil
}

func (h *Handler) ListFocus(ctx context.Context, req *connect.Request[ontologyv1.ListFocusRequest]) (*connect.Response[ontologyv1.ListFocusResponse], error) {
	items, err := h.service.ListFocus(ctx, req.Msg.GetLimit())
	if err != nil {
		return nil, toConnectError(err)
	}
	resp := &ontologyv1.ListFocusResponse{Items: make([]*ontologyv1.FocusItem, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, focusItemToProto(item))
	}
	return connect.NewResponse(resp), nil
}

func (h *Handler) GetCapabilityScenarios(ctx context.Context, req *connect.Request[ontologyv1.GetCapabilityScenariosRequest]) (*connect.Response[ontologyv1.CapabilityScenarios], error) {
	result, err := h.service.GetCapabilityScenarios(ctx, ontologydomain.CapabilityRef{ID: req.Msg.GetCapabilityId(), Slug: req.Msg.GetCapabilitySlug()}, req.Msg.GetIncludeDescendants())
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(capabilityScenariosToProto(result)), nil
}

func (h *Handler) GetScenarioCapabilities(ctx context.Context, req *connect.Request[ontologyv1.GetScenarioCapabilitiesRequest]) (*connect.Response[ontologyv1.ScenarioCapabilities], error) {
	result, err := h.service.GetScenarioCapabilities(ctx, req.Msg.GetScenarioSlug())
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(scenarioCapabilitiesToProto(result)), nil
}

func (h *Handler) DescribeOverlayGraph(ctx context.Context, req *connect.Request[ontologyv1.DescribeOverlayGraphRequest]) (*connect.Response[ontologyv1.DescribeOverlayGraphResponse], error) {
	graph, err := h.service.DescribeOverlayGraph(ctx, ontologydomain.OverlayGraphRequest{
		IncludeImplementation: req.Msg.GetIncludeImplementation(),
		IncludeOntology:       req.Msg.GetIncludeOntology(),
		IncludeFulfillment:    req.Msg.GetIncludeFulfillment(),
	})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&ontologyv1.DescribeOverlayGraphResponse{Graph: overlayGraphToProto(graph)}), nil
}

func toConnectError(err error) error {
	var invalid ontologydomain.ErrInvalidArgument
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w", err))
	}
	var cycle ontologydomain.ErrCapabilityCycle
	if errors.As(err, &cycle) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w", err))
	}
	var missing ontologydomain.ErrCapabilityNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("%w", err))
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%w", err))
}

func capabilityFromProto(capability *ontologyv1.Capability) ontologydomain.Capability {
	if capability == nil {
		return ontologydomain.Capability{}
	}
	return ontologydomain.Capability{
		ID:          capability.GetId(),
		Slug:        capability.GetSlug(),
		Name:        capability.GetName(),
		Description: capability.GetDescription(),
		Kind:        capability.GetKind(),
		ParentID:    capability.GetParentId(),
		SortOrder:   capability.GetSortOrder(),
		Importance:  capability.GetImportance(),
	}
}

func capabilityToProto(capability ontologydomain.Capability) *ontologyv1.Capability {
	return &ontologyv1.Capability{
		Id:          capability.ID,
		Slug:        capability.Slug,
		Name:        capability.Name,
		Description: capability.Description,
		Kind:        capability.Kind,
		ParentId:    capability.ParentID,
		SortOrder:   capability.SortOrder,
		Importance:  capability.Importance,
		CreatedAt:   capability.CreatedAt.Format(timeFormat),
		UpdatedAt:   capability.UpdatedAt.Format(timeFormat),
	}
}

func edgeFromProto(edge *ontologyv1.CapabilityEdge) ontologydomain.CapabilityEdge {
	if edge == nil {
		return ontologydomain.CapabilityEdge{}
	}
	return ontologydomain.CapabilityEdge{FromID: edge.GetFromId(), ToID: edge.GetToId(), Type: edge.GetType()}
}

func edgeToProto(edge ontologydomain.CapabilityEdge) *ontologyv1.CapabilityEdge {
	return &ontologyv1.CapabilityEdge{FromId: edge.FromID, ToId: edge.ToID, Type: edge.Type}
}

func fulfillmentFromProto(fulfillment *ontologyv1.Fulfillment) ontologydomain.Fulfillment {
	if fulfillment == nil {
		return ontologydomain.Fulfillment{}
	}
	return ontologydomain.Fulfillment{
		CapabilityID: fulfillment.GetCapabilityId(),
		ScenarioSlug: fulfillment.GetScenarioSlug(),
		Note:         fulfillment.GetNote(),
	}
}

func fulfillmentToProto(fulfillment ontologydomain.Fulfillment) *ontologyv1.Fulfillment {
	return &ontologyv1.Fulfillment{
		CapabilityId: fulfillment.CapabilityID,
		ScenarioSlug: fulfillment.ScenarioSlug,
		Note:         fulfillment.Note,
		CreatedAt:    fulfillment.CreatedAt.Format(timeFormat),
	}
}

func coverageToProto(coverage ontologydomain.CoverageSummary) *ontologyv1.CoverageSummary {
	resp := &ontologyv1.CoverageSummary{
		BuiltCapabilities:          coverage.BuiltCapabilities,
		InflightCapabilities:       coverage.InflightCapabilities,
		GapCapabilities:            coverage.GapCapabilities,
		UnmappedScenarios:          coverage.UnmappedScenarios,
		TotalCapabilities:          coverage.TotalCapabilities,
		TotalScenarios:             coverage.TotalScenarios,
		OntologyCompleteness:       coverage.OntologyCompleteness,
		ImplementationSituatedness: coverage.ImplementationSituatedness,
		GraphError:                 coverage.GraphError,
		Sectors:                    make([]*ontologyv1.SectorCoverage, 0, len(coverage.Sectors)),
		Classifications:            make([]*ontologyv1.CoverageClassification, 0, len(coverage.Classifications)),
	}
	for _, sector := range coverage.Sectors {
		resp.Sectors = append(resp.Sectors, &ontologyv1.SectorCoverage{
			SectorId:             sector.SectorID,
			SectorSlug:           sector.SectorSlug,
			SectorName:           sector.SectorName,
			BuiltCapabilities:    sector.BuiltCapabilities,
			InflightCapabilities: sector.InflightCapabilities,
			GapCapabilities:      sector.GapCapabilities,
			TotalCapabilities:    sector.TotalCapabilities,
			OntologyCompleteness: sector.OntologyCompleteness,
		})
	}
	for _, classification := range coverage.Classifications {
		resp.Classifications = append(resp.Classifications, &ontologyv1.CoverageClassification{
			CapabilityId:      classification.CapabilityID,
			CapabilitySlug:    classification.CapabilitySlug,
			State:             classification.State,
			DirectlyFulfilled: classification.DirectlyFulfilled,
			SubtreeCovered:    classification.SubtreeCovered,
			BuiltScenarios:    classification.BuiltScenarios,
			PlannedScenarios:  classification.PlannedScenarios,
		})
	}
	return resp
}

func focusItemToProto(item ontologydomain.FocusItem) *ontologyv1.FocusItem {
	return &ontologyv1.FocusItem{
		CapabilityId:         item.CapabilityID,
		CapabilitySlug:       item.CapabilitySlug,
		CapabilityName:       item.CapabilityName,
		Reason:               item.Reason,
		Score:                item.Score,
		DownstreamDependents: item.DownstreamDependents,
		RelatedScenarios:     item.RelatedScenarios,
	}
}

func capabilityScenariosToProto(result ontologydomain.CapabilityScenarios) *ontologyv1.CapabilityScenarios {
	resp := &ontologyv1.CapabilityScenarios{
		CapabilityId:     result.CapabilityID,
		CapabilitySlug:   result.CapabilitySlug,
		BuiltScenarios:   result.BuiltScenarios,
		PlannedScenarios: result.PlannedScenarios,
		Fulfillments:     make([]*ontologyv1.Fulfillment, 0, len(result.Fulfillments)),
	}
	for _, fulfillment := range result.Fulfillments {
		resp.Fulfillments = append(resp.Fulfillments, fulfillmentToProto(fulfillment))
	}
	return resp
}

func scenarioCapabilitiesToProto(result ontologydomain.ScenarioCapabilities) *ontologyv1.ScenarioCapabilities {
	resp := &ontologyv1.ScenarioCapabilities{
		ScenarioSlug: result.ScenarioSlug,
		Capabilities: make([]*ontologyv1.Capability, 0, len(result.Capabilities)),
		Fulfillments: make([]*ontologyv1.Fulfillment, 0, len(result.Fulfillments)),
	}
	for _, capability := range result.Capabilities {
		resp.Capabilities = append(resp.Capabilities, capabilityToProto(capability))
	}
	for _, fulfillment := range result.Fulfillments {
		resp.Fulfillments = append(resp.Fulfillments, fulfillmentToProto(fulfillment))
	}
	return resp
}

func overlayGraphToProto(graph *graphv1.TechTreeGraph) *ontologyv1.OverlayGraph {
	if graph == nil {
		return &ontologyv1.OverlayGraph{}
	}
	out := &ontologyv1.OverlayGraph{
		Nodes:  make([]*ontologyv1.OverlayNode, 0, len(graph.GetNodes())),
		Edges:  make([]*ontologyv1.OverlayEdge, 0, len(graph.GetEdges())),
		Errors: make([]*ontologyv1.OverlayError, 0, len(graph.GetErrors())),
	}
	for _, node := range graph.GetNodes() {
		out.Nodes = append(out.Nodes, &ontologyv1.OverlayNode{
			Scenario:       node.GetScenario(),
			Kind:           overlayNodeKind(node.GetKind()),
			DisplayName:    node.GetDisplayName(),
			TransportWorld: node.GetTransportWorld(),
			Stability:      node.GetStability(),
			Sector:         node.GetSector(),
			Tier:           node.GetTier(),
			Parent:         node.GetParent(),
		})
	}
	for _, edge := range graph.GetEdges() {
		outEdge := &ontologyv1.OverlayEdge{
			FromScenario:   edge.GetFromScenario(),
			ToScenario:     edge.GetToScenario(),
			TransportWorld: edge.GetTransportWorld(),
			Stability:      edge.GetStability(),
			Evidence:       make([]*ontologyv1.OverlayEvidence, 0, len(edge.GetEvidence())),
		}
		for _, evidence := range edge.GetEvidence() {
			outEdge.Evidence = append(outEdge.Evidence, &ontologyv1.OverlayEvidence{
				Source:     overlayEvidenceSource(evidence.GetSource()),
				ImportPath: evidence.GetImportPath(),
				FromFile:   evidence.GetFromFile(),
				ToFile:     evidence.GetToFile(),
				Path:       evidence.GetPath(),
				Analyzer:   evidence.GetAnalyzer(),
			})
		}
		out.Edges = append(out.Edges, outEdge)
	}
	for _, graphErr := range graph.GetErrors() {
		out.Errors = append(out.Errors, &ontologyv1.OverlayError{
			Source:   graphErr.GetSource(),
			Scenario: graphErr.GetScenario(),
			Message:  graphErr.GetMessage(),
		})
	}
	return out
}

func overlayNodeKind(kind graphv1.NodeKind) ontologyv1.OverlayNodeKind {
	switch kind {
	case graphv1.NodeKind_NODE_KIND_LIVE:
		return ontologyv1.OverlayNodeKind_OVERLAY_NODE_KIND_LIVE
	case graphv1.NodeKind_NODE_KIND_PLANNED:
		return ontologyv1.OverlayNodeKind_OVERLAY_NODE_KIND_PLANNED
	case graphv1.NodeKind_NODE_KIND_CAPABILITY:
		return ontologyv1.OverlayNodeKind_OVERLAY_NODE_KIND_CAPABILITY
	default:
		return ontologyv1.OverlayNodeKind_OVERLAY_NODE_KIND_UNSPECIFIED
	}
}

func overlayEvidenceSource(source graphv1.EvidenceSource) ontologyv1.OverlayEvidenceSource {
	switch source {
	case graphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT:
		return ontologyv1.OverlayEvidenceSource_OVERLAY_EVIDENCE_SOURCE_PROTO_IMPORT
	case graphv1.EvidenceSource_EVIDENCE_SOURCE_GO_IMPORT:
		return ontologyv1.OverlayEvidenceSource_OVERLAY_EVIDENCE_SOURCE_GO_IMPORT
	case graphv1.EvidenceSource_EVIDENCE_SOURCE_PLANNED_PROTO_IMPORT:
		return ontologyv1.OverlayEvidenceSource_OVERLAY_EVIDENCE_SOURCE_PLANNED_PROTO_IMPORT
	case graphv1.EvidenceSource_EVIDENCE_SOURCE_DECOMPOSES:
		return ontologyv1.OverlayEvidenceSource_OVERLAY_EVIDENCE_SOURCE_DECOMPOSES
	case graphv1.EvidenceSource_EVIDENCE_SOURCE_FULFILLS:
		return ontologyv1.OverlayEvidenceSource_OVERLAY_EVIDENCE_SOURCE_FULFILLS
	default:
		return ontologyv1.OverlayEvidenceSource_OVERLAY_EVIDENCE_SOURCE_UNSPECIFIED
	}
}
