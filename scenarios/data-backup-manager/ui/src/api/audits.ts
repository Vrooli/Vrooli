/**
 * Audits domain — UI ↔ API boundary over AuditsService. A snapshot audit is a
 * scenario-agnostic proof: it restores a snapshot to scratch, captures the live
 * target to scratch (read-only on live), and compares the two trees by generic
 * signals only (counts, bytes, path-list/content hashes, per-SQLite
 * integrity/schema). RunSnapshotAudit is async — it returns a requested record;
 * the hook layer polls getAudit for the verdict.
 */
import { createClient, type Client } from "@connectrpc/connect";
import { AuditsService } from "@vrooli/proto-types/data-backup-manager/v1/audits/audits_pb";
import { AuditStatus } from "@vrooli/proto-types/data-backup-manager/v1/audits/audits_pb";
import type { Audit } from "@vrooli/proto-types/data-backup-manager/v1/audits/audits_pb";

import { transport } from "./client";

export const auditsClient: Client<typeof AuditsService> = createClient(AuditsService, transport);

export async function listAudits(targetId = ""): Promise<Audit[]> {
  const res = await auditsClient.listAudits({ targetId });
  return res.audits;
}

export async function getAudit(id: string): Promise<Audit | undefined> {
  const res = await auditsClient.getAudit({ id });
  return res.audit;
}

export interface RunAuditInput {
  targetId: string;
  destinationId: string;
  snapshotId: string;
  includeContentHash?: boolean;
  includeSqliteChecks?: boolean;
}

/** Starts a snapshot audit; returns the requested (non-terminal) record. */
export async function runSnapshotAudit(input: RunAuditInput): Promise<Audit | undefined> {
  const res = await auditsClient.runSnapshotAudit({
    targetId: input.targetId,
    destinationId: input.destinationId,
    snapshotId: input.snapshotId,
    // Default the expensive-but-stronger proofs on; callers opt out for huge trees.
    includeContentHash: input.includeContentHash ?? true,
    includeSqliteChecks: input.includeSqliteChecks ?? true,
  });
  return res.audit;
}

/** True once an audit has reached a terminal state (completed or failed). */
export function isTerminalAudit(status: AuditStatus): boolean {
  return status === AuditStatus.COMPLETED || status === AuditStatus.FAILED;
}

export { AuditStatus };
export type { Audit };
