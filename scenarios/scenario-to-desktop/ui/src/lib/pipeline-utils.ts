/**
 * Shared utility functions for pipeline status mapping.
 */

/** UI-friendly build status */
export type MappedBuildStatus = "building" | "ready" | "partial" | "failed";

/**
 * Maps raw pipeline status strings to UI-friendly build status.
 * Used by components that poll pipeline status and display results.
 */
export function mapPipelineStatus(status: string): MappedBuildStatus {
  switch (status) {
    case "pending":
    case "running":
      return "building";
    case "completed":
      return "ready";
    case "failed":
    case "cancelled":
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
