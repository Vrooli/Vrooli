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
    buildApiUrl(`/api/v1/dependencies/analyze/${scenario}`, { baseUrl: getApiBaseUrl() }),
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
    buildApiUrl("/api/v1/fitness/score", { baseUrl: getApiBaseUrl() }),
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
    buildApiUrl("/api/v1/profiles", { baseUrl: getApiBaseUrl() }),
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
    buildApiUrl("/api/v1/profiles", { baseUrl: getApiBaseUrl() }),
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
    buildApiUrl(`/api/v1/profiles/${id}`, { baseUrl: getApiBaseUrl() }),
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
    buildApiUrl(`/api/v1/profiles/${id}`, { baseUrl: getApiBaseUrl() }),
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
    buildApiUrl(`/api/v1/deploy/${profileId}`, { baseUrl: getApiBaseUrl() }),
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
    buildApiUrl(`/api/v1/deployments/${deploymentId}`, { baseUrl: getApiBaseUrl() }),
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
    buildApiUrl(`/api/v1/swaps/analyze/${from}/${to}`, { baseUrl: getApiBaseUrl() }),
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
    buildApiUrl(`/api/v1/swaps/cascade/${from}/${to}`, { baseUrl: getApiBaseUrl() }),
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
