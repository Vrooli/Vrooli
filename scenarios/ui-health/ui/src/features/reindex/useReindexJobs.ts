// useReindexJobs — local tracking + polling for reindex jobs.
//
// The backend does not expose a list-jobs RPC, so this hook persists
// triggered jobs in localStorage and polls each non-terminal job via
// ReindexStatus. Terminal jobs stop polling but remain visible until
// the user clears them.
import { useCallback, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  isTerminal,
  reindex,
  reindexCancel,
  reindexStatus,
  type ReindexState,
  type ReindexStatus,
  type ReindexTrigger,
} from "../../api/reindex";

const TRACKED_STORAGE_KEY = "ui-health.reindex.tracked.v1";
const TRACKED_LIMIT = 50;
const POLL_MS = 1500;

export type TrackedJob = {
  jobId: string;
  scenario: string;
  dryRun: boolean;
  triggeredAt: string;
  plannedUpserts: number;
  plannedDeletes: number;
};

function isTrackedJob(value: unknown): value is TrackedJob {
  if (!value || typeof value !== "object") return false;
  const v = value as Record<string, unknown>;
  return (
    typeof v.jobId === "string" &&
    typeof v.scenario === "string" &&
    typeof v.dryRun === "boolean" &&
    typeof v.triggeredAt === "string" &&
    typeof v.plannedUpserts === "number" &&
    typeof v.plannedDeletes === "number"
  );
}

function readTracked(): TrackedJob[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(TRACKED_STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isTrackedJob);
  } catch {
    return [];
  }
}

function writeTracked(jobs: TrackedJob[]): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(TRACKED_STORAGE_KEY, JSON.stringify(jobs));
  } catch {
    // storage full / disabled — non-essential.
  }
}

export function trackedFromTrigger(
  scenario: string,
  trigger: ReindexTrigger,
): TrackedJob {
  return {
    jobId: trigger.jobId,
    scenario,
    dryRun: trigger.dryRun,
    triggeredAt: new Date().toISOString(),
    plannedUpserts: trigger.plannedUpserts,
    plannedDeletes: trigger.plannedDeletes,
  };
}

export function reindexStatusQueryKey(jobId: string): readonly unknown[] {
  return ["reindex", "status", jobId] as const;
}

export function useTrackedJobs(): {
  jobs: TrackedJob[];
  add: (job: TrackedJob) => void;
  remove: (jobId: string) => void;
  clearTerminal: (statuses: Record<string, ReindexState | undefined>) => void;
  clearAll: () => void;
} {
  const [jobs, setJobs] = useState<TrackedJob[]>(() => readTracked());

  useEffect(() => {
    if (typeof window === "undefined") return;
    const handler = (e: StorageEvent) => {
      if (e.key === TRACKED_STORAGE_KEY) setJobs(readTracked());
    };
    window.addEventListener("storage", handler);
    return () => window.removeEventListener("storage", handler);
  }, []);

  const add = useCallback((job: TrackedJob) => {
    setJobs((prev) => {
      const dedup = prev.filter((j) => j.jobId !== job.jobId);
      const next = [job, ...dedup].slice(0, TRACKED_LIMIT);
      writeTracked(next);
      return next;
    });
  }, []);

  const remove = useCallback((jobId: string) => {
    setJobs((prev) => {
      const next = prev.filter((j) => j.jobId !== jobId);
      writeTracked(next);
      return next;
    });
  }, []);

  const clearTerminal = useCallback(
    (statuses: Record<string, ReindexState | undefined>) => {
      setJobs((prev) => {
        const next = prev.filter((j) => {
          const s = statuses[j.jobId];
          return !(s && isTerminal(s));
        });
        writeTracked(next);
        return next;
      });
    },
    [],
  );

  const clearAll = useCallback(() => {
    setJobs([]);
    writeTracked([]);
  }, []);

  return { jobs, add, remove, clearTerminal, clearAll };
}

export function useJobStatus(jobId: string, enabled = true) {
  return useQuery({
    queryKey: reindexStatusQueryKey(jobId),
    queryFn: () => reindexStatus(jobId),
    enabled: enabled && jobId.length > 0,
    refetchInterval: (q) => {
      const data = q.state.data;
      if (!data) return POLL_MS;
      return isTerminal(data.state) ? false : POLL_MS;
    },
    refetchOnWindowFocus: false,
  });
}

export function useTriggerReindex() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ scenario, dryRun }: { scenario: string; dryRun: boolean }) =>
      reindex(scenario, dryRun),
    onSuccess: (trigger) => {
      // Seed the status cache so the jobs list immediately shows "queued"
      // before the first poll round-trip.
      queryClient.setQueryData(reindexStatusQueryKey(trigger.jobId), {
        jobId: trigger.jobId,
        state: "queued",
        processed: 0,
        total: 0,
        error: "",
      } satisfies ReindexStatus);
    },
  });
}

export function useCancelJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (jobId: string) => reindexCancel(jobId),
    onSuccess: (cancel) => {
      if (!cancel.cancelled) return;
      queryClient.setQueryData<ReindexStatus | undefined>(
        reindexStatusQueryKey(cancel.jobId),
        (prev) =>
          prev
            ? { ...prev, state: "cancelled" as const }
            : {
                jobId: cancel.jobId,
                state: "cancelled" as const,
                processed: 0,
                total: 0,
                error: "",
              },
      );
    },
  });
}

export type JobStatusMap = Record<string, ReindexStatus | undefined>;

export function useStatusMap(jobIds: string[]): JobStatusMap {
  const queryClient = useQueryClient();
  return useMemo<JobStatusMap>(() => {
    const out: JobStatusMap = {};
    for (const id of jobIds) {
      out[id] = queryClient.getQueryData<ReindexStatus>(reindexStatusQueryKey(id));
    }
    return out;
    // Re-derive when any tracked id list changes; per-job updates flow via
    // useJobStatus calls in `JobRow` triggering a re-render.
  }, [jobIds, queryClient]);
}
