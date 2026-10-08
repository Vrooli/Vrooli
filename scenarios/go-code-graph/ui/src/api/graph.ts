import { createClient, type Client } from "@connectrpc/connect";
import { GoCodeGraphService } from "@vrooli/proto-types/go-code-graph/v1/graph/graph_pb";
import type {
  ExtractRequest,
  ExtractResponse,
} from "@vrooli/proto-types/go-code-graph/v1/graph/graph_pb";
import type {
  CodeGraph,
  CodeGraphNode,
  CodeGraphEdge,
  CodeGraphWarning,
} from "@vrooli/proto-types/common/v1/code_graph_pb";

import { transport } from "./client";

/**
 * Connect-RPC client for the single GoCodeGraphService. Extract, RewritePlan,
 * RewriteApply (and the fixture-validation RPCs) all route through this one
 * client because the proto declares one service. Consumers call methods
 * (`goCodeGraphClient.extract(...)`) through React Query rather than wrapping
 * them in ad-hoc fetch helpers — the generated client owns proto encoding,
 * error parsing, and cancellation.
 *
 * The rewrite-specific helpers live in `./rewrite` but share this client.
 */
export const goCodeGraphClient: Client<typeof GoCodeGraphService> = createClient(
  GoCodeGraphService,
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
