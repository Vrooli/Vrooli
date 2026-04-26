/**
 * Agent Activity Utilities
 *
 * Shared helpers for classifying and labeling agent activity on a backlog
 * item. Centralizes the "is an agent currently active" predicate and the
 * purpose → human label mapping, so the dependency chip, backlog card,
 * and details page all agree on the semantics.
 */

import type { AgentActivityPurpose, AgentActivityStatus } from "../types";

/**
 * Statuses where an agent is still actively executing work. These are the only
 * states that should render live-run affordances like a pulse or Stop button.
 */
export const EXECUTING_AGENT_ACTIVITY_STATUSES: ReadonlySet<AgentActivityStatus> = new Set<AgentActivityStatus>([
  "pending",
  "starting",
  "running",
]);

/**
 * Statuses where the item is still blocked by an agent lifecycle, even if the
 * agent is no longer actively executing. `needs_review` remains here because
 * the user still has to resolve the run before the item is truly clear.
 */
export const BLOCKING_AGENT_ACTIVITY_STATUSES: ReadonlySet<AgentActivityStatus> = new Set<AgentActivityStatus>([
  ...EXECUTING_AGENT_ACTIVITY_STATUSES,
  "needs_review",
]);

/** True when the activity is still actively executing work. */
export function isAgentActivityExecuting(status: AgentActivityStatus | undefined | null): boolean {
  if (!status) return false;
  return EXECUTING_AGENT_ACTIVITY_STATUSES.has(status);
}

/** True when the activity still blocks the item, including awaiting review. */
export function isAgentActivityBlocking(status: AgentActivityStatus | undefined | null): boolean {
  if (!status) return false;
  return BLOCKING_AGENT_ACTIVITY_STATUSES.has(status);
}

/**
 * Human label for an agent activity purpose. Used in chips, badges, and
 * running-state CTAs. Keep labels short (one or two words) and in present
 * continuous form ("Workshopping") to read naturally in a chip.
 */
const PURPOSE_LABELS: Record<AgentActivityPurpose, string> = {
  initialize: "Initializing",
  workshop: "Workshopping",
  finalize: "Finalizing",
  research: "Researching",
  process: "Processing",
  fixup: "Fixing up",
  followup: "Following up",
  spec_sync: "Syncing spec",
  classify: "Classifying",
  clarify: "Clarifying",
  review: "Reviewing",
};

export function getAgentActivityLabel(purpose: AgentActivityPurpose | undefined | null): string {
  if (!purpose) return "Agent running";
  return PURPOSE_LABELS[purpose] ?? "Agent running";
}

/**
 * Visual tone the dependency chip should use for an active agent activity.
 * `needs_review` is semantically "your turn" (cyan, no pulse); all other
 * active statuses are "busy, don't touch" (amber, with pulse).
 */
export type AgentActivityTone = "busy" | "needs-review";

export function getAgentActivityTone(status: AgentActivityStatus): AgentActivityTone {
  return status === "needs_review" ? "needs-review" : "busy";
}
