import { createClient, type Client } from "@connectrpc/connect";
import { TypeScriptCodeGraphService } from "@vrooli/proto-types/typescript-code-graph/v1/graph/graph_pb";
import type {
  ExtractRequest,
  ExtractResponse,
} from "@vrooli/proto-types/typescript-code-graph/v1/graph/graph_pb";
import type {
  CodeGraph,
  CodeGraphNode,
  CodeGraphEdge,
  CodeGraphWarning,
} from "@vrooli/proto-types/common/v1/code_graph_pb";

import { transport } from "./client";

/**
 * Connect-RPC client for the single TypeScriptCodeGraphService. Extract,
 * RewritePlan, RewriteApply (and the fixture-validation RPCs) all route
 * through this one client because the proto declares one service. Consumers
 * call methods (`tsCodeGraphClient.extract(...)`) through React Query rather
 * than wrapping them in ad-hoc fetch helpers — the generated client owns
 * proto encoding, error parsing, and cancellation.
 *
 * The rewrite-specific helpers live in `./rewrite` but share this client.
 */
export const tsCodeGraphClient: Client<typeof TypeScriptCodeGraphService> = createClient(
  TypeScriptCodeGraphService,
  transport,
);

export type {
  ExtractRequest,
  ExtractResponse,
  CodeGraph,
  CodeGraphNode,
  CodeGraphEdge,
  CodeGraphWarning,
};
