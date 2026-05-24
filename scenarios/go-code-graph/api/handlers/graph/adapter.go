package graph

import (
	intgraph "go-code-graph/internal/graph"
	intrewrite "go-code-graph/internal/rewrite"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	rewrite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/rewrite"
)

// domainToProtoGraph projects the domain Graph onto the shared common.v1
// envelope. Pure data shaping; no I/O.
func domainToProtoGraph(g intgraph.Graph) *commonv1.CodeGraph {
	out := &commonv1.CodeGraph{
		Nodes: make([]*commonv1.CodeGraphNode, 0, len(g.Nodes)),
		Edges: make([]*commonv1.CodeGraphEdge, 0, len(g.Edges)),
	}
	for _, n := range g.Nodes {
		out.Nodes = append(out.Nodes, &commonv1.CodeGraphNode{
			Id:         n.ID,
			Kind:       nodeKindToProto(n.Kind),
			Name:       n.Name,
			Path:       n.Path,
			Attributes: cloneAttributes(n.Attributes, n.Kind),
		})
	}
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, &commonv1.CodeGraphEdge{
			Id:         e.ID,
			Kind:       edgeKindToProto(e.Kind),
			FromNodeId: e.From,
			ToNodeId:   e.To,
			Attributes: cloneAttributes(e.Attributes, ""),
		})
	}
	return out
}

// nodeKindToProto folds Go-specific symbol kinds onto the common
// envelope's primary NodeKind, recording the typed Go kind via the
// "kind" attribute (see common/v1/code_graph.proto comments).
func nodeKindToProto(k intgraph.NodeKind) commonv1.NodeKind {
	switch k {
	case intgraph.NodeKindFile:
		return commonv1.NodeKind_NODE_KIND_FILE
	case intgraph.NodeKindPackage:
		return commonv1.NodeKind_NODE_KIND_PACKAGE
	case intgraph.NodeKindModule:
		return commonv1.NodeKind_NODE_KIND_MODULE
	default:
		// Go-specific symbol kinds (go_type, go_func, ...) are
		// represented as PACKAGE-scoped attributes; we still need a
		// primary NodeKind, and per the envelope contract they ride
		// under the package-scoped node family.
		return commonv1.NodeKind_NODE_KIND_PACKAGE
	}
}

// cloneAttributes returns a defensive copy of attrs with the Go-typed
// node kind merged in when nodeKind is a Go-specific symbol kind.
// Returning nil for empty input keeps proto-zero-value semantics.
func cloneAttributes(attrs map[string]string, nodeKind intgraph.NodeKind) map[string]string {
	if len(attrs) == 0 && !isGoSymbolKind(nodeKind) {
		return nil
	}
	out := make(map[string]string, len(attrs)+1)
	for k, v := range attrs {
		out[k] = v
	}
	if isGoSymbolKind(nodeKind) {
		out["kind"] = string(nodeKind)
	}
	return out
}

func isGoSymbolKind(k intgraph.NodeKind) bool {
	switch k {
	case intgraph.NodeKindType, intgraph.NodeKindFunc, intgraph.NodeKindVar,
		intgraph.NodeKindConst, intgraph.NodeKindInterface, intgraph.NodeKindMethod:
		return true
	}
	return false
}

func edgeKindToProto(k intgraph.EdgeKind) commonv1.EdgeKind {
	switch k {
	case intgraph.EdgeKindImport:
		return commonv1.EdgeKind_EDGE_KIND_IMPORT
	case intgraph.EdgeKindIntraPackageRef:
		return commonv1.EdgeKind_EDGE_KIND_INTRA_PACKAGE_REF
	default:
		return commonv1.EdgeKind_EDGE_KIND_UNSPECIFIED
	}
}

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

// -----------------------------------------------------------------------------
// Rewrite operation adapters (proto ↔ domain)
// -----------------------------------------------------------------------------

// protoOperationsToDomain translates the proto Operation oneof list to
// the domain's sealed Operation interface. Unrecognized variants are
// dropped silently — the Service's ValidateOperations will then surface
// an empty-list error if every input was malformed.
func protoOperationsToDomain(in []*rewrite_v1.Operation) []intrewrite.Operation {
	if len(in) == 0 {
		return nil
	}
	out := make([]intrewrite.Operation, 0, len(in))
	for _, op := range in {
		if op == nil {
			continue
		}
		switch v := op.GetOp().(type) {
		case *rewrite_v1.Operation_FileMove:
			if v == nil || v.FileMove == nil {
				continue
			}
			out = append(out, intrewrite.FileMove{
				From: v.FileMove.GetFromPath(),
				To:   v.FileMove.GetToPath(),
			})
		case *rewrite_v1.Operation_ImportRewrite:
			if v == nil || v.ImportRewrite == nil {
				continue
			}
			out = append(out, intrewrite.ImportRewrite{
				Old: v.ImportRewrite.GetOldPath(),
				New: v.ImportRewrite.GetNewPath(),
			})
		}
	}
	return out
}

// domainOperationsToProto projects domain operations back onto the
// proto wire shape.
func domainOperationsToProto(in []intrewrite.Operation) []*rewrite_v1.Operation {
	if len(in) == 0 {
		return nil
	}
	out := make([]*rewrite_v1.Operation, 0, len(in))
	for _, op := range in {
		out = append(out, domainOperationToProto(op))
	}
	return out
}

// domainOperationToProto handles a single Operation. Returns a proto
// Operation with the correct oneof variant set.
func domainOperationToProto(op intrewrite.Operation) *rewrite_v1.Operation {
	switch o := op.(type) {
	case intrewrite.FileMove:
		return &rewrite_v1.Operation{
			Op: &rewrite_v1.Operation_FileMove{
				FileMove: &rewrite_v1.FileMove{FromPath: o.From, ToPath: o.To},
			},
		}
	case intrewrite.ImportRewrite:
		return &rewrite_v1.Operation{
			Op: &rewrite_v1.Operation_ImportRewrite{
				ImportRewrite: &rewrite_v1.ImportRewrite{OldPath: o.Old, NewPath: o.New},
			},
		}
	}
	return &rewrite_v1.Operation{}
}

// domainOperationResultsToProto projects per-op apply results back onto
// the proto wire shape.
func domainOperationResultsToProto(in []intrewrite.OperationResult) []*rewrite_v1.OperationResult {
	if len(in) == 0 {
		return nil
	}
	out := make([]*rewrite_v1.OperationResult, 0, len(in))
	for _, r := range in {
		out = append(out, &rewrite_v1.OperationResult{
			Operation: domainOperationToProto(r.Operation),
			Status:    domainOperationStatusToProto(r.Status),
			Message:   r.Message,
		})
	}
	return out
}

// domainOperationStatusToProto maps the domain enum to the proto enum.
func domainOperationStatusToProto(s intrewrite.OperationStatus) rewrite_v1.OperationStatus {
	switch s {
	case intrewrite.OperationStatusOK:
		return rewrite_v1.OperationStatus_OPERATION_STATUS_OK
	case intrewrite.OperationStatusFailed:
		return rewrite_v1.OperationStatus_OPERATION_STATUS_FAILED
	default:
		return rewrite_v1.OperationStatus_OPERATION_STATUS_UNSPECIFIED
	}
}
