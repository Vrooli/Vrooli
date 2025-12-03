import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchExecutionHistory, type SuiteExecutionResult } from "../lib/api";

export function useExecutions(limit = 6) {
  const query = useQuery<SuiteExecutionResult[]>({
    queryKey: ["executions", limit],
    queryFn: () => fetchExecutionHistory({ limit }),
    refetchInterval: 12000
  });

  const executions = query.data ?? [];

  const lastFailedExecution = useMemo(
    () => executions.find((execution) => execution.success === false),
    [executions]
  );

  return {
    ...query,
    executions,
    lastFailedExecution
  };
}

export function useScenarioHistory(scenarioName: string | null) {
  const query = useQuery<SuiteExecutionResult[]>({
    queryKey: ["scenario-history", scenarioName],
    queryFn: () =>
      scenarioName ? fetchExecutionHistory({ scenario: scenarioName, limit: 20 }) : Promise.resolve([]),
    enabled: Boolean(scenarioName)
  });

  return {
    ...query,
    historyExecutions: query.data ?? []
  };
}

export function useFilteredExecutions(focusScenario: string) {
  const { executions, ...rest } = useExecutions();
  const normalizedFocus = focusScenario.trim().toLowerCase();
  const focusActive = normalizedFocus.length > 0;

  const filteredExecutions = useMemo(() => {
    if (!focusActive) return executions;
    return executions.filter((execution) => execution.scenarioName.toLowerCase() === normalizedFocus);
  }, [executions, focusActive, normalizedFocus]);

  return {
    ...rest,
    executions,
    filteredExecutions,
    focusActive
  };
}
