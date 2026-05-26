/**
 * Runs domain — UI ↔ API boundary over RunsService. A run is one execution of
 * a plan: per target × destination it captures, cap-checks, snapshots, applies
 * retention, and records a per-target outcome. ListTargetStatus is the
 * owner-scoped posture rollup — it carries both "last backed up"
 * (`lastSuccessAt`) and "proven restorable" (`lastVerifiedAt`) so the Overview
 * needs a single call, not a per-target restore-history fan-out.
 */
import { createClient, type Client } from "@connectrpc/connect";
import { RunsService } from "@vrooli/proto-types/data-backup-manager/v1/runs/runs_pb";
import {
  RunStatus,
  TriggerSource,
  TargetOutcomeStatus,
} from "@vrooli/proto-types/data-backup-manager/v1/runs/runs_pb";
import type {
  Run,
  TargetOutcome,
  TargetStatus,
  SnapshotEntry,
} from "@vrooli/proto-types/data-backup-manager/v1/runs/runs_pb";

import { transport } from "./client";

export const runsClient: Client<typeof RunsService> = createClient(RunsService, transport);

export async function listRuns(planId = ""): Promise<Run[]> {
  const res = await runsClient.listRuns({ planId });
  return res.runs;
}

export async function getRun(id: string): Promise<Run | undefined> {
  const res = await runsClient.getRun({ id });
  return res.run;
}

/** Triggers an on-demand run of a plan; returns the (initially closed) run. */
export async function triggerRun(planId: string): Promise<Run | undefined> {
  const res = await runsClient.triggerRun({ planId });
  return res.run;
}

export async function listTargetStatus(owner = ""): Promise<TargetStatus[]> {
  const res = await runsClient.listTargetStatus({ owner });
  return res.statuses;
}

export async function browseSnapshot(
  destinationId: string,
  snapshotId: string,
  path = "",
): Promise<SnapshotEntry[]> {
  const res = await runsClient.browseSnapshot({ destinationId, snapshotId, path });
  return res.entries;
}

export { RunStatus, TriggerSource, TargetOutcomeStatus };
export type { Run, TargetOutcome, TargetStatus, SnapshotEntry };
