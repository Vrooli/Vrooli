package ontology

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	ontologyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology"
)

type Service struct {
	repo      Repository
	scenarios ScenarioSource
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func NewServiceWithScenarioSource(repo Repository, scenarios ScenarioSource) *Service {
	return &Service{repo: repo, scenarios: scenarios}
}

func (s *Service) ListCapabilities(ctx context.Context, filter CapabilityFilter) ([]Capability, error) {
	if filter.ParentID != "" {
		parentID, err := NormalizeID("parent_id", filter.ParentID)
		if err != nil {
			return nil, err
		}
		filter.ParentID = parentID
	}
	if filter.Kind != 0 {
		kind, err := NormalizeCapabilityKind(filter.Kind)
		if err != nil {
			return nil, err
		}
		filter.Kind = kind
	}
	return s.repo.ListCapabilities(ctx, filter)
}

func (s *Service) GetCapability(ctx context.Context, ref CapabilityRef) (Capability, error) {
	ref, err := normalizeRef(ref)
	if err != nil {
		return Capability{}, err
	}
	capability, err := s.repo.GetCapability(ctx, ref)
	if errors.Is(err, sql.ErrNoRows) {
		return Capability{}, ErrCapabilityNotFound{Ref: ref}
	}
	return capability, err
}

func (s *Service) UpsertCapability(ctx context.Context, capability Capability) (Capability, error) {
	normalized, err := normalizeCapability(capability)
	if err != nil {
		return Capability{}, err
	}
	if normalized.ParentID != "" {
		if err := s.ensureParentDoesNotCycle(ctx, normalized.ID, normalized.ParentID); err != nil {
			return Capability{}, err
		}
	}
	return s.repo.UpsertCapability(ctx, normalized)
}

func (s *Service) DeleteCapability(ctx context.Context, ref CapabilityRef) (bool, error) {
	ref, err := normalizeRef(ref)
	if err != nil {
		return false, err
	}
	return s.repo.DeleteCapability(ctx, ref)
}

func (s *Service) UpsertCapabilityEdge(ctx context.Context, edge CapabilityEdge) (CapabilityEdge, error) {
	edge, err := normalizeEdge(edge)
	if err != nil {
		return CapabilityEdge{}, err
	}
	if edge.FromID == edge.ToID {
		return CapabilityEdge{}, ErrInvalidArgument{Field: "edge.to_id", Reason: "cannot equal from_id"}
	}
	return s.repo.UpsertCapabilityEdge(ctx, edge)
}

func (s *Service) DeleteCapabilityEdge(ctx context.Context, edge CapabilityEdge) (bool, error) {
	edge, err := normalizeEdge(edge)
	if err != nil {
		return false, err
	}
	return s.repo.DeleteCapabilityEdge(ctx, edge)
}

func (s *Service) LinkFulfillment(ctx context.Context, fulfillment Fulfillment) (Fulfillment, error) {
	fulfillment.CapabilityID = strings.TrimSpace(strings.ToLower(fulfillment.CapabilityID))
	var err error
	fulfillment.CapabilityID, err = NormalizeID("capability_id", fulfillment.CapabilityID)
	if err != nil {
		return Fulfillment{}, err
	}
	fulfillment.ScenarioSlug, err = NormalizeScenarioSlug(fulfillment.ScenarioSlug)
	if err != nil {
		return Fulfillment{}, err
	}
	fulfillment.Note = strings.TrimSpace(fulfillment.Note)
	return s.repo.LinkFulfillment(ctx, fulfillment)
}

func (s *Service) UnlinkFulfillment(ctx context.Context, capabilityID, scenarioSlug string) (bool, error) {
	capabilityID, err := NormalizeID("capability_id", capabilityID)
	if err != nil {
		return false, err
	}
	scenarioSlug, err = NormalizeScenarioSlug(scenarioSlug)
	if err != nil {
		return false, err
	}
	return s.repo.UnlinkFulfillment(ctx, capabilityID, scenarioSlug)
}

func (s *Service) ListFulfillments(ctx context.Context, filter FulfillmentFilter) ([]Fulfillment, error) {
	var err error
	if filter.CapabilityID != "" {
		filter.CapabilityID, err = NormalizeID("capability_id", filter.CapabilityID)
		if err != nil {
			return nil, err
		}
	}
	if filter.ScenarioSlug != "" {
		filter.ScenarioSlug, err = NormalizeScenarioSlug(filter.ScenarioSlug)
		if err != nil {
			return nil, err
		}
	}
	return s.repo.ListFulfillments(ctx, filter)
}

func (s *Service) ImportTopology(ctx context.Context, data []byte) (TopologyImportResult, error) {
	topology, err := ParseTopology(data)
	if err != nil {
		return TopologyImportResult{}, err
	}
	existingCapabilities, err := s.repo.ListCapabilities(ctx, CapabilityFilter{})
	if err != nil {
		return TopologyImportResult{}, err
	}
	existingCapabilityIDs := map[string]struct{}{}
	for _, capability := range existingCapabilities {
		existingCapabilityIDs[capability.ID] = struct{}{}
	}
	existingEdges, err := s.repo.ListCapabilityEdges(ctx)
	if err != nil {
		return TopologyImportResult{}, err
	}
	existingEdgeKeys := map[string]struct{}{}
	for _, edge := range existingEdges {
		existingEdgeKeys[edgeKey(edge)] = struct{}{}
	}

	var result TopologyImportResult
	for _, capability := range topology.Capabilities {
		if _, found := existingCapabilityIDs[capability.ID]; !found {
			if capability.Kind == ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR {
				result.SectorsImported++
			} else {
				result.CapabilitiesImported++
			}
		}
		if _, err := s.UpsertCapability(ctx, capability); err != nil {
			return TopologyImportResult{}, err
		}
	}
	for _, edge := range topology.Edges {
		key := edgeKey(edge)
		if _, found := existingEdgeKeys[key]; !found {
			result.EdgesImported++
			existingEdgeKeys[key] = struct{}{}
		}
		if _, err := s.UpsertCapabilityEdge(ctx, edge); err != nil {
			return TopologyImportResult{}, err
		}
	}

	finalCapabilities, err := s.repo.ListCapabilities(ctx, CapabilityFilter{})
	if err != nil {
		return TopologyImportResult{}, err
	}
	finalEdges, err := s.repo.ListCapabilityEdges(ctx)
	if err != nil {
		return TopologyImportResult{}, err
	}
	for _, capability := range finalCapabilities {
		if capability.Kind == ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR {
			result.SectorsTotal++
		} else {
			result.CapabilitiesTotal++
		}
	}
	result.EdgesTotal = countInt32(len(finalEdges))
	return result, nil
}

func (s *Service) GetCoverage(ctx context.Context, req CoverageRequest) (CoverageSummary, error) {
	input, err := s.coverageInput(ctx)
	if err != nil {
		return CoverageSummary{}, err
	}
	classifications := classifyCapabilities(input.capabilities, input.fulfillments, input.scenarios, req.IncludeSubtreeRollup)
	summary := CoverageSummary{
		TotalCapabilities: countInt32(len(input.capabilities)),
		TotalScenarios:    countInt32(len(input.scenarios.all)),
		Classifications:   classifications,
		GraphError:        input.graphError,
	}
	for _, classification := range classifications {
		switch classification.State {
		case ontologyv1.CoverageState_COVERAGE_STATE_BUILT:
			summary.BuiltCapabilities++
		case ontologyv1.CoverageState_COVERAGE_STATE_IN_FLIGHT:
			summary.InflightCapabilities++
		case ontologyv1.CoverageState_COVERAGE_STATE_GAP:
			summary.GapCapabilities++
		}
	}
	if summary.TotalCapabilities > 0 {
		summary.OntologyCompleteness = float64(summary.BuiltCapabilities+summary.InflightCapabilities) / float64(summary.TotalCapabilities)
	}
	mappedScenarios := mappedScenarioSet(input.fulfillments)
	for scenario := range input.scenarios.all {
		if _, ok := mappedScenarios[scenario]; !ok {
			summary.UnmappedScenarios++
		}
	}
	if summary.TotalScenarios > 0 {
		summary.ImplementationSituatedness = float64(summary.TotalScenarios-summary.UnmappedScenarios) / float64(summary.TotalScenarios)
	}
	summary.Sectors = sectorCoverage(input.capabilities, classifications)
	return summary, nil
}

func (s *Service) ListFocus(ctx context.Context, limit int32) ([]FocusItem, error) {
	if limit <= 0 {
		limit = 10
	}
	input, err := s.coverageInput(ctx)
	if err != nil {
		return nil, err
	}
	classifications := classifyCapabilities(input.capabilities, input.fulfillments, input.scenarios, true)
	byID := capabilityByID(input.capabilities)
	downstream := downstreamDependents(input.edges)
	var items []FocusItem
	for _, classification := range classifications {
		if classification.State != ontologyv1.CoverageState_COVERAGE_STATE_GAP {
			continue
		}
		capability := byID[classification.CapabilityID]
		dependents := downstream[classification.CapabilityID]
		score := capability.Importance * (1 + float64(dependents))
		items = append(items, FocusItem{
			CapabilityID:         capability.ID,
			CapabilitySlug:       capability.Slug,
			CapabilityName:       capability.Name,
			Reason:               ontologyv1.FocusReason_FOCUS_REASON_GAP,
			Score:                score,
			DownstreamDependents: countInt32(dependents),
		})
	}
	mapped := mappedScenarioSet(input.fulfillments)
	for scenario := range input.scenarios.all {
		if _, ok := mapped[scenario]; ok {
			continue
		}
		items = append(items, FocusItem{
			Reason:           ontologyv1.FocusReason_FOCUS_REASON_UNMAPPED_SCENARIO,
			Score:            1,
			RelatedScenarios: []string{scenario},
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if items[i].Reason != items[j].Reason {
			return items[i].Reason < items[j].Reason
		}
		if items[i].CapabilitySlug != items[j].CapabilitySlug {
			return items[i].CapabilitySlug < items[j].CapabilitySlug
		}
		return strings.Join(items[i].RelatedScenarios, ",") < strings.Join(items[j].RelatedScenarios, ",")
	})
	if len(items) > int(limit) {
		items = items[:limit]
	}
	return items, nil
}

func (s *Service) GetCapabilityScenarios(ctx context.Context, ref CapabilityRef, includeDescendants bool) (CapabilityScenarios, error) {
	capability, err := s.GetCapability(ctx, ref)
	if err != nil {
		return CapabilityScenarios{}, err
	}
	input, err := s.coverageInput(ctx)
	if err != nil {
		return CapabilityScenarios{}, err
	}
	ids := map[string]struct{}{capability.ID: {}}
	if includeDescendants {
		for _, descendant := range descendants(input.capabilities, capability.ID) {
			ids[descendant.ID] = struct{}{}
		}
	}
	var fulfillments []Fulfillment
	for _, fulfillment := range input.fulfillments {
		if _, ok := ids[fulfillment.CapabilityID]; ok {
			fulfillments = append(fulfillments, fulfillment)
		}
	}
	built, planned := splitScenarioSlugs(fulfillments, input.scenarios)
	return CapabilityScenarios{
		CapabilityID:     capability.ID,
		CapabilitySlug:   capability.Slug,
		BuiltScenarios:   built,
		PlannedScenarios: planned,
		Fulfillments:     fulfillments,
	}, nil
}

func (s *Service) GetScenarioCapabilities(ctx context.Context, scenarioSlug string) (ScenarioCapabilities, error) {
	scenarioSlug, err := NormalizeScenarioSlug(scenarioSlug)
	if err != nil {
		return ScenarioCapabilities{}, err
	}
	fulfillments, err := s.repo.ListFulfillments(ctx, FulfillmentFilter{ScenarioSlug: scenarioSlug})
	if err != nil {
		return ScenarioCapabilities{}, err
	}
	var capabilities []Capability
	for _, fulfillment := range fulfillments {
		capability, err := s.repo.GetCapability(ctx, CapabilityRef{ID: fulfillment.CapabilityID})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return ScenarioCapabilities{}, err
		}
		capabilities = append(capabilities, capability)
	}
	sort.Slice(capabilities, func(i, j int) bool { return capabilities[i].Slug < capabilities[j].Slug })
	return ScenarioCapabilities{ScenarioSlug: scenarioSlug, Capabilities: capabilities, Fulfillments: fulfillments}, nil
}

func (s *Service) DescribeOverlayGraph(ctx context.Context, req OverlayGraphRequest) (*graphv1.TechTreeGraph, error) {
	if !req.IncludeImplementation && !req.IncludeOntology && !req.IncludeFulfillment {
		req.IncludeImplementation = true
		req.IncludeOntology = true
		req.IncludeFulfillment = true
	}
	out := &graphv1.TechTreeGraph{}
	if req.IncludeImplementation && s.scenarios != nil {
		graph, err := s.scenarios.ScenarioGraph(ctx)
		if err != nil {
			out.Errors = append(out.Errors, &graphv1.GraphError{Source: "graph", Message: err.Error()})
		} else {
			out.Nodes = append(out.Nodes, graph.GetNodes()...)
			out.Edges = append(out.Edges, graph.GetEdges()...)
			out.Errors = append(out.Errors, graph.GetErrors()...)
		}
	}
	if req.IncludeOntology {
		capabilities, err := s.repo.ListCapabilities(ctx, CapabilityFilter{})
		if err != nil {
			return nil, err
		}
		edges, err := s.repo.ListCapabilityEdges(ctx)
		if err != nil {
			return nil, err
		}
		for _, capability := range capabilities {
			out.Nodes = append(out.Nodes, &graphv1.TechNode{
				Scenario:    capability.ID,
				Kind:        graphv1.NodeKind_NODE_KIND_CAPABILITY,
				DisplayName: firstNonEmpty(capability.Name, capability.Slug),
				Sector:      capability.Kind.String(),
				Tier:        capability.Slug,
				Parent:      capability.ParentID,
			})
			if capability.ParentID != "" {
				out.Edges = append(out.Edges, overlayEdge(capability.ParentID, capability.ID, graphv1.EvidenceSource_EVIDENCE_SOURCE_DECOMPOSES, "parent"))
			}
		}
		for _, edge := range edges {
			out.Edges = append(out.Edges, overlayEdge(edge.FromID, edge.ToID, graphv1.EvidenceSource_EVIDENCE_SOURCE_DECOMPOSES, EdgeTypeToStorage(edge.Type)))
		}
	}
	if req.IncludeFulfillment {
		fulfillments, err := s.repo.ListFulfillments(ctx, FulfillmentFilter{})
		if err != nil {
			return nil, err
		}
		for _, fulfillment := range fulfillments {
			out.Edges = append(out.Edges, overlayEdge(fulfillment.CapabilityID, fulfillment.ScenarioSlug, graphv1.EvidenceSource_EVIDENCE_SOURCE_FULFILLS, "fulfillment"))
		}
	}
	return normalizeOverlayGraph(out), nil
}

func (s *Service) ensureParentDoesNotCycle(ctx context.Context, id, parentID string) error {
	if id == parentID {
		return ErrCapabilityCycle{ID: id, ParentID: parentID}
	}
	capabilities, err := s.repo.ListCapabilities(ctx, CapabilityFilter{})
	if err != nil {
		return err
	}
	parents := map[string]string{}
	for _, capability := range capabilities {
		parents[capability.ID] = capability.ParentID
	}
	for current := parentID; current != ""; current = parents[current] {
		if current == id {
			return ErrCapabilityCycle{ID: id, ParentID: parentID}
		}
	}
	return nil
}

func normalizeCapability(capability Capability) (Capability, error) {
	var err error
	capability.Slug, err = NormalizeID("slug", capability.Slug)
	if err != nil {
		return Capability{}, err
	}
	if strings.TrimSpace(capability.ID) == "" {
		capability.ID = capability.Slug
	} else {
		capability.ID, err = NormalizeID("id", capability.ID)
		if err != nil {
			return Capability{}, err
		}
	}
	capability.ParentID, err = NormalizeOptionalID("parent_id", capability.ParentID)
	if err != nil {
		return Capability{}, err
	}
	capability.Name = strings.TrimSpace(capability.Name)
	if capability.Name == "" {
		capability.Name = DefaultName(capability.Slug)
	}
	capability.Description = strings.TrimSpace(capability.Description)
	capability.Kind, err = NormalizeCapabilityKind(capability.Kind)
	if err != nil {
		return Capability{}, err
	}
	if capability.Importance <= 0 {
		capability.Importance = 1
	}
	return capability, nil
}

func normalizeRef(ref CapabilityRef) (CapabilityRef, error) {
	var err error
	if strings.TrimSpace(ref.ID) != "" {
		ref.ID, err = NormalizeID("id", ref.ID)
		if err != nil {
			return CapabilityRef{}, err
		}
	}
	if strings.TrimSpace(ref.Slug) != "" {
		ref.Slug, err = NormalizeID("slug", ref.Slug)
		if err != nil {
			return CapabilityRef{}, err
		}
	}
	if ref.ID == "" && ref.Slug == "" {
		return CapabilityRef{}, ErrInvalidArgument{Field: "id", Reason: "or slug is required"}
	}
	return ref, nil
}

func normalizeEdge(edge CapabilityEdge) (CapabilityEdge, error) {
	var err error
	edge.FromID, err = NormalizeID("edge.from_id", edge.FromID)
	if err != nil {
		return CapabilityEdge{}, err
	}
	edge.ToID, err = NormalizeID("edge.to_id", edge.ToID)
	if err != nil {
		return CapabilityEdge{}, err
	}
	edge.Type, err = NormalizeCapabilityEdgeType(edge.Type)
	if err != nil {
		return CapabilityEdge{}, err
	}
	return edge, nil
}

func edgeKey(edge CapabilityEdge) string {
	return edge.FromID + "\x00" + edge.ToID + "\x00" + EdgeTypeToStorage(edge.Type)
}

type coverageInput struct {
	capabilities []Capability
	edges        []CapabilityEdge
	fulfillments []Fulfillment
	scenarios    scenarioSets
	graphError   string
}

type scenarioSets struct {
	all     map[string]struct{}
	built   map[string]struct{}
	planned map[string]struct{}
}

func (s *Service) coverageInput(ctx context.Context) (coverageInput, error) {
	capabilities, err := s.repo.ListCapabilities(ctx, CapabilityFilter{})
	if err != nil {
		return coverageInput{}, err
	}
	edges, err := s.repo.ListCapabilityEdges(ctx)
	if err != nil {
		return coverageInput{}, err
	}
	fulfillments, err := s.repo.ListFulfillments(ctx, FulfillmentFilter{})
	if err != nil {
		return coverageInput{}, err
	}
	input := coverageInput{
		capabilities: capabilities,
		edges:        edges,
		fulfillments: fulfillments,
		scenarios: scenarioSets{
			all:     map[string]struct{}{},
			built:   map[string]struct{}{},
			planned: map[string]struct{}{},
		},
	}
	if s.scenarios == nil {
		return input, nil
	}
	graph, err := s.scenarios.ScenarioGraph(ctx)
	if err != nil {
		input.graphError = err.Error()
		return input, nil
	}
	input.scenarios = scenarioSetsFromGraph(graph)
	return input, nil
}

func scenarioSetsFromGraph(graph *graphv1.TechTreeGraph) scenarioSets {
	sets := scenarioSets{
		all:     map[string]struct{}{},
		built:   map[string]struct{}{},
		planned: map[string]struct{}{},
	}
	for _, node := range graph.GetNodes() {
		scenario := strings.TrimSpace(node.GetScenario())
		if scenario == "" || node.GetKind() == graphv1.NodeKind_NODE_KIND_CAPABILITY {
			continue
		}
		sets.all[scenario] = struct{}{}
		switch node.GetKind() {
		case graphv1.NodeKind_NODE_KIND_PLANNED:
			sets.planned[scenario] = struct{}{}
		default:
			sets.built[scenario] = struct{}{}
		}
	}
	return sets
}

func classifyCapabilities(capabilities []Capability, fulfillments []Fulfillment, scenarios scenarioSets, includeSubtree bool) []CoverageClassification {
	byCapability := fulfillmentsByCapability(fulfillments)
	byID := capabilityByID(capabilities)
	out := make([]CoverageClassification, 0, len(capabilities))
	for _, capability := range capabilities {
		ids := []string{capability.ID}
		if includeSubtree {
			for _, descendant := range descendants(capabilities, capability.ID) {
				ids = append(ids, descendant.ID)
			}
		}
		directBuilt, directPlanned := splitScenarioSlugs(byCapability[capability.ID], scenarios)
		var scoped []Fulfillment
		for _, id := range ids {
			scoped = append(scoped, byCapability[id]...)
		}
		built, planned := splitScenarioSlugs(scoped, scenarios)
		classification := CoverageClassification{
			CapabilityID:      capability.ID,
			CapabilitySlug:    capability.Slug,
			DirectlyFulfilled: len(directBuilt)+len(directPlanned) > 0,
			SubtreeCovered:    len(built)+len(planned) > 0,
			BuiltScenarios:    built,
			PlannedScenarios:  planned,
		}
		switch {
		case len(built) > 0:
			classification.State = ontologyv1.CoverageState_COVERAGE_STATE_BUILT
		case len(planned) > 0:
			classification.State = ontologyv1.CoverageState_COVERAGE_STATE_IN_FLIGHT
		default:
			classification.State = ontologyv1.CoverageState_COVERAGE_STATE_GAP
		}
		if _, ok := byID[classification.CapabilityID]; ok {
			out = append(out, classification)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CapabilitySlug < out[j].CapabilitySlug })
	return out
}

func sectorCoverage(capabilities []Capability, classifications []CoverageClassification) []SectorCoverage {
	byClassification := map[string]CoverageClassification{}
	for _, classification := range classifications {
		byClassification[classification.CapabilityID] = classification
	}
	var sectors []SectorCoverage
	for _, capability := range capabilities {
		if capability.Kind != ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR {
			continue
		}
		ids := map[string]struct{}{capability.ID: {}}
		for _, descendant := range descendants(capabilities, capability.ID) {
			ids[descendant.ID] = struct{}{}
		}
		coverage := SectorCoverage{
			SectorID:   capability.ID,
			SectorSlug: capability.Slug,
			SectorName: capability.Name,
		}
		for id := range ids {
			classification, ok := byClassification[id]
			if !ok {
				continue
			}
			coverage.TotalCapabilities++
			switch classification.State {
			case ontologyv1.CoverageState_COVERAGE_STATE_BUILT:
				coverage.BuiltCapabilities++
			case ontologyv1.CoverageState_COVERAGE_STATE_IN_FLIGHT:
				coverage.InflightCapabilities++
			case ontologyv1.CoverageState_COVERAGE_STATE_GAP:
				coverage.GapCapabilities++
			}
		}
		if coverage.TotalCapabilities > 0 {
			coverage.OntologyCompleteness = float64(coverage.BuiltCapabilities+coverage.InflightCapabilities) / float64(coverage.TotalCapabilities)
		}
		sectors = append(sectors, coverage)
	}
	sort.Slice(sectors, func(i, j int) bool { return sectors[i].SectorSlug < sectors[j].SectorSlug })
	return sectors
}

func splitScenarioSlugs(fulfillments []Fulfillment, scenarios scenarioSets) ([]string, []string) {
	builtSet := map[string]struct{}{}
	plannedSet := map[string]struct{}{}
	for _, fulfillment := range fulfillments {
		if _, ok := scenarios.built[fulfillment.ScenarioSlug]; ok {
			builtSet[fulfillment.ScenarioSlug] = struct{}{}
			continue
		}
		if _, ok := scenarios.planned[fulfillment.ScenarioSlug]; ok {
			plannedSet[fulfillment.ScenarioSlug] = struct{}{}
		}
	}
	return sortedKeys(builtSet), sortedKeys(plannedSet)
}

func fulfillmentsByCapability(fulfillments []Fulfillment) map[string][]Fulfillment {
	out := map[string][]Fulfillment{}
	for _, fulfillment := range fulfillments {
		out[fulfillment.CapabilityID] = append(out[fulfillment.CapabilityID], fulfillment)
	}
	return out
}

func mappedScenarioSet(fulfillments []Fulfillment) map[string]struct{} {
	out := map[string]struct{}{}
	for _, fulfillment := range fulfillments {
		out[fulfillment.ScenarioSlug] = struct{}{}
	}
	return out
}

func capabilityByID(capabilities []Capability) map[string]Capability {
	out := map[string]Capability{}
	for _, capability := range capabilities {
		out[capability.ID] = capability
	}
	return out
}

func descendants(capabilities []Capability, rootID string) []Capability {
	byParent := map[string][]Capability{}
	for _, capability := range capabilities {
		byParent[capability.ParentID] = append(byParent[capability.ParentID], capability)
	}
	var out []Capability
	var visit func(string)
	visit = func(parentID string) {
		children := byParent[parentID]
		sort.Slice(children, func(i, j int) bool { return children[i].Slug < children[j].Slug })
		for _, child := range children {
			out = append(out, child)
			visit(child.ID)
		}
	}
	visit(rootID)
	return out
}

func downstreamDependents(edges []CapabilityEdge) map[string]int {
	outgoing := map[string][]string{}
	for _, edge := range edges {
		outgoing[edge.FromID] = append(outgoing[edge.FromID], edge.ToID)
	}
	out := map[string]int{}
	for id := range outgoing {
		seen := map[string]struct{}{}
		queue := append([]string(nil), outgoing[id]...)
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if _, ok := seen[cur]; ok {
				continue
			}
			seen[cur] = struct{}{}
			queue = append(queue, outgoing[cur]...)
		}
		out[id] = len(seen)
	}
	return out
}

func overlayEdge(from, to string, source graphv1.EvidenceSource, analyzer string) *graphv1.TechEdge {
	return &graphv1.TechEdge{
		FromScenario: from,
		ToScenario:   to,
		Evidence: []*graphv1.GraphEvidence{{
			Source:   source,
			Analyzer: analyzer,
		}},
	}
}

func normalizeOverlayGraph(graph *graphv1.TechTreeGraph) *graphv1.TechTreeGraph {
	nodeByID := map[string]*graphv1.TechNode{}
	for _, node := range graph.GetNodes() {
		if node.GetScenario() == "" {
			continue
		}
		if _, ok := nodeByID[node.GetScenario()]; !ok {
			nodeByID[node.GetScenario()] = node
		}
	}
	for _, edge := range graph.GetEdges() {
		if edge.GetFromScenario() != "" {
			if _, ok := nodeByID[edge.GetFromScenario()]; !ok {
				nodeByID[edge.GetFromScenario()] = &graphv1.TechNode{Scenario: edge.GetFromScenario(), DisplayName: displayName(edge.GetFromScenario())}
			}
		}
		if edge.GetToScenario() != "" {
			if _, ok := nodeByID[edge.GetToScenario()]; !ok {
				nodeByID[edge.GetToScenario()] = &graphv1.TechNode{Scenario: edge.GetToScenario(), DisplayName: displayName(edge.GetToScenario())}
			}
		}
	}
	nodes := make([]*graphv1.TechNode, 0, len(nodeByID))
	for _, node := range nodeByID {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].GetScenario() < nodes[j].GetScenario() })
	edges := append([]*graphv1.TechEdge(nil), graph.GetEdges()...)
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].GetFromScenario() != edges[j].GetFromScenario() {
			return edges[i].GetFromScenario() < edges[j].GetFromScenario()
		}
		if edges[i].GetToScenario() != edges[j].GetToScenario() {
			return edges[i].GetToScenario() < edges[j].GetToScenario()
		}
		return edgeEvidenceKey(edges[i]) < edgeEvidenceKey(edges[j])
	})
	return &graphv1.TechTreeGraph{Nodes: nodes, Edges: edges, Errors: graph.GetErrors()}
}

func edgeEvidenceKey(edge *graphv1.TechEdge) string {
	var parts []string
	for _, evidence := range edge.GetEvidence() {
		parts = append(parts, evidence.GetSource().String()+":"+evidence.GetAnalyzer())
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func displayName(slug string) string {
	return DefaultName(strings.ReplaceAll(slug, "_", "-"))
}

func countInt32(n int) int32 {
	const maxInt32 = int(^uint32(0) >> 1)
	if n > maxInt32 {
		return int32(maxInt32)
	}
	if n < 0 {
		return 0
	}
	return int32(n)
}
