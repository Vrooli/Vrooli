import { useQuery } from "@tanstack/react-query";
import { fetchScenarioIsolation, type IsolationResponse, type IsolationStatus } from "../lib/api-isolation";

export type ScenarioIsolationStatus = IsolationStatus | "loading";

export interface ScenarioIsolation {
  status: ScenarioIsolationStatus;
  reasons: string[];
  violations: NonNullable<IsolationResponse["violations"]>;
  refetch: () => void;
}

// useScenarioIsolation returns the routed-test-db eligibility for a scenario.
// The badge UI keys colour and copy off `status`; `loading` is surfaced as a
// distinct visual state so the panel can render a skeleton instead of a
// placeholder badge.
export function useScenarioIsolation(scenarioSlug: string | undefined): ScenarioIsolation {
  const query = useQuery<IsolationResponse, Error>({
    queryKey: ["scenario", "isolation", scenarioSlug ?? "(none)"],
    queryFn: () => fetchScenarioIsolation(scenarioSlug as string),
    enabled: !!scenarioSlug,
    staleTime: 25_000,
    refetchOnWindowFocus: false,
  });

  if (!scenarioSlug || query.isLoading) {
    return { status: "loading", reasons: [], violations: [], refetch: () => query.refetch() };
  }

  const data = query.data;
  if (!data) {
    return { status: "unknown", reasons: ["No response from GCT API"], violations: [], refetch: () => query.refetch() };
  }

  return {
    status: data.status,
    reasons: data.reasons ?? [],
    violations: data.violations ?? [],
    refetch: () => query.refetch(),
  };
}
