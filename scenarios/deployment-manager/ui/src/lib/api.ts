import { buildApiUrl } from "@vrooli/api-base";
import { scenarioApiBase } from "../api/transport";
import {
  createProfile as createProfilesConnect,
  getProfile as getProfileConnect,
  listProfiles as listProfilesConnect,
  updateProfile as updateProfileConnect,
  type DeploymentProfile,
} from "../api/profiles";
import { operatorApi } from "../api/operator";

export type { DeploymentProfile } from "../api/profiles";

let API_BASE_URL: string | null = null;

const getApiBaseUrl = () => {
  if (API_BASE_URL === null) {
    API_BASE_URL = scenarioApiBase;
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

export interface EvidenceRef {
  producer: string;
  artifact_id: string;
  kind: string;
  checksum: string;
  size_bytes: number;
  created_at?: string;
}

export interface EvidenceTarget {
  ramp: string;
  platform: string;
  os: string;
  device_kind: string;
  bridge_node_id?: string;
  bridge_job_id?: string;
}

export interface TargetVerdict {
  target?: EvidenceTarget;
  disposition: string;
  refs: EvidenceRef[];
  run_id: string;
  detail: string;
}

export interface EvidenceReview {
  profile_id: string;
  git_commit_hash: string;
  verdicts: TargetVerdict[];
  ready: boolean;
  reason: string;
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
  return operatorApi.analyzeDependencies(scenario) as Promise<DependencyAnalysisResponse>;
}

export function scoreFitness(request: FitnessScoreRequest): Promise<FitnessScoreResponse> {
  return operatorApi.scoreFitness(request) as Promise<FitnessScoreResponse>;
}

export function listProfiles(): Promise<DeploymentProfile[]> {
  return listProfilesConnect();
}

export function createProfile(request: {
  name: string;
  scenario: string;
  tiers: number[];
  swaps?: Record<string, unknown>;
  secrets?: Record<string, unknown>;
  settings?: Record<string, unknown>;
}): Promise<{ id: string; version: number }> {
  return createProfilesConnect(request);
}

export function getProfile(id: string): Promise<DeploymentProfile> {
  return getProfileConnect(id);
}

export function updateProfile(id: string, updates: Partial<DeploymentProfile>): Promise<DeploymentProfile> {
  return updateProfileConnect(id, updates);
}

export function deployProfile(profileId: string): Promise<DeployResponse> {
  return operatorApi.deploy(profileId) as Promise<DeployResponse>;
}

export function getDeploymentStatus(deploymentId: string): Promise<DeploymentStatus> {
  return operatorApi.deploymentStatus(deploymentId) as Promise<DeploymentStatus>;
}

export function analyzeSwap(from: string, to: string): Promise<SwapAnalysis> {
  return operatorApi.swapAnalyze(from, to) as Promise<SwapAnalysis>;
}

export function analyzeSwapCascade(from: string, to: string): Promise<SwapCascade> {
  return operatorApi.swapCascade(from, to) as Promise<SwapCascade>;
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
  const body = {
    scenario_name: scenario || "unknown",
    deployment_mode: "bundled",
    source: "deployment-manager-ui",
    events,
  };
  const data = await operatorApi.telemetryUpload(body) as { path: string };
  return { path: data.path };
}

export function listTelemetry(): Promise<TelemetrySummary[]> {
  return operatorApi.telemetry() as Promise<TelemetrySummary[]>;
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
  return operatorApi.migrationReport(req) as Promise<MigrationTaskFeedback>;
}

export function getMigrationTaskStatus(name: string, kind = "fix"): Promise<MigrationTaskFeedback> {
  return operatorApi.migrationStatus(name, kind) as Promise<MigrationTaskFeedback>;
}

// ============================================================================
// Approval API Client
// ============================================================================

export function listApprovals(profileId: string, commit?: string): Promise<DeploymentApproval[]> {
  return operatorApi.approvals(profileId, commit) as Promise<DeploymentApproval[]>;
}

export function getApproval(id: string): Promise<DeploymentApproval> {
  return operatorApi.approval(id) as Promise<DeploymentApproval>;
}

export function createApproval(profileId: string, req: CreateApprovalRequest): Promise<DeploymentApproval> {
  return operatorApi.createApproval(profileId, req) as Promise<DeploymentApproval>;
}

export function decideApproval(id: string, req: ApprovalDecisionRequest): Promise<DeploymentApproval> {
  return operatorApi.decideApproval(id, req) as Promise<DeploymentApproval>;
}

export function checkReleaseGate(profileId: string, commit: string): Promise<ReleaseGateStatus> {
  return operatorApi.releaseGate(profileId, commit) as Promise<ReleaseGateStatus>;
}

export function getEvidenceReview(profileId: string, commit: string): Promise<EvidenceReview> {
  return operatorApi.evidenceReview(profileId, commit).then((review) => ({
    profile_id: review.profileId,
    git_commit_hash: review.gitCommitHash,
    verdicts: review.verdicts.map((verdict) => ({
      target: verdict.target ? {
        ramp: verdict.target.ramp,
        platform: verdict.target.platform,
        os: verdict.target.os,
        device_kind: String(verdict.target.deviceKind),
        bridge_node_id: verdict.target.bridgeNodeId,
        bridge_job_id: verdict.target.bridgeJobId,
      } : undefined,
      disposition: String(verdict.disposition),
      refs: verdict.refs.map((ref) => ({ producer: ref.producer, artifact_id: ref.artifactId, kind: ref.kind, checksum: ref.checksum, size_bytes: Number(ref.sizeBytes) })),
      run_id: verdict.runId,
      detail: verdict.detail,
    })),
    ready: review.ready,
    reason: review.reason,
  }));
}

export function setRequiredPlatforms(profileId: string, platforms: string[]): Promise<RequiredPlatformsResponse> {
  return operatorApi.setRequiredPlatforms(profileId, platforms) as Promise<RequiredPlatformsResponse>;
}

export function getRequiredPlatforms(profileId: string): Promise<RequiredPlatformsResponse> {
  return operatorApi.requiredPlatforms(profileId) as Promise<RequiredPlatformsResponse>;
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

export interface ReleaseTargetIdentity {
  id: string;
  platform: string;
  os: string;
  architecture: string;
  format: string;
}

export interface ReleaseCandidateArtifact {
  target: ReleaseTargetIdentity;
  immutable_ref: string;
  digest: string;
  size_bytes: number;
  signature_digest: string;
  signer_ref: string;
}

export interface ReleaseCapabilityDeclaration {
  support_owner: string;
  incident_owner: string;
  customer_contact: string;
  release_authority: string;
  rollback_authority: string;
  degraded_mode_authority: string;
}

export interface ReleaseCandidate {
  candidate_id?: string;
  source_revision: string;
  profile_revision: string;
  build_inputs?: Record<string, string>;
  dependency_lock_digest: string;
  policy_digest: string;
  artifacts: ReleaseCandidateArtifact[];
  capability_declaration?: ReleaseCapabilityDeclaration;
}

export interface ReleaseDestinationRevision {
  destination_revision_id?: string;
  kind: string;
  destination_id: string;
  configuration_digest: string;
  channel: string;
  expected_channel_revision?: string;
}

export interface ReleaseReviewBinding {
  review_id?: string;
  candidate_id: string;
  destination_revision_id: string;
  targets: string[];
  channel: string;
  evidence_set_digest: string;
  policy_digest: string;
  authorization_epoch: number;
}

export interface ReleasePublicationReceipt {
  candidate_id: string;
  destination_revision_id: string;
  target_id: string;
  artifact_digest: string;
  destination_object: string;
  producer: string;
  external_receipt: string;
  outcome: string;
  observed_at: string;
}

export interface ReleaseClientUpdateReceipt {
  candidate_id: string;
  predecessor_ref: string;
  successor_digest: string;
  target_id: string;
  verified_version: string;
  outcome: string;
  producer: string;
  external_receipt: string;
  observed_at: string;
}

export interface ReleaseRecoveryReceipt {
  release_id: string;
  candidate_id: string;
  destination_revision_id: string;
  deployment_id: string;
  action: string;
  outcome: string;
  health: string;
  external_receipt: string;
  observed_at: string;
  dry_run: boolean;
}

export interface ReleaseHealthAlert {
  code: string;
  severity: string;
  target?: string;
  message: string;
  next_action: string;
}

export interface ReleaseHealth {
  release_id: string;
  status: string;
  observed_at: string;
  publication_verified: boolean;
  client_updates_healthy: boolean;
  recovery_standing?: string;
  alerts: ReleaseHealthAlert[];
  known_durations_millis: Record<string, number>;
  supported_controls: string[];
  unsupported_controls: string[];
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
  candidate_id?: string;
  artifact_digest?: string;
  destination_revision_id?: string;
  authorization_epoch?: number;
  readiness_review_key?: string;
  candidate?: ReleaseCandidate;
  destination_revision?: ReleaseDestinationRevision;
  review_binding?: ReleaseReviewBinding;
  publication_receipts?: ReleasePublicationReceipt[];
  client_update_receipts?: ReleaseClientUpdateReceipt[];
  recovery_receipts?: ReleaseRecoveryReceipt[];
  operation_id?: string;
}

export interface ReleaseDossier {
  schema_version: number;
  generated_at: string;
  release: Release;
  health: ReleaseHealth;
  missing_proof: string[];
}

export interface ReleaseOperation {
  operation_id: string;
  release_id: string;
  profile_id: string;
  idempotency_key?: string;
  status: string;
  active_stage?: string;
  error?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
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
  artifact_digest?: string;
  candidate_id?: string;
  destination_revision_id?: string;
  authorization_epoch?: number;
  idempotency_key?: string;
  readiness_review_key?: string;
  cloud_manifest?: unknown;
  cloud_deployment_name?: string;
  cloud_bundle_path?: string;
  cloud_bundle_sha256?: string;
  cloud_bundle_size_bytes?: number;
  cloud_run_preflight?: boolean;
}

export interface StartReleaseResponse {
  operation_id?: string;
  release_id?: string;
  status?: string;
  release?: Release;
  steps?: Array<{ name: string; status: string; message?: string; error?: string }>;
}

export function getProfileLPBSConfig(profileId: string): Promise<LPBSReleaseConfig> {
  return operatorApi.lpbsConfig(profileId) as Promise<LPBSReleaseConfig>;
}

export function saveProfileLPBSConfig(profileId: string, cfg: Partial<LPBSReleaseConfig>): Promise<LPBSReleaseConfig> {
  return operatorApi.saveLpbsConfig(profileId, cfg) as Promise<LPBSReleaseConfig>;
}

export function listProfileReleases(profileId: string, limit = 10): Promise<ReleaseListResponse> {
  return operatorApi.releases(profileId, limit) as Promise<ReleaseListResponse>;
}

export function getRelease(releaseId: string): Promise<Release> {
  return operatorApi.release(releaseId) as Promise<Release>;
}

export function getReleaseDossier(releaseId: string): Promise<ReleaseDossier> {
  return operatorApi.releaseDossier(releaseId) as Promise<ReleaseDossier>;
}

export function getReleaseOperation(operationId: string): Promise<ReleaseOperation> {
  return operatorApi.releaseOperation(operationId) as Promise<ReleaseOperation>;
}

export function reverifyRelease(releaseId: string, deep = false): Promise<Release> {
  return operatorApi.reverify(releaseId, deep) as Promise<Release>;
}

export function startRelease(profileId: string, req: StartReleaseRequest): Promise<StartReleaseResponse> {
  return operatorApi.startRelease(profileId, req) as Promise<StartReleaseResponse>;
}
