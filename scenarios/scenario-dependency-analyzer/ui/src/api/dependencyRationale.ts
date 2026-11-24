import { buildScenarioApiUrl } from "./client";

export interface DependencyRationaleSource {
  owner: string;
  basis: string;
  path: string;
  fingerprint: string;
  observedAt: string;
}

export interface DependencyRationaleFields {
  description?: string;
  purpose?: string;
  runtimeOnlyRationale?: string;
  degradedBehavior?: string;
}

export interface DependencyRationaleResult {
  id: string;
  title: string;
  score: number;
  snippet: string;
  path: string;
  consumer: string;
  dependency: string;
  type?: string;
  required: boolean;
  enabled: boolean;
  runtimeOnly: boolean;
  evidence: DependencyRationaleFields;
  source: DependencyRationaleSource;
}

export interface DependencyRationaleResponse {
  query: string;
  scenario: string;
  results: DependencyRationaleResult[];
  scan: {
    basis: string;
    scenariosScanned: number;
    scenariosFailed: number;
    complete: boolean;
    runtimeStatusKnown: boolean;
  };
  degradations: { scenario?: string; code: string; reason: string }[];
}

export async function searchDependencyRationale(
  query: string,
  scenario: string,
  limit = 10,
): Promise<DependencyRationaleResponse> {
  const response = await fetch(buildScenarioApiUrl("/dependencies/rationale/search"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    body: JSON.stringify({ query, scenario, limit: Math.min(Math.max(limit, 1), 10) }),
  });
  if (!response.ok) throw new Error(`Dependency rationale request failed (${response.status})`);
  const payload = await response.json() as Partial<DependencyRationaleResponse>;
  return {
    query: payload.query ?? query,
    scenario: payload.scenario ?? scenario,
    results: Array.isArray(payload.results) ? payload.results : [],
    scan: payload.scan ?? { basis: "unknown", scenariosScanned: 0, scenariosFailed: 0, complete: false, runtimeStatusKnown: false },
    degradations: Array.isArray(payload.degradations) ? payload.degradations : [],
  };
}
