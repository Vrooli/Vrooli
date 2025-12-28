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

export type BundleInfo = {
  path: string;
  filename: string;
  scenario_id: string;
  sha256: string;
  size_bytes: number;
  created_at: string;
};
export type ListBundlesResponse = { bundles: BundleInfo[]; timestamp: string };

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

export async function listBundles() {
  const url = buildApiUrl("/bundles", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list bundles: ${res.status} ${text}`);
  }
  return res.json() as Promise<ListBundlesResponse>;
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

export type PreflightCheckStatus = "pass" | "warn" | "fail";

export type PreflightCheck = {
  id: string;
  title: string;
  status: PreflightCheckStatus;
  details?: string;
  hint?: string;
  data?: Record<string, string>;
};

export type PreflightResponse = {
  ok: boolean;
  checks: PreflightCheck[];
  issues?: ValidationIssue[];
  timestamp: string;
};

export async function runPreflight(manifest: unknown) {
  const url = buildApiUrl("/preflight", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(manifest)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Preflight check failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<PreflightResponse>;
}

// ============================================================================
// Deployment Management Types & Functions
// ============================================================================

export type DeploymentStatus =
  | "pending"
  | "setup_running"
  | "setup_complete"
  | "deploying"
  | "deployed"
  | "failed"
  | "stopped";

export type DeploymentSummary = {
  id: string;
  name: string;
  scenario_id: string;
  status: DeploymentStatus;
  domain?: string;
  host?: string;
  error_message?: string;
  created_at: string;
  last_deployed_at?: string;
};

export type Deployment = {
  id: string;
  name: string;
  scenario_id: string;
  status: DeploymentStatus;
  manifest: unknown;
  bundle_path?: string;
  bundle_sha256?: string;
  bundle_size_bytes?: number;
  setup_result?: unknown;
  deploy_result?: unknown;
  last_inspect_result?: VPSInspectResult;
  error_message?: string;
  error_step?: string;
  created_at: string;
  updated_at: string;
  last_deployed_at?: string;
  last_inspected_at?: string;
};

export type VPSInspectResult = {
  ok: boolean;
  scenario_status?: unknown;
  resource_status?: unknown;
  scenario_logs?: string;
  error?: string;
  timestamp: string;
};

export type ListDeploymentsResponse = {
  deployments: DeploymentSummary[];
  timestamp: string;
};

export type DeploymentResponse = {
  deployment: Deployment;
  created?: boolean;
  updated?: boolean;
  timestamp: string;
};

export type ExecuteDeploymentResponse = {
  deployment: Deployment;
  success: boolean;
  error?: string;
  timestamp: string;
};

export type InspectDeploymentResponse = {
  result: VPSInspectResult;
  timestamp: string;
};

export async function listDeployments(): Promise<ListDeploymentsResponse> {
  const url = buildApiUrl("/deployments", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to list deployments: ${res.status} ${text}`);
  }
  return res.json() as Promise<ListDeploymentsResponse>;
}

export async function createDeployment(
  manifest: unknown,
  name?: string
): Promise<DeploymentResponse> {
  const url = buildApiUrl("/deployments", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ manifest, name })
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to create deployment: ${res.status} ${text}`);
  }
  return res.json() as Promise<DeploymentResponse>;
}

export async function getDeployment(id: string): Promise<DeploymentResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to get deployment: ${res.status} ${text}`);
  }
  return res.json() as Promise<DeploymentResponse>;
}

export async function executeDeployment(id: string): Promise<ExecuteDeploymentResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(id)}/execute`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to execute deployment: ${res.status} ${text}`);
  }
  return res.json() as Promise<ExecuteDeploymentResponse>;
}

export async function inspectDeployment(id: string): Promise<InspectDeploymentResponse> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(id)}/inspect`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to inspect deployment: ${res.status} ${text}`);
  }
  return res.json() as Promise<InspectDeploymentResponse>;
}

export async function stopDeployment(id: string): Promise<{ success: boolean; error?: string; timestamp: string }> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(id)}/stop`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to stop deployment: ${res.status} ${text}`);
  }
  return res.json();
}

export async function deleteDeployment(id: string, stopOnVPS = false): Promise<{ deleted: boolean; timestamp: string }> {
  const url = buildApiUrl(`/deployments/${encodeURIComponent(id)}${stopOnVPS ? "?stop=true" : ""}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Failed to delete deployment: ${res.status} ${text}`);
  }
  return res.json();
}
