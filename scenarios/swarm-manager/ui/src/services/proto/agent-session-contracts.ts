import type {
  AgentSession,
  AgentSessionArtifact,
  AgentSessionAttribution,
  AgentSessionAttachment,
  AgentSessionContextItem,
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
  GetAgentSessionStartupBriefResponseSchema,
  GetArtifactsByEntityResponseSchema,
  ListAgentSessionEventsResponseSchema,
  ListAgentSessionArtifactsResponseSchema,
  ListAgentSessionsResponseSchema,
  RefreshAgentSessionResponseSchema,
  StartAgentSessionResponseSchema,
  UploadAgentSessionAttachmentsResponseSchema,
  type AgentSessionRunEvent,
} from "@vrooli/proto-types/swarm-manager/v1/api/agent_session_pb";
import type {
  AgentSession as AgentSessionDomain,
  AgentSessionArtifact as AgentSessionArtifactDomain,
  AgentSessionAttachment as AgentSessionAttachmentDomain,
  AgentSessionAttribution as AgentSessionAttributionDomain,
  AgentSessionContextItem as AgentSessionContextItemDomain,
  AgentSessionContextType,
  AgentSessionKind,
  AgentSessionMessage as AgentSessionMessageDomain,
  AgentSessionProposal as AgentSessionProposalDomain,
  AgentSessionProposalTarget,
  AgentSessionRunEvent as AgentSessionRunEventDomain,
  AgentSessionStatus,
} from "../../types";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import { createProtoSchema } from "./shared";

const sessionKinds = new Set<string>(["meta_orchestration", "operating_mode_authoring", "swarm_operations"]);
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
const contextTypes = new Set<string>([
  "backlog_item",
  "initiative",
  "capture",
  "execution",
  "agent_activity",
  "scenario",
  "operating_mode",
  "session",
  "operations_briefing",
  "startup_brief",
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
const eventTypes = new Set<string>([
  "message",
  "tool_call",
  "tool_result",
  "status",
  "progress",
  "error",
  "log",
  "metric",
  "artifact",
  "compaction",
  "lifecycle",
  "message_deleted",
  "unknown",
]);

export const listAgentSessionsResponseSchema = createProtoSchema(
  ListAgentSessionsResponseSchema,
  "agent sessions list"
);
export const getAgentSessionResponseSchema = createProtoSchema(
  GetAgentSessionResponseSchema,
  "agent session"
);
export const getAgentSessionStartupBriefResponseSchema = createProtoSchema(
  GetAgentSessionStartupBriefResponseSchema,
  "agent session startup brief"
);
export const createAgentSessionResponseSchema = createProtoSchema(
  CreateAgentSessionResponseSchema,
  "agent session create"
);
export const continueAgentSessionResponseSchema = createProtoSchema(
  ContinueAgentSessionResponseSchema,
  "agent session continue"
);
export const startAgentSessionResponseSchema = createProtoSchema(
  StartAgentSessionResponseSchema,
  "agent session start"
);
export const listAgentSessionEventsResponseSchema = createProtoSchema(
  ListAgentSessionEventsResponseSchema,
  "agent session events"
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
export const uploadAgentSessionAttachmentsResponseSchema = createProtoSchema(
  UploadAgentSessionAttachmentsResponseSchema,
  "agent session attachment upload"
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
    attachments: protoSession.attachments?.map((attachment) => ({
      ...mapProtoAgentSessionAttachment(attachment),
      url: API_ENDPOINTS.agentSessionAttachment(protoSession.id ?? "", attachment.id ?? ""),
    })) ?? [],
    ...(protoSession.createdBy ? { createdBy: mapProtoAgentSessionAttribution(protoSession.createdBy) } : {}),
    ...(protoSession.proposalTarget ? { proposalTarget: mapProtoProposalTarget(protoSession.proposalTarget) } : {}),
  };
}

function mapProtoProposalTarget(target: NonNullable<AgentSession["proposalTarget"]>): AgentSessionProposalTarget {
  return {
    type: target.type === "initiative" ? "initiative" : "backlog_item",
    ref: target.ref ?? "",
    name: target.name ?? "",
  };
}

export function mapProtoAgentSessionMessage(message: AgentSessionMessage): AgentSessionMessageDomain {
  return {
    id: message.id ?? "",
    role: messageRoles.has(message.role) ? (message.role as AgentSessionMessageDomain["role"]) : "assistant",
    content: message.content ?? "",
    createdAt: message.createdAt ?? "",
    attachmentIds: message.attachmentIds ?? [],
    context: message.context?.map(mapProtoAgentSessionContextItem) ?? [],
  };
}

export function mapProtoAgentSessionContextItem(item: AgentSessionContextItem): AgentSessionContextItemDomain {
  return {
    type: contextTypes.has(item.type) ? (item.type as AgentSessionContextType) : "capture",
    ref: item.ref ?? "",
    title: item.title ?? "",
    summary: item.summary ?? "",
    ...(item.nodeId ? { nodeId: item.nodeId } : {}),
    ...(item.metadataJson ? { metadataJson: item.metadataJson } : {}),
    selectedAt: item.selectedAt ?? "",
  };
}

export function mapProtoAgentSessionAttachment(attachment: AgentSessionAttachment): AgentSessionAttachmentDomain {
  return {
    id: attachment.id ?? "",
    filename: attachment.filename ?? "",
    ...(attachment.contentType ? { contentType: attachment.contentType } : {}),
    ...(attachment.sizeBytes !== undefined ? { sizeBytes: attachment.sizeBytes } : {}),
    createdAt: attachment.createdAt ?? "",
  };
}

export function mapProtoAgentSessionProposal(proposal: AgentSessionProposal): AgentSessionProposalDomain {
  return {
    id: proposal.id ?? "",
    kind: proposal.kind ?? "unknown",
    status: proposal.status ?? "unknown",
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

export function mapProtoAgentSessionRunEvent(event: AgentSessionRunEvent): AgentSessionRunEventDomain {
  return {
    id: event.id ?? "",
    runId: event.runId ?? "",
    sequence: event.sequence ?? 0n,
    createdAt: event.createdAt ?? "",
    eventType: eventTypes.has(event.eventType) ? (event.eventType as AgentSessionRunEventDomain["eventType"]) : "unknown",
    role: event.role ?? "",
    content: event.content ?? "",
    toolName: event.toolName ?? "",
    toolCallId: event.toolCallId ?? "",
    input: event.input ?? "",
    output: event.output ?? "",
    error: event.error ?? "",
    status: event.status ?? "",
    previousStatus: event.previousStatus ?? "",
    progressPhase: event.progressPhase ?? "",
    progressPercent: event.progressPercent ?? 0,
    progressMessage: event.progressMessage ?? "",
    summary: event.summary ?? "",
    rawJson: event.rawJson ?? "",
  };
}

function asSessionKind(value: string): AgentSessionKind {
  return sessionKinds.has(value) ? (value as AgentSessionKind) : "meta_orchestration";
}

function asSessionStatus(value: string): AgentSessionStatus {
  return sessionStatuses.has(value) ? (value as AgentSessionStatus) : "draft";
}
