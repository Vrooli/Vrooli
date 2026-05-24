package graph

import (
	intgraph "typescript-code-graph/internal/graph"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
)

// domainToProtoGraph projects the domain Graph onto the shared
// common.v1 envelope. Pure data shaping; no I/O.
//
// The CodeGraphNode.kind primary enum uses the shared common range
// (FILE/PACKAGE/MODULE); TS-specific kinds (component/hook/...) ride
// under attributes["kind"] = "<TsNodeKind name>" per plan §8.3.
// LeadingComments[] survives verbatim — REQ-P0-003.
func domainToProtoGraph(g intgraph.Graph) *commonv1.CodeGraph {
	out := &commonv1.CodeGraph{
		Nodes: make([]*commonv1.CodeGraphNode, 0, len(g.Nodes)),
		Edges: make([]*commonv1.CodeGraphEdge, 0, len(g.Edges)),
	}
	for _, n := range g.Nodes {
		out.Nodes = append(out.Nodes, &commonv1.CodeGraphNode{
			Id:              n.ID,
			Kind:            nodeKindToProto(n.Kind),
			Name:            n.Name,
			Path:            n.Path,
			Attributes:      cloneAttributesWithKind(n.Attributes, n.Kind),
			LeadingComments: append([]string(nil), n.LeadingComments...),
		})
	}
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, &commonv1.CodeGraphEdge{
			Id:         e.ID,
			Kind:       edgeKindToProto(e.Kind),
			FromNodeId: e.From,
			ToNodeId:   e.To,
			Attributes: cloneAttributesWithKind(e.Attributes, ""),
		})
	}
	return out
}

// nodeKindToProto folds the TS-specific domain kind onto the common
// envelope's primary NodeKind. Symbol kinds land under PACKAGE (the
// envelope's catch-all for non-file, non-module structures) and the
// TS-specific kind is preserved via attributes["kind"].
func nodeKindToProto(k intgraph.NodeKind) commonv1.NodeKind {
	switch k {
	case intgraph.NodeKindFile:
		return commonv1.NodeKind_NODE_KIND_FILE
	case intgraph.NodeKindModule:
		return commonv1.NodeKind_NODE_KIND_MODULE
	default:
		return commonv1.NodeKind_NODE_KIND_PACKAGE
	}
}

// cloneAttributesWithKind defensively copies attrs; when nodeKind is a
// TS-specific symbol kind, ensures attributes["kind"] is set to the
// canonical TsNodeKind enum-name string (Normalize already does this
// for nodes that arrived with a TS_NODE_KIND_* string, but this is
// belt-and-suspenders for adapters that mint Graph values directly in
// tests).
func cloneAttributesWithKind(attrs map[string]string, nodeKind intgraph.NodeKind) map[string]string {
	if len(attrs) == 0 && !isTsSymbolKind(nodeKind) {
		return nil
	}
	out := make(map[string]string, len(attrs)+1)
	for k, v := range attrs {
		out[k] = v
	}
	if isTsSymbolKind(nodeKind) {
		if _, ok := out["kind"]; !ok {
			out["kind"] = tsKindEnumName(nodeKind)
		}
	}
	return out
}

func isTsSymbolKind(k intgraph.NodeKind) bool {
	switch k {
	case intgraph.NodeKindComponent, intgraph.NodeKindHook,
		intgraph.NodeKindClass, intgraph.NodeKindInterface,
		intgraph.NodeKindType, intgraph.NodeKindFunction,
		intgraph.NodeKindVar, intgraph.NodeKindConst,
		intgraph.NodeKindReExport:
		return true
	}
	return false
}

func tsKindEnumName(k intgraph.NodeKind) string {
	switch k {
	case intgraph.NodeKindComponent:
		return "TS_NODE_KIND_COMPONENT"
	case intgraph.NodeKindHook:
		return "TS_NODE_KIND_HOOK"
	case intgraph.NodeKindClass:
		return "TS_NODE_KIND_CLASS"
	case intgraph.NodeKindInterface:
		return "TS_NODE_KIND_INTERFACE"
	case intgraph.NodeKindType:
		return "TS_NODE_KIND_TYPE"
	case intgraph.NodeKindFunction:
		return "TS_NODE_KIND_FUNCTION"
	case intgraph.NodeKindVar:
		return "TS_NODE_KIND_VAR"
	case intgraph.NodeKindConst:
		return "TS_NODE_KIND_CONST"
	case intgraph.NodeKindReExport:
		return "TS_NODE_KIND_RE_EXPORT"
	}
	return ""
}

func edgeKindToProto(k intgraph.EdgeKind) commonv1.EdgeKind {
	switch k {
	case intgraph.EdgeKindImport:
		return commonv1.EdgeKind_EDGE_KIND_IMPORT
	case intgraph.EdgeKindReExport:
		return commonv1.EdgeKind_EDGE_KIND_RE_EXPORT
	default:
		return commonv1.EdgeKind_EDGE_KIND_UNSPECIFIED
	}
}

// warningsToProto folds domain Warnings onto the shared envelope.
func warningsToProto(ws []intgraph.Warning) []*commonv1.CodeGraphWarning {
	if len(ws) == 0 {
		return nil
	}
	out := make([]*commonv1.CodeGraphWarning, 0, len(ws))
	for _, w := range ws {
		out = append(out, &commonv1.CodeGraphWarning{
			Kind:    warningKindToProto(w.Kind),
			File:    w.File,
			Message: w.Message,
		})
	}
	return out
}

func warningKindToProto(k intgraph.WarningKind) commonv1.CodeGraphWarningKind {
	switch k {
	case intgraph.WarningKindParseError:
		return commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_PARSE_ERROR
	case intgraph.WarningKindUnresolvedImport:
		return commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_UNRESOLVED_IMPORT
	case intgraph.WarningKindTypeCheckFailure:
		return commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_TYPE_CHECK_FAILURE
	default:
		return commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_UNSPECIFIED
	}
}

// requestToInput translates the proto ExtractRequest into the domain
// ExtractInput. Centralized so the handler's flow reads top-to-bottom
// without inline payload shaping.
func requestToInput(req *graphv1.ExtractRequest) intgraph.ExtractInput {
	return intgraph.ExtractInput{
		ScenarioPath: req.GetScenarioPath(),
	}
}
