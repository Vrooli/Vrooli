/**
 * Agent domain types.
 */

import type { AgentManagerStatusResponse as ProtoAgentManagerStatusResponse } from "@vrooli/proto-types/swarm-manager/v1/api/agent_manager_pb";
import type { AgentActivity as ProtoAgentActivity } from "@vrooli/proto-types/swarm-manager/v1/domain/agent_activity_pb";
import type { ProtoMessage } from "./shared";

export type AgentManagerStatus = ProtoMessage<ProtoAgentManagerStatusResponse>;

export type AgentRunStatus =
  | "pending"
  | "starting"
  | "running"
  | "needs_review"
  | "complete"
  | "failed"
  | "cancelled"
  | "unspecified";

export interface AgentRunState {
  runId: string;
  taskId?: string;
  status: AgentRunStatus;
  startedAt?: string;
  finishedAt?: string;
  errorMessage?: string;
  durationSeconds?: number;
  active: boolean;
  tokensUsed?: number;
  turnsUsed?: number;
  costEstimate?: number;
  changedFiles?: number;
  contextTokens?: number;
}

export type AgentActivityStatus =
  | "pending"
  | "starting"
  | "running"
  | "needs_review"
  | "complete"
  | "failed"
  | "cancelled"
  | "unspecified";


/**
 * All known agent activity purposes.
 *
 * IMPORTANT: When adding a new purpose, add it here. The `AGENT_ACTIVITY_PURPOSES`
 * array below is derived from this type and used for runtime validation, so the
 * type system will enforce that both stay in sync.
 */
export type AgentActivityPurpose =
  | "initialize"
  | "workshop"
  | "finalize"
  | "research"
  | "process"
  | "fixup"
  | "followup"
  | "spec_sync"
  | "classify"
  | "clarify"
  | "review"
  | "feedback"
  | "feedback_continue"
  | "initiative_review"
  | "meta_orchestration"
  | "operating_mode_authoring";

/**
 * Exhaustive array of all AgentActivityPurpose values.
 * TypeScript enforces this array matches the union type via `satisfies`.
 * Used at runtime for validation sets — keeps type and runtime in sync.
 */
export const AGENT_ACTIVITY_PURPOSES = [
  "initialize",
  "workshop",
  "finalize",
  "research",
  "process",
  "fixup",
  "followup",
  "spec_sync",
  "classify",
  "clarify",
  "review",
  "feedback",
  "feedback_continue",
  "initiative_review",
  "meta_orchestration",
  "operating_mode_authoring",
] as const satisfies readonly AgentActivityPurpose[];

export type AgentActivityInteractionType = "spawn" | "continue";
/** Exhaustive array of all AgentActivityInteractionType values. */
export const AGENT_ACTIVITY_INTERACTION_TYPES = ["spawn", "continue"] as const satisfies readonly AgentActivityInteractionType[];

export type AgentActivityOwnerType = "backlog" | "capture" | "scenario" | "initiative" | "session";
/** Exhaustive array of all AgentActivityOwnerType values. */
export const AGENT_ACTIVITY_OWNER_TYPES = ["backlog", "capture", "scenario", "initiative", "session"] as const satisfies readonly AgentActivityOwnerType[];

export type AgentActivity = Omit<
  ProtoMessage<ProtoAgentActivity>,
  "ownerType" | "purpose" | "interactionType" | "status" | "metadata"
> & {
  ownerType: AgentActivityOwnerType;
  purpose: AgentActivityPurpose;
  interactionType: AgentActivityInteractionType;
  status: AgentActivityStatus;
  metadata?: Record<string, string>;
};
