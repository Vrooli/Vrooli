/**
 * Runs query + mutation hooks. A run is a real, potentially slow engine
 * operation, so `useRun` polls while the run is in-flight and stops once it
 * reaches a terminal state. Triggering a run invalidates run history and the
 * posture rollup.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { browseSnapshot, getRun, listRuns, triggerRun } from "../api/runs";
import { isRunInFlight } from "../lib/status";
import { queryKeys } from "./keys";

/** Poll cadence for in-flight runs (ms). */
const RUN_POLL_INTERVAL = 2000;

export function useRuns(planId = "") {
  return useQuery({
    queryKey: queryKeys.runs(planId),
    queryFn: () => listRuns(planId),
    // Poll while any listed run is still in-flight so the history table and the
    // per-plan "last run" badge update live as a run progresses.
    refetchInterval: (query) =>
      (query.state.data ?? []).some((r) => isRunInFlight(r.status)) ? RUN_POLL_INTERVAL : false,
  });
}

export function useRun(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.run(id ?? ""),
    queryFn: () => getRun(id ?? ""),
    enabled: Boolean(id),
    refetchInterval: (query) => {
      const run = query.state.data;
      return run && isRunInFlight(run.status) ? RUN_POLL_INTERVAL : false;
    },
  });
}

export function useTriggerRun() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (planId: string) => triggerRun(planId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["runs"] });
      void qc.invalidateQueries({ queryKey: ["targetStatus"] });
    },
  });
}

export function useSnapshotEntries(
  destinationId: string | undefined,
  snapshotId: string | undefined,
  path = "",
) {
  return useQuery({
    queryKey: ["snapshot", destinationId ?? "", snapshotId ?? "", path],
    queryFn: () => browseSnapshot(destinationId ?? "", snapshotId ?? "", path),
    enabled: Boolean(destinationId && snapshotId),
  });
}
