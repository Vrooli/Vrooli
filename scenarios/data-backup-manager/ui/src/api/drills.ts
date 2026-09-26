import { createClient, type Client } from "@connectrpc/connect";
import {
  DrillStatus,
  RecoveryDrillsService,
} from "@vrooli/proto-types/data-backup-manager/v1/drills/drills_pb";
import type { RecoveryDrill } from "@vrooli/proto-types/data-backup-manager/v1/drills/drills_pb";

import { transport } from "./client";

export const drillsClient: Client<typeof RecoveryDrillsService> = createClient(RecoveryDrillsService, transport);

export async function listDrills(planId = "", targetId = ""): Promise<RecoveryDrill[]> {
  const res = await drillsClient.listDrills({ planId, targetId });
  return res.drills;
}

export async function runDrill(planId: string, targetId = "", destinationId = "", idempotencyKey = ""): Promise<RecoveryDrill | undefined> {
  const res = await drillsClient.runDrill({ planId, targetId, destinationId, idempotencyKey });
  return res.drill;
}

export { DrillStatus };
export type { RecoveryDrill };
