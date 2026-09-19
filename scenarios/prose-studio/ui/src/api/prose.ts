import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE, decodeApiError } from "./client";

export type Candidate = {
  id: string;
  text: string;
  measurements?: { word_count?: number; readability?: { flesch_kincaid?: number }; mattr?: number };
  eligibility?: { eligible: boolean; reason?: string };
};

export type Generation = {
  session: { id: string };
  round: { measurement_basis?: string; measurement_fallback?: string; semantic_set_measurements?: { diversity?: number; basis?: string }; lexical_set_measurements?: { diversity?: number; basis?: string } };
  candidates?: Candidate[];
  selected?: Candidate;
};

async function post<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body) });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as T;
}

export const proseApi = {
  generate: (profileKey: string, query: string) => post<Generation>("/api/v1/prose/generate", { profile_key: profileKey, query, include_candidates: true }),
  reroll: (sessionId: string) => post<Generation>("/api/v1/prose/sessions/reroll", { session_id: sessionId, include_candidates: true }),
  createDocument: (title: string, profileKey: string) => post<{ document: Record<string, unknown> }>("/api/v1/prose/documents", { document: { title, profile_key: profileKey } }),
  assemble: (id: string) => post<{ document: Record<string, unknown> }>("/api/v1/prose/documents/assemble", { id }),
  registry: () => post<Record<string, unknown>>("/api/v1/prose/registry", {}),
};
