package graph

import (
	intgraph "go-code-graph/internal/graph"
	intrewrite "go-code-graph/internal/rewrite"

	"github.com/vrooli/api-core/gograph"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	rewrite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/rewrite"
)

// domainToProtoGraph projects the domain Graph onto the shared common.v1
// envelope. Pure data shaping; no I/O.
func domainToProtoGraph(g intgraph.Graph) *commonv1.CodeGraph {
	return gograph.ToProtoGraph(g)
}

func warningsToProto(ws []intgraph.Warning) []*commonv1.CodeGraphWarning {
	return gograph.WarningsToProto(ws)
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
