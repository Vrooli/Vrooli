import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Use /api prefix without /v1 suffix since our health endpoint is at /health (not /api/v1/health)
// The /api prefix ensures requests go through the UI server proxy to the API server
const API_BASE = resolveApiBase({ appendSuffix: false });

export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json() as Promise<{ status: string; service: string; timestamp: string }>;
}

export interface SearchRequest {
  query: string;
  collection?: string;
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
    cache: "no-store"
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Search failed" }));
    throw new Error(error.error || `Search failed: ${res.status}`);
  }

  const data = (await res.json().catch(() => null)) as any;
  const results = Array.isArray(data?.results) ? data.results : [];
  return {
    ...data,
    results: results.map((result: any) => ({
      ...result,
      metadata: result?.metadata && typeof result.metadata === "object" ? result.metadata : {},
    })),
  } as SearchResponse;
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
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Knowledge health check failed: ${res.status}`);
  }

  return res.json() as Promise<HealthResponse>;
}
