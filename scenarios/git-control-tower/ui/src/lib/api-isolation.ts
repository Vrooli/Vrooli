// API client for the scenario isolation endpoint, which proxies test-genie's
// routed-test-db eligibility decision.

import { API_BASE, buildApiUrl, handleResponse } from "./api-internals";

export type IsolationStatus = "routed" | "not_routed" | "unknown";

export interface IsolationViolation {
  rule_id: string;
  severity: string;
  file?: string;
  line?: number;
  excerpt?: string;
}

export interface IsolationResponse {
  status: IsolationStatus;
  reasons?: string[];
  violations?: IsolationViolation[];
}

export async function fetchScenarioIsolation(slug: string): Promise<IsolationResponse> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(slug)}/isolation`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  return handleResponse<IsolationResponse>(res);
}
