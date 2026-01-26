// DOC: docs/reference/api-endpoints.md#scenario-list
// DOC: docs/reference/api-endpoints.md#scenario-documentation-tree
// DOC: docs/reference/api-endpoints.md#documentation-health
import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { buildDocHealthViewModel, buildScenarioSummaryViews } from "../controllers/documentationController";
import {
  fetchScenarioDocHealth,
  fetchScenarioDocTree,
  fetchScenarioSummaries,
  type DocTreeNode,
} from "../services/documentationApi";

export function useScenarioExplorer() {
  const [filter, setFilter] = useState("");
  const [selectedScenario, setSelectedScenario] = useState<string | null>(null);
  const [selectedDocPath, setSelectedDocPath] = useState<string | null>(null);

  const scenariosQuery = useQuery({
    queryKey: ["docScenarios"],
    queryFn: fetchScenarioSummaries,
  });

  useEffect(() => {
    const scenarios = scenariosQuery.data ?? [];
    if (scenarios.length === 0) return;
    if (!selectedScenario || !scenarios.some((scenario) => scenario.name === selectedScenario)) {
      setSelectedScenario(scenarios[0]?.name ?? null);
    }
  }, [scenariosQuery.data, selectedScenario]);

  useEffect(() => {
    setSelectedDocPath(null);
  }, [selectedScenario]);

  const scenarioViews = useMemo(
    () => buildScenarioSummaryViews(scenariosQuery.data, filter),
    [scenariosQuery.data, filter]
  );

  const activeScenario = selectedScenario ?? "";

  const docTreeQuery = useQuery({
    queryKey: ["docTree", activeScenario],
    queryFn: () => fetchScenarioDocTree(activeScenario),
    enabled: Boolean(activeScenario),
  });

  const docHealthQuery = useQuery({
    queryKey: ["docHealth", activeScenario],
    queryFn: () => fetchScenarioDocHealth(activeScenario),
    enabled: Boolean(activeScenario),
  });

  const healthViewModel = useMemo(
    () => buildDocHealthViewModel(docHealthQuery.data),
    [docHealthQuery.data]
  );

  return {
    filter,
    setFilter,
    scenarios: scenarioViews,
    selectedScenario,
    setSelectedScenario,
    selectedDocPath,
    setSelectedDocPath,
    docTree: docTreeQuery.data as DocTreeNode | undefined,
    healthViewModel,
    scenariosState: {
      isLoading: scenariosQuery.isLoading,
      hasError: Boolean(scenariosQuery.error),
      errorMessage:
        scenariosQuery.error instanceof Error
          ? scenariosQuery.error.message
          : "Unable to load scenarios.",
      refetch: scenariosQuery.refetch,
    },
    docTreeState: {
      isLoading: docTreeQuery.isLoading,
      hasError: Boolean(docTreeQuery.error),
      errorMessage:
        docTreeQuery.error instanceof Error ? docTreeQuery.error.message : "Unable to load doc tree.",
      refetch: docTreeQuery.refetch,
    },
    docHealthState: {
      isLoading: docHealthQuery.isLoading,
      hasError: Boolean(docHealthQuery.error),
      errorMessage:
        docHealthQuery.error instanceof Error ? docHealthQuery.error.message : "Unable to load doc health.",
      refetch: docHealthQuery.refetch,
    },
  };
}
