import { buildUrl, throwIfNotOk } from "./client";
import type { BundlePreflightResponse } from "./types";

// ==================== Pipeline API Types ====================

export interface PipelineConfig {
  scenario_name: string;
  platforms?: string[];
  deployment_mode?: "bundled" | "proxy";
  template_type?: string;
  proxy_url?: string;
  bundle_manifest_path?: string;
  skip_preflight?: boolean;
  skip_smoke_test?: boolean;
  stop_after_stage?: "bundle" | "preflight" | "generate" | "build" | "smoketest" | "distribution";
  clean?: boolean;
  sign?: boolean;
  distribute?: boolean;
  preflight_timeout_seconds?: number;
  preflight_secrets?: Record<string, string>;
  /**
   * Client-provided key for request deduplication.
   * If a pipeline with the same idempotency key already exists and is running or completed,
   * the existing pipeline status will be returned instead of starting a new pipeline.
   * This enables safe retries where "running twice is no worse than running once".
   */
  idempotency_key?: string;
}

export interface PipelineStageResult {
  stage: string;
  status: string;
  started_at: number;
  completed_at?: number;
  error?: string;
  details?: unknown;
}

export interface PipelineStatus {
  pipeline_id: string;
  scenario_name?: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled" | string;
  current_stage?: string;
  /** Completion percentage (0-100) */
  progress_percent?: number;
  /** Human-readable progress message (e.g., "Running bundle stage (1/6)") */
  progress_message?: string;
  stages?: Record<string, PipelineStageResult>;
  stage_order?: string[];
  config?: PipelineConfig;
  started_at?: number;
  completed_at?: number;
  error?: string;
  final_artifacts?: Record<string, string>;
  stopped_after_stage?: string;
}

export interface PipelineRunResponse {
  pipeline_id: string;
  status_url?: string;
  message?: string;
}

export interface PipelineResumeResponse {
  pipeline_id: string;
  parent_pipeline_id: string;
  status_url: string;
  resume_from_stage: string;
  message?: string;
}

// ==================== Verbose Stage Details ====================

/** Bundle stage details (from bundle/types.go PackageResult) */
export interface BundleStageDetails {
  bundle_dir?: string;
  manifest_path?: string;
  manifest_content?: Record<string, unknown>;
  runtime_binaries?: Record<string, string>;
  copied_artifacts?: string[];
  total_size_bytes?: number;
  total_size_human?: string;
  size_warning?: {
    level?: string;
    message?: string;
    total_bytes?: number;
    total_human?: string;
    large_files?: { path: string; size_bytes: number; size_human?: string }[];
  };
}

/** Generate stage details (from generation/types.go GenerateResponse) */
export interface GenerateStageDetails {
  build_id?: string;
  pipeline_id?: string;
  status?: string;
  scenario_name?: string;
  desktop_path?: string;
  install_instructions?: string;
  test_command?: string;
  status_url?: string;
  detected_metadata?: {
    name: string;
    display_name?: string;
    description?: string;
    version?: string;
    author?: string;
    license?: string;
    app_id?: string;
    has_ui?: boolean;
    ui_dist_path?: string;
    ui_port?: number;
    api_port?: number;
    scenario_path?: string;
    category?: string;
    tags?: string[];
  };
}

/** Build stage platform result (from build/types.go PlatformResult) */
export interface BuildPlatformResult {
  platform: string;
  status: "pending" | "building" | "ready" | "failed" | "skipped";
  started_at?: string;
  completed_at?: string;
  artifact?: string;
  file_size?: number;
  error_log?: string[];
  skip_reason?: string;
}

/** Build stage details (from build/types.go Status) */
export interface BuildStageDetails {
  build_id?: string;
  scenario_name?: string;
  status?: string;
  framework?: string;
  template_type?: string;
  platforms?: string[];
  requested_platforms?: string[];
  platform_results?: Record<string, BuildPlatformResult>;
  output_path?: string;
  created_at?: string;
  completed_at?: string;
  build_log?: string[];
  error_log?: string[];
  artifacts?: Record<string, string>;
  metadata?: Record<string, unknown>;
}

/** SmokeTest stage details (from smoketest/types.go Status) */
export interface SmokeTestStageDetails {
  smoke_test_id?: string;
  scenario_name?: string;
  platform?: string;
  status?: string;
  artifact_path?: string;
  started_at?: string;
  completed_at?: string;
  logs?: string[];
  error?: string;
  telemetry_uploaded?: boolean;
  telemetry_upload_error?: string;
}

/** Distribution platform upload (from distribution/types.go PlatformUpload) */
export interface DistributionPlatformUpload {
  platform?: string;
  status?: string;
  local_path?: string;
  remote_key?: string;
  url?: string;
  size?: number;
  bytes_uploaded?: number;
  error?: string;
}

/** Distribution target status (from distribution/types.go TargetDistribution) */
export interface DistributionTargetStatus {
  target_name?: string;
  status?: string;
  started_at?: number;
  completed_at?: number;
  uploads?: Record<string, DistributionPlatformUpload>;
  error?: string;
}

/** Distribution stage details (from distribution/types.go DistributionStatus) */
export interface DistributionStageDetails {
  distribution_id?: string;
  scenario_name?: string;
  version?: string;
  status?: string;
  started_at?: number;
  completed_at?: number;
  targets?: Record<string, DistributionTargetStatus>;
  error?: string;
}

/** Union type for all possible stage details */
export type StageDetails =
  | BundleStageDetails
  | GenerateStageDetails
  | BuildStageDetails
  | SmokeTestStageDetails
  | DistributionStageDetails;

/** Verbose stage result includes details and logs */
export interface VerboseStageResult extends Omit<PipelineStageResult, "details"> {
  details?: StageDetails;
  logs?: string[];
}

/** Verbose pipeline status includes full stage details */
export interface VerbosePipelineStatus extends Omit<PipelineStatus, "stages"> {
  stages: Record<string, VerboseStageResult>;
}

// ==================== Pipeline API Functions ====================

export async function runPipeline(config: PipelineConfig): Promise<PipelineRunResponse> {
  const response = await fetch(buildUrl("/pipeline/run"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config)
  });
  await throwIfNotOk(response);
  return response.json();
}

/** Options for getPipelineStatus */
export interface GetPipelineStatusOptions {
  /** Request verbose output with stage Details and Logs (default: false) */
  verbose?: boolean;
}

/**
 * Get pipeline status by ID.
 * @param pipelineId - The pipeline ID
 * @param options - Options including verbose flag
 * @returns Pipeline status. When verbose=true, includes stage details and logs.
 */
export async function getPipelineStatus(
  pipelineId: string,
  options?: { verbose: true }
): Promise<VerbosePipelineStatus>;
export async function getPipelineStatus(
  pipelineId: string,
  options?: { verbose?: false } | undefined
): Promise<PipelineStatus>;
export async function getPipelineStatus(
  pipelineId: string,
  options?: GetPipelineStatusOptions
): Promise<PipelineStatus | VerbosePipelineStatus> {
  const params = new URLSearchParams();
  if (options?.verbose) {
    params.set("verbose", "true");
  }

  const queryString = params.toString();
  const url = buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}`) +
    (queryString ? `?${queryString}` : "");

  const response = await fetch(url);
  await throwIfNotOk(response);
  return response.json();
}

export async function resumePipeline(pipelineId: string): Promise<PipelineResumeResponse> {
  const response = await fetch(buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/resume`), {
    method: "POST"
  });
  await throwIfNotOk(response);
  return response.json();
}

export async function cancelPipeline(pipelineId: string): Promise<{ status: string; message?: string }> {
  const response = await fetch(buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/cancel`), {
    method: "POST"
  });
  await throwIfNotOk(response);
  return response.json();
}

export async function listPipelines(): Promise<{ pipelines: PipelineStatus[] }> {
  const response = await fetch(buildUrl("/pipelines"));
  await throwIfNotOk(response);
  return response.json();
}

/**
 * Runs preflight validation via the pipeline.
 * Uses stop_after_stage: "preflight" to run only bundle and preflight stages.
 * Returns the pipeline run response which can be polled for status.
 */
export async function runPreflightPipeline(
  scenarioName: string,
  config: Partial<PipelineConfig> = {}
): Promise<PipelineRunResponse> {
  return runPipeline({
    scenario_name: scenarioName,
    stop_after_stage: "preflight",
    ...config
  });
}

/**
 * Extracts the preflight result from a pipeline status.
 * Returns null if the preflight stage hasn't completed or failed.
 */
export function extractPreflightResult(status: PipelineStatus): BundlePreflightResponse | null {
  const preflightStage = status.stages?.preflight;
  if (!preflightStage || preflightStage.status !== "completed") {
    return null;
  }
  // The preflight stage stores the response in Details
  return preflightStage.details as BundlePreflightResponse | null;
}

// ==================== Scenario-Based Pipeline Management API ====================

/** Response from getting active pipeline */
export interface ActivePipelineResponse {
  pipeline: PipelineStatus | null;
  created: boolean;
}

/** Response from creating new pipeline */
export interface CreatePipelineResponse {
  pipeline: PipelineStatus;
  archived_id?: string;
}

/** Response from resetting pipeline */
export interface ResetPipelineResponse {
  archived_id?: string;
  cleared: boolean;
}

/** Response from getting pipeline history */
export interface PipelineHistoryResponse {
  pipelines: PipelineStatus[];
  total: number;
}

/** Options for getting active pipeline */
export interface GetActivePipelineOptions {
  /** If true, creates a new pipeline if none exists. Default: true */
  autoCreate?: boolean;
}

/**
 * Get the active pipeline for a scenario.
 * If autoCreate is true (default), creates a new pipeline if none exists.
 */
export async function getActivePipeline(
  scenarioName: string,
  options?: GetActivePipelineOptions
): Promise<ActivePipelineResponse> {
  const params = new URLSearchParams();
  if (options?.autoCreate === false) {
    params.set("auto_create", "false");
  }

  const queryString = params.toString();
  const url = buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/pipeline/active`) +
    (queryString ? `?${queryString}` : "");

  const response = await fetch(url);
  await throwIfNotOk(response);
  return response.json();
}

/**
 * Create a new active pipeline for a scenario.
 * Archives the current active pipeline if one exists.
 */
export async function createNewPipeline(
  scenarioName: string,
  config?: Partial<PipelineConfig>
): Promise<CreatePipelineResponse> {
  const response = await fetch(buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/pipeline`), {
    method: "POST",
    headers: config ? { "Content-Type": "application/json" } : undefined,
    body: config ? JSON.stringify(config) : undefined,
  });
  await throwIfNotOk(response);
  return response.json();
}

/**
 * Reset the active pipeline for a scenario.
 * Archives the current active pipeline and clears the active slot.
 */
export async function resetPipeline(scenarioName: string): Promise<ResetPipelineResponse> {
  const response = await fetch(buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/pipeline/reset`), {
    method: "POST",
  });
  await throwIfNotOk(response);
  return response.json();
}

/** Options for getting pipeline history */
export interface GetPipelineHistoryOptions {
  /** Maximum number of pipelines to return. Default: 10 */
  limit?: number;
}

/**
 * Get the history of pipelines for a scenario.
 */
export async function getPipelineHistory(
  scenarioName: string,
  options?: GetPipelineHistoryOptions
): Promise<PipelineHistoryResponse> {
  const params = new URLSearchParams();
  if (options?.limit && options.limit > 0) {
    params.set("limit", options.limit.toString());
  }

  const queryString = params.toString();
  const url = buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/pipeline/history`) +
    (queryString ? `?${queryString}` : "");

  const response = await fetch(url);
  await throwIfNotOk(response);
  return response.json();
}
