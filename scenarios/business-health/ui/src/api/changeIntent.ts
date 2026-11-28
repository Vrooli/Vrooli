import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";

export interface ChangeIntentAnchor {
  path: string;
  line: number;
  sha256: string;
  kind: string;
  attestation: string;
}

export interface ChangeIntentResult {
  scenario: string;
  requirement_id: string;
  title: string;
  description: string;
  declared_status: string;
  relation: "support" | "contradiction" | "unknown";
  relation_basis: string;
  lexical_overlap: number;
  requirement_source: ChangeIntentAnchor;
  implementation_anchors: ChangeIntentAnchor[];
}

export interface ChangeIntentResponse {
  scenario: string;
  intent: string;
  scope: string;
  results: ChangeIntentResult[];
  limitations: string[];
  degradations: string[];
}

export async function assessChangeIntent(intent: string, limit = 10): Promise<ChangeIntentResponse> {
  const response = await fetch(buildApiUrl("/api/v1/change-intent/assess", { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    body: JSON.stringify({ scenario: "business-health", intent, limit: Math.min(Math.max(limit, 1), 20) }),
  });
  if (!response.ok) throw await decodeApiError(response);
  const payload = await response.json() as Partial<ChangeIntentResponse>;
  return {
    scenario: payload.scenario ?? "business-health",
    intent: payload.intent ?? intent,
    scope: payload.scope ?? "",
    results: Array.isArray(payload.results) ? payload.results : [],
    limitations: Array.isArray(payload.limitations) ? payload.limitations : [],
    degradations: Array.isArray(payload.degradations) ? payload.degradations : [],
  };
}
