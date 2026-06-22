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

interface ApiErrorResponse {
  error?: string;
}

function isApiErrorResponse(value: unknown): value is ApiErrorResponse {
  return Boolean(value && typeof value === "object");
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
    const errorPayload: unknown = await res.json().catch(() => ({ error: "Unknown error" }));
    const message = isApiErrorResponse(errorPayload) ? errorPayload.error : undefined;
    throw new Error(message || `${errorPrefix}: ${res.status}`);
  }

  return (await res.json()) as T;
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
// Approval Types
// ============================================================================

export interface DeploymentApproval {
  id: string;
  profile_id: string;
  git_commit_hash: string;
  platform: string;
  status: "pending" | "approved" | "rejected" | "stale";
  approved_by?: string;
  approved_at?: string;
  notes?: string;
  validation_id?: string;
  created_at: string;
  updated_at: string;
}

export interface ApprovalDecisionRequest {
  decision: "approved" | "rejected";
  reviewer: string;
  notes?: string;
}

export interface CreateApprovalRequest {
  git_commit_hash: string;
  platform: string;
  validation_id?: string;
}

export interface ReleaseGateStatus {
  profile_id: string;
  git_commit_hash: string;
  ready: boolean;
  platforms: PlatformGateStatus[];
}

export interface PlatformGateStatus {
  platform: string;
  required: boolean;
  status: "pending" | "approved" | "rejected" | "stale" | "missing";
}

export interface RequiredPlatformsResponse {
  profile_id: string;
  platforms: string[];
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
      return JSON.parse(line) as unknown;
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

// ============================================================================
// Migration Task API Client
// ============================================================================
//
// When a dependency swap is approved the source-code migration must be done by a
// developer. deployment-manager files it as a swarm-manager backlog `fix` item
// and surfaces the item's live status + queue position back here.

export interface MigrationTaskRequest {
  scenario: string;
  from_dependency: string;
  to_dependency: string;
  profile_id?: string;
  title?: string;
  notes?: string;
}

// MigrationTaskFeedback mirrors the swarm-manager backlog feedback contract.
export interface MigrationTaskFeedback {
  item_id: string;
  kind: string;
  name: string;
  deep_link: string;
  status: string;
  queue_position?: number;
  priority: number;
  deduped: boolean;
}

export function reportMigrationTask(req: MigrationTaskRequest): Promise<MigrationTaskFeedback> {
  return apiFetch("/migration-tasks", { method: "POST", body: req, errorPrefix: "Failed to file migration task" });
}

export function getMigrationTaskStatus(name: string, kind = "fix"): Promise<MigrationTaskFeedback> {
  const query = `?name=${encodeURIComponent(name)}&kind=${encodeURIComponent(kind)}`;
  return apiFetch(`/migration-tasks/status${query}`, { errorPrefix: "Failed to get migration task status" });
}

// ============================================================================
// Approval API Client
// ============================================================================

export function listApprovals(profileId: string, commit?: string): Promise<DeploymentApproval[]> {
  const query = commit ? `?commit=${encodeURIComponent(commit)}` : "";
  return apiFetch(`/profiles/${profileId}/approvals${query}`, { errorPrefix: "Failed to list approvals" });
}

export function getApproval(id: string): Promise<DeploymentApproval> {
  return apiFetch(`/approvals/${id}`, { errorPrefix: "Failed to get approval" });
}

export function createApproval(profileId: string, req: CreateApprovalRequest): Promise<DeploymentApproval> {
  return apiFetch(`/profiles/${profileId}/approvals`, { method: "POST", body: req, errorPrefix: "Failed to create approval" });
}

export function decideApproval(id: string, req: ApprovalDecisionRequest): Promise<DeploymentApproval> {
  return apiFetch(`/approvals/${id}/decide`, { method: "POST", body: req, errorPrefix: "Failed to decide approval" });
}

export function checkReleaseGate(profileId: string, commit: string): Promise<ReleaseGateStatus> {
  return apiFetch(`/profiles/${profileId}/release-gate?commit=${encodeURIComponent(commit)}`, { errorPrefix: "Failed to check release gate" });
}

export function setRequiredPlatforms(profileId: string, platforms: string[]): Promise<RequiredPlatformsResponse> {
  return apiFetch(`/profiles/${profileId}/required-platforms`, { method: "PUT", body: { platforms }, errorPrefix: "Failed to set required platforms" });
}

export function getRequiredPlatforms(profileId: string): Promise<RequiredPlatformsResponse> {
  return apiFetch(`/profiles/${profileId}/required-platforms`, { errorPrefix: "Failed to get required platforms" });
}

// ============================================================================
// LPBS Release Config + Releases API Client
// ============================================================================

export interface LPBSReleaseConfig {
  profile_id: string;
  lpbs_domain: string;
  lpbs_remote_profile: string;
  lpbs_app_key: string;
  default_channel: string;
  update_url: string;
  created_at?: string;
  updated_at?: string;
}

export interface VerificationItem {
  platform: string;
  channel: string;
  expected_version: string;
  observed_version?: string;
  sha512_match: boolean;
  match: boolean;
  error?: string;
  checked_at: string;
}

export interface ReleasePlatform {
  release_id: string;
  platform: string;
  status: "pending" | "uploading" | "published" | "failed" | "verify_failed";
  approval_id?: string;
  lpbs_artifact_id?: number;
  published_at?: string;
  verified_at?: string;
  error?: string;
}

export interface Release {
  id: string;
  profile_id: string;
  deployment_id?: string;
  profile_version?: number;
  git_commit_hash: string;
  release_version: string;
  channel: string;
  status: "pending" | "publishing" | "published" | "failed" | "superseded" | "verify_failed";
  release_notes?: string;
  released_by?: string;
  promoted_from_release_id?: string;
  verification_evidence?: VerificationItem[];
  platforms?: ReleasePlatform[];
  created_at: string;
  published_at?: string;
  updated_at: string;
}

export interface ReleaseListResponse {
  releases: Release[];
}

export interface StartReleaseRequest {
  channel?: string;
  git_commit_hash: string;
  release_version: string;
  release_notes?: string;
  released_by?: string;
  platforms?: string[];
}

export interface StartReleaseResponse {
  release: Release;
  steps?: Array<{ name: string; status: string; message?: string; error?: string }>;
}

export function getProfileLPBSConfig(profileId: string): Promise<LPBSReleaseConfig> {
  return apiFetch(`/profiles/${profileId}/lpbs-config`, { errorPrefix: "Failed to load LPBS config" });
}

export function saveProfileLPBSConfig(profileId: string, cfg: Partial<LPBSReleaseConfig>): Promise<LPBSReleaseConfig> {
  return apiFetch(`/profiles/${profileId}/lpbs-config`, { method: "PUT", body: cfg, errorPrefix: "Failed to save LPBS config" });
}

export function listProfileReleases(profileId: string, limit = 10): Promise<ReleaseListResponse> {
  return apiFetch(`/profiles/${profileId}/releases?limit=${limit}`, { errorPrefix: "Failed to list releases" });
}

export function getRelease(releaseId: string): Promise<Release> {
  return apiFetch(`/releases/${releaseId}`, { errorPrefix: "Failed to get release" });
}

export function reverifyRelease(releaseId: string, deep = false): Promise<Release> {
  const q = deep ? "?deep=true" : "";
  return apiFetch(`/releases/${releaseId}/verify${q}`, { method: "POST", errorPrefix: "Failed to re-verify release" });
}

export function startRelease(profileId: string, req: StartReleaseRequest): Promise<StartReleaseResponse> {
  return apiFetch(`/profiles/${profileId}/releases/start`, { method: "POST", body: req, errorPrefix: "Failed to start release" });
}
