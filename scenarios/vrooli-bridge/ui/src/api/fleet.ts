import { createClient } from "@connectrpc/connect";
import {
  FleetService,
  RolloutStatus,
  NodeRolloutDisposition,
  type Rollout,
  type NodeRolloutResult,
} from "@vrooli/proto-types/vrooli-bridge/v1/fleet/fleet_pb";

import { transport } from "./client";

/**
 * Typed client for the FleetService — owner-gated fleet-wide rollouts
 * (fleet domain, OT-P1-001). RollFleet stages a typed job across many nodes;
 * GetRollout/ListRollouts report per-node disposition. Phase 5 surfaces the
 * fleet dashboard; rollout history is consumed here so the dashboard can show
 * what was dispatched across the fleet.
 */
export const fleetClient = createClient(FleetService, transport);

export { RolloutStatus, NodeRolloutDisposition };
export type { Rollout, NodeRolloutResult };
