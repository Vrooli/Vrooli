import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import type { JsonObject } from "@bufbuild/protobuf";
import {
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
