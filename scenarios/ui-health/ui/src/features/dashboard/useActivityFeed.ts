// useActivityFeed — merges recent validation runs and tracked reindex
// jobs into a single time-sorted feed for the Dashboard.
//
// Both inputs already live in localStorage via their respective feature
// hooks; this hook is a pure aggregator + sorter.
import { useMemo } from "react";

import { useRecentRuns, type RecentRun } from "../validation/useValidation";
import { useTrackedJobs, type TrackedJob } from "../reindex/useReindexJobs";

export type ActivityItem =
  | {
      kind: "validation";
      id: string;
      timestamp: string;
      run: RecentRun;
    }
  | {
      kind: "reindex";
      id: string;
      timestamp: string;
      job: TrackedJob;
    };

const MAX_ITEMS = 12;

export function mergeActivity(
  runs: RecentRun[],
  jobs: TrackedJob[],
  limit = MAX_ITEMS,
): ActivityItem[] {
  const validation: ActivityItem[] = runs.map((r) => ({
    kind: "validation",
    id: `validation:${r.scenario}:${r.ranAt}`,
    timestamp: r.ranAt,
    run: r,
  }));
  const reindex: ActivityItem[] = jobs.map((j) => ({
    kind: "reindex",
    id: `reindex:${j.jobId}`,
    timestamp: j.triggeredAt,
    job: j,
  }));
  const merged = [...validation, ...reindex];
  merged.sort((a, b) => (a.timestamp < b.timestamp ? 1 : a.timestamp > b.timestamp ? -1 : 0));
  return merged.slice(0, limit);
}

export function useActivityFeed(limit = MAX_ITEMS): ActivityItem[] {
  const { runs } = useRecentRuns();
  const { jobs } = useTrackedJobs();
  return useMemo(() => mergeActivity(runs, jobs, limit), [runs, jobs, limit]);
}
