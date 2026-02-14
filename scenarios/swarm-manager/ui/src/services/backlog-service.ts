/**
 * Backlog Service - Data access layer for backlog operations
 *
 * This service encapsulates all backlog-related API operations behind a clean seam.
 * It accepts an API client as a dependency, making it easy to substitute for testing.
 *
 * Responsibilities:
 * - Backlog CRUD operations
 * - File tree operations
 * - Queue/research/convert actions
 *
 * NOT responsible for:
 * - HTTP implementation details (delegated to api client)
 * - UI state or caching (delegated to React Query)
 * - Domain validation (delegated to API)
 *
 * DOC: docs/internal/SEAMS.md#ui-to-api-seam-improved-in-phase-3
 * DOC: docs/internal/INTENT.md#key-flows
 */

import {
  CreateBacklogItemRequestSchema,
  UpdateBacklogItemRequestSchema,
  ConvertBacklogItemRequestSchema,
  BacklogResearchRequestSchema,
  QueueBacklogItemRequestSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import {
  backlogResearchResponseSchema,
  backlogFileResponseSchema,
  backlogFilesResponseSchema,
  backlogItemResponseSchema,
  listBacklogResponseSchema,
  mapProtoBacklogFile,
  mapProtoBacklogItem,
  parseProtoResponse,
  queueBacklogResponseSchema,
  requireProtoField,
  buildMessage,
  toProtoJson,
} from "./proto-contracts";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { BacklogItem, BacklogFile, BacklogKind, BacklogResearchTarget, IdeaAgentMode, ResearchResponse } from "../types";

/**
 * Response from queueing a backlog item for processing.
 */
export interface QueueResponse {
  item: BacklogItem;
  taskId: string;
  runId: string;
  baseUrl: string;
  created: string;
}

/**
 * Interface for the backlog service.
 * This is the seam - implementations can be swapped for testing.
 */
export interface IBacklogService {
  list(kinds?: BacklogKind[]): Promise<BacklogItem[]>;
  get(kind: BacklogKind, name: string): Promise<BacklogItem>;
  create(item: Omit<BacklogItem, "created" | "updated">): Promise<BacklogItem>;
  update(
    kind: BacklogKind,
    name: string,
    item: Pick<BacklogItem, "title" | "description" | "status" | "priority" | "tags" | "researchTarget">
  ): Promise<BacklogItem>;
  delete(kind: BacklogKind, name: string): Promise<void>;
  getFiles(kind: BacklogKind, name: string): Promise<BacklogFile[]>;
  getFileContent(kind: BacklogKind, name: string, filePath: string): Promise<string>;
  uploadFile(kind: BacklogKind, name: string, file: File, path?: string): Promise<BacklogFile>;
  saveFileContent(
    kind: BacklogKind,
    name: string,
    filePath: string,
    content: string,
    contentType?: string
  ): Promise<BacklogFile>;
  queue(
    kind: BacklogKind,
    name: string,
    options?: {
      operation?: "generator" | "improver";
      mode?: "manual" | "scheduled" | "yolo";
      delaySeconds?: number;
      startedBy?: string;
    }
  ): Promise<QueueResponse>;
  research(
    kind: BacklogKind,
    name: string,
    payload?: {
      prompt?: string;
      scopePath?: string;
      projectRoot?: string;
      mode?: IdeaAgentMode;
      targetKind?: BacklogResearchTarget;
    }
  ): Promise<ResearchResponse>;
  convert(
    kind: BacklogKind,
    name: string,
    payload: { targetKind: BacklogKind; targetName?: string }
  ): Promise<BacklogItem>;
}

export function createBacklogService(apiClient: IApiClient = defaultApiClient): IBacklogService {
  const uploadFile = async (
    kind: BacklogKind,
    name: string,
    file: File,
    path?: string
  ): Promise<BacklogFile> => {
    const formData = new FormData();
    formData.append("file", file);
    if (path) {
      formData.append("path", path);
    }
    const data = await apiClient.post<unknown>(API_ENDPOINTS.backlogFiles(kind, name), formData, {
      headers: {},
    });
    const parsed = parseProtoResponse(backlogFileResponseSchema, data, "backlog file");
    return mapProtoBacklogFile(requireProtoField(parsed.file, "backlog file"));
  };

  return {
    async list(kinds?: BacklogKind[]): Promise<BacklogItem[]> {
      const query = kinds && kinds.length > 0 ? `?kinds=${kinds.join(",")}` : "";
      const data = await apiClient.get<unknown>(`${API_ENDPOINTS.backlog}${query}`);
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
        researchTarget: item.researchTarget || undefined,
      });
      const payload = toProtoJson(CreateBacklogItemRequestSchema, message);
      const data = await apiClient.post<unknown>(API_ENDPOINTS.backlog, payload);
      const parsed = parseProtoResponse(backlogItemResponseSchema, data, "backlog item");
      return mapProtoBacklogItem(requireProtoField(parsed.item, "backlog item"));
    },

    async update(
      kind: BacklogKind,
      name: string,
      item: Pick<BacklogItem, "title" | "description" | "status" | "priority" | "tags" | "researchTarget">
    ): Promise<BacklogItem> {
      const message = buildMessage(UpdateBacklogItemRequestSchema, {
        title: item.title,
        description: item.description,
        status: item.status,
        priority: item.priority,
        tags: item.tags,
        researchTarget: item.researchTarget || undefined,
      });
      const payload = toProtoJson(UpdateBacklogItemRequestSchema, message);
      const data = await apiClient.put<unknown>(API_ENDPOINTS.backlogItem(kind, name), payload);
      const parsed = parseProtoResponse(backlogItemResponseSchema, data, "backlog item");
      return mapProtoBacklogItem(requireProtoField(parsed.item, "backlog item"));
    },

    async delete(kind: BacklogKind, name: string): Promise<void> {
      return apiClient.delete<void>(API_ENDPOINTS.backlogItem(kind, name));
    },

    async getFiles(kind: BacklogKind, name: string): Promise<BacklogFile[]> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.backlogFiles(kind, name));
      const parsed = parseProtoResponse(backlogFilesResponseSchema, data, "backlog files");
      return parsed.files.map(mapProtoBacklogFile);
    },

    async getFileContent(kind: BacklogKind, name: string, filePath: string): Promise<string> {
      return apiClient.get<string>(API_ENDPOINTS.backlogFileContent(kind, name, filePath), {
        responseType: "text",
      });
    },

    async uploadFile(kind: BacklogKind, name: string, file: File, path?: string): Promise<BacklogFile> {
      return uploadFile(kind, name, file, path);
    },

    async saveFileContent(
      kind: BacklogKind,
      name: string,
      filePath: string,
      content: string,
      contentType = "text/plain"
    ): Promise<BacklogFile> {
      const normalizedPath = filePath.replace(/^[\\/]+/, "");
      const segments = normalizedPath.split("/");
      const fileName = segments.pop() || "notes.txt";
      const directory = segments.length > 0 ? segments.join("/") : undefined;
      const file = new File([content], fileName, { type: contentType });
      return uploadFile(kind, name, file, directory);
    },

    async queue(
      kind: BacklogKind,
      name: string,
      options?: {
        operation?: "generator" | "improver";
        mode?: "manual" | "scheduled" | "yolo";
        delaySeconds?: number;
        startedBy?: string;
      }
    ): Promise<QueueResponse> {
      const msg = buildMessage(QueueBacklogItemRequestSchema, {
        operation: options?.operation ?? "generator",
        mode: options?.mode ?? "yolo",
        ...(options?.delaySeconds !== undefined ? { delaySeconds: BigInt(options.delaySeconds) } : {}),
        ...(options?.startedBy ? { startedBy: options.startedBy } : {}),
      });
      const data = await apiClient.post<unknown>(
        API_ENDPOINTS.backlogQueue(kind, name),
        toProtoJson(QueueBacklogItemRequestSchema, msg),
      );
      const parsed = parseProtoResponse(queueBacklogResponseSchema, data, "backlog queue");
      const item = requireProtoField(parsed.item, "backlog queue");
      return {
        item: mapProtoBacklogItem(item),
        taskId: parsed.taskId ?? "",
        runId: parsed.runId ?? "",
        baseUrl: parsed.baseUrl ?? "",
        created: parsed.created ?? "",
      };
    },

    async research(
      kind: BacklogKind,
      name: string,
      payload?: {
        prompt?: string;
        scopePath?: string;
        projectRoot?: string;
        mode?: IdeaAgentMode;
        targetKind?: BacklogResearchTarget;
      }
    ): Promise<ResearchResponse> {
      const message = buildMessage(BacklogResearchRequestSchema, {
        prompt: payload?.prompt,
        scopePath: payload?.scopePath,
        projectRoot: payload?.projectRoot,
        mode: payload?.mode,
        targetKind: payload?.targetKind,
      });
      const data = await apiClient.post<unknown>(
        API_ENDPOINTS.backlogResearch(kind, name),
        toProtoJson(BacklogResearchRequestSchema, message)
      );
      return parseProtoResponse(backlogResearchResponseSchema, data, "backlog research");
    },

    async convert(
      kind: BacklogKind,
      name: string,
      payload: { targetKind: BacklogKind; targetName?: string }
    ): Promise<BacklogItem> {
      const message = buildMessage(ConvertBacklogItemRequestSchema, {
        targetKind: payload.targetKind,
        targetName: payload.targetName || undefined,
      });
      const data = await apiClient.post<unknown>(API_ENDPOINTS.backlogConvert(kind, name), toProtoJson(ConvertBacklogItemRequestSchema, message));
      const parsed = parseProtoResponse(backlogItemResponseSchema, data, "backlog item");
      return mapProtoBacklogItem(requireProtoField(parsed.item, "backlog item"));
    },
  };
}

/**
 * Default backlog service instance.
 * Uses the default API client for production use.
 */
export const backlogService = createBacklogService();
