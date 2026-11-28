/**
 * Shared utility functions for pipeline status mapping.
 */

import { StageStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

/** UI-friendly build status */
export type MappedBuildStatus = "building" | "ready" | "partial" | "failed";

/**
 * Maps generated pipeline statuses to UI-friendly build status.
 * Used by components that poll pipeline status and display results.
 */
export function mapPipelineStatus(status: StageStatus): MappedBuildStatus {
  switch (status) {
    case StageStatus.PENDING:
    case StageStatus.RUNNING:
      return "building";
    case StageStatus.COMPLETED:
      return "ready";
    case StageStatus.FAILED:
    case StageStatus.CANCELLED:
      return "failed";
    default:
      return "building";
  }
}

// Session ID is generated once per page load to scope idempotency keys
let _sessionId: string | null = null;

/**
 * Resets the session ID, forcing new idempotency keys for future requests.
 * Useful when the user explicitly wants to retry a failed operation.
 */
export function resetSessionId(): void {
  _sessionId = null;
}
