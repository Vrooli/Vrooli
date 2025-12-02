import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

let API_BASE_URL: string | null = null;

const getApiBaseUrl = () => {
  if (API_BASE_URL === null) {
    API_BASE_URL = resolveApiBase({
      appendSuffix: true,
    });
  }
  return API_BASE_URL;
};

// ============================================================================
// Types
// ============================================================================

export interface HealthResponse {
  status: string;
  service: string;
  version: string;
  readiness: boolean;
  timestamp: string;
  dependencies: {
    database: string;
  };
}

export interface DependencyAnalysisResponse {
  scenario: string;
  dependencies: Record<string, unknown>;
  circular_dependencies: string[];
  aggregate_requirements: {
    memory: string;
    cpu: string;
    gpu: string;
    storage: string;
    network: string;
  };
  tiers: Record<string, TierFitnessScore>;
}

export interface TierFitnessScore {
  overall: number;
  portability: number;
  resources: number;
  licensing: number;
  platform_support: number;
}

export interface FitnessScoreRequest {
  scenario: string;
  tiers: number[];
}

export interface FitnessScoreResponse {
  scenario: string;
  scores: Record<number, TierFitnessScore>;
  blockers: string[];
  warnings: string[];
}

export interface DeploymentProfile {
  id: string;
  name: string;
  scenario: string;
  tiers: number[];
  swaps?: Record<string, string>;
  secrets?: Record<string, unknown>;
  settings?: Record<string, unknown>;
  version: number;
  created_at?: string;
  updated_at?: string;
}

export interface CreateProfileRequest {
  name: string;
  scenario: string;
  tiers: number[];
  swaps?: Record<string, string>;
  secrets?: Record<string, unknown>;
  settings?: Record<string, unknown>;
}

export interface CreateProfileResponse {
  id: string;
  version: number;
}

export interface DeployRequest {
  profile_id: string;
}

export interface DeployResponse {
  deployment_id: string;
  profile_id: string;
  status: string;
  logs_url: string;
  message?: string;
}

export interface DeploymentStatus {
  id: string;
  status: string;
  profile_id: string;
  started_at: string;
  completed_at: string | null;
  artifacts: string[];
  message?: string;
}

export interface SwapAnalysis {
  from: string;
  to: string;
  fitness_delta: Record<string, number>;
  impact: string;
  pros: string[];
  cons: string[];
  migration_effort: string;
  applicable_tiers: string[];
}

export interface SwapCascade {
  from: string;
  to: string;
  cascading_impacts: Array<{
    affected_scenario: string;
    reason: string;
    severity: string;
    remediation: string;
  }>;
  warnings: string[];
}

export interface TelemetryEvent {
  event?: string;
  timestamp?: string;
  details?: Record<string, unknown>;
  scenario_name?: string;
  deployment_mode?: string;
  source?: string;
}

export interface TelemetrySummary {
  scenario: string;
  path: string;
  total_events: number;
  last_event?: string;
  last_timestamp?: string;
  failure_counts?: Record<string, number>;
  recent_failures?: TelemetryEvent[];
  recent_events?: TelemetryEvent[];
}

// ============================================================================
// API Client
// ============================================================================

export async function fetchHealth(): Promise<HealthResponse> {
  const res = await fetch(buildApiUrl("/health", { baseUrl: getApiBaseUrl() }), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json();
}

export async function analyzeDependencies(scenario: string): Promise<DependencyAnalysisResponse> {
  const res = await fetch(
    buildApiUrl(`/dependencies/analyze/${scenario}`, { baseUrl: getApiBaseUrl() }),
    {
      headers: { "Content-Type": "application/json" },
      cache: "no-store"
    }
  );

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Dependency analysis failed: ${res.status}`);
  }

  return res.json();
}

export async function scoreFitness(request: FitnessScoreRequest): Promise<FitnessScoreResponse> {
  const res = await fetch(
    buildApiUrl("/fitness/score", { baseUrl: getApiBaseUrl() }),
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
      cache: "no-store"
    }
  );

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Fitness scoring failed: ${res.status}`);
  }

  return res.json();
}

export async function listProfiles(): Promise<DeploymentProfile[]> {
  const res = await fetch(
    buildApiUrl("/profiles", { baseUrl: getApiBaseUrl() }),
    {
      headers: { "Content-Type": "application/json" },
      cache: "no-store"
    }
  );

  if (!res.ok) {
    throw new Error(`Failed to list profiles: ${res.status}`);
  }

  return res.json();
}

export async function createProfile(request: CreateProfileRequest): Promise<CreateProfileResponse> {
  const res = await fetch(
    buildApiUrl("/profiles", { baseUrl: getApiBaseUrl() }),
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
      cache: "no-store"
    }
  );

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Failed to create profile: ${res.status}`);
  }

  return res.json();
}

export async function getProfile(id: string): Promise<DeploymentProfile> {
  const res = await fetch(
    buildApiUrl(`/profiles/${id}`, { baseUrl: getApiBaseUrl() }),
    {
      headers: { "Content-Type": "application/json" },
      cache: "no-store"
    }
  );

  if (!res.ok) {
    throw new Error(`Failed to get profile: ${res.status}`);
  }

  return res.json();
}

export async function updateProfile(id: string, updates: Partial<DeploymentProfile>): Promise<DeploymentProfile> {
  const res = await fetch(
    buildApiUrl(`/profiles/${id}`, { baseUrl: getApiBaseUrl() }),
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(updates),
      cache: "no-store"
    }
  );

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Failed to update profile: ${res.status}`);
  }

  return res.json();
}

export async function deployProfile(profileId: string): Promise<DeployResponse> {
  const res = await fetch(
    buildApiUrl(`/deploy/${profileId}`, { baseUrl: getApiBaseUrl() }),
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      cache: "no-store"
    }
  );

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Deployment failed: ${res.status}`);
  }

  return res.json();
}

export async function getDeploymentStatus(deploymentId: string): Promise<DeploymentStatus> {
  const res = await fetch(
    buildApiUrl(`/deployments/${deploymentId}`, { baseUrl: getApiBaseUrl() }),
    {
      headers: { "Content-Type": "application/json" },
      cache: "no-store"
    }
  );

  if (!res.ok) {
    throw new Error(`Failed to get deployment status: ${res.status}`);
  }

  return res.json();
}

export async function analyzeSwap(from: string, to: string): Promise<SwapAnalysis> {
  const res = await fetch(
    buildApiUrl(`/swaps/analyze/${from}/${to}`, { baseUrl: getApiBaseUrl() }),
    {
      headers: { "Content-Type": "application/json" },
      cache: "no-store"
    }
  );

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Swap analysis failed: ${res.status}`);
  }

  return res.json();
}

export async function analyzeSwapCascade(from: string, to: string): Promise<SwapCascade> {
  const res = await fetch(
    buildApiUrl(`/swaps/cascade/${from}/${to}`, { baseUrl: getApiBaseUrl() }),
    {
      headers: { "Content-Type": "application/json" },
      cache: "no-store"
    }
  );

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `Cascade analysis failed: ${res.status}`);
  }

  return res.json();
}

export async function uploadTelemetry(scenario: string | undefined, file: File): Promise<{ path: string }> {
  const raw = await file.text();
  const lines = raw
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  if (lines.length === 0) {
    throw new Error("File is empty");
  }
  const events = lines.map((line, idx) => {
    try {
      return JSON.parse(line);
    } catch {
      throw new Error(`Line ${idx + 1} is not valid JSON`);
    }
  });

  const res = await fetch(buildApiUrl(`/telemetry/upload${scenario ? `?scenario=${encodeURIComponent(scenario)}` : ""}`, { baseUrl: getApiBaseUrl() }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      scenario_name: scenario || "unknown",
      deployment_mode: "bundled",
      source: "deployment-manager-ui",
      events,
    }),
    cache: "no-store",
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Telemetry upload failed (${res.status})`);
  }

  const data = await res.json();
  return { path: data.path as string };
}

export async function listTelemetry(): Promise<TelemetrySummary[]> {
  const res = await fetch(buildApiUrl("/telemetry", { baseUrl: getApiBaseUrl() }), {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Failed to list telemetry (${res.status})`);
  }

  return res.json();
}
