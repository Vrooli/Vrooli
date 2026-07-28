/**
 * Backlog CRUD Service — list, get, create, update, delete operations
 */

import {
  CreateBacklogItemRequestSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import {
  backlogItemResponseSchema,
  listBacklogResponseSchema,
  mapProtoBacklogItem,
  parseProtoResponse,
  requireProtoField,
  buildMessage,
  toProtoJson,
} from "../proto-contracts";
import type { IApiClient } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import { buildQueryString } from "../../lib/query-utils";
import type { BacklogItem, BacklogKind, ItemBlockingInfo } from "../../types";
import type { BacklogNextAction, BacklogUpdatePatch } from "./types";

const MAX_NEXT_ACTION_BATCH_SIZE = 100;

function stringField(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function mapNextAction(raw: Record<string, unknown>): BacklogNextAction {
	const followUp = raw.follow_up ?? raw.followUp;
  return {
    id: String(raw.id) as BacklogNextAction["id"],
    compactLabel: stringField(raw.compact_label ?? raw.compactLabel),
    expandedLabel: stringField(raw.expanded_label ?? raw.expandedLabel),
    enabled: raw.enabled === true,
    reason: typeof raw.reason === "string" ? raw.reason : undefined,
    blockers: Array.isArray(raw.blockers) ? raw.blockers as BacklogNextAction["blockers"] : [],
    target: typeof raw.target === "string" ? raw.target : undefined,
	followUp: typeof followUp === "object" && followUp !== null ? followUp as BacklogNextAction["followUp"] : undefined,
  };
}

export function buildBacklogUpdatePayload(patch: BacklogUpdatePatch): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  if (patch.title !== undefined) payload.title = patch.title;
  if (patch.description !== undefined) payload.description = patch.description;
  if (patch.status !== undefined) payload.status = patch.status;
  if (patch.priority !== undefined) payload.priority = patch.priority;
  if (patch.tags !== undefined) payload.tags = patch.tags;
  if (patch.dependsOn !== undefined) payload.depends_on = patch.dependsOn;
  if (patch.milestone !== undefined) payload.milestone = patch.milestone;
  if (patch.effort !== undefined) payload.effort = patch.effort;
  if (patch.acceptanceAllow !== undefined) payload.acceptance_allow = patch.acceptanceAllow;
  if (patch.acceptanceDeny !== undefined) payload.acceptance_deny = patch.acceptanceDeny;
  if (patch.note !== undefined) payload.note = patch.note;
  return payload;
}

export function createCrudMethods(apiClient: IApiClient) {
  return {
    async list(kinds?: BacklogKind[]): Promise<{ items: BacklogItem[]; blocking: Record<string, ItemBlockingInfo> }> {
      const query = buildQueryString({ kinds, archived: "all" });
      const data = await apiClient.get<unknown>(`${API_ENDPOINTS.backlog}${query}`);
      const parsed = parseProtoResponse(listBacklogResponseSchema, data, "backlog list");
      const items = parsed.items.map(mapProtoBacklogItem);
      const blocking: Record<string, ItemBlockingInfo> = {};
      for (const [key, info] of Object.entries(parsed.blocking ?? {})) {
        blocking[key] = {
          blocked: info.blocked ?? false,
          blockingDepKeys: info.blockingDepKeys ?? [],
          allForceable: info.allForceable ?? false,
        };
      }
      return { items, blocking };
    },

    async listBySpawnedFrom(spawnedFrom: string): Promise<BacklogItem[]> {
      const data = await apiClient.get<unknown>(`${API_ENDPOINTS.backlog}?spawned_from=${encodeURIComponent(spawnedFrom)}`);
      const parsed = parseProtoResponse(listBacklogResponseSchema, data, "backlog list");
      return parsed.items.map(mapProtoBacklogItem);
    },

    async get(kind: BacklogKind, name: string): Promise<BacklogItem> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.backlogItem(kind, name));
      const parsed = parseProtoResponse(backlogItemResponseSchema, data, "backlog item");
      return mapProtoBacklogItem(requireProtoField(parsed.item, "backlog item"));
    },

    async getNextAction(kind: BacklogKind, name: string): Promise<BacklogNextAction> {
      const data = await apiClient.get<{ action?: Record<string, unknown> }>(API_ENDPOINTS.backlogNextAction(kind, name));
      if (!data.action) throw new Error("next action response is missing action");
      return mapNextAction(data.action);
    },

    async getNextActions(items: Array<{ kind: BacklogKind; name: string }>): Promise<Record<string, BacklogNextAction>> {
      const references = items.map(({ kind, name }) => `${kind}/${name}`);
      const batches = Array.from(
        { length: Math.ceil(references.length / MAX_NEXT_ACTION_BATCH_SIZE) },
        (_, index) => references.slice(index * MAX_NEXT_ACTION_BATCH_SIZE, (index + 1) * MAX_NEXT_ACTION_BATCH_SIZE),
      );
      const responses = await Promise.all(batches.map((batch) => apiClient.post<{ results?: Array<{ item: string; action?: Record<string, unknown> }> }>(
        API_ENDPOINTS.backlogNextActions,
        { items: batch },
      )));
      const actions: Record<string, BacklogNextAction> = {};
      for (const data of responses) {
        for (const result of data.results ?? []) if (result.action) actions[result.item] = mapNextAction(result.action);
      }
      return actions;
    },

    async create(item: Omit<BacklogItem, "created" | "updated">): Promise<BacklogItem> {
      const message = buildMessage(CreateBacklogItemRequestSchema, {
        name: item.name,
        title: item.title,
        description: item.description || undefined,
        priority: item.priority || undefined,
        tags: item.tags,
        kind: item.kind,
        dependsOn: item.dependsOn ?? [],
        milestone: item.milestone || undefined,
      });
      const payload = toProtoJson(CreateBacklogItemRequestSchema, message);
      const data = await apiClient.post<unknown>(API_ENDPOINTS.backlog, payload);
      const parsed = parseProtoResponse(backlogItemResponseSchema, data, "backlog item");
      return mapProtoBacklogItem(requireProtoField(parsed.item, "backlog item"));
    },

    async update(
      kind: BacklogKind,
      name: string,
      patch: BacklogUpdatePatch
    ): Promise<BacklogItem> {
      const payload = buildBacklogUpdatePayload(patch);
      const data = await apiClient.patch<unknown>(API_ENDPOINTS.backlogItem(kind, name), payload);
      const parsed = parseProtoResponse(backlogItemResponseSchema, data, "backlog item");
      return mapProtoBacklogItem(requireProtoField(parsed.item, "backlog item"));
    },

    async delete(kind: BacklogKind, name: string): Promise<void> {
      return apiClient.delete<void>(API_ENDPOINTS.backlogItem(kind, name));
    },

    async archiveItem(kind: BacklogKind, name: string): Promise<BacklogItem> {
      const data = await apiClient.patch<unknown>(API_ENDPOINTS.backlogArchiveItem(kind, name), {});
      const parsed = parseProtoResponse(backlogItemResponseSchema, data, "backlog item");
      return mapProtoBacklogItem(requireProtoField(parsed.item, "backlog item"));
    },

    async unarchiveItem(kind: BacklogKind, name: string): Promise<BacklogItem> {
      const data = await apiClient.delete<unknown>(API_ENDPOINTS.backlogArchiveItem(kind, name));
      const parsed = parseProtoResponse(backlogItemResponseSchema, data, "backlog item");
      return mapProtoBacklogItem(requireProtoField(parsed.item, "backlog item"));
    },
  };
}
