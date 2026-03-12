import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true });

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

export interface Reference {
  id: string;
  slug: string;
  name: string;
  template: string;
  path: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface ReferenceListResponse {
  references: Reference[];
  count: number;
}

export interface CreateReferenceInput {
  slug: string;
  name: string;
  template: string;
  path: string;
  description?: string;
}

export interface UpdateReferenceInput {
  name?: string;
  template?: string;
  path?: string;
  description?: string;
}

export interface ApiError {
  error: string;
}

// ─────────────────────────────────────────────────────────────────────────────
// Request Utilities
// ─────────────────────────────────────────────────────────────────────────────
//
// Decision boundary: Error handling for all API requests
// - If response is not ok, extract error message from body or use fallback
// - Error messages are user-facing, so keep them informative
// ─────────────────────────────────────────────────────────────────────────────

interface RequestOptions<T> {
  /** HTTP method (default: GET) */
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  /** Request body for POST/PATCH */
  body?: unknown;
  /** Fallback error message when API error extraction fails */
  errorContext: string;
  /** Transform successful response (default: return response as-is) */
  transform?: (data: unknown) => T;
}

/**
 * Centralized request helper that consolidates error handling.
 *
 * This is the single responsibility point for:
 * - Building URLs with correct base
 * - Setting headers (Content-Type, cache control)
 * - Extracting error messages from failed responses
 * - Optional response transformation
 */
async function apiRequest<T>(
  path: string,
  options: RequestOptions<T>
): Promise<T> {
  const { method = "GET", body, errorContext, transform } = options;

  const url = buildApiUrl(path, { baseUrl: API_BASE });

  const fetchOptions: RequestInit = {
    method,
    headers: { "Content-Type": "application/json" },
    cache: method === "GET" ? "no-store" : undefined
  };

  if (body !== undefined) {
    fetchOptions.body = JSON.stringify(body);
  }

  const res = await fetch(url, fetchOptions);

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: "Unknown error" })) as ApiError;
    throw new Error(errorBody.error ?? `${errorContext}: ${res.status}`);
  }

  // For DELETE, no body expected
  if (method === "DELETE") {
    return undefined as T;
  }

  const data = await res.json();
  return transform ? transform(data) : data as T;
}

// ─────────────────────────────────────────────────────────────────────────────
// Skill Connections
// [REQ:P0-003] Skill Connection Management
// ─────────────────────────────────────────────────────────────────────────────

export interface SkillConnection {
  id: string;
  reference_id: string;
  skill_id: string;
  skill_version?: string;
  skill_content_hash?: string;
  connected_at: string;
  updated_at: string;
}

export interface SkillConnectionListResponse {
  connections: SkillConnection[];
  count: number;
}

// ─────────────────────────────────────────────────────────────────────────────
// Health
// ─────────────────────────────────────────────────────────────────────────────

export function fetchHealth(): Promise<HealthResponse> {
  return apiRequest<HealthResponse>("/health", {
    errorContext: "API health check failed"
  });
}

// ─────────────────────────────────────────────────────────────────────────────
// References - CRUD operations
// [REQ:P0-002] Reference Scenario API Endpoints
// ─────────────────────────────────────────────────────────────────────────────

export function fetchReferences(template?: string): Promise<Reference[]> {
  const params = template ? `?template=${encodeURIComponent(template)}` : "";
  return apiRequest<Reference[]>(`/references${params}`, {
    errorContext: "Failed to fetch references",
    transform: (data) => (data as ReferenceListResponse).references ?? []
  });
}

export function fetchReferenceById(id: string): Promise<Reference> {
  return apiRequest<Reference>(`/references/${encodeURIComponent(id)}`, {
    errorContext: "Failed to fetch reference"
  });
}

export function fetchReferenceBySlug(slug: string): Promise<Reference> {
  return apiRequest<Reference>(`/references/by-slug/${encodeURIComponent(slug)}`, {
    errorContext: "Failed to fetch reference"
  });
}

export function createReference(input: CreateReferenceInput): Promise<Reference> {
  return apiRequest<Reference>("/references", {
    method: "POST",
    body: input,
    errorContext: "Failed to create reference"
  });
}

export function updateReference(id: string, input: UpdateReferenceInput): Promise<Reference> {
  return apiRequest<Reference>(`/references/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: input,
    errorContext: "Failed to update reference"
  });
}

export function deleteReference(id: string): Promise<void> {
  return apiRequest<void>(`/references/${encodeURIComponent(id)}`, {
    method: "DELETE",
    errorContext: "Failed to delete reference"
  });
}

// ─────────────────────────────────────────────────────────────────────────────
// Skill Connections - API operations
// [REQ:P0-003] Skill Connection Management
// ─────────────────────────────────────────────────────────────────────────────

export function fetchConnections(referenceId?: string): Promise<SkillConnection[]> {
  const params = referenceId ? `?reference_id=${encodeURIComponent(referenceId)}` : "";
  return apiRequest<SkillConnection[]>(`/connections${params}`, {
    errorContext: "Failed to fetch connections",
    transform: (data) => (data as SkillConnectionListResponse).connections ?? []
  });
}

export function fetchConnectionsByReference(referenceId: string): Promise<SkillConnection[]> {
  return fetchConnections(referenceId);
}
