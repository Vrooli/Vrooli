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
import type { BacklogItem, BacklogKind } from "../../types";
import type { BacklogUpdatePatch } from "./types";

export function buildBacklogUpdatePayload(patch: BacklogUpdatePatch): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  if (patch.title !== undefined) payload.title = patch.title;
  if (patch.description !== undefined) payload.description = patch.description;
  if (patch.status !== undefined) payload.status = patch.status;
  if (patch.priority !== undefined) payload.priority = patch.priority;
  if (patch.tags !== undefined) payload.tags = patch.tags;
  if (patch.dependsOn !== undefined) payload.depends_on = patch.dependsOn;
  if (patch.initiative !== undefined) payload.initiative = patch.initiative;
  if (patch.effort !== undefined) payload.effort = patch.effort;
  if (patch.acceptanceAllow !== undefined) payload.acceptance_allow = patch.acceptanceAllow;
  if (patch.acceptanceDeny !== undefined) payload.acceptance_deny = patch.acceptanceDeny;
  return payload;
}

export function createCrudMethods(apiClient: IApiClient) {
  return {
    async list(kinds?: BacklogKind[]): Promise<BacklogItem[]> {
      const query = buildQueryString({ kinds });
      const data = await apiClient.get<unknown>(`${API_ENDPOINTS.backlog}${query}`);
      const parsed = parseProtoResponse(listBacklogResponseSchema, data, "backlog list");
      return parsed.items.map(mapProtoBacklogItem);
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

    async create(item: Omit<BacklogItem, "created" | "updated">): Promise<BacklogItem> {
      const message = buildMessage(CreateBacklogItemRequestSchema, {
        name: item.name,
        title: item.title,
        description: item.description || undefined,
        priority: item.priority || undefined,
        tags: item.tags,
        kind: item.kind,
        dependsOn: item.dependsOn ?? [],
        initiative: item.initiative || undefined,
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
  };
}
