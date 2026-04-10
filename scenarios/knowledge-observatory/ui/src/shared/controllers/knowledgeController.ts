// DOC: docs/reference/api-endpoints.md#search
// DOC: docs/reference/api-endpoints.md#health-metrics
import {
  fetchKnowledgeGraph,
  fetchHealth,
  fetchKnowledgeHealth,
  searchKnowledge,
  type ApiHealthResponse,
  type CollectionHealth,
  type GraphRequest,
  type GraphResponse,
  type SearchResponse,
  type SearchResult,
  type HealthResponse,
} from "../services/api";

export type HealthViewModel = {
  status: string;
  service: string;
  lastUpdated: string;
  statusLabel: "Syncing..." | "Offline" | "Online";
  statusPulse: boolean;
};

export type SearchResultView = {
  id: string;
  content: string;
  metadata: Record<string, unknown>;
  scoreLabel: string;
  hasMetadata: boolean;
};

export type SearchViewModel = {
  results: SearchResultView[];
  totalResults: number;
  tookMsLabel: string;
  displayQuery: string;
  errorMessage: string;
  hasResults: boolean;
};

export type MetricTone = "good" | "medium" | "poor";

export type MetricCardView = {
  label: string;
  description: string;
  percentageLabel: string;
  tone: MetricTone;
};

export type CollectionMetricView = {
  label: string;
  percentageLabel: string;
};

export type CollectionView = {
  name: string;
  sizeLabel: string;
  metrics: CollectionMetricView[];
};

export type MetricsViewModel = {
  metricCards: MetricCardView[];
  collections: CollectionView[];
  overallHealth: string;
  lastUpdated: string;
  totalEntriesLabel: string;
  hasMetrics: boolean;
};

export type GraphNodeView = {
  id: string;
  label: string;
  scoreLabel: string;
  hasMetadata: boolean;
  metadata: Record<string, unknown>;
};

export type GraphEdgeView = {
  source: string;
  target: string;
  relationship: string;
  weightLabel: string;
};

export type GraphViewModel = {
  center: string;
  nodeCount: number;
  edgeCount: number;
  tookMsLabel: string;
  hasGraph: boolean;
  nodes: GraphNodeView[];
  edges: GraphEdgeView[];
  errorMessage: string;
};

const EMPTY_RESULTS: SearchResultView[] = [];
const EMPTY_COLLECTIONS: CollectionView[] = [];
const EMPTY_METRIC_CARDS: MetricCardView[] = [];
const EMPTY_GRAPH_NODES: GraphNodeView[] = [];
const EMPTY_GRAPH_EDGES: GraphEdgeView[] = [];
const DEFAULT_SEARCH_LIMIT = 10;
const DEFAULT_GRAPH_LIMIT = 50;
const DEFAULT_GRAPH_DEPTH = 1;
const DEFAULT_GRAPH_THRESHOLD = 0.35;

export const loadHealth = () => fetchHealth();

export const loadKnowledgeMetrics = () => fetchKnowledgeHealth();
export const loadKnowledgeGraph = (request: GraphRequest) => fetchKnowledgeGraph(request);

export const runSearchQuery = async (queryValue: unknown) => {
  if (typeof queryValue !== "string" || !queryValue.trim()) {
    throw new Error("Search query is missing.");
  }
  return searchKnowledge({ query: queryValue, limit: DEFAULT_SEARCH_LIMIT });
};

export const runGraphQuery = async (request: unknown) => {
  if (!request || typeof request !== "object" || Array.isArray(request)) {
    throw new Error("Graph request is missing.");
  }

  const shape = request as Record<string, unknown>;
  const center = safeString(shape.center_concept);
  if (!center) {
    throw new Error("Center concept is required.");
  }

  const normalizeStringArray = (value: unknown) =>
    Array.isArray(value)
      ? value
          .map((entry) => safeString(entry))
          .filter((entry): entry is string => Boolean(entry))
      : undefined;

  const depth = safeNumber(shape.depth);
  const limit = safeNumber(shape.limit);
  const threshold = safeNumber(shape.threshold);

  return fetchKnowledgeGraph({
    center_concept: center,
    collection: safeString(shape.collection),
    namespaces: normalizeStringArray(shape.namespaces),
    visibility: normalizeStringArray(shape.visibility),
    tags: normalizeStringArray(shape.tags),
    depth: depth && depth > 0 ? Math.round(depth) : DEFAULT_GRAPH_DEPTH,
    limit: limit && limit > 0 ? Math.round(limit) : DEFAULT_GRAPH_LIMIT,
    threshold: threshold && threshold > 0 ? threshold : DEFAULT_GRAPH_THRESHOLD,
  });
};

export function formatTimestamp(value?: string) {
  if (!value) return "Unknown";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Unknown" : date.toLocaleTimeString();
}

const formatPercent = (value: number, digits: number) => `${(value * 100).toFixed(digits)}%`;

const safeString = (value: unknown) =>
  typeof value === "string" && value.trim() ? value : undefined;

const safeNumber = (value: unknown) =>
  typeof value === "number" && Number.isFinite(value) ? value : undefined;

const safeRecord = (value: unknown): Record<string, unknown> =>
  value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : {};

const METRIC_DEFINITIONS = [
  {
    key: "coherence",
    label: "Coherence",
    description: "Topical consistency across knowledge",
    invert: false,
    digits: 1,
  },
  {
    key: "freshness",
    label: "Freshness",
    description: "Recency of knowledge entries",
    invert: false,
    digits: 1,
  },
  {
    key: "coverage",
    label: "Coverage",
    description: "Domain topic distribution",
    invert: false,
    digits: 1,
  },
  {
    key: "redundancy",
    label: "Redundancy",
    description: "Duplicate detection (lower is better)",
    invert: true,
    digits: 1,
  },
] as const;

type MetricDefinition = (typeof METRIC_DEFINITIONS)[number];

const getMetricTone = (value: number, invert: boolean): MetricTone => {
  if (invert) {
    if (value < 0.2) return "good";
    if (value < 0.4) return "medium";
    return "poor";
  }
  if (value >= 0.6) return "good";
  if (value >= 0.4) return "medium";
  return "poor";
};

export function buildHealthViewModel(params: {
  data?: ApiHealthResponse | null;
  isLoading: boolean;
  hasError: boolean;
}): HealthViewModel {
  const { data, isLoading, hasError } = params;
  return {
    status: safeString(data?.status) ?? "Unknown",
    service: safeString(data?.service) ?? "Unknown",
    lastUpdated: formatTimestamp(safeString(data?.timestamp)),
    statusLabel: isLoading ? "Syncing..." : hasError ? "Offline" : "Online",
    statusPulse: isLoading,
  };
}

export function buildSearchViewModel(params: {
  data?: SearchResponse | null;
  fallbackQuery: string;
  error: unknown;
}): SearchViewModel {
  const { data, fallbackQuery, error } = params;
  const rawResults = Array.isArray(data?.results) ? data?.results : [];
  const displayQuery =
    safeString(data?.query) ?? safeString(fallbackQuery) ?? "your query";

  const results = rawResults.length
    ? rawResults.map((result: SearchResult, index) => {
        const id = safeString(result.id) ?? `result-${index + 1}`;
        const content = safeString(result.content) ?? "No content available";
        const metadata = safeRecord(result.metadata);
        const scoreValue = safeNumber(result.score);
        return {
          id,
          content,
          metadata,
          scoreLabel: scoreValue === undefined ? "N/A" : formatPercent(scoreValue, 1),
          hasMetadata: Object.keys(metadata).length > 0,
        };
      })
    : EMPTY_RESULTS;

  return {
    results,
    totalResults: results.length,
    tookMsLabel: (() => {
      const tookMs = safeNumber(data?.took_ms);
      return tookMs === undefined ? "?ms" : `${tookMs}ms`;
    })(),
    displayQuery,
    errorMessage: error instanceof Error ? error.message : "Search failed. Please try again.",
    hasResults: results.length > 0,
  };
}

export function buildMetricsViewModel(data?: HealthResponse | null): MetricsViewModel {
  const metrics =
    data?.overall_metrics && typeof data.overall_metrics === "object" ? data.overall_metrics : null;
  const metricCards = metrics
    ? METRIC_DEFINITIONS.flatMap((definition: MetricDefinition) => {
        const value = safeNumber(metrics[definition.key]);
        if (value === undefined) return [];
        return [
          {
            label: definition.label,
            description: definition.description,
            percentageLabel: formatPercent(value, definition.digits),
            tone: getMetricTone(value, definition.invert),
          },
        ];
      })
    : EMPTY_METRIC_CARDS;

  const collections = Array.isArray(data?.collections)
    ? data.collections.map((collection: CollectionHealth, index) => {
        const name = safeString(collection.name) ?? `Collection ${index + 1}`;
        const sizeLabel =
          typeof collection.size === "number" ? `${collection.size} vectors` : "Vectors: unknown";
        const metricsList = collection.metrics
          ? METRIC_DEFINITIONS.flatMap((definition: MetricDefinition) => {
              const value = safeNumber(collection.metrics?.[definition.key]);
              if (value === undefined) return [];
              return [
                {
                  label: definition.label,
                  percentageLabel: formatPercent(value, 0),
                },
              ];
            })
          : [];
        return { name, sizeLabel, metrics: metricsList };
      })
    : EMPTY_COLLECTIONS;
  const overallHealth = safeString(data?.overall_health) ?? "unknown";
  const lastUpdated = formatTimestamp(safeString(data?.timestamp));
  const totalEntriesLabel =
    typeof data?.total_entries === "number" ? data.total_entries.toLocaleString() : "Unknown";

  return {
    collections,
    metricCards,
    overallHealth,
    lastUpdated,
    totalEntriesLabel,
    hasMetrics: metricCards.length > 0,
  };
}

export function buildGraphViewModel(params: {
  data?: GraphResponse | null;
  fallbackCenter: string;
  error: unknown;
}): GraphViewModel {
  const { data, fallbackCenter, error } = params;
  const center = safeString(data?.center) ?? safeString(fallbackCenter) ?? "center concept";

  const nodes = Array.isArray(data?.nodes)
    ? data.nodes.map((node, index) => {
        const id = safeString(node.id) ?? `node-${index + 1}`;
        const label = safeString(node.label) ?? id;
        const metadata = safeRecord(node.metadata);
        const score = safeNumber(node.score);
        return {
          id,
          label,
          metadata,
          hasMetadata: Object.keys(metadata).length > 0,
          scoreLabel: score === undefined ? "N/A" : score.toFixed(3),
        };
      })
    : EMPTY_GRAPH_NODES;

  const edges = Array.isArray(data?.edges)
    ? data.edges.flatMap((edge) => {
        const source = safeString(edge.source);
        const target = safeString(edge.target);
        if (!source || !target) return [];
        const weight = safeNumber(edge.weight);
        return [
          {
            source,
            target,
            relationship: safeString(edge.relationship) ?? "semantic_similarity",
            weightLabel: weight === undefined ? "N/A" : weight.toFixed(3),
          },
        ];
      })
    : EMPTY_GRAPH_EDGES;

  return {
    center,
    nodeCount: nodes.length,
    edgeCount: edges.length,
    tookMsLabel: (() => {
      const tookMs = safeNumber(data?.took_ms);
      return tookMs === undefined ? "?ms" : `${Math.round(tookMs)}ms`;
    })(),
    hasGraph: nodes.length > 0,
    nodes,
    edges,
    errorMessage: error instanceof Error ? error.message : "Unable to generate graph.",
  };
}
