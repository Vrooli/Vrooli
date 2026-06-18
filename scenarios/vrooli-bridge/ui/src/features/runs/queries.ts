import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { runsClient, RunStatus, type Run, type RunEvent } from "../../api/runs";

/** Canonical react-query key for the fleet-wide run-history feed. */
export const RUNS_QUERY_KEY = ["runs", "list"] as const;

/** react-query key for a single run's detail (run + persisted event history). */
export const runDetailKey = (id: string) => ["runs", "detail", id] as const;

/** A run is in-flight (not yet terminal) when it is QUEUED or RUNNING. */
export function isRunActive(status: RunStatus): boolean {
  return status === RunStatus.QUEUED || status === RunStatus.RUNNING;
}

/**
 * The newest-first run-history feed (RunsService.ListRuns). Polled on a short
 * interval so in-flight runs advance toward a terminal status without a manual
 * refresh — there is always either progress or a terminal state, never a frozen
 * spinner.
 */
export function useRunsQuery() {
  return useQuery({
    queryKey: RUNS_QUERY_KEY,
    queryFn: async (): Promise<Run[]> => {
      const resp = await runsClient.listRuns({});
      return resp.runs;
    },
    refetchInterval: 5_000,
  });
}

export interface RunDetail {
  run?: Run;
  events: RunEvent[];
}

/**
 * A single run plus its full persisted event history (GetRun). While the run is
 * active we keep polling so the output grows and the status advances; once
 * terminal we stop. `enabled` lets the drill-in mount lazily.
 */
export function useRunDetailQuery(id: string | null) {
  return useQuery({
    queryKey: runDetailKey(id ?? ""),
    enabled: id != null,
    queryFn: async (): Promise<RunDetail> => {
      const resp = await runsClient.getRun({ id: id ?? "" });
      return { run: resp.run, events: resp.events };
    },
    refetchInterval: (query) => {
      const status = query.state.data?.run?.status;
      return status != null && isRunActive(status) ? 2_000 : false;
    },
  });
}

/**
 * Abort an in-flight run (RunsService.AbortRun). The node stops the job and the
 * run transitions to ABORTED. Refreshes both the feed and the run's detail so
 * the cancel reflects immediately.
 */
export function useAbortRunMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => runsClient.abortRun({ id }),
    onSuccess: (_resp, id) => {
      void queryClient.invalidateQueries({ queryKey: RUNS_QUERY_KEY });
      void queryClient.invalidateQueries({ queryKey: runDetailKey(id) });
    },
  });
}

export { RunStatus };
export type { Run, RunEvent };
