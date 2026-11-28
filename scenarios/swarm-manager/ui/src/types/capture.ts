/**
 * Capture domain types.
 */

/**
 * Lifecycle status of a capture.
 */
export type CaptureStatus = "classifying" | "failed";

/**
 * Categorized failure reasons for classification.
 * Each category implies a different recovery path for the user.
 *
 * - dependency_unavailable: agent-manager or prompt-manager not running (transient — retry later)
 * - classification_timeout: agent didn't finish within the allowed window (transient — retry)
 * - prompt_missing: classification skill not found in prompt catalog (config issue)
 * - agent_error: agent spawn call failed unexpectedly (check agent logs)
 * - internal_error: catch-all for unexpected server errors
 */
export type CaptureFailureReason =
  | "dependency_unavailable"
  | "classification_timeout"
  | "prompt_missing"
  | "agent_error"
  | "internal_error";

/**
 * A raw, unclassified thought from the user.
 */
export interface Capture {
  id: string;
  text: string;
  attachments: string[];
  created: string;
  status: CaptureStatus;
  failureReason?: CaptureFailureReason;
  workflowExecutionId?: string;
  workflowDefinitionDigest?: string;
  workflowEntityVersion?: string;
  note?: string;
}
