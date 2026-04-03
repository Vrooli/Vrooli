import {
  ListAgentActivitiesResponseSchema,
  AgentActivityResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/agent_activity_pb";
import type { AgentActivity as ProtoAgentActivity } from "@vrooli/proto-types/swarm-manager/v1/domain/agent_activity_pb";
import type {
  AgentActivity,
  AgentActivityInteractionType,
  AgentActivityOwnerType,
  AgentActivityPurpose,
  AgentActivityStatus,
} from "../../types";
import { createProtoSchema } from "./shared";

const agentActivityStatusSet = new Set<string>([
  "pending",
  "starting",
  "running",
  "needs_review",
  "complete",
  "failed",
  "cancelled",
  "unspecified",
]);
const agentActivityPurposeSet = new Set<string>([
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
]);
const agentActivityInteractionTypeSet = new Set<string>(["spawn", "continue"]);
const agentActivityOwnerTypeSet = new Set<string>(["backlog", "capture", "scenario"]);

function isAgentActivityStatus(value: unknown): value is AgentActivityStatus {
  return typeof value === "string" && agentActivityStatusSet.has(value);
}

function isAgentActivityPurpose(value: unknown): value is AgentActivityPurpose {
  return typeof value === "string" && agentActivityPurposeSet.has(value);
}

function isAgentActivityInteractionType(value: unknown): value is AgentActivityInteractionType {
  return typeof value === "string" && agentActivityInteractionTypeSet.has(value);
}

function isAgentActivityOwnerType(value: unknown): value is AgentActivityOwnerType {
  return typeof value === "string" && agentActivityOwnerTypeSet.has(value);
}

export const listAgentActivitiesResponseSchema = createProtoSchema(
  ListAgentActivitiesResponseSchema,
  "agent activities"
);
export const agentActivityResponseSchema = createProtoSchema(
  AgentActivityResponseSchema,
  "agent activity"
);

export function mapProtoAgentActivity(proto: ProtoAgentActivity): AgentActivity {
  return {
    activityId: proto.activityId ?? "",
    ownerType: isAgentActivityOwnerType(proto.ownerType) ? proto.ownerType : "backlog",
    ownerKind: proto.ownerKind,
    ownerName: proto.ownerName ?? "",
    ownerTitle: proto.ownerTitle,
    executionId: proto.executionId,
    purpose: isAgentActivityPurpose(proto.purpose) ? proto.purpose : "process",
    interactionType: isAgentActivityInteractionType(proto.interactionType)
      ? proto.interactionType
      : "spawn",
    taskId: proto.taskId,
    runId: proto.runId,
    status: isAgentActivityStatus(proto.status) ? proto.status : "unspecified",
    requestedAt: proto.requestedAt ?? "",
    startedAt: proto.startedAt,
    finishedAt: proto.finishedAt,
    failureReason: proto.failureReason,
    requestedBy: proto.requestedBy,
    metadata: proto.metadata ?? {},
    updatedAt: proto.updatedAt ?? "",
  };
}
