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
  | "clarify";

export type AgentActivityInteractionType = "spawn" | "continue";
export type AgentActivityOwnerType = "backlog" | "capture" | "scenario";

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
