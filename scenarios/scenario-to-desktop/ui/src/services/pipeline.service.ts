/**
 * Pipeline service - pure functions for pipeline state management.
 * Extracted from pipelineStore.ts for testability and reuse.
 */

import type { VerbosePipelineStatus } from "../lib/api";
import type { PipelineRunStatus, PipelineErrorInfo } from "../store/pipelineStore";

// ============================================================================
// Constants
// ============================================================================

/** Polling interval in milliseconds */
export const POLL_INTERVAL_MS = 2000;

/** Terminal pipeline states that should stop polling */
export const TERMINAL_STATES = ["completed", "failed", "cancelled"] as const;

export type TerminalState = (typeof TERMINAL_STATES)[number];

/** States that don't require polling (either haven't started or are done) */
export const NON_POLLING_STATES = ["idle", "completed", "failed", "cancelled"] as const;

// ============================================================================
// Status Mapping
// ============================================================================

/**
 * Map a verbose pipeline status to a simplified run status for UI consumption.
 */
export function mapPipelineToRunStatus(status: string | undefined | null): PipelineRunStatus {
  if (!status) return "idle";

  switch (status) {
    case "idle":
      // Pipeline created but not started - ready for configuration
      return "idle";
    case "pending":
    case "running":
      return "running";
    case "completed":
      return "completed";
    case "failed":
      return "failed";
    case "cancelled":
      return "cancelled";
    default:
      return "idle";
  }
}

/**
 * Check if a pipeline status is a terminal state.
 * Terminal states mean the pipeline has finished (can't be changed).
 */
export function isTerminalState(status: string | undefined | null): boolean {
  if (!status) return false;
  return TERMINAL_STATES.includes(status as TerminalState);
}

/**
 * Check if a pipeline is idle (created but not started).
 */
export function isIdleState(status: string | undefined | null): boolean {
  return status === "idle";
}

/**
 * Check if a pipeline is actively running (needs polling).
 */
export function isActivelyRunning(status: string | undefined | null): boolean {
  return status === "running" || status === "pending";
}

/**
 * Determine if polling should continue based on current status.
 * Don't poll for idle pipelines (haven't started) or terminal states (finished).
 */
export function shouldContinuePolling(status: VerbosePipelineStatus | null): boolean {
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
  sessionId: string
): string {
  return `${scenarioName}:${stage}:${sessionId}:${Date.now()}`;
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

  const message = error instanceof Error ? error.message.toLowerCase() : String(error).toLowerCase();

  if (message.includes("network") || message.includes("fetch") || message.includes("connect")) {
    return "network";
  }
  if (message.includes("valid") || message.includes("invalid") || message.includes("required")) {
    return "validation";
  }
  if (message.includes("permission") || message.includes("denied") || message.includes("unauthorized")) {
    return "permission";
  }
  if (message.includes("timeout") || message.includes("timed out")) {
    return "timeout";
  }
  if (message.includes("resource") || message.includes("memory") || message.includes("disk")) {
    return "resource";
  }

  return "unknown";
}

/**
 * Get recovery suggestions based on error category.
 */
export function getRecoverySuggestions(category: PipelineErrorCategory): string[] {
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
  const message = error instanceof Error ? error.message : String(error);
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
export function calculatePipelineProgress(status: VerbosePipelineStatus | null): number {
  if (!status) return 0;

  const { stage_order, stages } = status;
  if (!stage_order?.length) return 0;

  const completed = stage_order.filter(
    (s) => stages?.[s]?.status === "completed" || stages?.[s]?.status === "skipped"
  ).length;

  return completed / stage_order.length;
}

/**
 * Get the current active stage name from pipeline status.
 */
export function getCurrentStage(status: VerbosePipelineStatus | null): string | null {
  return status?.current_stage ?? null;
}

/**
 * Get the status of a specific stage.
 */
export function getStageStatus(
  status: VerbosePipelineStatus | null,
  stage: string
): string {
  return status?.stages?.[stage]?.status ?? "pending";
}

/**
 * Check if a pipeline can be resumed (stopped after a stage).
 */
export function canResumePipeline(status: VerbosePipelineStatus | null): boolean {
  if (!status) return false;
  return status.status === "completed" && Boolean(status.stopped_after_stage);
}

/**
 * Get the stage where the pipeline stopped (for resume functionality).
 */
export function getStoppedAfterStage(status: VerbosePipelineStatus | null): string | null {
  return status?.stopped_after_stage ?? null;
}
