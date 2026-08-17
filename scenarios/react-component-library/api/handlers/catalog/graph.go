package catalog

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"connectrpc.com/connect"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/assetgraph"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/graphreconcile"
	"react-component-library/internal/portcontract"
)

func (h *handler) graph() (*assetgraph.Index, error) {
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog"))
	if err != nil {
		return nil, err
	}
	return assetgraph.Build(assets)
}

func (h *handler) GetAssetRelationships(_ context.Context, req *connect.Request[catalogv1.GetAssetRelationshipsRequest]) (*connect.Response[catalogv1.GetAssetRelationshipsResponse], error) {
	index, err := h.graph()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	id := req.Msg.GetAssetId()
	root, err := index.Node(id)
	if err != nil {
		return nil, graphError(err)
	}
	dependencies, err := index.Dependencies(id)
	if err != nil {
		return nil, graphError(err)
	}
	closure, err := index.Closure(id)
	if err != nil {
		return nil, graphError(err)
	}
	direct, transitive, err := index.Dependents(id)
	if err != nil {
		return nil, graphError(err)
	}
	bands := assetgraph.Bands(closure)
	return connect.NewResponse(&catalogv1.GetAssetRelationshipsResponse{Relationships: &catalogv1.AssetRelationships{
		Root:                     nodeProto(root),
		DirectDependencies:       nodesProto(dependencies),
		Closure:                  nodesProto(closure),
		ClosureBands:             bandsProto(bands),
		DirectDependents:         nodesProto(direct),
		TransitiveDependents:     nodesProto(transitive),
		TransitiveDependentCount: int32(len(transitive)),
	}}), nil
}

func (h *handler) GetCatalogStructure(_ context.Context, _ *connect.Request[catalogv1.GetCatalogStructureRequest]) (*connect.Response[catalogv1.GetCatalogStructureResponse], error) {
	index, err := h.graph()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	population := map[int32]*catalogv1.RungPopulation{}
	for _, node := range index.Nodes() {
		if node.Rung < 0 {
			continue
		}
		key := int32(node.Rung)
		item := population[key]
		if item == nil {
			item = &catalogv1.RungPopulation{Rung: key, RungName: node.RungName}
			population[key] = item
		}
		item.Count++
	}
	pop := make([]*catalogv1.RungPopulation, 0, len(population))
	for _, item := range population {
		pop = append(pop, item)
	}
	sort.Slice(pop, func(i, j int) bool { return pop[i].Rung > pop[j].Rung })
	rankHolds, noDependencies := true, 0
	for _, node := range index.Nodes() {
		dependencies, dependenciesErr := index.Dependencies(node.ID)
		if dependenciesErr != nil {
			return nil, connect.NewError(connect.CodeInternal, dependenciesErr)
		}
		if len(dependencies) == 0 {
			noDependencies++
		}
		for _, dependency := range dependencies {
			if node.Kind != "generator" && dependency.Rung > node.Rung {
				rankHolds = false
			}
		}
	}
	invariants := []*catalogv1.StructureInvariant{
		{Id: "rank-ordering", Label: "Rank ordering holds", Status: invariantStatus(rankHolds), Detail: "All catalog requires edges point to the same or a lower rung."},
		{Id: "requires-acyclic", Label: "Requires graph is acyclic", Status: "holds", Detail: "Every asset closure completed without a cycle."},
		{Id: "graph-reconciliation", Label: "Catalog and manifest graphs are reconciled", Status: "not-run", Detail: "The non-blocking graph-reconciled gate supplies this measurement."},
		{Id: "dependency-declarations", Label: "Assets declaring no dependencies", Status: "measured", Detail: fmt.Sprintf("%d catalog assets declare no requires dependency.", noDependencies)},
	}
	blast := make([]*catalogv1.BlastRadiusRow, 0, len(index.Nodes()))
	for _, node := range index.Nodes() {
		if node.Rung < 0 {
			continue
		}
		_, transitive, dependentsErr := index.Dependents(node.ID)
		if dependentsErr != nil {
			return nil, connect.NewError(connect.CodeInternal, dependentsErr)
		}
		blast = append(blast, &catalogv1.BlastRadiusRow{Asset: nodeProto(node), TransitiveDependentCount: int32(len(transitive))})
	}
	sort.Slice(blast, func(i, j int) bool {
		if blast[i].TransitiveDependentCount != blast[j].TransitiveDependentCount {
			return blast[i].TransitiveDependentCount > blast[j].TransitiveDependentCount
		}
		return blast[i].Asset.AssetId < blast[j].Asset.AssetId
	})
	if len(blast) > 10 {
		blast = blast[:10]
	}
	return connect.NewResponse(&catalogv1.GetCatalogStructureResponse{Structure: &catalogv1.CatalogStructure{Population: pop, Invariants: invariants, BlastRadius: blast}}), nil
}

func invariantStatus(holds bool) string {
	if holds {
		return "holds"
	}
	return "fails"
}

func nodeProto(node assetgraph.Node) *catalogv1.AssetNode {
	return &catalogv1.AssetNode{AssetId: node.ID, Name: node.Name, Kind: node.Kind, Rung: int32(node.Rung), RungName: node.RungName, Domain: node.Domain, DomainOrder: int32(node.DomainOrder)}
}

func nodesProto(nodes []assetgraph.Node) []*catalogv1.AssetNode {
	out := make([]*catalogv1.AssetNode, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, nodeProto(node))
	}
	return out
}

func bandsProto(bands []assetgraph.Band) []*catalogv1.RungBand {
	out := make([]*catalogv1.RungBand, 0, len(bands))
	for _, band := range bands {
		out = append(out, &catalogv1.RungBand{Rung: int32(band.Rung), RungName: band.Name, Assets: nodesProto(band.Assets), Count: int32(band.Count)})
	}
	return out
}

func graphError(err error) error {
	var unknown assetgraph.UnknownAssetError
	if errors.As(err, &unknown) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var cycle assetgraph.CycleError
	if errors.As(err, &cycle) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("catalog graph: %w", err))
}

func (h *handler) ReconcileGraph(ctx context.Context, _ *connect.Request[catalogv1.ReconcileGraphRequest]) (*connect.Response[catalogv1.ReconcileGraphResponse], error) {
	report, err := graphreconcile.Reconcile(ctx, h.repoRoot)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("reconcile catalog graph: %w", err))
	}
	assets := make([]*catalogv1.ReconciliationAsset, 0, len(report.Assets))
	for _, row := range report.Assets {
		assets = append(assets, &catalogv1.ReconciliationAsset{AssetId: row.AssetID, Verdict: string(row.Verdict), Cause: row.Cause, CatalogEdges: row.CatalogEdges, ManifestEdges: row.ManifestEdges, ImportEdges: row.ImportEdges})
	}
	counts := map[string]int32{}
	for verdict, count := range report.Distribution {
		counts[string(verdict)] = int32(count)
	}
	return connect.NewResponse(&catalogv1.ReconcileGraphResponse{Assets: assets, Distribution: &catalogv1.ReconciliationDistribution{Counts: counts}}), nil
}

func (h *handler) GetAssetPortContract(_ context.Context, req *connect.Request[catalogv1.GetAssetPortContractRequest]) (*connect.Response[catalogv1.GetAssetPortContractResponse], error) {
	contract, err := portcontract.Build(h.repoRoot, req.Msg.GetAssetId())
	if err != nil {
		return nil, graphError(err)
	}
	ports := make([]*catalogv1.UnmetPort, 0, len(contract.UnmetPorts))
	for _, port := range contract.UnmetPorts {
		ports = append(ports, &catalogv1.UnmetPort{CapabilityId: port.CapabilityID, DemandingAssets: nodesProto(port.DemandingAssets), CandidateSatisfiers: nodesProto(port.CandidateSatisfiers)})
	}
	return connect.NewResponse(&catalogv1.GetAssetPortContractResponse{Contract: &catalogv1.AssetPortContract{AssetId: contract.AssetID, ClosureCount: int32(contract.ClosureCount), SelfContained: contract.SelfContained, UnmetPorts: ports}}), nil
}
