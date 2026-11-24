import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";

export interface PersistenceEvidence { layer: string; path: string; line: number; sha256: string; basis: string; status: string }
export interface PersistencePattern {
  id: string; title: string; summary: string; scenario: string; entity: string; source_path: string; line: number; source_sha256: string;
  classification: string; schema: PersistenceEvidence; schema_binding: PersistenceEvidence; migration: PersistenceEvidence; repository: PersistenceEvidence; limits: string[]; score: number;
}
export interface PersistenceUnknown { entity: string; status: "UNKNOWN"; reason: string; schema: PersistenceEvidence }
export interface PersistencePatternResponse { query: string; scenario: string; scope: string; results: PersistencePattern[]; unknowns?: PersistenceUnknown[]; degradations: string[] }

export async function searchPersistencePatterns(query: string, scenario: string, limit = 5): Promise<PersistencePatternResponse> {
  const response = await fetch(buildApiUrl("/api/v1/persistence-patterns/search", { baseUrl: API_BASE }), {
    method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ query, scenario, limit }),
  });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as PersistencePatternResponse;
}
