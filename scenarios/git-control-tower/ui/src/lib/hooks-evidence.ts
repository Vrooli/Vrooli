import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getRun, listEvidence, listRuns, startRun, type ArtifactFilters, type EvidenceFilters, type RunFilters, type StartRunInput } from "./api-evidence";
import { queryKeys } from "./hooks-query-keys";

export function useRuns(scenario: string, filters: RunFilters = {}, enabled = true, repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.testRuns(scenario, filters, repoId),
    queryFn: () => listRuns(scenario, filters),
    enabled: enabled && Boolean(scenario),
    refetchInterval: 15_000,
  });
}

export function useRun(scenario: string, runId: string, filters: ArtifactFilters = {}, enabled = true, repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.testRun(scenario, runId, filters, repoId),
    queryFn: () => getRun(scenario, runId, filters),
    enabled: enabled && Boolean(scenario) && Boolean(runId),
    staleTime: 30_000,
  });
}

export function useEvidence(scenario: string, filters: EvidenceFilters = {}, enabled = true, repoId?: string | null) {
  return useQuery({
    queryKey: queryKeys.evidence(scenario, filters, repoId),
    queryFn: () => listEvidence(scenario, filters),
    enabled: enabled && Boolean(scenario),
    refetchInterval: 15_000,
  });
}

export function useStartRun(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: StartRunInput) => startRun(input),
    onSuccess: (_result, input) => {
      queryClient.invalidateQueries({ queryKey: ["test-runs", repoId ?? "default", input.scenario] });
      queryClient.invalidateQueries({ queryKey: ["evidence", repoId ?? "default", input.scenario] });
    },
  });
}
