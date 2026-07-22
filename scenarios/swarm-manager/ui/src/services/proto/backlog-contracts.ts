import type {
  BacklogItem,
  BacklogFile,
} from "@vrooli/proto-types/swarm-manager/v1/domain/backlog_pb";
import {
  BacklogItemResponseSchema,
  BacklogFilesResponseSchema,
  BacklogFileResponseSchema,
  BacklogFileOperationResponseSchema,
  ListBacklogItemsResponseSchema,
  QueueBacklogItemResponseSchema,
  BacklogResearchResponseSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import type {
  BacklogItem as BacklogItemDomain,
  BacklogFile as BacklogFileDomain,
} from "../../types";
import { BACKLOG_KINDS, BACKLOG_STATUSES } from "../../types";
import { mapProtoAgentSessionAttribution } from "./agent-session-contracts";
import { createProtoSchema, isFileType, toFiniteNumber } from "./shared";

const backlogStatusSet = new Set<string>(BACKLOG_STATUSES);
const backlogKindSet = new Set<string>(BACKLOG_KINDS);

function isBacklogStatus(value: unknown): value is BacklogItemDomain["status"] {
  return typeof value === "string" && backlogStatusSet.has(value);
}

function isBacklogKind(value: unknown): value is BacklogItemDomain["kind"] {
  return typeof value === "string" && backlogKindSet.has(value);
}

export const listBacklogResponseSchema = createProtoSchema(
  ListBacklogItemsResponseSchema,
  "backlog list"
);
export const backlogItemResponseSchema = createProtoSchema(
  BacklogItemResponseSchema,
  "backlog item"
);
export const backlogFilesResponseSchema = createProtoSchema(
  BacklogFilesResponseSchema,
  "backlog files"
);
export const backlogFileResponseSchema = createProtoSchema(
  BacklogFileResponseSchema,
  "backlog file"
);
export const backlogFileOperationResponseSchema = createProtoSchema(
  BacklogFileOperationResponseSchema,
  "backlog file operation"
);
export const queueBacklogResponseSchema = createProtoSchema(
  QueueBacklogItemResponseSchema,
  "backlog queue"
);
export const backlogResearchResponseSchema = createProtoSchema(
  BacklogResearchResponseSchema,
  "backlog research"
);

export function mapProtoBacklogItem(protoItem: BacklogItem): BacklogItemDomain {
  const status = isBacklogStatus(protoItem.status) ? protoItem.status : "backlog";
  const kind = isBacklogKind(protoItem.kind) ? protoItem.kind : "idea";
  const planRef = (protoItem as BacklogItem & { planRef?: BacklogItemDomain["planRef"] }).planRef;
  return {
    name: protoItem.name ?? "",
    title: protoItem.title ?? "",
    description: protoItem.description ?? "",
    status,
    priority: protoItem.priority ?? 0,
    tags: protoItem.tags ?? [],
    created: protoItem.created ?? "",
    updated: protoItem.updated ?? "",
    kind,
    ...(protoItem.dependsOn?.length ? { dependsOn: protoItem.dependsOn } : {}),
    ...(protoItem.milestone ? { milestone: protoItem.milestone } : {}),
    ...(protoItem.acceptanceAllow?.length ? { acceptanceAllow: protoItem.acceptanceAllow } : {}),
    ...(protoItem.acceptanceDeny?.length ? { acceptanceDeny: protoItem.acceptanceDeny } : {}),
    ...(protoItem.effort ? { effort: protoItem.effort } : {}),
    ...(protoItem.spawnedFrom ? { spawnedFrom: protoItem.spawnedFrom } : {}),
    ...(planRef ? { planRef } : {}),
    ...(protoItem.note ? { note: protoItem.note } : {}),
    ...(protoItem.archivedAt ? { archivedAt: protoItem.archivedAt } : {}),
    ...(protoItem.createdBy ? { createdBy: mapProtoAgentSessionAttribution(protoItem.createdBy) } : {}),
    suggestedSkills: protoItem.suggestedSkills ?? [],
  };
}

export function mapProtoBacklogFile(protoFile: BacklogFile): BacklogFileDomain {
  const size = toFiniteNumber(protoFile.size);
  const children = protoFile.children?.map(mapProtoBacklogFile) ?? [];
  const fileType = isFileType(protoFile.type) ? protoFile.type : "file";
  return {
    name: protoFile.name ?? "",
    path: protoFile.path ?? "",
    type: fileType,
    ...(size !== undefined ? { size } : {}),
    ...(children.length > 0 ? { children } : {}),
  };
}
