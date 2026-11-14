import { createClient, type Client } from "@connectrpc/connect";
import { GraphService } from "@vrooli/proto-types/architecture-cartographer/v1/graph/graph_pb";

import { transport } from "./client";

/**
 * Connect-RPC client for the cartographer GraphService. Consumers should
 * call methods (`graphClient.extractGraph(...)`) directly through React
 * Query rather than wrapping them in ad-hoc fetch helpers — the generated
 * client owns proto encoding, error parsing, and cancellation.
 */
export const graphClient: Client<typeof GraphService> = createClient(GraphService, transport);
