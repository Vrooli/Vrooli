package graph

import (
	"sort"

	"typescript-code-graph/internal/sidecar"
)

// Normalize folds the sidecar's RawGraph into the domain Graph and
// sorts the result for byte-stability. The function is pure (no I/O,
// no clocks) so the same RawGraph always produces the same Graph
// bytes — a precondition for GraphHash's determinism guarantee.
//
// Decoding rules (per plan §8.3 and §6.3):
//
//   - The sidecar may emit Kind as either the common.v1.NodeKind enum
//     name (e.g. "NODE_KIND_FILE") or the TS-specific TsNodeKind enum
//     name (e.g. "TS_NODE_KIND_COMPONENT"). Strings outside the known
//     set are accepted as raw values; the canonical mapping is below.
//   - Unknown kinds bubble up via the attributes["sidecar_kind"] key
//     so downstream code can still see them.
//   - LeadingComments survive verbatim — REQ-P0-003 — no whitespace
//     trim, no JSDoc parsing.
func Normalize(raw sidecar.RawGraph) Graph {
	nodes := make([]Node, 0, len(raw.Nodes))
	for _, rn := range raw.Nodes {
		kind := decodeNodeKind(rn.Kind)
		attrs := cloneAttrs(rn.Attributes)
		// The TS-specific enum *name* (TS_NODE_KIND_*) rides on
		// attributes["kind"] per the proto envelope contract; the
		// sidecar populates it directly. Nothing to splice in here.
		_ = attrs
		nodes = append(nodes, Node{
			ID:              rn.ID,
			Kind:            kind,
			Name:            rn.Name,
			Path:            rn.Path,
			Attributes:      attrs,
			LeadingComments: append([]string(nil), rn.LeadingComments...),
		})
	}

	edges := make([]Edge, 0, len(raw.Edges))
	for _, re := range raw.Edges {
		edges = append(edges, Edge{
			ID:         re.ID,
			Kind:       decodeEdgeKind(re.Kind),
			From:       re.FromNodeID,
			To:         re.ToNodeID,
			Attributes: cloneAttrs(re.Attributes),
		})
	}

	sortGraph(nodes, edges)
	return Graph{Nodes: nodes, Edges: edges}
}

// sortGraph orders nodes by id and edges by (from, to, kind) so two
// runs against the same raw input produce identical byte streams.
func sortGraph(nodes []Node, edges []Edge) {
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Kind < edges[j].Kind
	})
}

// decodeNodeKind maps a numeric common.v1.NodeKind enum value to the
// domain NodeKind. Values 1..3 are the common envelope (FILE, PACKAGE,
// MODULE); 200..209 are TS-specific (TsNodeKind). Unknown values fall
// back to NodeKindFile.
func decodeNodeKind(k int32) NodeKind {
	switch k {
	case 1:
		return NodeKindFile
	case 2, 3, 200: // PACKAGE, MODULE, TS_NODE_KIND_MODULE
		return NodeKindModule
	case 201:
		return NodeKindComponent
	case 202:
		return NodeKindHook
	case 203:
		return NodeKindClass
	case 204:
		return NodeKindInterface
	case 205:
		return NodeKindType
	case 206:
		return NodeKindFunction
	case 207:
		return NodeKindVar
	case 208:
		return NodeKindConst
	case 209:
		return NodeKindReExport
	default:
		return NodeKindFile
	}
}

// decodeEdgeKind maps a numeric common.v1.EdgeKind enum value to the
// domain EdgeKind. Unknown values default to import.
func decodeEdgeKind(k int32) EdgeKind {
	switch k {
	case 3:
		return EdgeKindReExport
	case 1:
		return EdgeKindImport
	default:
		return EdgeKindImport
	}
}

// cloneAttrs returns a defensive copy of in, or nil if in is empty.
func cloneAttrs(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
