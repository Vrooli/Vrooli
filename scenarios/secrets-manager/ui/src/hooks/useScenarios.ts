import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchScenarios, type ScenarioSummary } from "../lib/api";

export const useScenarios = () => {
  const [search, setSearch] = useState("");

  const query = useQuery({
    queryKey: ["scenarios"],
    queryFn: fetchScenarios,
    staleTime: 5 * 60 * 1000
  });

  const scenarios: ScenarioSummary[] = query.data?.scenarios ?? [];

  const filtered = useMemo(() => {
    if (!search.trim()) return scenarios;
    const needle = search.toLowerCase();
    return scenarios.filter((scenario) => scenario.name.toLowerCase().includes(needle));
  }, [search, scenarios]);

  return {
    search,
    setSearch,
    query,
    scenarios,
    filtered
  };
};
