// Read hooks for the Usage Intelligence Measures procedures. Every surface
// (Usage page, dashboard strip, run usage panel) reads the same queries so
// the numbers agree wherever they are shown.

import { createClient } from "@connectrpc/connect";
import { createScenarioConnectTransport } from "@vrooli/api-base";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { EpisodesService } from "@vrooli/proto-types/agent-manager/v1/domain/episode_pb";
import { REQUEST_DEADLINE_MS, resolveAgentManagerApiBase } from "../../lib/api";
import { measuresClient } from "../stats/api/measuresClient";

export const USAGE_GROUP_BY = ["session", "run", "profile", "workload", "model", "runner", "modality", "tool_kind"] as const;
export type UsageGroupBy = (typeof USAGE_GROUP_BY)[number];

/** The provider window a request is narrowed to. */
export interface UsageWindowScope {
  provider: string;
  pool: string;
  window: string;
}

const REFRESH_MS = 60_000;

/** Attribution can take seconds while the API compacts; past the shared deadline it is an error, not a spinner. */
export const CALL_OPTIONS = { timeoutMs: REQUEST_DEADLINE_MS };

const episodesClient = createClient(EpisodesService, createScenarioConnectTransport({ baseUrl: resolveAgentManagerApiBase() }));

const usageQueryKeys = {
  forecast: ["usage", "forecast"] as const,
  burners: (groupBy: UsageGroupBy, runId: string) => ["usage", "burners", groupBy, runId] as const,
  recommendations: (runId: string, scope?: UsageWindowScope) => ["usage", "recommendations", runId, scope?.provider ?? "", scope?.pool ?? "", scope?.window ?? ""] as const,
  coverage: ["usage", "coverage"] as const,
  friction: (status: string) => ["usage", "friction", status] as const,
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

/** Grouped recommendations for a run, or for one provider window when `scope` is given. */
export function useUsageRecommendations(runId = "", scope?: UsageWindowScope) {
  return useQuery({
    queryKey: usageQueryKeys.recommendations(runId, scope),
    queryFn: () => measuresClient.usageRecommendations({ runId, ...scope }, CALL_OPTIONS),
    placeholderData: keepPreviousData,
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

/** Every recurring friction problem shown on one page; the server's default is a short CLI list. */
export const FRICTION_PAGE_LIMIT = 500;

/** Recurring friction problems; an empty status returns every status. */
export function useUsageFriction(status = "") {
  return useQuery({
    queryKey: usageQueryKeys.friction(status),
    queryFn: () => measuresClient.usageFriction({ status, limit: FRICTION_PAGE_LIMIT }, CALL_OPTIONS),
    refetchInterval: REFRESH_MS,
    retry: false,
  });
}

/** Report files the problem to its owner and resolves with the bug id; dismiss resolves with the new status. */
export function useFrictionActions() {
  const queryClient = useQueryClient();
  const refresh = () => queryClient.invalidateQueries({ queryKey: ["usage", "friction"] });
  const report = useMutation({ mutationFn: (fingerprint: string) => episodesClient.reportFrictionProblem({ fingerprint }, CALL_OPTIONS), onSuccess: refresh });
  const dismiss = useMutation({ mutationFn: (fingerprint: string) => episodesClient.dismissFrictionProblem({ fingerprint }, CALL_OPTIONS), onSuccess: refresh });
  return { report, dismiss };
}
