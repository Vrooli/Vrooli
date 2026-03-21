// DOC: docs/internal/SEAMS.md#ui--api
// DOC: docs/reference/api-endpoints.md
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import type { Scheme, Information, Thought, ThoughtEdge, Suggestion } from "./types";

const API_BASE = resolveApiBase({ appendSuffix: true });

/** Full API route paths used by this client (for documentation and tooling detection) */
export const API_ROUTES: Readonly<Record<string, string>> = {
  health: "/api/v1/health",
  schemes: "/api/v1/schemes",
  schemeById: "/api/v1/schemes/:id",
  information: "/api/v1/schemes/:schemeId/information",
  informationById: "/api/v1/schemes/:schemeId/information/:infoId",
  thoughts: "/api/v1/thoughts",
  thoughtById: "/api/v1/thoughts/:id",
  edges: "/api/v1/thoughts/:id/edges",
  edgeById: "/api/v1/thoughts/:id/edges/:edgeId",
  export: "/api/v1/schemes/:id/export",
  providers: "/api/v1/providers",
  suggestions: "/api/v1/schemes/:id/suggestions",
};

/** Error categories returned by the API */
export type ErrorCategory = "validation" | "not_found" | "conflict" | "dependency" | "internal";

/** Structured error from the API */
export interface APIError {
  category: ErrorCategory;
  message: string;
  retryable: boolean;
}

/** Extended Error that carries structured API error details */
export class ApiRequestError extends Error {
  status: number;
  category: ErrorCategory;
  retryable: boolean;

  constructor(status: number, apiError: APIError) {
    super(apiError.message);
    this.name = "ApiRequestError";
    this.status = status;
    this.category = apiError.category;
    this.retryable = apiError.retryable;
  }
}

function apiUrl(path: string): string {
  return buildApiUrl(path, { baseUrl: API_BASE });
}

const ERROR_CATEGORIES: ReadonlySet<string> = new Set<ErrorCategory>(["validation", "not_found", "conflict", "dependency", "internal"]);

function isErrorCategory(value: unknown): value is ErrorCategory {
  return typeof value === "string" && ERROR_CATEGORIES.has(value);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

async function parseErrorBody(res: Response): Promise<APIError> {
  try {
    const json: unknown = await res.json();
    if (isRecord(json)) {
      if (isErrorCategory(json["category"]) && typeof json["message"] === "string") {
        return {
          category: json["category"],
          message: json["message"],
          retryable: typeof json["retryable"] === "boolean" ? json["retryable"] : false,
        };
      }
      const errMsg = typeof json["error"] === "string" ? json["error"] : undefined;
      return {
        category: res.status >= 500 ? "internal" : "validation",
        message: errMsg ?? "An unexpected error occurred",
        retryable: res.status >= 500,
      };
    }
  } catch { /* non-JSON body */ }
  return {
    category: "internal",
    message: "An unexpected error occurred",
    retryable: res.status >= 500,
  };
}

async function apiFetch<T = void>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(apiUrl(path), {
      headers: { "Content-Type": "application/json" },
      ...init,
    });
  } catch {
    throw new ApiRequestError(0, {
      category: "dependency",
      message: "Unable to reach the server. Check your connection and try again.",
      retryable: true,
    });
  }
  if (!res.ok) {
    throw new ApiRequestError(res.status, await parseErrorBody(res));
  }
  if (res.status === 204) return undefined as never;
  const body = (await res.json()) as T;
  return body;
}

export async function fetchHealth() {
  return apiFetch<{ status: string; service: string; timestamp: string }>("/health");
}

// Schemes
export const listSchemes = () => apiFetch<Scheme[]>("/schemes");
export const createScheme = (name: string) =>
  apiFetch<Scheme>("/schemes", { method: "POST", body: JSON.stringify({ name }) });
export const getScheme = (id: string) => apiFetch<Scheme>(`/schemes/${id}`);
export const updateScheme = (id: string, name: string) =>
  apiFetch<Scheme>(`/schemes/${id}`, { method: "PUT", body: JSON.stringify({ name }) });
export const deleteScheme = (id: string) =>
  apiFetch<void>(`/schemes/${id}`, { method: "DELETE" });

// Information
export const listInformation = (schemeId: string) =>
  apiFetch<Information[]>(`/schemes/${schemeId}/information`);
export const createInformation = (schemeId: string, data: { type: string; content: string; canvas_x: number; canvas_y: number }) =>
  apiFetch<Information>(`/schemes/${schemeId}/information`, { method: "POST", body: JSON.stringify(data) });
export const updateInformation = (schemeId: string, id: string, data: Partial<Pick<Information, "type" | "content" | "canvas_x" | "canvas_y">>) =>
  apiFetch<Information>(`/schemes/${schemeId}/information/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deleteInformation = (schemeId: string, id: string) =>
  apiFetch<void>(`/schemes/${schemeId}/information/${id}`, { method: "DELETE" });

// Thoughts
export const listThoughts = (schemeId?: string) =>
  apiFetch<Thought[]>(`/thoughts${schemeId ? `?scheme_id=${schemeId}` : ""}`);
export const createThought = (data: { scheme_id?: string; title: string; body: string; canvas_x: number; canvas_y: number }) =>
  apiFetch<Thought>("/thoughts", { method: "POST", body: JSON.stringify(data) });
export const updateThought = (id: string, data: Partial<Pick<Thought, "title" | "body" | "canvas_x" | "canvas_y">>) =>
  apiFetch<Thought>(`/thoughts/${id}`, { method: "PUT", body: JSON.stringify(data) });
export const deleteThought = (id: string) =>
  apiFetch<void>(`/thoughts/${id}`, { method: "DELETE" });

// Edges
export const listEdges = (thoughtId: string) => apiFetch<ThoughtEdge[]>(`/thoughts/${thoughtId}/edges`);
export const createEdge = (sourceId: string, data: { target_id: string; label: string }) =>
  apiFetch<ThoughtEdge>(`/thoughts/${sourceId}/edges`, { method: "POST", body: JSON.stringify(data) });
export const deleteEdge = (thoughtId: string, edgeId: string) =>
  apiFetch<void>(`/thoughts/${thoughtId}/edges/${edgeId}`, { method: "DELETE" });

// Export
export const exportScheme = (schemeId: string) =>
  apiFetch<{ scheme: Scheme; information: Information[]; thoughts: Thought[]; edges: ThoughtEdge[]; export_format: string }>(`/schemes/${schemeId}/export`);

// Suggestions
export const generateSuggestions = (schemeId: string) =>
  apiFetch<{ suggestions: Suggestion[]; provider: string }>(`/schemes/${schemeId}/suggestions`, { method: "POST" });

// Providers
export const listProviders = () =>
  apiFetch<Array<{ name: string; url: string; active: boolean; fallback: boolean }>>("/providers");
