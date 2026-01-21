/**
 * Pure domain functions for build progress calculation and status mapping.
 * These functions have no side effects and can be tested in isolation.
 *
 * Responsibility: Build status domain logic
 * - Progress calculation from build logs
 * - Stage completion detection
 * - Build status interpretation
 */

import type {
  VerbosePipelineStatus,
  BuildStageDetails,
  GenerateStageDetails,
  BundleStageDetails,
  BundlePreflightResponse,
  SmokeTestStageDetails,
  DistributionStageDetails,
  BuildStatus as BuildStatusType,
  PipelineStatus,
} from "../lib/api";
import { mapPipelineStatus } from "../lib/pipeline-utils";

// ============================================================================
// Build Stage Definitions
// ============================================================================

/**
 * Definition of a build stage for progress tracking.
 * Separates the domain knowledge of what stages exist from UI rendering.
 */
export interface BuildStageDefinition {
  /** Stage identifier */
  name: string;
  /** Keywords that indicate this stage has started/completed */
  keywords: string[];
  /** Progress percentage when this stage is reached */
  progress: number;
}

/**
 * Canonical list of build stages with detection keywords.
 * This is domain knowledge about the build process that should not be scattered in UI code.
 */
export const BUILD_STAGES: BuildStageDefinition[] = [
  {
    name: "Template Generation",
    keywords: ["Generating", "Creating"],
    progress: 25,
  },
  {
    name: "Installing Dependencies",
    keywords: ["npm install", "Installing"],
    progress: 50,
  },
  {
    name: "Compiling TypeScript",
    keywords: ["compiling", "building"],
    progress: 75,
  },
  {
    name: "Packaging Application",
    keywords: ["packaging", "electron-builder"],
    progress: 90,
  },
];

// ============================================================================
// Progress Calculation
// ============================================================================

/**
 * Checks if a log array contains any of the specified keywords.
 */
function logsContainKeyword(logs: string[], keywords: string[]): boolean {
  return keywords.some((keyword) =>
    logs.some((log) => log.includes(keyword))
  );
}

/**
 * Determines if a build stage has been reached based on log content.
 */
export function isStageReached(logs: string[], stage: BuildStageDefinition): boolean {
  return logsContainKeyword(logs, stage.keywords);
}

/**
 * Calculate build progress based on build log content.
 * Returns a number 0-100 representing completion percentage.
 *
 * ASSUMPTION: Build logs are appended in order; earlier stages appear before later ones.
 * This heuristic approach works because build tools (npm, tsc, electron-builder) emit
 * predictable log patterns.
 */
export function calculateBuildProgress(status: BuildStatusType | null | undefined): number {
  if (!status) return 0;
  if (status.status === "ready") return 100;
  if (status.status === "failed") return 0;

  const logs = status.build_log ?? [];
  if (logs.length === 0) return 10; // Just started, baseline progress

  // Find the highest progress stage that has been reached
  let progress = 10;
  for (const stage of BUILD_STAGES) {
    if (isStageReached(logs, stage)) {
      progress = stage.progress;
    }
  }

  return progress;
}

/**
 * Get the completion status of each build stage.
 * Returns an array matching BUILD_STAGES with completion and active state.
 */
export interface BuildStageStatus {
  stage: BuildStageDefinition;
  completed: boolean;
  active: boolean;
}

export function getBuildStageStatuses(logs: string[], currentProgress: number): BuildStageStatus[] {
  return BUILD_STAGES.map((stage) => {
    const completed = isStageReached(logs, stage);
    // A stage is active if we're in its progress range (progress - 25 to progress)
    const active = currentProgress >= stage.progress - 25 && currentProgress < stage.progress;
    return { stage, completed, active };
  });
}

// ============================================================================
// Pipeline Status Transformation
// ============================================================================

/**
 * Extracted stage results from a verbose pipeline status.
 * Separates the data transformation logic from store state management.
 */
export interface ExtractedStageResults {
  bundleResult: BundleStageDetails | null;
  preflightResult: BundlePreflightResponse | null;
  generateResult: GenerateStageDetails | null;
  buildResult: BuildStageDetails | null;
  smokeTestResult: SmokeTestStageDetails | null;
  distributionResult: DistributionStageDetails | null;
  stageLogs: Record<string, string[]>;
}

/**
 * Extract stage results from a verbose pipeline status.
 * This is pure data transformation - no side effects.
 *
 * ASSUMPTION: Stage details types match the expected TypeScript interfaces.
 * Backend schema changes would require updating these casts.
 */
export function extractStageResults(status: VerbosePipelineStatus): ExtractedStageResults {
  const stages = status.stages ?? {};
  const logs: Record<string, string[]> = {};

  const results: ExtractedStageResults = {
    bundleResult: null,
    preflightResult: null,
    generateResult: null,
    buildResult: null,
    smokeTestResult: null,
    distributionResult: null,
    stageLogs: {},
  };

  // Extract stage details
  if (stages.bundle?.details) {
    results.bundleResult = stages.bundle.details as BundleStageDetails;
  }
  if (stages.preflight?.details) {
    results.preflightResult = stages.preflight.details as BundlePreflightResponse;
  }
  if (stages.generate?.details) {
    results.generateResult = stages.generate.details as GenerateStageDetails;
  }
  if (stages.build?.details) {
    results.buildResult = stages.build.details as BuildStageDetails;
  }
  if (stages.smoketest?.details) {
    results.smokeTestResult = stages.smoketest.details as SmokeTestStageDetails;
  }
  if (stages.distribution?.details) {
    results.distributionResult = stages.distribution.details as DistributionStageDetails;
  }

  // Extract logs from all stages
  for (const [stageName, stage] of Object.entries(stages)) {
    if (stage?.logs?.length) {
      logs[stageName] = stage.logs;
    }
  }
  results.stageLogs = logs;

  return results;
}

/**
 * Convert a VerbosePipelineStatus to a UI-friendly BuildStatus.
 * Encapsulates the complex mapping logic that was scattered in BuildStatus.tsx.
 */
export function pipelineStatusToBuildStatus(
  pipelineStatus: VerbosePipelineStatus | null
): BuildStatusType | null {
  if (!pipelineStatus) return null;

  const generateDetails = pipelineStatus.stages?.generate?.details as GenerateStageDetails | undefined;
  const buildDetails = pipelineStatus.stages?.build?.details as BuildStageDetails | undefined;

  // Collect logs from all stages
  const allLogs: string[] = [];
  if (pipelineStatus.stages) {
    for (const stage of Object.values(pipelineStatus.stages)) {
      if (stage?.logs) {
        allLogs.push(...stage.logs);
      }
    }
  }
  // Also include build_log from build details
  if (buildDetails?.build_log) {
    allLogs.push(...buildDetails.build_log);
  }

  return {
    status: mapPipelineStatus(pipelineStatus.status),
    build_id: pipelineStatus.pipeline_id,
    scenario_name: pipelineStatus.scenario_name,
    template_type: buildDetails?.template_type || generateDetails?.detected_metadata?.name || "basic",
    framework: buildDetails?.framework || "electron",
    platforms: buildDetails?.platforms || pipelineStatus.config?.platforms || [],
    output_path: generateDetails?.desktop_path || buildDetails?.output_path || "",
    build_log: allLogs,
    error_log: buildDetails?.error_log,
    created_at: pipelineStatus.started_at
      ? new Date(pipelineStatus.started_at * 1000).toISOString()
      : new Date().toISOString(),
  };
}

// ============================================================================
// UI Status Mapping
// ============================================================================

/** Maps pipeline status to legacy UI-friendly status */
export type UiStatus = "building" | "ready" | "partial" | "failed";

/** UI-friendly build status derived from pipeline status */
export interface UiBuildStatus {
  status: UiStatus;
  output_path?: string;
  pipeline_id: string;
  scenario_name: string;
}

/**
 * Map a PipelineStatus to a simplified UI-friendly status.
 * Used by App.tsx for the bottom action bar display.
 */
export function mapPipelineToUiStatus(pipeline: PipelineStatus): UiBuildStatus {
  let status: UiStatus;
  switch (pipeline.status) {
    case "pending":
    case "running":
      status = "building";
      break;
    case "completed":
      status = "ready";
      break;
    case "failed":
    case "cancelled":
      status = "failed";
      break;
    default:
      status = "building";
  }

  // Extract output_path from generate stage details if available
  const generateDetails = pipeline.stages?.generate?.details as GenerateStageDetails | undefined;
  const outputPath = generateDetails?.desktop_path;

  return {
    status,
    output_path: outputPath,
    pipeline_id: pipeline.pipeline_id,
    scenario_name: pipeline.scenario_name,
  };
}
