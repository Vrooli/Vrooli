import { createClient } from "@connectrpc/connect";
import {
  NodeRegistryService,
  NodeStatus,
  type Node,
} from "@vrooli/proto-types/vrooli-bridge/v1/registry/registry_pb";

import { transport } from "./client";

/**
 * Typed client for the NodeRegistryService — the fleet's durable node identity
 * (registry domain, OT-P0-001). The owner-gated RPCs (register/list/get/update/
 * revoke) require an owner JWT; the live presence overlay (online / status /
 * last-seen) is stamped server-side, so the UI just reads `online` / `status`
 * off each Node. Phase 2 adds pairing; Phase 5 grows this into the full fleet
 * dashboard.
 */
export const nodesClient = createClient(NodeRegistryService, transport);

export { NodeStatus };
export type { Node };
