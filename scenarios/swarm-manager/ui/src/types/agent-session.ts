/**
 * Agent session domain types.
 */

import type {
  AgentSession as ProtoAgentSession,
  AgentSessionArtifact as ProtoAgentSessionArtifact,
  AgentSessionAttribution as ProtoAgentSessionAttribution,
  AgentSessionAttachment as ProtoAgentSessionAttachment,
  AgentSessionContextItem as ProtoAgentSessionContextItem,
  AgentSessionMessage as ProtoAgentSessionMessage,
  AgentSessionProposal as ProtoAgentSessionProposal,
} from "@vrooli/proto-types/swarm-manager/v1/domain/agent_session_pb";
import type { AgentSessionRunEvent as ProtoAgentSessionRunEvent } from "@vrooli/proto-types/swarm-manager/v1/api/agent_session_pb";
import type { ProtoMessage } from "./shared";

export type AgentSessionKind = "meta_orchestration" | "operating_mode_authoring" | "swarm_operations" | "workflow_authoring";

/** Session kinds that can be started by current UI controls. The retired kind
 * remains in AgentSessionKind solely so historical records can be displayed. */
export type CreatableAgentSessionKind = Exclude<AgentSessionKind, "operating_mode_authoring">;

export type AgentSessionStatus =
  | "draft"
  | "starting"
  | "running"
  | "waiting_for_user"
  | "proposal_ready"
  | "applying"
  | "complete"
  | "failed"
  | "canceled";

export type AgentSessionMessageRole = "user" | "assistant" | "system";

export type AgentSessionContextType =
  | "backlog_item"
  | "initiative"
  | "capture"
  | "execution"
  | "agent_activity"
  | "scenario"
  | "operating_mode"
  | "session"
  | "operations_briefing"
  | "startup_brief"
  | "plan_dependency_cycles"
  | "plan_eta"
  | "goal";

// Proposal kinds and statuses are service-owned, forward-compatible strings.
// A newer server must not make an older UI reject the entire sessions response.
export type AgentSessionProposalKind = string;
export type AgentSessionProposalStatus = string;

export interface AgentSessionProposalTarget {
  type: "backlog_item" | "initiative";
  ref: string;
  name: string;
}

export type AgentSessionArtifactType =
  | "backlog_item"
  | "initiative"
  | "operating_mode_proposal"
  | "operating_mode_definition"
  | "capture"
  | "file"
  | "agent_activity";

export type AgentSessionArtifactAction =
  | "proposed"
  | "created"
  | "updated"
  | "deleted"
  | "linked";

export type AgentSessionAttribution = Omit<
  ProtoMessage<ProtoAgentSessionAttribution>,
  "type" | "sessionKind"
> & {
  type: "operator" | "agent";
  sessionKind?: AgentSessionKind;
};

export type AgentSessionContextItem = Omit<ProtoMessage<ProtoAgentSessionContextItem>, "type"> & {
  type: AgentSessionContextType;
};

export interface AgentSessionContextRef {
  type: AgentSessionContextType;
  ref: string;
}

export type AgentSessionAttachment = ProtoMessage<ProtoAgentSessionAttachment> & {
  url?: string;
};

export type AgentSessionMessage = Omit<ProtoMessage<ProtoAgentSessionMessage>, "role" | "context"> & {
  role: AgentSessionMessageRole;
  context?: AgentSessionContextItem[];
};

export type AgentSessionProposal = Omit<
  ProtoMessage<ProtoAgentSessionProposal>,
  "kind" | "status" | "attribution"
> & {
  kind: AgentSessionProposalKind;
  status: AgentSessionProposalStatus;
  attribution?: AgentSessionAttribution;
};

export type AgentSessionArtifact = Omit<
  ProtoMessage<ProtoAgentSessionArtifact>,
  "artifactType" | "action" | "attribution"
> & {
  artifactType: AgentSessionArtifactType;
  action: AgentSessionArtifactAction;
  attribution?: AgentSessionAttribution;
};

export type AgentSession = Omit<
  ProtoMessage<ProtoAgentSession>,
  "kind" | "status" | "messages" | "proposals" | "artifacts" | "attachments" | "createdBy" | "proposalTarget"
> & {
  kind: AgentSessionKind;
  status: AgentSessionStatus;
  messages: AgentSessionMessage[];
  proposals: AgentSessionProposal[];
  artifacts: AgentSessionArtifact[];
  attachments?: AgentSessionAttachment[];
  createdBy?: AgentSessionAttribution;
  proposalTarget?: AgentSessionProposalTarget;
};

export type AgentSessionEventType =
  | "message"
  | "tool_call"
  | "tool_result"
  | "status"
  | "progress"
  | "error"
  | "log"
  | "metric"
  | "artifact"
  | "compaction"
  | "lifecycle"
  | "message_deleted"
  | "unknown";

export type AgentSessionRunEvent = Omit<ProtoMessage<ProtoAgentSessionRunEvent>, "eventType"> & {
  eventType: AgentSessionEventType;
};
