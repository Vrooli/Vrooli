// API client for the reindex domain. Thin wrapper over the generated
// ReindexService Connect client; exports plain TS types and a small
// state enum so callers don't depend on the protobuf shape.
//
// The backend does not expose a list-jobs RPC — job lifetime is the
// caller's problem. `useReindexJobs` (in features/reindex) tracks
// triggered jobs in localStorage and polls each via ReindexStatus.
import { createClient } from "@connectrpc/connect";

import {
  ReindexService,
  type ReindexResponse as ProtoTriggerResponse,
  type ReindexStatusResponse as ProtoStatusResponse,
  type ReindexCancelResponse as ProtoCancelResponse,
} from "@vrooli/proto-types/ui-health/v1/reindex/reindex_pb";

import { transport } from "./client";

export const reindexClient = createClient(ReindexService, transport);

export type ReindexState =
  | "queued"
  | "running"
  | "succeeded"
  | "failed"
  | "cancelled"
  | "unknown";

export const TERMINAL_STATES: readonly ReindexState[] = [
  "succeeded",
  "failed",
  "cancelled",
];

export type ReindexTrigger = {
  jobId: string;
  plannedUpserts: number;
  plannedDeletes: number;
  dryRun: boolean;
};

export type ReindexStatus = {
  jobId: string;
  state: ReindexState;
  processed: number;
  total: number;
  error: string;
};

export type ReindexCancel = {
  jobId: string;
  cancelled: boolean;
};

export async function reindex(
  scenario: string,
  dryRun: boolean,
): Promise<ReindexTrigger> {
  const resp = await reindexClient.reindex({ scenario, dryRun });
  return triggerFromProto(resp);
}

export async function reindexStatus(jobId: string): Promise<ReindexStatus> {
  const resp = await reindexClient.reindexStatus({ jobId });
  return statusFromProto(resp);
}

export async function reindexCancel(jobId: string): Promise<ReindexCancel> {
  const resp = await reindexClient.reindexCancel({ jobId });
  return cancelFromProto(resp);
}

export function reindexStateFromString(s: string): ReindexState {
  switch (s) {
    case "queued":
    case "running":
    case "succeeded":
    case "failed":
    case "cancelled":
      return s;
    default:
      return "unknown";
  }
}

export function isTerminal(state: ReindexState): boolean {
  return TERMINAL_STATES.includes(state);
}

function triggerFromProto(p: ProtoTriggerResponse): ReindexTrigger {
  return {
    jobId: p.jobId,
    plannedUpserts: p.plannedUpserts,
    plannedDeletes: p.plannedDeletes,
    dryRun: p.dryRun,
  };
}

function statusFromProto(p: ProtoStatusResponse): ReindexStatus {
  return {
    jobId: p.jobId,
    state: reindexStateFromString(p.state),
    processed: p.processed,
    total: p.total,
    error: p.error,
  };
}

function cancelFromProto(p: ProtoCancelResponse): ReindexCancel {
  return {
    jobId: p.jobId,
    cancelled: p.cancelled,
  };
}
