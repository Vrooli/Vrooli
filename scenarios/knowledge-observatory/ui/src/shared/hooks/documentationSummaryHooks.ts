// DOC: docs/reference/api-endpoints.md#scenario-list
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { buildDocumentationSummaryView } from "../controllers/documentationController";
import { fetchScenarioSummaries } from "../services/documentationApi";

export function useDocumentationSummary() {
  const query = useQuery({
    queryKey: ["docSummary"],
    queryFn: fetchScenarioSummaries,
    refetchInterval: 60000,
  });

  const viewModel = useMemo(
    () => buildDocumentationSummaryView(query.data),
    [query.data]
  );

  return {
    viewModel,
    isLoading: query.isLoading,
    hasError: Boolean(query.error),
    errorMessage:
      query.error instanceof Error ? query.error.message : "Unable to load documentation summary.",
    refetch: query.refetch,
  };
}
