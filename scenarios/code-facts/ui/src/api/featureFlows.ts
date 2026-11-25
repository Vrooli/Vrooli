import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";

export interface FeatureFlowStep {
  call_site_id: string;
  from: string;
  to?: string;
  call: string;
  path: string;
  to_path?: string;
  line: number;
  graph_hash?: string;
  to_graph_hash?: string;
  resolution: string;
}

export interface FeatureFlowHit {
  id: string;
  title: string;
  text: string;
  score: number;
  path: string;
  paths: string[];
  graph_hash: string;
  attestation: string;
  depth_limited: boolean;
  branch_truncated: boolean;
  steps: FeatureFlowStep[];
}

export interface FeatureFlowResponse {
  query: string;
  scenario: string;
  path: string;
  results: FeatureFlowHit[];
  degradations: { code: string; message: string }[];
}

// Search only the caller-selected scenario subtree. This is a bounded source
// call-graph query, not a runtime execution or reachability probe.
export async function searchFeatureFlows(
  query: string,
  scenario: string,
  path: string,
  limit = 5,
): Promise<FeatureFlowResponse> {
  const response = await fetch(buildApiUrl("/api/v1/flows/search", { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, scenario, path, limit: Math.min(Math.max(limit, 1), 5) }),
  });
  if (!response.ok) throw await decodeApiError(response);
  const payload = (await response.json()) as Partial<FeatureFlowResponse>;
  return {
    query: payload.query ?? query,
    scenario: payload.scenario ?? scenario,
    path,
    results: Array.isArray(payload.results) ? payload.results : [],
    // Go omits this optional field when the successful response has no
    // degradations. Keep the UI consumer's collection contract stable.
    degradations: Array.isArray(payload.degradations) ? payload.degradations : [],
  };
}
