import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true });

export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json() as Promise<{ status: string; service: string; timestamp: string }>;
}

export type ValidationIssue = {
  path: string;
  message: string;
  hint?: string;
  severity: "error" | "warn";
};

export type ManifestValidateResponse = {
  valid: boolean;
  issues?: ValidationIssue[];
  timestamp: string;
};

export type PlanStep = { id: string; title: string; description: string };
export type PlanResponse = { plan: PlanStep[]; timestamp: string };

export type BundleArtifact = { path: string; sha256: string; size_bytes: number };
export type BundleBuildResponse = { artifact: BundleArtifact; timestamp: string };

export type PortConfig = {
  env_var?: string;
  description?: string;
  range?: string;
  port?: number;
};

export type ScenarioInfo = {
  id: string;
  displayName?: string;
  description?: string;
  ports?: Record<string, PortConfig>;
};

export type ScenariosResponse = {
  scenarios: ScenarioInfo[];
  timestamp: string;
};

export type ScenarioPortsResponse = {
  scenario_id: string;
  ports: Record<string, PortConfig>;
  timestamp: string;
};

export type ReachabilityResult = {
  target: string;
  type: "host" | "domain";
  reachable: boolean;
  message?: string;
  hint?: string;
};

export type ReachabilityResponse = {
  results: ReachabilityResult[];
  timestamp: string;
};

export async function validateManifest(manifest: unknown) {
  const url = buildApiUrl("/manifest/validate", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Manifest validation failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<ManifestValidateResponse>;
}

export async function buildPlan(manifest: unknown) {
  const url = buildApiUrl("/plan", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Plan generation failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<PlanResponse>;
}

export async function buildBundle(manifest: unknown) {
  const url = buildApiUrl("/bundle/build", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Bundle build failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<BundleBuildResponse>;
}

export async function listScenarios() {
  const url = buildApiUrl("/scenarios", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list scenarios: ${res.status} ${text}`);
  }
  return res.json() as Promise<ScenariosResponse>;
}

export async function getScenarioPorts(scenarioId: string) {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(scenarioId)}/ports`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get scenario ports: ${res.status} ${text}`);
  }
  return res.json() as Promise<ScenarioPortsResponse>;
}

export async function checkReachability(host?: string, domain?: string) {
  const url = buildApiUrl("/validate/reachability", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ host, domain })
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Reachability check failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<ReachabilityResponse>;
}
