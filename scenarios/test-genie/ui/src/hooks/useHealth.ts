import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchHealth, type ApiHealthResponse, type SuiteExecutionResult } from "../lib/api";

export function useHealth() {
  const query = useQuery<ApiHealthResponse>({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 10000
  });

  const queueSnapshot = query.data?.operations?.queue;
  const lastExecution = query.data?.operations?.lastExecution;

  const queueMetrics = useMemo(
    () => [
      { label: "Pending", value: queueSnapshot?.pending ?? 0 },
      { label: "Running", value: queueSnapshot?.running ?? 0 },
      { label: "Completed (24h)", value: queueSnapshot?.completed ?? 0 },
      { label: "Failed (24h)", value: queueSnapshot?.failed ?? 0 }
    ],
    [queueSnapshot?.completed, queueSnapshot?.failed, queueSnapshot?.pending, queueSnapshot?.running]
  );

  const heroExecution = useMemo<SuiteExecutionResult | undefined>(() => {
    if (lastExecution) {
      return {
        executionId: lastExecution.executionId,
        suiteRequestId: undefined,
        scenarioName: lastExecution.scenario,
        startedAt: lastExecution.startedAt,
        completedAt: lastExecution.completedAt,
        success: lastExecution.success,
        preset: lastExecution.preset,
        phases: [],
        phaseSummary: lastExecution.phaseSummary
      };
    }
    return undefined;
  }, [lastExecution]);

  return {
    ...query,
    queueSnapshot,
    queueMetrics,
    heroExecution
  };
}
