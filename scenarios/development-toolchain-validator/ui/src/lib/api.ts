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
// Health
// ─────────────────────────────────────────────────────────────────────────────

export async function fetchHealth(): Promise<HealthResponse> {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json() as Promise<HealthResponse>;
}

// ─────────────────────────────────────────────────────────────────────────────
// References - CRUD operations
// [REQ:P0-002] Reference Scenario API Endpoints
// ─────────────────────────────────────────────────────────────────────────────

export async function fetchReferences(template?: string): Promise<Reference[]> {
  const params = template ? `?template=${encodeURIComponent(template)}` : "";
  const url = buildApiUrl(`/references${params}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: "Unknown error" })) as ApiError;
    throw new Error(errorBody.error ?? `Failed to fetch references: ${res.status}`);
  }

  const data = await res.json() as ReferenceListResponse;
  return data.references ?? [];
}

export async function fetchReferenceById(id: string): Promise<Reference> {
  const url = buildApiUrl(`/references/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: "Unknown error" })) as ApiError;
    throw new Error(errorBody.error ?? `Failed to fetch reference: ${res.status}`);
  }

  return res.json() as Promise<Reference>;
}

export async function fetchReferenceBySlug(slug: string): Promise<Reference> {
  const url = buildApiUrl(`/references/by-slug/${encodeURIComponent(slug)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: "Unknown error" })) as ApiError;
    throw new Error(errorBody.error ?? `Failed to fetch reference: ${res.status}`);
  }

  return res.json() as Promise<Reference>;
}

export async function createReference(input: CreateReferenceInput): Promise<Reference> {
  const url = buildApiUrl("/references", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: "Unknown error" })) as ApiError;
    throw new Error(errorBody.error ?? `Failed to create reference: ${res.status}`);
  }

  return res.json() as Promise<Reference>;
}

export async function updateReference(id: string, input: UpdateReferenceInput): Promise<Reference> {
  const url = buildApiUrl(`/references/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input)
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: "Unknown error" })) as ApiError;
    throw new Error(errorBody.error ?? `Failed to update reference: ${res.status}`);
  }

  return res.json() as Promise<Reference>;
}

export async function deleteReference(id: string): Promise<void> {
  const url = buildApiUrl(`/references/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: "Unknown error" })) as ApiError;
    throw new Error(errorBody.error ?? `Failed to delete reference: ${res.status}`);
  }
}
