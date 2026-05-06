import type {
  AgentSession,
  AgentSessionArtifact,
  AgentSessionAttribution,
  AgentSessionMessage,
  AgentSessionProposal,
} from "@vrooli/proto-types/swarm-manager/v1/domain/agent_session_pb";
import {
  ApplyAgentSessionProposalResponseSchema,
  CancelAgentSessionResponseSchema,
  ContinueAgentSessionResponseSchema,
  CreateAgentSessionResponseSchema,
  DeleteAgentSessionResponseSchema,
  GetAgentSessionResponseSchema,
  GetArtifactsByEntityResponseSchema,
  ListAgentSessionArtifactsResponseSchema,
  ListAgentSessionsResponseSchema,
  RefreshAgentSessionResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/agent_session_pb";
import type {
  AgentSession as AgentSessionDomain,
  AgentSessionArtifact as AgentSessionArtifactDomain,
  AgentSessionAttribution as AgentSessionAttributionDomain,
  AgentSessionKind,
  AgentSessionMessage as AgentSessionMessageDomain,
  AgentSessionProposal as AgentSessionProposalDomain,
  AgentSessionStatus,
} from "../../types";
import { createProtoSchema } from "./shared";

const sessionKinds = new Set<string>(["meta_orchestration", "operating_mode_authoring"]);
const sessionStatuses = new Set<string>([
  "draft",
  "starting",
  "running",
  "waiting_for_user",
  "proposal_ready",
  "applying",
  "complete",
  "failed",
  "canceled",
]);
const messageRoles = new Set<string>(["user", "assistant", "system"]);
const proposalKinds = new Set<string>([
  "backlog_batch_import",
  "operating_mode_draft",
  "operating_mode_implementation_plan",
]);
const proposalStatuses = new Set<string>([
  "draft",
  "ready",
  "applied",
  "rejected",
  "superseded",
  "failed",
]);
const artifactTypes = new Set<string>([
  "backlog_item",
  "initiative",
  "operating_mode_proposal",
  "operating_mode_definition",
  "capture",
  "file",
  "agent_activity",
]);
const artifactActions = new Set<string>(["proposed", "created", "updated", "deleted", "linked"]);

export const listAgentSessionsResponseSchema = createProtoSchema(
  ListAgentSessionsResponseSchema,
  "agent sessions list"
);
export const getAgentSessionResponseSchema = createProtoSchema(
  GetAgentSessionResponseSchema,
  "agent session"
);
export const createAgentSessionResponseSchema = createProtoSchema(
  CreateAgentSessionResponseSchema,
  "agent session create"
);
export const continueAgentSessionResponseSchema = createProtoSchema(
  ContinueAgentSessionResponseSchema,
  "agent session continue"
);
export const refreshAgentSessionResponseSchema = createProtoSchema(
  RefreshAgentSessionResponseSchema,
  "agent session refresh"
);
export const cancelAgentSessionResponseSchema = createProtoSchema(
  CancelAgentSessionResponseSchema,
  "agent session cancel"
);
export const deleteAgentSessionResponseSchema = createProtoSchema(
  DeleteAgentSessionResponseSchema,
  "agent session delete"
);
export const applyAgentSessionProposalResponseSchema = createProtoSchema(
  ApplyAgentSessionProposalResponseSchema,
  "agent session proposal apply"
);
export const listAgentSessionArtifactsResponseSchema = createProtoSchema(
  ListAgentSessionArtifactsResponseSchema,
  "agent session artifacts"
);
export const getArtifactsByEntityResponseSchema = createProtoSchema(
  GetArtifactsByEntityResponseSchema,
  "agent session artifacts by entity"
);

export function mapProtoAgentSession(protoSession: AgentSession): AgentSessionDomain {
  return {
    id: protoSession.id ?? "",
    title: protoSession.title ?? "",
    kind: asSessionKind(protoSession.kind),
    status: asSessionStatus(protoSession.status),
    skillId: protoSession.skillId ?? "",
    ...(protoSession.taskId ? { taskId: protoSession.taskId } : {}),
    ...(protoSession.runId ? { runId: protoSession.runId } : {}),
    ...(protoSession.profileKey ? { profileKey: protoSession.profileKey } : {}),
    ...(protoSession.failureReason ? { failureReason: protoSession.failureReason } : {}),
    createdAt: protoSession.createdAt ?? "",
    updatedAt: protoSession.updatedAt ?? "",
    messages: protoSession.messages?.map(mapProtoAgentSessionMessage) ?? [],
    proposals: protoSession.proposals?.map(mapProtoAgentSessionProposal) ?? [],
    artifacts: protoSession.artifacts?.map(mapProtoAgentSessionArtifact) ?? [],
    ...(protoSession.createdBy ? { createdBy: mapProtoAgentSessionAttribution(protoSession.createdBy) } : {}),
  };
}

export function mapProtoAgentSessionMessage(message: AgentSessionMessage): AgentSessionMessageDomain {
  return {
    id: message.id ?? "",
    role: messageRoles.has(message.role) ? (message.role as AgentSessionMessageDomain["role"]) : "assistant",
    content: message.content ?? "",
    createdAt: message.createdAt ?? "",
    attachmentIds: message.attachmentIds ?? [],
  };
}

export function mapProtoAgentSessionProposal(proposal: AgentSessionProposal): AgentSessionProposalDomain {
  return {
    id: proposal.id ?? "",
    kind: proposalKinds.has(proposal.kind)
      ? (proposal.kind as AgentSessionProposalDomain["kind"])
      : "backlog_batch_import",
    status: proposalStatuses.has(proposal.status)
      ? (proposal.status as AgentSessionProposalDomain["status"])
      : "draft",
    summary: proposal.summary ?? "",
    payloadJson: proposal.payloadJson ?? "",
    createdAt: proposal.createdAt ?? "",
    updatedAt: proposal.updatedAt ?? "",
    ...(proposal.attribution ? { attribution: mapProtoAgentSessionAttribution(proposal.attribution) } : {}),
  };
}

export function mapProtoAgentSessionArtifact(artifact: AgentSessionArtifact): AgentSessionArtifactDomain {
  return {
    id: artifact.id ?? "",
    sessionId: artifact.sessionId ?? "",
    artifactType: artifactTypes.has(artifact.artifactType)
      ? (artifact.artifactType as AgentSessionArtifactDomain["artifactType"])
      : "file",
    action: artifactActions.has(artifact.action)
      ? (artifact.action as AgentSessionArtifactDomain["action"])
      : "linked",
    entityRef: artifact.entityRef ?? "",
    ...(artifact.title ? { title: artifact.title } : {}),
    ...(artifact.proposalId ? { proposalId: artifact.proposalId } : {}),
    ...(artifact.activityId ? { activityId: artifact.activityId } : {}),
    ...(artifact.runId ? { runId: artifact.runId } : {}),
    ...(artifact.mutationSource ? { mutationSource: artifact.mutationSource } : {}),
    ...(artifact.attribution ? { attribution: mapProtoAgentSessionAttribution(artifact.attribution) } : {}),
    createdAt: artifact.createdAt ?? "",
  };
}

export function mapProtoAgentSessionAttribution(
  attribution: AgentSessionAttribution
): AgentSessionAttributionDomain {
  const type = attribution.type === "agent" ? "agent" : "operator";
  return {
    type,
    ...(attribution.runId ? { runId: attribution.runId } : {}),
    ...(attribution.taskId ? { taskId: attribution.taskId } : {}),
    ...(attribution.profileKey ? { profileKey: attribution.profileKey } : {}),
    ...(attribution.sessionId ? { sessionId: attribution.sessionId } : {}),
    ...(sessionKinds.has(attribution.sessionKind ?? "")
      ? { sessionKind: attribution.sessionKind as AgentSessionKind }
      : {}),
    ...(attribution.source ? { source: attribution.source } : {}),
  };
}

function asSessionKind(value: string): AgentSessionKind {
  return sessionKinds.has(value) ? (value as AgentSessionKind) : "meta_orchestration";
}

function asSessionStatus(value: string): AgentSessionStatus {
  return sessionStatuses.has(value) ? (value as AgentSessionStatus) : "draft";
}
