import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE, decodeApiError } from "./client";

export interface DriftEvidence { path: string; line: number; sha256: string; basis: string }
export interface DocCodeDrift { id: string; title: string; summary: string; scenario: string; domain: string; claim: string; classification: string; status: string; documentation: DriftEvidence; code: DriftEvidence[]; api_status?: string; api?: DriftEvidence[]; limits: string[]; score: number }
export interface DocCodeDriftResponse { query: string; scope: string; authority: string; results: DocCodeDrift[]; degradations: string[] }

export async function searchDocCodeDrift(query: string, scope: string, limit = 10): Promise<DocCodeDriftResponse> {
  const response = await fetch(buildApiUrl("/api/v1/doc-code-drift/search", { baseUrl: API_BASE }), { method:"POST", headers:{"Content-Type":"application/json"}, body:JSON.stringify({query,scope,limit}) });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as DocCodeDriftResponse;
}
