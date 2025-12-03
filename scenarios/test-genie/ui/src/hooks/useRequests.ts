import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchSuiteRequests, type SuiteRequest, type SuiteExecutionResult } from "../lib/api";
import { selectActionableRequest, isActionableRequest } from "../lib/requestHelpers";
import { parseTimestamp } from "../lib/formatters";
import type { FocusQueueStats } from "../types";

export function useRequests() {
  const query = useQuery<SuiteRequest[]>({
    queryKey: ["suite-requests"],
    queryFn: fetchSuiteRequests,
    refetchInterval: 10000
  });

  const requests = query.data ?? [];

  const actionableRequest = useMemo(() => selectActionableRequest(requests), [requests]);

  const queuePendingCount = useMemo(
    () => requests.filter((req) => isActionableRequest(req.status)).length,
    [requests]
  );

  return {
    ...query,
    requests,
    actionableRequest,
    queuePendingCount
  };
}

export function useFilteredRequests(focusScenario: string) {
  const { requests, ...rest } = useRequests();
  const normalizedFocus = focusScenario.trim().toLowerCase();
  const focusActive = normalizedFocus.length > 0;

  const filteredRequests = useMemo(() => {
    if (!focusActive) return requests;
    return requests.filter((req) => req.scenarioName.toLowerCase() === normalizedFocus);
  }, [focusActive, normalizedFocus, requests]);

  return {
    ...rest,
    requests,
    filteredRequests,
    focusActive
  };
}

export function useFocusQueueStats(
  focusScenario: string,
  filteredRequests: SuiteRequest[],
  filteredExecutions: SuiteExecutionResult[]
): FocusQueueStats | null {
  const focusActive = focusScenario.trim().length > 0;

  return useMemo(() => {
    if (!focusActive) return null;

    const actionableCount = filteredRequests.filter((req) => isActionableRequest(req.status)).length;
    const mostRecentRequest = [...filteredRequests].sort(
      (a, b) =>
        parseTimestamp(b.updatedAt ?? b.createdAt) - parseTimestamp(a.updatedAt ?? a.createdAt)
    )[0];
    const recentExecution = filteredExecutions[0] ?? null;
    const failedExecution =
      filteredExecutions.find((execution) => execution.success === false) ?? null;
    const nextRequest = selectActionableRequest(filteredRequests);

    return {
      actionableCount,
      totalCount: filteredRequests.length,
      mostRecentRequest: mostRecentRequest ?? null,
      recentExecution,
      failedExecution,
      nextRequest
    };
  }, [filteredExecutions, filteredRequests, focusActive]);
}
