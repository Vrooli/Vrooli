/**
 * Restores domain — UI ↔ API boundary over RestoresService. Two modes share
 * the snapshot-selection flow:
 *
 *  - **verify** (`verifyTarget`): non-destructive test-restore to scratch +
 *    checksum. It is the gate that proves a backup is restorable, and it is
 *    safe to encourage — one click from a target row.
 *  - **restore** (`restoreTarget`): writes real data to a caller-chosen
 *    location. Consequential, so the UI gates it behind explicit confirmation.
 */
import { createClient, type Client } from "@connectrpc/connect";
import { RestoresService } from "@vrooli/proto-types/data-backup-manager/v1/restores/restores_pb";
import {
  RestoreMode,
  RestoreStatus,
} from "@vrooli/proto-types/data-backup-manager/v1/restores/restores_pb";
import type { Restore } from "@vrooli/proto-types/data-backup-manager/v1/restores/restores_pb";

import { transport } from "./client";

export const restoresClient: Client<typeof RestoresService> = createClient(
  RestoresService,
  transport,
);

export async function listRestores(targetId = ""): Promise<Restore[]> {
  const res = await restoresClient.listRestores({ targetId });
  return res.restores;
}

export async function getRestore(id: string): Promise<Restore | undefined> {
  const res = await restoresClient.getRestore({ id });
  return res.restore;
}

/** Non-destructive verify (test-restore + checksum). */
export async function verifyTarget(
  targetId: string,
  destinationId: string,
  snapshotId: string,
): Promise<Restore | undefined> {
  const res = await restoresClient.verifyTarget({ targetId, destinationId, snapshotId });
  return res.restore;
}

/** Destructive restore to a caller-chosen location. */
export async function restoreTarget(
  targetId: string,
  destinationId: string,
  snapshotId: string,
  location: string,
): Promise<Restore | undefined> {
  const res = await restoresClient.restoreTarget({ targetId, destinationId, snapshotId, location });
  return res.restore;
}

export { RestoreMode, RestoreStatus };
export type { Restore };
