/**
 * Pipeline service - pure functions for pipeline state management.
 * Extracted from pipelineStore.ts for testability and reuse.
 */

import type { VerbosePipelineStatus } from "../lib/api";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import type {
  PipelineRunStatus,
  PipelineErrorInfo,
} from "../store/pipelineTypes";

// ============================================================================
// Constants
// ============================================================================

/** Polling interval in milliseconds */
export const POLL_INTERVAL_MS = 2000;

/** Terminal pipeline states that should stop polling */
export const TERMINAL_STATES = [
  StageStatus.COMPLETED,
  StageStatus.FAILED,
  StageStatus.CANCELLED,
] as const;

/** States that don't require polling (either haven't started or are done) */
export const NON_POLLING_STATES = [
  StageStatus.IDLE,
  StageStatus.COMPLETED,
  StageStatus.FAILED,
  StageStatus.CANCELLED,
] as const;

// ============================================================================
// Status Mapping
// ============================================================================

/**
 * Map a verbose pipeline status to a simplified run status for UI consumption.
 */
export function mapPipelineToRunStatus(
  status: StageStatus | undefined,
): PipelineRunStatus {
  if (status === undefined) return "idle";

  switch (status) {
    case StageStatus.IDLE:
      // Pipeline created but not started - ready for configuration
      return "idle";
    case StageStatus.PENDING:
    case StageStatus.RUNNING:
      return "running";
    case StageStatus.COMPLETED:
      return "completed";
    case StageStatus.FAILED:
      return "failed";
    case StageStatus.CANCELLED:
      return "cancelled";
    default:
      return "idle";
  }
}

/**
 * Check if a pipeline status is a terminal state.
 * Terminal states mean the pipeline has finished (can't be changed).
 */
export function isTerminalState(status: StageStatus | undefined): boolean {
  return (
    status === StageStatus.COMPLETED ||
    status === StageStatus.FAILED ||
    status === StageStatus.CANCELLED
  );
}

/**
 * Check if a pipeline is idle (created but not started).
 */
export function isIdleState(status: StageStatus | undefined): boolean {
  return status === StageStatus.IDLE;
}

/**
 * Check if a pipeline is actively running (needs polling).
 */
export function isActivelyRunning(status: StageStatus | undefined): boolean {
  return status === StageStatus.RUNNING || status === StageStatus.PENDING;
}

/**
 * Determine if polling should continue based on current status.
 * Don't poll for idle pipelines (haven't started) or terminal states (finished).
 */
export function shouldContinuePolling(
  status: VerbosePipelineStatus | null,
): boolean {
  if (!status) return false;
  return isActivelyRunning(status.status);
}

// ============================================================================
// Idempotency
// ============================================================================

/**
 * Generate a unique idempotency key for a pipeline request.
 * Format: `{scenario}:{stage}:{sessionId}:{timestamp}`
 */
export function generateRequestIdempotencyKey(
  scenarioName: string,
  stage: string,
  sessionId: string,
): string {
  return `${scenarioName}:${stage}:${sessionId}:${String(Date.now())}`;
}

// ============================================================================
// Error Handling
// ============================================================================

/**
 * Categories for pipeline errors to enable targeted recovery UI.
 */
export type PipelineErrorCategory =
  | "network"
  | "validation"
  | "permission"
  | "timeout"
  | "resource"
  | "unknown";

/**
 * Categorize a pipeline error for UI recovery guidance.
 */
export function categorizeError(error: unknown): PipelineErrorCategory {
  if (!error) return "unknown";

  const message =
    error instanceof Error
      ? error.message.toLowerCase()
      : typeof error === "string"
        ? error.toLowerCase()
        : "unknown error";

  if (
    message.includes("network") ||
    message.includes("fetch") ||
    message.includes("connect")
  ) {
    return "network";
  }
  if (
    message.includes("valid") ||
    message.includes("invalid") ||
    message.includes("required")
  ) {
    return "validation";
  }
  if (
    message.includes("permission") ||
    message.includes("denied") ||
    message.includes("unauthorized")
  ) {
    return "permission";
  }
  if (message.includes("timeout") || message.includes("timed out")) {
    return "timeout";
  }
  if (
    message.includes("resource") ||
    message.includes("memory") ||
    message.includes("disk")
  ) {
    return "resource";
  }

  return "unknown";
}

/**
 * Get recovery suggestions based on error category.
 */
export function getRecoverySuggestions(
  category: PipelineErrorCategory,
): string[] {
  switch (category) {
    case "network":
      return [
        "Check your internet connection",
        "Verify the API server is running",
        "Try again in a few moments",
      ];
    case "validation":
      return [
        "Review your form inputs",
        "Ensure all required fields are filled",
        "Check that file paths exist",
      ];
    case "permission":
      return [
        "Check file and directory permissions",
        "Verify you have write access to output directories",
        "Run with appropriate privileges if needed",
      ];
    case "timeout":
      return [
        "The operation took too long",
        "Try with fewer platforms selected",
        "Check for resource constraints on your system",
      ];
    case "resource":
      return [
        "Free up disk space",
        "Close other applications to free memory",
        "Try building for fewer platforms at once",
      ];
    default:
      return [
        "An unexpected error occurred",
        "Check the logs for more details",
        "Try running the operation again",
      ];
  }
}

/**
 * Create a structured error info object from an error.
 */
export function createPipelineErrorInfo(error: unknown): PipelineErrorInfo {
  const message =
    error instanceof Error
      ? error.message
      : typeof error === "string"
        ? error
        : "Unknown pipeline error";
  const category = categorizeError(error);
  const suggestions = getRecoverySuggestions(category);

  return {
    message,
    category,
    suggestions,
    raw: error,
  };
}

// ============================================================================
// Progress Calculation
// ============================================================================

/**
 * Calculate overall progress (0-1) from a verbose pipeline status.
 */
export function calculatePipelineProgress(
  status: VerbosePipelineStatus | null,
): number {
  if (!status) return 0;

  const { stageOrder, stages } = status;
  if (!stageOrder.length) return 0;

  const completed = stageOrder.filter(
    (s) =>
      stages[stageResultKey(s)]?.status === StageStatus.COMPLETED ||
      stages[stageResultKey(s)]?.status === StageStatus.SKIPPED,
  ).length;

  return completed / stageOrder.length;
}

function stageResultKey(stage: StageName): string {
  switch (stage) {
    case StageName.BUNDLE:
      return "bundle";
    case StageName.PREFLIGHT:
      return "preflight";
    case StageName.GENERATE:
      return "generate";
    case StageName.BUILD:
      return "build";
    case StageName.SMOKE_TEST:
      return "smoketest";
    case StageName.DEPLOY:
      return "deploy";
    case StageName.RESOLVE_DEPLOYMENT:
      return "resolve-deployment";
    default:
      return "";
  }
}

/**
 * Get the current active stage name from pipeline status.
 */
export function getCurrentStage(
  status: VerbosePipelineStatus | null,
): StageName | null {
  return status?.currentStage ?? null;
}

/**
 * Get the status of a specific stage.
 */
export function getStageStatus(
  status: VerbosePipelineStatus | null,
  stage: string,
): StageStatus {
  return status?.stages[stage]?.status ?? StageStatus.PENDING;
}

/**
 * Check if a pipeline can be resumed (stopped after a stage).
 */
export function canResumePipeline(
  status: VerbosePipelineStatus | null,
): boolean {
  if (!status) return false;
  return (
    status.status === StageStatus.COMPLETED && Boolean(status.stoppedAfterStage)
  );
}

/**
 * Get the stage where the pipeline stopped (for resume functionality).
 */
export function getStoppedAfterStage(
  status: VerbosePipelineStatus | null,
): StageName | null {
  return status?.stoppedAfterStage ?? null;
}
