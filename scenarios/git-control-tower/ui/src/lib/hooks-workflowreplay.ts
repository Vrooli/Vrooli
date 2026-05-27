// ============================================================================
// Workflow replay hooks (Plan B §4.4) — React Query over WorkflowReplayService
// ============================================================================

import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "./hooks-query-keys";
import { listRecentRuns, getRunDetail } from "./api-workflowreplay";
import type {
  RunSummary,
  GetRunDetailResponse,
} from "./api-workflowreplay";

export function useRecentRuns(
  scenario: string,
  opts: { limit?: number; enabled?: boolean; repoId?: string | null } = {},
) {
  const { limit = 10, enabled = true, repoId } = opts;
  return useQuery<RunSummary[], Error>({
    queryKey: queryKeys.workflowRuns(scenario, repoId),
    queryFn: () => listRecentRuns(scenario, limit),
    enabled: enabled && Boolean(scenario),
    refetchInterval: 15_000,
  });
}

export function useRunDetail(
  scenario: string,
  runId: string,
  opts: { enabled?: boolean; repoId?: string | null } = {},
) {
  const { enabled = true, repoId } = opts;
  return useQuery<GetRunDetailResponse, Error>({
    queryKey: queryKeys.workflowRunDetail(scenario, runId, repoId),
    queryFn: () => getRunDetail(scenario, runId),
    enabled: enabled && Boolean(scenario) && Boolean(runId),
    staleTime: 30_000,
  });
}
