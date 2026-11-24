import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";

export interface AmbientSeam {
  id: string;
  title: string;
  summary: string;
  scenario: string;
  kind: string;
  expression: string;
  context_symbol: string;
  source_layer: "TEST_SOURCE_CANDIDATE" | "NON_TEST_SOURCE_CANDIDATE";
  source_path: string;
  line: number;
  source_sha256: string;
  argument_context_status: string;
  argument_callee?: string;
  argument_line?: number;
  matched_terms: string[];
  matched_fields: string[];
  match_evidence?: { term: string; field: string; value: string; source_path: string; line: number; source_sha256: string }[];
  match_status?: "MATCHED" | "UNKNOWN";
  basis: string;
  limit: string;
  score: number;
}

export interface AmbientSeamResponse {
  query: string;
  scenario: string;
  scope: string;
  match_status?: "MATCHED" | "UNKNOWN";
  results: AmbientSeam[];
  degradations: { code: string; message: string }[];
}

export async function searchAmbientSeams(query: string, scenario: string, limit = 12): Promise<AmbientSeamResponse> {
  const response = await fetch(buildApiUrl("/api/v1/ambient-seams/search", { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, scenario, limit }),
  });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as AmbientSeamResponse;
}
