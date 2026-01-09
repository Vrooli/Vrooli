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
// Shared Fetch Helper
// ============================================================================

interface FetchOptions {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  errorPrefix?: string;
}

async function apiFetch<T>(path: string, options: FetchOptions = {}): Promise<T> {
  const { method = "GET", body, errorPrefix = "Request failed" } = options;

  const fetchOptions: RequestInit = {
    method,
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  };

  if (body !== undefined) {
    fetchOptions.body = JSON.stringify(body);
  }

  const res = await fetch(buildApiUrl(path, { baseUrl: getApiBaseUrl() }), fetchOptions);

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(error.error || `${errorPrefix}: ${res.status}`);
  }

  return res.json();
}

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

export function fetchHealth(): Promise<HealthResponse> {
  return apiFetch("/health", { errorPrefix: "API health check failed" });
}

export function analyzeDependencies(scenario: string): Promise<DependencyAnalysisResponse> {
  return apiFetch(`/dependencies/analyze/${scenario}`, { errorPrefix: "Dependency analysis failed" });
}

export function scoreFitness(request: FitnessScoreRequest): Promise<FitnessScoreResponse> {
  return apiFetch("/fitness/score", { method: "POST", body: request, errorPrefix: "Fitness scoring failed" });
}

export function listProfiles(): Promise<DeploymentProfile[]> {
  return apiFetch("/profiles", { errorPrefix: "Failed to list profiles" });
}

export function createProfile(request: CreateProfileRequest): Promise<CreateProfileResponse> {
  return apiFetch("/profiles", { method: "POST", body: request, errorPrefix: "Failed to create profile" });
}

export function getProfile(id: string): Promise<DeploymentProfile> {
  return apiFetch(`/profiles/${id}`, { errorPrefix: "Failed to get profile" });
}

export function updateProfile(id: string, updates: Partial<DeploymentProfile>): Promise<DeploymentProfile> {
  return apiFetch(`/profiles/${id}`, { method: "PUT", body: updates, errorPrefix: "Failed to update profile" });
}

export function deployProfile(profileId: string): Promise<DeployResponse> {
  return apiFetch(`/deploy/${profileId}`, { method: "POST", errorPrefix: "Deployment failed" });
}

export function getDeploymentStatus(deploymentId: string): Promise<DeploymentStatus> {
  return apiFetch(`/deployments/${deploymentId}`, { errorPrefix: "Failed to get deployment status" });
}

export function analyzeSwap(from: string, to: string): Promise<SwapAnalysis> {
  return apiFetch(`/swaps/analyze/${from}/${to}`, { errorPrefix: "Swap analysis failed" });
}

export function analyzeSwapCascade(from: string, to: string): Promise<SwapCascade> {
  return apiFetch(`/swaps/cascade/${from}/${to}`, { errorPrefix: "Cascade analysis failed" });
}

function parseJsonLines(raw: string): unknown[] {
  const lines = raw.split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
  if (lines.length === 0) {
    throw new Error("File is empty");
  }
  return lines.map((line, idx) => {
    try {
      return JSON.parse(line);
    } catch {
      throw new Error(`Line ${idx + 1} is not valid JSON`);
    }
  });
}

export async function uploadTelemetry(scenario: string | undefined, file: File): Promise<{ path: string }> {
  const events = parseJsonLines(await file.text());
  const queryParam = scenario ? `?scenario=${encodeURIComponent(scenario)}` : "";
  const body = {
    scenario_name: scenario || "unknown",
    deployment_mode: "bundled",
    source: "deployment-manager-ui",
    events,
  };
  const data = await apiFetch<{ path: string }>(`/telemetry/upload${queryParam}`, {
    method: "POST",
    body,
    errorPrefix: "Telemetry upload failed",
  });
  return { path: data.path };
}

export function listTelemetry(): Promise<TelemetrySummary[]> {
  return apiFetch("/telemetry", { errorPrefix: "Failed to list telemetry" });
}
