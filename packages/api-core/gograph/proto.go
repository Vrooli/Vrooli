package gograph

import commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"

// ToProtoGraph is the single projection from the Go extraction graph to the
// shared wire envelope used by Go Code Graph and Code Facts.
func ToProtoGraph(g Graph) *commonv1.CodeGraph {
	out := &commonv1.CodeGraph{
		Nodes: make([]*commonv1.CodeGraphNode, 0, len(g.Nodes)),
		Edges: make([]*commonv1.CodeGraphEdge, 0, len(g.Edges)),
	}
	for _, node := range g.Nodes {
		out.Nodes = append(out.Nodes, &commonv1.CodeGraphNode{
			Id: node.ID, Kind: protoNodeKind(node.Kind), Name: node.Name, Path: node.Path,
			Attributes: protoAttributes(node.Attributes, node.Kind),
		})
	}
	for _, edge := range g.Edges {
		out.Edges = append(out.Edges, &commonv1.CodeGraphEdge{
			Id: edge.ID, Kind: protoEdgeKind(edge.Kind), FromNodeId: edge.From, ToNodeId: edge.To,
			Attributes: protoAttributes(edge.Attributes, ""),
		})
	}
	return out
}

// WarningsToProto projects extraction warnings onto the shared wire envelope.
func WarningsToProto(warnings []Warning) []*commonv1.CodeGraphWarning {
	if len(warnings) == 0 {
		return nil
	}
	out := make([]*commonv1.CodeGraphWarning, 0, len(warnings))
	for _, warning := range warnings {
		kind := commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_UNSPECIFIED
		switch warning.Kind {
		case WarningKindParseError:
			kind = commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_PARSE_ERROR
		case WarningKindUnresolvedImport:
			kind = commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_UNRESOLVED_IMPORT
		case WarningKindTypeCheckFailure:
			kind = commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_TYPE_CHECK_FAILURE
		case WarningKindUnresolvedSymbol:
			kind = commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_AMBIGUOUS_DECLARATION
		}
		out = append(out, &commonv1.CodeGraphWarning{Kind: kind, File: warning.File, Message: warning.Message})
	}
	return out
}

func protoNodeKind(kind NodeKind) commonv1.NodeKind {
	switch kind {
	case NodeKindFile:
		return commonv1.NodeKind_NODE_KIND_FILE
	case NodeKindPackage:
		return commonv1.NodeKind_NODE_KIND_PACKAGE
	case NodeKindModule:
		return commonv1.NodeKind_NODE_KIND_MODULE
	default:
		return commonv1.NodeKind_NODE_KIND_PACKAGE
	}
}

func protoEdgeKind(kind EdgeKind) commonv1.EdgeKind {
	switch kind {
	case EdgeKindImport:
		return commonv1.EdgeKind_EDGE_KIND_IMPORT
	case EdgeKindIntraPackageRef:
		return commonv1.EdgeKind_EDGE_KIND_INTRA_PACKAGE_REF
	default:
		return commonv1.EdgeKind_EDGE_KIND_UNSPECIFIED
	}
}

func protoAttributes(attrs map[string]string, nodeKind NodeKind) map[string]string {
	if len(attrs) == 0 && !isGoSymbolKind(nodeKind) {
		return nil
	}
	out := make(map[string]string, len(attrs)+1)
	for key, value := range attrs {
		out[key] = value
	}
	if isGoSymbolKind(nodeKind) {
		out["kind"] = string(nodeKind)
	}
	return out
}

func isGoSymbolKind(kind NodeKind) bool {
	switch kind {
	case NodeKindType, NodeKindFunc, NodeKindVar, NodeKindConst, NodeKindInterface,
		NodeKindMethod, NodeKindImportSpec, NodeKindReference, NodeKindCall,
		NodeKindTypeUsage, NodeKindRouteRegistration:
		return true
	default:
		return false
	}
}
