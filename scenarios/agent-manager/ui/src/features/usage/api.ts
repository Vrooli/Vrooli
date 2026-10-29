// Read hooks for the Usage Intelligence Measures procedures. Every surface
// (Usage page, dashboard strip, run usage panel) reads the same four queries so
// the numbers agree wherever they are shown.

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { measuresClient } from "../stats/api/measuresClient";

export const USAGE_GROUP_BY = ["session", "run", "profile", "workload", "model", "runner", "modality", "tool_kind"] as const;
export type UsageGroupBy = (typeof USAGE_GROUP_BY)[number];

const REFRESH_MS = 60_000;

/** Attribution can take seconds while the API compacts; past this it is a timeout, not a spinner. */
const CALL_OPTIONS = { timeoutMs: 30_000 };

const usageQueryKeys = {
  forecast: ["usage", "forecast"] as const,
  burners: (groupBy: UsageGroupBy, runId: string) => ["usage", "burners", groupBy, runId] as const,
  recommendations: (runId: string) => ["usage", "recommendations", runId] as const,
  coverage: ["usage", "coverage"] as const,
};

export function useUsageForecast() {
  return useQuery({
    queryKey: usageQueryKeys.forecast,
    queryFn: () => measuresClient.usageForecast({}, CALL_OPTIONS),
    refetchInterval: REFRESH_MS,
    retry: false,
  });
}

export function useUsageBurners(groupBy: UsageGroupBy = "session", runId = "") {
  return useQuery({
    queryKey: usageQueryKeys.burners(groupBy, runId),
    queryFn: () => measuresClient.usageBurners({ groupBy, runId }, CALL_OPTIONS),
    // Switching group-by keeps the previous grouping on screen until the new one answers.
    placeholderData: keepPreviousData,
    refetchInterval: REFRESH_MS,
    retry: false,
  });
}

export function useUsageRecommendations(runId = "") {
  return useQuery({
    queryKey: usageQueryKeys.recommendations(runId),
    queryFn: () => measuresClient.usageRecommendations({ runId }, CALL_OPTIONS),
    refetchInterval: REFRESH_MS,
    retry: false,
  });
}

export function useUsageCoverage() {
  return useQuery({
    queryKey: usageQueryKeys.coverage,
    queryFn: () => measuresClient.usageCoverage({}, CALL_OPTIONS),
    refetchInterval: 5 * REFRESH_MS,
    retry: false,
  });
}
