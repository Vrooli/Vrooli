// DOC: docs/reference/api-endpoints.md#health
// DOC: docs/reference/api-endpoints.md#search
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import type { JsonObject } from "@bufbuild/protobuf";
import {
  graphResponseSchema,
  infrastructureHealthResponseSchema,
  knowledgeHealthResponseSchema,
  parseProtoResponse,
  searchResponseSchema,
} from "./knowledgeContracts";

// Use /api prefix without /v1 suffix since our health endpoint is at /health (not /api/v1/health)
// The /api prefix ensures requests go through the UI server proxy to the API server
const API_BASE = resolveApiBase({ appendSuffix: false });

type JsonRecord = Record<string, unknown>;

type QualityMetricsShape = {
  coherence?: number;
  freshness?: number;
  redundancy?: number;
  coverage?: number;
};

export interface ApiHealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

function isRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function toNonEmptyString(value?: string): string | undefined {
  return value && value.trim() ? value : undefined;
}

function toFiniteNumber(value: number | bigint | undefined): number | undefined {
  if (typeof value === "bigint") {
    const asNumber = Number(value);
    return Number.isFinite(asNumber) ? asNumber : undefined;
  }
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function normalizeJsonObject(value?: JsonObject): Record<string, unknown> {
  if (!value || typeof value !== "object") return {};
  return value as Record<string, unknown>;
}

function normalizeMetrics(value?: QualityMetricsShape): QualityMetrics | undefined {
  if (!value) return undefined;
  return {
    coherence: typeof value.coherence === "number" ? value.coherence : undefined,
    freshness: typeof value.freshness === "number" ? value.freshness : undefined,
    redundancy: typeof value.redundancy === "number" ? value.redundancy : undefined,
    coverage: typeof value.coverage === "number" ? value.coverage : undefined,
  };
}

export async function fetchHealth(): Promise<ApiHealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  const data = (await res.json().catch(() => null)) as unknown;
  const parsed = parseProtoResponse(infrastructureHealthResponseSchema, data, "health");

  return {
    status: toNonEmptyString(parsed.status) ?? "unknown",
    service: toNonEmptyString(parsed.service) ?? "unknown",
    timestamp: toNonEmptyString(parsed.timestamp) ?? new Date().toISOString(),
  };
}

export interface SearchRequest {
  query: string;
  collection?: string;
  namespaces?: string[];
  visibility?: string[];
  tags?: string[];
  ingested_after?: string;
  ingested_before?: string;
  limit?: number;
  threshold?: number;
}

export interface SearchResult {
  id: string;
  score: number;
  content: string;
  metadata: Record<string, unknown>;
}

export interface SearchResponse {
  results: SearchResult[];
  query: string;
  took_ms: number;
}

export interface GraphRequest {
  center_concept: string;
  collection?: string;
  namespaces?: string[];
  visibility?: string[];
  tags?: string[];
  depth?: number;
  limit?: number;
  threshold?: number;
}

export interface GraphNode {
  id: string;
  label: string;
  score?: number;
  metadata: Record<string, unknown>;
}

export interface GraphEdge {
  source: string;
  target: string;
  weight: number;
  relationship: string;
}

export interface GraphResponse {
  center: string;
  nodes: GraphNode[];
  edges: GraphEdge[];
  took_ms: number;
}

export async function searchKnowledge(request: SearchRequest): Promise<SearchResponse> {
  const url = buildApiUrl("/api/v1/knowledge/search", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    cache: "no-store",
  });

  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    const errorMessage =
      isRecord(errorPayload) && typeof errorPayload.error === "string"
        ? errorPayload.error
        : `Search failed: ${res.status}`;
    throw new Error(errorMessage);
  }

  const data = (await res.json().catch(() => null)) as unknown;
  const parsed = parseProtoResponse(searchResponseSchema, data, "search");

  const normalizedResults: SearchResult[] = parsed.results.map((result, index) => {
    const id = toNonEmptyString(result.id) ?? `result-${index + 1}`;
    const score = toFiniteNumber(result.score) ?? Number.NaN;
    const content = toNonEmptyString(result.content) ?? "";
    const metadata = normalizeJsonObject(result.metadata);
    return { id, score, content, metadata };
  });

  const responseQuery = toNonEmptyString(parsed.query);
  const normalizedQuery = responseQuery ?? request.query;
  const tookMs = toFiniteNumber(parsed.tookMs) ?? 0;

  return {
    results: normalizedResults,
    query: normalizedQuery,
    took_ms: tookMs,
  };
}

export async function fetchKnowledgeGraph(request: GraphRequest): Promise<GraphResponse> {
  const url = buildApiUrl("/api/v1/knowledge/graph", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    cache: "no-store",
  });

  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    const errorMessage =
      isRecord(errorPayload) && typeof errorPayload.error === "string"
        ? errorPayload.error
        : `Graph fetch failed: ${res.status}`;
    throw new Error(errorMessage);
  }

  const data = (await res.json().catch(() => null)) as unknown;
  const parsed = parseProtoResponse(graphResponseSchema, data, "graph");

  const nodes = (parsed.nodes ?? []).map((node, index) => {
    const fallbackID = `node-${index + 1}`;
    const id = toNonEmptyString(node.id) ?? fallbackID;
    const label = toNonEmptyString(node.label) ?? id;
    const score = typeof node.score === "number" ? node.score : undefined;
    const metadata = isRecord(node.metadata) ? node.metadata : {};
    return { id, label, score, metadata };
  });

  const edges = (parsed.edges ?? []).flatMap((edge) => {
    const source = toNonEmptyString(edge.source);
    const target = toNonEmptyString(edge.target);
    if (!source || !target) return [];
    return [
      {
        source,
        target,
        weight: toFiniteNumber(edge.weight) ?? 0,
        relationship: toNonEmptyString(edge.relationship) ?? "semantic_similarity",
      },
    ];
  });

  return {
    center: toNonEmptyString(parsed.center) ?? request.center_concept,
    nodes,
    edges,
    took_ms: toFiniteNumber(parsed.took_ms) ?? 0,
  };
}

export interface QualityMetrics {
  coherence?: number;
  freshness?: number;
  redundancy?: number;
  coverage?: number;
}

export interface CollectionHealth {
  name: string;
  size?: number;
  metrics?: QualityMetrics;
}

export interface HealthResponse {
  total_entries?: number;
  collections: CollectionHealth[];
  overall_health: string;
  overall_metrics?: QualityMetrics;
  timestamp: string;
}

export async function fetchKnowledgeHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/api/v1/knowledge/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`Knowledge health check failed: ${res.status}`);
  }

  const data = (await res.json().catch(() => null)) as unknown;
  const parsed = parseProtoResponse(knowledgeHealthResponseSchema, data, "knowledge health");

  const collections = parsed.collections.map((collection, index) => {
    const name = toNonEmptyString(collection.name) ?? `Collection ${index + 1}`;
    const size = typeof collection.size === "number" ? collection.size : undefined;
    return {
      name,
      size,
      metrics: normalizeMetrics(collection.metrics),
    };
  });

  return {
    total_entries: typeof parsed.totalEntries === "number" ? parsed.totalEntries : undefined,
    collections,
    overall_health: toNonEmptyString(parsed.overallHealth) ?? "unknown",
    overall_metrics: normalizeMetrics(parsed.overallMetrics),
    timestamp: toNonEmptyString(parsed.timestamp) ?? new Date().toISOString(),
  };
}

export interface CollectionDiagnostics {
  collection: string;
  mode: string;
  total_points?: number;
  analyzed_points: number;
  vector_dimensions: Array<{ dimension: number; count: number }>;
  namespaces: Array<{ name: string; count: number }>;
  chunk_length: {
    min_characters: number;
    max_characters: number;
    avg_characters: number;
  };
  missing_payload_fields: Record<string, number>;
  redundancy: {
    duplicate_content_hashes: number;
    duplicate_point_count: number;
    duplicate_ratio: number;
  };
  stale_chunks: {
    groups_detected: number;
    candidate_delete_rows: number;
    top_documents: Array<{ name: string; count: number }>;
  };
  ingest_history?: {
    total_attempts: number;
    success_count: number;
    failure_count: number;
    failure_count_last_24h: number;
    failure_rate: number;
    last_failure_at?: string;
  };
  recommendations: string[];
  timestamp: string;
}

export interface CollectionMaintenanceResponse {
  collection: string;
  action: string;
  dry_run: boolean;
  analyzed_points: number;
  candidate_delete_count: number;
  deleted_count: number;
  took_ms: number;
}

export interface DocumentDeleteResponse {
  collection: string;
  namespace: string;
  document_id: string;
  external_id?: string;
  dry_run: boolean;
  candidate_delete_count: number;
  deleted_count: number;
  took_ms: number;
}

export interface DocumentDeleteRequest {
  namespace: string;
  collection?: string;
  document_id?: string;
  external_id?: string;
  dry_run?: boolean;
}

export interface IngestHealthResponse {
  runner_interval_ms: number;
  pending_jobs: number;
  running_jobs: number;
  failed_jobs: number;
  successful_jobs: number;
  failures_last_24h: number;
  oldest_pending_age_ms?: number;
  status: string;
  timestamp: string;
}

export interface CollectionInventoryItem {
  name: string;
  total_points?: number;
  ownership: "knowledge_observatory" | "mixed" | "external_or_unknown" | string;
  ownership_label: string;
  ingest_attempts: number;
  metadata_rows: number;
  distinct_namespaces: number;
  last_ingest_at?: string;
}

export interface CollectionInventoryResponse {
  collections: CollectionInventoryItem[];
  timestamp: string;
}

export interface CollectionRecordPreview {
  id: string;
  namespace?: string;
  document_id?: string;
  chunk_index?: number;
  external_id?: string;
  visibility?: string;
  content_hash?: string;
  ingested_at?: string;
  source?: string;
  source_type?: string;
  tags?: string[];
  content_preview?: string;
  metadata?: Record<string, unknown>;
}

export interface CollectionRecordsResponse {
  collection: string;
  total_count: number;
  offset: number;
  limit: number;
  next_offset?: number;
  records: CollectionRecordPreview[];
}

export interface CollectionDeleteResponse {
  collection: string;
  deleted: boolean;
  metadata_rows_deleted: number;
  ingest_history_rows_deleted: number;
  warning?: string;
  timestamp: string;
}

type MaintenanceAction = "prune-stale-chunks" | "dedupe-content";

export async function fetchCollectionDiagnostics(
  collection: string,
  mode: "sample" | "full" = "sample",
  limit = 500
): Promise<CollectionDiagnostics> {
  const params = new URLSearchParams({
    mode,
    limit: String(limit),
  });
  const url = buildApiUrl(`/api/v1/knowledge/collections/${encodeURIComponent(collection)}/diagnostics?${params.toString()}`, {
    baseUrl: API_BASE,
  });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`Collection diagnostics failed: ${res.status}`);
  }
  return (await res.json()) as CollectionDiagnostics;
}

export async function runCollectionMaintenance(
  collection: string,
  action: MaintenanceAction,
  request: { dry_run?: boolean; max_deletes?: number } = {}
): Promise<CollectionMaintenanceResponse> {
  const url = buildApiUrl(`/api/v1/knowledge/collections/${encodeURIComponent(collection)}/maintenance/${action}`, {
    baseUrl: API_BASE,
  });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    cache: "no-store",
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    const errorMessage =
      isRecord(errorPayload) && typeof errorPayload.error === "string"
        ? errorPayload.error
        : `Collection maintenance failed: ${res.status}`;
    throw new Error(errorMessage);
  }
  return (await res.json()) as CollectionMaintenanceResponse;
}

export async function runDocumentDelete(request: DocumentDeleteRequest): Promise<DocumentDeleteResponse> {
  const url = buildApiUrl("/api/v1/knowledge/documents/delete", {
    baseUrl: API_BASE,
  });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    cache: "no-store",
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    const errorMessage =
      isRecord(errorPayload) && typeof errorPayload.error === "string"
        ? errorPayload.error
        : `Document delete failed: ${res.status}`;
    throw new Error(errorMessage);
  }
  return (await res.json()) as DocumentDeleteResponse;
}

export async function fetchIngestHealth(): Promise<IngestHealthResponse> {
  const url = buildApiUrl("/api/v1/ingest/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`Ingest health fetch failed: ${res.status}`);
  }
  return (await res.json()) as IngestHealthResponse;
}

export async function fetchCollectionInventory(): Promise<CollectionInventoryResponse> {
  const url = buildApiUrl("/api/v1/knowledge/collections", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`Collection inventory fetch failed: ${res.status}`);
  }
  return (await res.json()) as CollectionInventoryResponse;
}

export async function fetchCollectionRecords(
  collection: string,
  request: {
    limit?: number;
    offset?: number;
    namespace?: string;
    document_id?: string;
    search?: string;
  } = {}
): Promise<CollectionRecordsResponse> {
  const params = new URLSearchParams();
  if (request.limit && request.limit > 0) params.set("limit", String(request.limit));
  if (request.offset && request.offset >= 0) params.set("offset", String(request.offset));
  if (request.namespace?.trim()) params.set("namespace", request.namespace.trim());
  if (request.document_id?.trim()) params.set("document_id", request.document_id.trim());
  if (request.search?.trim()) params.set("search", request.search.trim());
  const suffix = params.toString();
  const url = buildApiUrl(
    `/api/v1/knowledge/collections/${encodeURIComponent(collection)}/records${suffix ? `?${suffix}` : ""}`,
    { baseUrl: API_BASE }
  );
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    const errorMessage =
      isRecord(errorPayload) && typeof errorPayload.error === "string"
        ? errorPayload.error
        : `Collection records fetch failed: ${res.status}`;
    throw new Error(errorMessage);
  }
  return (await res.json()) as CollectionRecordsResponse;
}

export async function runCollectionDelete(collection: string): Promise<CollectionDeleteResponse> {
  const url = buildApiUrl(`/api/v1/knowledge/collections/${encodeURIComponent(collection)}`, {
    baseUrl: API_BASE,
  });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    const errorPayload = (await res.json().catch(() => null)) as unknown;
    const errorMessage =
      isRecord(errorPayload) && typeof errorPayload.error === "string"
        ? errorPayload.error
        : `Collection delete failed: ${res.status}`;
    throw new Error(errorMessage);
  }
  return (await res.json()) as CollectionDeleteResponse;
}
