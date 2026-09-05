// Package rewrite is the Connect-transport edge for the rewrite RPCs.
//
// The two rewrite RPCs (RewritePlan, RewriteApply) live on the same
// TypeScriptCodeGraphService as Extract — so there is no separate
// Connect mount here. Instead, this package exposes RewritePlan and
// RewriteApply helper functions that handlers/graph/handler.go
// delegates to, plus the EndpointDescriptors the codegen registry
// reads.
//
// Substrate boundary: this file owns proto ↔ domain translation and
// nothing else. Connect mount happens in handlers/graph/module.go;
// timing happens in handlers/graph/handler.go.
package rewrite

import (
	rewritedom "typescript-code-graph/internal/rewrite"

	rewritepb "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/rewrite"
)

// protoToDomainOperations translates a slice of proto rewrite.Operation
// into the domain Operation type. Operations whose oneof is unset
// surface as a zero-valued domain Operation; the domain Normalize step
// then rejects them with RewriteErrorInvalidOperation.
func protoToDomainOperations(in []*rewritepb.Operation) []rewritedom.Operation {
	if len(in) == 0 {
		return nil
	}
	out := make([]rewritedom.Operation, 0, len(in))
	for _, op := range in {
		out = append(out, protoToDomainOperation(op))
	}
	return out
}

// protoToDomainOperation is the single-op variant — public-ish so
// adapter_test.go can exercise it directly.
func protoToDomainOperation(op *rewritepb.Operation) rewritedom.Operation {
	if op == nil {
		return rewritedom.Operation{}
	}
	out := rewritedom.Operation{}
	if fm := op.GetFileMove(); fm != nil {
		out.FileMove = &rewritedom.FileMove{
			FromPath: fm.GetFromPath(),
			ToPath:   fm.GetToPath(),
		}
	}
	if ir := op.GetImportRewrite(); ir != nil {
		out.ImportRewrite = &rewritedom.ImportRewrite{
			OldPath: ir.GetOldPath(),
			NewPath: ir.GetNewPath(),
		}
	}
	return out
}

// domainToProtoOperations is the inverse of protoToDomainOperations.
func domainToProtoOperations(in []rewritedom.Operation) []*rewritepb.Operation {
	if len(in) == 0 {
		return nil
	}
	out := make([]*rewritepb.Operation, 0, len(in))
	for _, op := range in {
		out = append(out, domainToProtoOperation(op))
	}
	return out
}

func domainToProtoOperation(op rewritedom.Operation) *rewritepb.Operation {
	if op.FileMove != nil {
		return &rewritepb.Operation{
			Op: &rewritepb.Operation_FileMove{
				FileMove: &rewritepb.FileMove{
					FromPath: op.FileMove.FromPath,
					ToPath:   op.FileMove.ToPath,
				},
			},
		}
	}
	if op.ImportRewrite != nil {
		return &rewritepb.Operation{
			Op: &rewritepb.Operation_ImportRewrite{
				ImportRewrite: &rewritepb.ImportRewrite{
					OldPath: op.ImportRewrite.OldPath,
					NewPath: op.ImportRewrite.NewPath,
				},
			},
		}
	}
	// Empty oneof — return an empty proto so the wire shape stays
	// well-formed; downstream consumers detect the missing arm via
	// GetOp() == nil.
	return &rewritepb.Operation{}
}

// domainResultsToProto projects per-op apply results onto the proto
// OperationResult slice. The Status string is mapped back onto the
// canonical proto enum.
func domainResultsToProto(in []rewritedom.ApplyResult) []*rewritepb.OperationResult {
	if len(in) == 0 {
		return nil
	}
	out := make([]*rewritepb.OperationResult, 0, len(in))
	for _, r := range in {
		out = append(out, &rewritepb.OperationResult{
			Operation: domainToProtoOperation(r.Operation),
			Status:    statusToProto(r.Status),
			Message:   r.Message,
		})
	}
	return out
}

func statusToProto(s string) rewritepb.OperationStatus {
	switch s {
	case rewritedom.StatusOK:
		return rewritepb.OperationStatus_OPERATION_STATUS_OK
	case rewritedom.StatusFailed:
		return rewritepb.OperationStatus_OPERATION_STATUS_FAILED
	default:
		return rewritepb.OperationStatus_OPERATION_STATUS_UNSPECIFIED
	}
}
