import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchScenarioSummaries, type ScenarioSummary } from "../lib/api";
import { timestampOrZero } from "../lib/formatters";

export interface ScenarioDirectoryEntry extends ScenarioSummary {
  lastActivity: number;
}

export function useScenarios() {
  const query = useQuery<ScenarioSummary[]>({
    queryKey: ["scenario-summaries"],
    queryFn: fetchScenarioSummaries,
    refetchInterval: 15000
  });

  const scenarioDirectoryEntries = useMemo<ScenarioDirectoryEntry[]>(() => {
    const summaries = query.data ?? [];
    if (summaries.length === 0) return [];

    return summaries
      .map((summary) => {
        const lastActivity = Math.max(
          timestampOrZero(summary.lastExecutionAt),
          timestampOrZero(summary.lastRequestAt)
        );
        return { ...summary, lastActivity };
      })
      .sort((a, b) => b.lastActivity - a.lastActivity);
  }, [query.data]);

  const catalogStats = useMemo(() => {
    if (scenarioDirectoryEntries.length === 0) {
      return { tracked: 0, pending: 0, failing: 0, idle: 0 };
    }
    const tracked = scenarioDirectoryEntries.length;
    const pending = scenarioDirectoryEntries.filter((entry) => entry.pendingRequests > 0).length;
    const failing = scenarioDirectoryEntries.filter((entry) => entry.lastExecutionSuccess === false).length;
    const idle = scenarioDirectoryEntries.filter(
      (entry) => entry.pendingRequests === 0 && !entry.lastExecutionAt
    ).length;
    return { tracked, pending, failing, idle };
  }, [scenarioDirectoryEntries]);

  return {
    ...query,
    scenarioDirectoryEntries,
    catalogStats
  };
}

export function useFilteredScenarios(scenarioSearch: string) {
  const { scenarioDirectoryEntries, ...rest } = useScenarios();

  const filteredScenarioDirectory = useMemo(() => {
    const trimmed = scenarioSearch.trim().toLowerCase();
    if (!trimmed) return scenarioDirectoryEntries;

    return scenarioDirectoryEntries.filter((entry) =>
      entry.scenarioName.toLowerCase().includes(trimmed)
    );
  }, [scenarioDirectoryEntries, scenarioSearch]);

  return {
    ...rest,
    scenarioDirectoryEntries,
    filteredScenarioDirectory
  };
}
