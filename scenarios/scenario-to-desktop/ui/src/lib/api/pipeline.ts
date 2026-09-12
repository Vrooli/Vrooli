import type {
  PipelineCancelResponse as ProtoPipelineCancelResponse,
  GenerateResponse as ProtoGenerateResponse,
  PipelineConfig as ProtoPipelineConfig,
  PipelineRunResponse as ProtoPipelineRunResponse,
  PipelineResumeResponse as ProtoPipelineResumeResponse,
  PipelineStatus as ProtoPipelineStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
import type { BundleStageDetails } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
import type { DeployStageDetails as ProtoDeployStageDetails } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
import type { ResourceDeploymentPlan as ProtoResourceDeploymentPlan } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
import type { SmokeTestStatusResponse as ProtoSmokeTestStatusResponse } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/operation_results_pb";
import type {
  BuildStatusResponse as ProtoBuildStatusResponse,
  PlatformBuildResult,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/operation_results_pb";
import type { PreflightResponse as ProtoPreflightResponse } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { PipelineConfigSchema } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { pipelineConnectClient } from "./connect";

// ==================== Pipeline API Types ====================

export type PipelineConfig = Exclude<
  MessageInitShape<typeof PipelineConfigSchema>,
  ProtoPipelineConfig
>;

export interface BuildProvenance {
  git_commit_hash: string;
  git_branch: string;
  git_dirty: boolean;
  built_at: string;
  version: string;
}

export type PipelineStatus = ProtoPipelineStatus;

export type PipelineRunResponse = ProtoPipelineRunResponse;

export type PipelineResumeResponse = ProtoPipelineResumeResponse;
export type { BundleStageDetails } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";

// ==================== Verbose Stage Details ====================

/** Bundle stage details (from bundle/types.go PackageResult) */
/** Generate stage details (from generation/types.go GenerateResponse) */
export type GenerateStageDetails = ProtoGenerateResponse;

export type BuildPlatformResult = PlatformBuildResult;
export type BuildStageDetails = ProtoBuildStatusResponse;

/** SmokeTest stage details (from smoketest/types.go Status) */
export type SmokeTestStageDetails = ProtoSmokeTestStatusResponse;

export type PreflightStageDetails = ProtoPreflightResponse;

export type DeployStageDetails = ProtoDeployStageDetails;
export type ResourceDeploymentPlan = ProtoResourceDeploymentPlan;

/** Union type for all possible stage details */
export type StageDetails =
  | BundleStageDetails
  | PreflightStageDetails
  | GenerateStageDetails
  | BuildStageDetails
  | SmokeTestStageDetails
  | DeployStageDetails
  | ResourceDeploymentPlan;

/** Pipeline status is the generated wire message; display projections live in domain/. */
export type VerbosePipelineStatus = ProtoPipelineStatus;

// ==================== Pipeline API Functions ====================

export async function runPipeline(
  config: PipelineConfig,
): Promise<PipelineRunResponse> {
  return pipelineConnectClient.run({
    config: create(PipelineConfigSchema, config),
  });
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
  options?: { verbose: true },
): Promise<VerbosePipelineStatus>;
export async function getPipelineStatus(
  pipelineId: string,
  options?: { verbose?: false },
): Promise<PipelineStatus>;
export async function getPipelineStatus(
  pipelineId: string,
  options?: GetPipelineStatusOptions,
): Promise<PipelineStatus> {
  // The Proto status always includes stage details and logs; `verbose` remains
  // only as a source-compatible UI display option, not a wire-level branch.
  void options;
  return pipelineConnectClient.get({ pipelineId });
}

export async function resumePipeline(
  pipelineId: string,
): Promise<PipelineResumeResponse> {
  return pipelineConnectClient.resume({ pipelineId });
}

export async function cancelPipeline(
  pipelineId: string,
): Promise<ProtoPipelineCancelResponse> {
  return pipelineConnectClient.cancel({ pipelineId });
}

export function stageDetailsFromProto(
  value: ProtoPipelineStatus["stages"][string]["details"],
): StageDetails | undefined {
  if (!value?.kind.case) return undefined;

  switch (value.kind.case) {
    case "bundle":
      return value.kind.value;
    case "generate":
      return value.kind.value;
    case "build":
      return value.kind.value;
    case "smokeTest":
      return value.kind.value;
    case "deploy":
      return value.kind.value;
    case "preflight":
      return value.kind.value;
    case "resolveDeployment":
      return value.kind.value;
  }
}

/**
 * Runs preflight validation via the pipeline.
 * Uses stop_after_stage: "preflight" to run only bundle and preflight stages.
 * Returns the pipeline run response which can be polled for status.
 */
export async function runPreflightPipeline(
  scenarioName: string,
  config: Partial<PipelineConfig> = {},
): Promise<PipelineRunResponse> {
  return runPipeline({
    scenarioName,
    stopAfterStage: StageName.PREFLIGHT,
    ...config,
  });
}

/**
 * Extracts the preflight result from a pipeline status.
 * Returns null if the preflight stage hasn't completed or failed.
 */
export function extractPreflightResult(
  status: PipelineStatus,
): PreflightStageDetails | null {
  const preflightStage = status.stages.preflight;
  if (!preflightStage || preflightStage.status !== StageStatus.COMPLETED) {
    return null;
  }
  return preflightStage.details?.kind.case === "preflight"
    ? preflightStage.details.kind.value
    : null;
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
  archivedPipelineId?: string;
}

/** Response from resetting pipeline */
export interface ResetPipelineResponse {
  archivedPipelineId?: string;
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
  options?: GetActivePipelineOptions,
): Promise<ActivePipelineResponse> {
  const response = await pipelineConnectClient.getActive({
    scenarioName,
    autoCreate: options?.autoCreate !== false,
  });
  return {
    pipeline: response.pipeline ?? null,
    created: response.created,
  };
}

/**
 * Create a new active pipeline for a scenario.
 * Archives the current active pipeline if one exists.
 */
export async function createNewPipeline(
  scenarioName: string,
  config?: Partial<PipelineConfig>,
): Promise<CreatePipelineResponse> {
  const response = await pipelineConnectClient.createActive({
    scenarioName,
    config: config
      ? create(PipelineConfigSchema, { scenarioName, ...config })
      : undefined,
  });
  if (!response.pipeline) {
    throw new Error("CreateActive returned no pipeline");
  }
  return {
    pipeline: response.pipeline,
    archivedPipelineId: response.archivedPipelineId,
  };
}

/**
 * Reset the active pipeline for a scenario.
 * Archives the current active pipeline and clears the active slot.
 */
export async function resetPipeline(
  scenarioName: string,
): Promise<ResetPipelineResponse> {
  const response = await pipelineConnectClient.resetActive({ scenarioName });
  return {
    archivedPipelineId: response.archivedPipelineId,
    cleared: response.cleared,
  };
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
  options?: GetPipelineHistoryOptions,
): Promise<PipelineHistoryResponse> {
  const response = await pipelineConnectClient.getHistory({
    scenarioName,
    limit: options?.limit,
  });
  return {
    pipelines: response.pipelines,
    total: response.total,
  };
}

/** Response from starting the active pipeline */
export interface StartActivePipelineResponse {
  pipeline: PipelineStatus;
  message?: string;
}

/**
 * Start the active pipeline for a scenario with optional config overrides.
 * This is the correct way to run stages - it uses the existing active pipeline
 * rather than creating orphaned new ones.
 *
 * @param scenarioName The scenario to start the pipeline for
 * @param config Optional config overrides (e.g., stop_after_stage)
 */
export async function startActivePipeline(
  scenarioName: string,
  config?: Partial<PipelineConfig>,
): Promise<StartActivePipelineResponse> {
  const response = await pipelineConnectClient.startActive({
    scenarioName,
    configOverrides: config
      ? create(PipelineConfigSchema, { scenarioName, ...config })
      : undefined,
  });
  if (!response.pipeline) {
    throw new Error("StartActive returned no pipeline");
  }
  return {
    pipeline: response.pipeline,
    message: response.message,
  };
}
