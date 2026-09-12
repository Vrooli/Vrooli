import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { applyClient } from "../../../api/apply";

/** Stable React Query cache keys for the apply feature. */
export const applyKeys = {
  all: () => ["apply"] as const,
  baseline: (scenario: string) => [...applyKeys.all(), "baseline", scenario] as const,
  history: (scenario: string, domain: string) =>
    [...applyKeys.all(), "history", scenario, domain] as const,
  plan: (scenario: string, domain: string) =>
    [...applyKeys.all(), "plan", scenario, domain] as const,
};

export interface UseBuildBaselineArgs {
  scenario: string;
  enabled?: boolean;
}

export function useBuildBaseline({ scenario, enabled = true }: UseBuildBaselineArgs) {
  return useQuery({
    queryKey: applyKeys.baseline(scenario),
    queryFn: () => applyClient.getBuildBaseline({ scenario }),
    enabled: enabled && scenario.length > 0,
  });
}

export interface UseApplyHistoryArgs {
  scenario: string;
  domain: string;
  enabled?: boolean;
}

export function useApplyHistory({ scenario, domain, enabled = true }: UseApplyHistoryArgs) {
  return useQuery({
    queryKey: applyKeys.history(scenario, domain),
    queryFn: () =>
      applyClient.listApplyHistory({
        scenario,
        domain,
        pageSize: 50,
        pageToken: "",
      }),
    enabled: enabled && scenario.length > 0,
  });
}

export interface PlanApplyArgs {
  scenario: string;
  domain: string;
  conflictIds?: readonly string[];
  dryRun?: boolean;
}

export function usePlanApply() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ scenario, domain, conflictIds, dryRun }: PlanApplyArgs) =>
      applyClient.planApply({
        scenario,
        domain,
        conflictIds: [...(conflictIds ?? [])],
        dryRun: dryRun ?? false,
      }),
    onSuccess: (data, vars) => {
      queryClient.setQueryData(applyKeys.plan(vars.scenario, vars.domain), data);
    },
  });
}

export interface RunApplyArgs {
  scenario: string;
  domain: string;
  planId: string;
}

export function useRunApply() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ planId }: RunApplyArgs) =>
      applyClient.runApply({ planId, acknowledgeV01Unimplemented: true }),
    onSuccess: (_data, vars) => {
      void queryClient.invalidateQueries({
        queryKey: applyKeys.history(vars.scenario, vars.domain),
      });
    },
  });
}

/**
 * Composed read surface for the apply workspace: baseline + history for the
 * (scenario, domain) pair. Plan + run live as mutations so callers control
 * timing explicitly.
 */
export function useApplyWorkspace(scenario: string, domain: string) {
  const baseline = useBuildBaseline({ scenario });
  const history = useApplyHistory({ scenario, domain });
  return { baseline, history };
}
