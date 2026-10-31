import { createClient } from "@connectrpc/connect";
import {
  QueueService,
  QueueState,
  type NodeQueue,
  type QueueEntry,
} from "@vrooli/proto-types/vrooli-bridge/v1/queue/queue_pb";

import { transport } from "./client";

/**
 * Typed client for the QueueService — the per-node scheduler view
 * (queue domain, OT-P1-003). ListQueue returns each node's running/queued
 * jobs so the fleet dashboard can show live job status per node without
 * polling individual runs.
 */
export const queueClient = createClient(QueueService, transport);

export { QueueState };
export type { NodeQueue, QueueEntry };
