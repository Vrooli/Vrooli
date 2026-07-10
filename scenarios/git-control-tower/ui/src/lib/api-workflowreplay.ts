// ============================================================================
// Workflow replay API (Plan B §4.4) — typed wrappers over WorkflowReplayService
// ============================================================================
//
// GCT's single-origin proxy over typed Test Genie workflow evidence. Structured run data
// comes over Connect; binary video bytes stream from a GCT REST route, whose
// URL is built by workflowVideoUrl (same-origin, so the UI owns its base).

import { buildApiUrl } from "@vrooli/api-base";
import { workflowReplayClient } from "./connect";
import { API_BASE } from "./api-internals";
import type {
  RunSummary,
  GetRunDetailResponse,
} from "@vrooli/proto-types/git-control-tower/v1/workflowreplay/workflowreplay_pb";

export type { RunSummary, GetRunDetailResponse };

export async function listRecentRuns(scenario: string, limit = 10): Promise<RunSummary[]> {
  const res = await workflowReplayClient.listRecentRuns({ scenario, limit });
  return res.runs;
}

export async function getRunDetail(scenario: string, runId: string): Promise<GetRunDetailResponse> {
  return workflowReplayClient.getRunDetail({ scenario, runId });
}

// workflowVideoUrl builds the same-origin proxy URL from a run-scoped opaque ID.
export function workflowVideoUrl(scenario: string, runId: string, artifactId: string): string {
  const qs = new URLSearchParams({ scenario, artifact_id: artifactId }).toString();
  return buildApiUrl(`/repo/workflow-runs/${encodeURIComponent(runId)}/video?${qs}`, {
    baseUrl: API_BASE,
  });
}
