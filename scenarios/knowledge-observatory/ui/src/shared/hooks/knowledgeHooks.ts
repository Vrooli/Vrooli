// DOC: docs/reference/api-endpoints.md#search
// DOC: docs/reference/api-endpoints.md#health-metrics
import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  buildGraphViewModel,
  buildHealthViewModel,
  buildMetricsViewModel,
  buildSearchViewModel,
  loadHealth,
  loadKnowledgeMetrics,
  runGraphQuery,
  runSearchQuery,
} from "../controllers/knowledgeController";
import { recordActivity } from "../lib/activityStore";

const SAMPLE_QUERIES = [
  "recent knowledge health status",
  "semantic drift detection playbooks",
  "qdrant collections overview",
];

const DEFAULT_GRAPH_DEPTH = "1";
const DEFAULT_GRAPH_LIMIT = "50";
const DEFAULT_GRAPH_THRESHOLD = "0.35";
const DEFAULT_GRAPH_VISIBILITY = "shared,global";

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
  const lastRecordedRef = useRef<string | null>(null);

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

  useEffect(() => {
    if (!searchQuery.data || !searchTrigger) return;
    if (lastRecordedRef.current === searchTrigger) return;
    lastRecordedRef.current = searchTrigger;
    recordActivity({
      type: "semantic-search",
      title: "Semantic search",
      description: searchTrigger,
      status: "completed",
      meta: {
        results: `${searchQuery.data.results.length}`,
      },
    });
  }, [searchQuery.data, searchTrigger]);

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

type GraphQueryInput = {
  center_concept: string;
  collection?: string;
  namespaces?: string[];
  visibility?: string[];
  tags?: string[];
  depth?: number;
  limit?: number;
  threshold?: number;
};

const parseCSV = (value: string) =>
  value
    .split(",")
    .map((entry) => entry.trim())
    .filter(Boolean);

const normalizePositiveInteger = (value: string, fallback: number) => {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isFinite(parsed) || parsed <= 0) return fallback;
  return parsed;
};

const normalizePositiveFloat = (value: string, fallback: number) => {
  const parsed = Number.parseFloat(value);
  if (!Number.isFinite(parsed) || parsed <= 0) return fallback;
  return parsed;
};

export function useKnowledgeGraphController() {
  const [centerConcept, setCenterConcept] = useState("");
  const [collection, setCollection] = useState("");
  const [namespacesInput, setNamespacesInput] = useState("");
  const [tagsInput, setTagsInput] = useState("");
  const [visibilityInput, setVisibilityInput] = useState(DEFAULT_GRAPH_VISIBILITY);
  const [depthInput, setDepthInput] = useState(DEFAULT_GRAPH_DEPTH);
  const [limitInput, setLimitInput] = useState(DEFAULT_GRAPH_LIMIT);
  const [thresholdInput, setThresholdInput] = useState(DEFAULT_GRAPH_THRESHOLD);
  const [graphTrigger, setGraphTrigger] = useState<GraphQueryInput | null>(null);
  const lastRecordedRef = useRef<string | null>(null);

  const graphQuery = useQuery({
    queryKey: ["graph", graphTrigger],
    queryFn: ({ queryKey }) => runGraphQuery(queryKey[1]),
    enabled: Boolean(graphTrigger),
  });

  const buildRequest = (center: string): GraphQueryInput => ({
    center_concept: center,
    collection: collection.trim() || undefined,
    namespaces: parseCSV(namespacesInput),
    tags: parseCSV(tagsInput),
    visibility: parseCSV(visibilityInput),
    depth: normalizePositiveInteger(depthInput, 1),
    limit: normalizePositiveInteger(limitInput, 50),
    threshold: normalizePositiveFloat(thresholdInput, 0.35),
  });

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const center = centerConcept.trim();
    if (!center) return;

    setGraphTrigger(buildRequest(center));
  };

  const clear = () => {
    setCenterConcept("");
    setCollection("");
    setNamespacesInput("");
    setTagsInput("");
    setVisibilityInput(DEFAULT_GRAPH_VISIBILITY);
    setDepthInput(DEFAULT_GRAPH_DEPTH);
    setLimitInput(DEFAULT_GRAPH_LIMIT);
    setThresholdInput(DEFAULT_GRAPH_THRESHOLD);
    setGraphTrigger(null);
  };

  const refetch = () => {
    if (!graphTrigger) return;
    graphQuery.refetch();
  };

  const queryGraph = (center: string) => runGraphQuery(buildRequest(center));

  const viewModel = useMemo(
    () =>
      buildGraphViewModel({
        data: graphQuery.data,
        fallbackCenter: centerConcept,
        error: graphQuery.error,
      }),
    [graphQuery.data, graphQuery.error, centerConcept]
  );

  useEffect(() => {
    if (!graphQuery.data || !graphTrigger) return;
    const signature = JSON.stringify([graphTrigger.center_concept, graphTrigger.depth, graphTrigger.limit]);
    if (lastRecordedRef.current === signature) return;
    lastRecordedRef.current = signature;
    recordActivity({
      type: "knowledge-graph",
      title: "Knowledge graph generated",
      description: graphTrigger.center_concept,
      status: "completed",
      meta: {
        nodes: `${graphQuery.data.nodes.length}`,
        edges: `${graphQuery.data.edges.length}`,
      },
    });
  }, [graphQuery.data, graphTrigger]);

  const isSubmitDisabled = graphQuery.isLoading || !centerConcept.trim();
  const isClearDisabled = !centerConcept && !graphQuery.data && !graphTrigger;

  return {
    centerConcept,
    setCenterConcept,
    collection,
    setCollection,
    namespacesInput,
    setNamespacesInput,
    tagsInput,
    setTagsInput,
    visibilityInput,
    setVisibilityInput,
    depthInput,
    setDepthInput,
    limitInput,
    setLimitInput,
    thresholdInput,
    setThresholdInput,
    submit,
    clear,
    refetch,
    queryGraph,
    isLoading: graphQuery.isLoading,
    hasError: Boolean(graphQuery.error),
    hasData: Boolean(graphQuery.data),
    isSubmitDisabled,
    isClearDisabled,
    graphData: graphQuery.data,
    graphRequest: graphTrigger,
    viewModel,
  };
}
