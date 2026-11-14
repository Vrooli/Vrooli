import { useCallback, useMemo, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";

import { perfClient } from "../../api/perf";
import { ScenarioContext, type ScenarioContextValue } from "./scenarioContextValue";

const STORAGE_KEY = "performance-health.selected-scenario";
const DEFAULT_SCENARIO = "performance-health";

const readStored = (): string => {
  if (typeof window === "undefined") return DEFAULT_SCENARIO;
  return window.localStorage.getItem(STORAGE_KEY) ?? DEFAULT_SCENARIO;
};

/**
 * Owns the "current scenario" the per-scenario workflows (audit, trends,
 * readiness, budgets, trace) act on. The picker list comes from the same
 * `ScanFleet` RPC the fleet dashboard uses — the fleet scan is the authoritative
 * enumerator of discoverable scenarios — so the dropdown is always real data.
 */
export function ScenarioProvider({ children }: { children: ReactNode }) {
  const [scenario, setScenarioState] = useState<string>(() => readStored());

  const fleetQuery = useQuery({
    queryKey: ["fleet-scan", "scenario-picker"],
    queryFn: () => perfClient.scanFleet({}),
    staleTime: 60_000,
  });

  const scenarios = useMemo(() => {
    const fromFleet = (fleetQuery.data?.entries ?? []).map((e) => e.scenario);
    // Always include the currently-selected scenario so a stored value that the
    // scan didn't surface (e.g. a not-yet-discovered scenario) stays selectable.
    const set = new Set<string>([...fromFleet, scenario]);
    return Array.from(set).sort((a, b) => a.localeCompare(b));
  }, [fleetQuery.data, scenario]);

  const setScenario = useCallback((next: string) => {
    setScenarioState(next);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, next);
    }
  }, []);

  const value = useMemo<ScenarioContextValue>(
    () => ({
      scenario,
      setScenario,
      scenarios,
      isLoadingScenarios: fleetQuery.isLoading,
    }),
    [scenario, setScenario, scenarios, fleetQuery.isLoading],
  );

  return <ScenarioContext.Provider value={value}>{children}</ScenarioContext.Provider>;
}
