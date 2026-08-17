package components

import (
	"path/filepath"

	"react-component-library/internal/assetgraph"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
)

// catalogIndex builds the desired-state graph once per request. It was
// previously rebuilt inside enrichCatalogProjection, so a list of 200
// components re-read all 410 catalog JSON files and re-walked the 848-edge
// graph 200 times. Callers that enrich more than one row must resolve the
// index once and pass it in.
func (h *connectHandler) catalogIndex() (*assetgraph.Index, error) {
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(filepath.Dir(h.deps.SourceRoot), "catalog"))
	if err != nil {
		return nil, err
	}
	return assetgraph.Build(assets)
}

// enrichCatalogProjection joins the indexed implementation row to the
// desired-state catalog read model. The components registry remains the
// source of implementation metadata; this projection only adds placement and
// graph facts needed by catalog navigation.
//
// A nil index means the catalog could not be read at all; that is a whole-
// request condition rather than a per-row one, so the caller decides how to
// report it and this function simply leaves the projection unset.
func (h *connectHandler) enrichCatalogProjection(index *assetgraph.Index, component *components.Component) {
	if index == nil {
		return
	}
	assetID := component.CatalogID
	if assetID == "" {
		return
	}
	node, err := index.Node(assetID)
	if err != nil {
		return
	}
	component.CatalogDomain = node.Domain
	component.CatalogDomainOrder = node.DomainOrder
	component.CatalogRung = int(node.Rung)
	component.CatalogRungName = node.RungName
	_, transitive, err := index.Dependents(assetID)
	if err == nil {
		component.TransitiveDependentCount = len(transitive)
	}
}
