import { useCallback, useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  fetchScenarioDetail,
  fetchScenarioSummaries,
  optimizeScenario as optimizeScenarioRequest,
  scanScenario as scanScenarioRequest
} from "../../api/client";

interface ScanOptions {
  apply?: boolean;
  applyResources?: boolean;
  applyScenarios?: boolean;
}

export function useScenarioCatalog() {
  const [selectedScenario, setSelectedScenario] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const summariesQuery = useQuery({
    queryKey: ["scenario-summaries"],
    queryFn: fetchScenarioSummaries,
    refetchOnWindowFocus: false
  });

  const detailQuery = useQuery({
    queryKey: ["scenario-detail", selectedScenario],
    queryFn: () => fetchScenarioDetail(selectedScenario ?? ""),
    enabled: Boolean(selectedScenario),
    refetchOnWindowFocus: false
  });

  const invalidateScenarioData = useCallback(
    async (name: string) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["scenario-summaries"] }),
        queryClient.invalidateQueries({ queryKey: ["scenario-detail", name] })
      ]);
    },
    [queryClient]
  );

  const selectScenario = useCallback(
    (name: string) => {
      setSelectedScenario(name);
    },
    []
  );

  const scanMutation = useMutation({
    mutationFn: ({ name, options }: { name: string; options?: ScanOptions }) =>
      scanScenarioRequest(name, options),
    onSuccess: (_result, variables) => invalidateScenarioData(variables.name)
  });

  const scanScenario = useCallback(
    async (name: string, options?: ScanOptions) => {
      await scanMutation.mutateAsync({ name, options });
    },
    [scanMutation]
  );

  const optimizeMutation = useMutation({
    mutationFn: ({ name, options }: { name: string; options?: { apply?: boolean } }) =>
      optimizeScenarioRequest(name, options),
    onSuccess: (_result, variables) => invalidateScenarioData(variables.name)
  });

  const optimizeScenario = useCallback(
    async (name: string, options?: { apply?: boolean }) => {
      await optimizeMutation.mutateAsync({ name, options });
    },
    [optimizeMutation]
  );

  useEffect(() => {
    if (summariesQuery.error) {
      console.error("Failed to fetch scenarios", summariesQuery.error);
    }
  }, [summariesQuery.error]);

  useEffect(() => {
    if (detailQuery.error) {
      console.error("Failed to fetch scenario detail", detailQuery.error);
    }
  }, [detailQuery.error]);

  const refreshSummaries = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: ["scenario-summaries"] });
  }, [queryClient]);

  return {
    summaries: summariesQuery.data ?? [],
    loadingSummaries: summariesQuery.isLoading || summariesQuery.isFetching,
    selectedScenario,
    detail: detailQuery.data ?? null,
    detailLoading: detailQuery.isLoading || detailQuery.isFetching,
    scanLoading: scanMutation.isPending,
    optimizeLoading: optimizeMutation.isPending,
    selectScenario,
    refreshSummaries,
    scanScenario,
    optimizeScenario
  };
}
