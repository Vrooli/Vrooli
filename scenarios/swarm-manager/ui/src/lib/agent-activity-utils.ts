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
 * Statuses where an agent is either doing work or awaiting human input.
 * When the latest activity for a backlog item is in one of these states,
 * the user cannot usefully interact with the item until it resolves.
 */
export const ACTIVE_AGENT_ACTIVITY_STATUSES: ReadonlySet<AgentActivityStatus> = new Set<AgentActivityStatus>([
  "pending",
  "starting",
  "running",
  "needs_review",
]);

/** True when the activity is still doing work or awaiting user input. */
export function isAgentActivityActive(status: AgentActivityStatus | undefined | null): boolean {
  if (!status) return false;
  return ACTIVE_AGENT_ACTIVITY_STATUSES.has(status);
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
