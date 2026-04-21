/**
 * ai-search — typed client + React hook for the swarm-manager AI search API.
 *
 * The hook polls /search/ai/status every 60s so callers can disable UI
 * affordances when Ollama or Qdrant is unreachable. searchAI() is the
 * one-shot query helper.
 */

import { useEffect, useState } from "react";
import { defaultApiClient, isApiError, type IApiClient } from "./api-client";
import { API_ENDPOINTS } from "./api-endpoints";

export type AISearchEntity = "backlog" | "initiative" | "both";
export type AISearchFallback = "none" | "text-search" | "unavailable";

export interface AISearchFilters {
  status?: string[];
  kind?: string[];
  initiative?: string;
  include_archived?: boolean;
}

export interface AISearchRequest {
  query: string;
  entity?: AISearchEntity;
  limit?: number;
  threshold?: number;
  filters?: AISearchFilters;
}

export interface AISearchResult {
  entity: AISearchEntity;
  id: string;
  score: number;
  scorePercent: number;
  payload: Record<string, unknown>;
}

export interface AISearchResponse {
  results: AISearchResult[];
  total: number;
  query: string;
  entity: AISearchEntity;
  fallback: AISearchFallback;
  latencyMs: number;
}

export interface AISearchStatus {
  available: boolean;
  ollama: boolean;
  qdrant: boolean;
  indexedBacklog: number;
  indexedInitiatives: number;
  onDiskBacklog: number;
  onDiskInitiatives: number;
  message?: string;
}

/** searchAI posts a query to the semantic search endpoint. */
export async function searchAI(
  req: AISearchRequest,
  client: IApiClient = defaultApiClient,
): Promise<AISearchResponse> {
  const raw = await client.post<unknown>(API_ENDPOINTS.searchAI, req);
  return normalizeAISearchResponse(raw, req.entity);
}

/**
 * Normalize an AI search response at the system boundary.
 *
 * The server *should* always return `results: []` for no matches, but a nil
 * Go slice marshals to JSON `null` in a single missed code path and crashes
 * any client that does `results.length`. We defend against that (and any
 * future regression) by coercing the shape here once, so every consumer
 * downstream can trust the invariants in the TypeScript type.
 */
export function normalizeAISearchResponse(
  raw: unknown,
  requestedEntity: AISearchEntity | undefined,
): AISearchResponse {
  const r = (raw ?? {}) as Partial<AISearchResponse> & Record<string, unknown>;
  const results = Array.isArray(r.results) ? r.results : [];
  const total = typeof r.total === "number" && Number.isFinite(r.total) ? r.total : results.length;
  const latencyMs = typeof r.latencyMs === "number" && Number.isFinite(r.latencyMs) ? r.latencyMs : 0;
  const fallback: AISearchFallback =
    r.fallback === "none" || r.fallback === "text-search" || r.fallback === "unavailable"
      ? r.fallback
      : "unavailable";
  const entity: AISearchEntity =
    r.entity === "backlog" || r.entity === "initiative" || r.entity === "both"
      ? r.entity
      : (requestedEntity ?? "both");
  const query = typeof r.query === "string" ? r.query : "";
  return { results, total, query, entity, fallback, latencyMs };
}

/** getAISearchStatus fetches availability and index coverage. */
export async function getAISearchStatus(
  client: IApiClient = defaultApiClient,
): Promise<AISearchStatus> {
  return client.get<AISearchStatus>(API_ENDPOINTS.searchAIStatus);
}

const STATUS_POLL_INTERVAL_MS = 60_000;

/**
 * useAISearchStatus polls /status periodically so UI can gate AI-only
 * affordances. Returns a loading flag on first fetch; `status` stays the last
 * successful reading after that. `error` is set when every attempt has failed
 * since mount.
 */
export function useAISearchStatus(
  pollIntervalMs: number = STATUS_POLL_INTERVAL_MS,
  client: IApiClient = defaultApiClient,
) {
  const [status, setStatus] = useState<AISearchStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const fetchOnce = async () => {
      try {
        const next = await getAISearchStatus(client);
        if (cancelled) return;
        setStatus(next);
        setError(null);
      } catch (err) {
        if (cancelled) return;
        const message = isApiError(err) ? err.message : String(err);
        setError(message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    fetchOnce();
    const id = window.setInterval(fetchOnce, pollIntervalMs);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [pollIntervalMs, client]);

  return { status, loading, error };
}
