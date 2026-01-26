// DOC: docs/reference/api-endpoints.md#search
// DOC: docs/reference/api-endpoints.md#health-metrics
import { useMemo, useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  buildHealthViewModel,
  buildMetricsViewModel,
  buildSearchViewModel,
  loadHealth,
  loadKnowledgeMetrics,
  runSearchQuery,
} from "../controllers/knowledgeController";

const SAMPLE_QUERIES = [
  "recent knowledge health status",
  "semantic drift detection playbooks",
  "qdrant collections overview",
];

export function useHealthStatus() {
  const query = useQuery({
    queryKey: ["health"],
    queryFn: loadHealth,
    refetchInterval: 5000,
  });

  const hasError = Boolean(query.error);
  const hasData = Boolean(query.data);

  const viewModel = useMemo(
    () =>
      buildHealthViewModel({
        data: query.data,
        isLoading: query.isLoading,
        hasError,
      }),
    [query.data, query.isLoading, hasError]
  );

  return {
    viewModel,
    isLoading: query.isLoading,
    hasError,
    hasData,
    refetch: query.refetch,
  };
}

export function useKnowledgeMetrics() {
  const query = useQuery({
    queryKey: ["knowledgeHealth"],
    queryFn: loadKnowledgeMetrics,
    refetchInterval: 30000,
  });

  const viewModel = useMemo(() => buildMetricsViewModel(query.data), [query.data]);

  const hasError = Boolean(query.error);
  const hasData = Boolean(query.data);
  const errorMessage =
    query.error instanceof Error ? query.error.message : "Unable to load metrics.";

  return {
    viewModel,
    isLoading: query.isLoading,
    hasError,
    hasData,
    errorMessage,
    refetch: query.refetch,
  };
}

export function useSearchController() {
  const [query, setQuery] = useState("");
  const [searchTrigger, setSearchTrigger] = useState<string | null>(null);

  const searchQuery = useQuery({
    queryKey: ["search", searchTrigger],
    queryFn: ({ queryKey }) => runSearchQuery(queryKey[1]),
    enabled: Boolean(searchTrigger),
  });

  const runSearch = (nextQuery: string) => {
    const trimmed = nextQuery.trim();
    if (!trimmed) return;
    setQuery(trimmed);
    setSearchTrigger(trimmed);
  };

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    runSearch(query);
  };

  const clear = () => {
    setQuery("");
    setSearchTrigger(null);
  };

  const viewModel = useMemo(
    () =>
      buildSearchViewModel({
        data: searchQuery.data,
        fallbackQuery: query,
        error: searchQuery.error,
      }),
    [searchQuery.data, searchQuery.error, query]
  );

  const trimmedQuery = query.trim();
  const isSubmitDisabled = searchQuery.isLoading || !trimmedQuery;
  const isClearDisabled = !query && !searchQuery.data;

  return {
    query,
    setQuery,
    sampleQueries: SAMPLE_QUERIES,
    runSearch,
    handleSubmit,
    clear,
    isLoading: searchQuery.isLoading,
    hasError: Boolean(searchQuery.error),
    hasData: Boolean(searchQuery.data),
    isSubmitDisabled,
    isClearDisabled,
    viewModel,
  };
}
