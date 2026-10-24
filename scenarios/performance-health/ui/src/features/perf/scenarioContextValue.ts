import { createContext, useContext } from "react";

/**
 * Context object + hook for the "current scenario" selection, kept in a
 * component-free module so `ScenarioContext.tsx` (which exports the provider
 * *component*) doesn't trip the react-refresh "only export components" rule.
 */
export interface ScenarioContextValue {
  /** The scenario the per-scenario screens act on. */
  scenario: string;
  setScenario: (scenario: string) => void;
  /** Discovered scenario ids (from ScanFleet), sorted; empty until loaded. */
  scenarios: string[];
  isLoadingScenarios: boolean;
}

export const ScenarioContext = createContext<ScenarioContextValue | null>(null);

export function useScenario(): ScenarioContextValue {
  const ctx = useContext(ScenarioContext);
  if (!ctx) {
    throw new Error("useScenario must be called inside <ScenarioProvider>");
  }
  return ctx;
}
