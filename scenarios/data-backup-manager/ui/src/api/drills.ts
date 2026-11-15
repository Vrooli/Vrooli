import { createClient, type Client } from "@connectrpc/connect";
import {
  DrillStatus,
  RecoveryDrillsService,
} from "@vrooli/proto-types/data-backup-manager/v1/drills/drills_pb";
import type { RecoveryDrill } from "@vrooli/proto-types/data-backup-manager/v1/drills/drills_pb";

import { transport } from "./client";

export const drillsClient: Client<typeof RecoveryDrillsService> = createClient(RecoveryDrillsService, transport);

export async function listDrills(planId = "", targetId = ""): Promise<RecoveryDrill[]> {
  const drills: RecoveryDrill[] = [];
  let pageToken = "";
  const seenPageTokens = new Set<string>();
  do {
    if (pageToken !== "" && seenPageTokens.has(pageToken)) {
      throw new Error("recovery drill history pagination repeated a cursor");
    }
    if (pageToken !== "") seenPageTokens.add(pageToken);
    const res = await drillsClient.listDrills({ planId, targetId, pageSize: 100, pageToken });
    drills.push(...res.drills);
    if (res.nextPageToken !== "" && seenPageTokens.has(res.nextPageToken)) {
      throw new Error("recovery drill history pagination repeated a cursor");
    }
    pageToken = res.nextPageToken;
  } while (pageToken !== "");
  return drills;
}

export async function runDrill(planId: string, targetId = "", destinationId = "", idempotencyKey = ""): Promise<RecoveryDrill | undefined> {
  const res = await drillsClient.runDrill({ planId, targetId, destinationId, idempotencyKey });
  return res.drill;
}

export { DrillStatus };
export type { RecoveryDrill };
