import { create } from "@bufbuild/protobuf";
import {
  OperationSchema,
  FileMoveSchema,
  ImportRewriteSchema,
  OperationStatus,
} from "@vrooli/proto-types/go-code-graph/v1/rewrite/rewrite_pb";
import type {
  Operation,
  OperationResult,
} from "@vrooli/proto-types/go-code-graph/v1/rewrite/rewrite_pb";
import type {
  RewritePlanResponse,
  RewriteApplyResponse,
} from "@vrooli/proto-types/go-code-graph/v1/graph/graph_pb";

// Re-export the shared GoCodeGraphService client so rewrite controllers do not
// reach across to ./graph for the transport. Same client, narrower import site.
export { goCodeGraphClient } from "./graph";

/**
 * Build a proto FileMove Operation from a typed editor row. Centralizing the
 * oneof construction keeps the `op.case` discriminant out of UI components.
 */
export function makeFileMoveOp(fromPath: string, toPath: string): Operation {
  return create(OperationSchema, {
    op: {
      case: "fileMove",
      value: create(FileMoveSchema, { fromPath, toPath }),
    },
  });
}

/**
 * Build a proto ImportRewrite Operation from a typed editor row.
 */
export function makeImportRewriteOp(oldPath: string, newPath: string): Operation {
  return create(OperationSchema, {
    op: {
      case: "importRewrite",
      value: create(ImportRewriteSchema, { oldPath, newPath }),
    },
  });
}

export { OperationStatus };
export type { Operation, OperationResult, RewritePlanResponse, RewriteApplyResponse };
