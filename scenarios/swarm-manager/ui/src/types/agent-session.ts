/**
 * Agent session domain types.
 */

import type {
  AgentSession as ProtoAgentSession,
  AgentSessionArtifact as ProtoAgentSessionArtifact,
  AgentSessionAttribution as ProtoAgentSessionAttribution,
  AgentSessionMessage as ProtoAgentSessionMessage,
  AgentSessionProposal as ProtoAgentSessionProposal,
} from "@vrooli/proto-types/swarm-manager/v1/domain/agent_session_pb";
import type { ProtoMessage } from "./shared";

export type AgentSessionKind = "meta_orchestration" | "operating_mode_authoring";

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

export type AgentSessionProposalKind =
  | "backlog_batch_import"
  | "operating_mode_draft"
  | "operating_mode_implementation_plan";

export type AgentSessionProposalStatus =
  | "draft"
  | "ready"
  | "applied"
  | "rejected"
  | "superseded"
  | "failed";

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

export type AgentSessionMessage = Omit<ProtoMessage<ProtoAgentSessionMessage>, "role"> & {
  role: AgentSessionMessageRole;
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
  "kind" | "status" | "messages" | "proposals" | "artifacts" | "createdBy"
> & {
  kind: AgentSessionKind;
  status: AgentSessionStatus;
  messages: AgentSessionMessage[];
  proposals: AgentSessionProposal[];
  artifacts: AgentSessionArtifact[];
  createdBy?: AgentSessionAttribution;
};
