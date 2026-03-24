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
  BacklogFileOperationRequestSchema,
  QueueBacklogItemRequestSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import {
  backlogResearchResponseSchema,
  backlogFileResponseSchema,
  backlogFileOperationResponseSchema,
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
import type { ArchiveRequirementRecord, ArchiveTargetFormValues, ArchiveTargetsResponse, BacklogItem, BacklogFile, BacklogKind, BacklogResearchTarget, FeedbackSummaryResponse, MaturitySummaryResponse, ModuleFormValues, ResearchResponse, ReviewUpdate } from "../types";

/**
 * Response from queueing a backlog item for processing.
 * When `dryRun` is true the queue did not execute — check `blockingReasons`.
 */
export interface QueueResponse {
  item?: BacklogItem;
  taskId: string;
  runId: string;
  baseUrl: string;
  created: string;
  dryRun: boolean;
  queued: boolean;
  message: string;
  blockingReasons: string[];
  pendingDecisions: number;
  pendingSuggestions: number;
}

export interface BacklogFileOperationResult {
  file?: BacklogFile;
  deletedPath?: string;
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
  renameFile(kind: BacklogKind, name: string, sourcePath: string, destinationPath: string): Promise<BacklogFileOperationResult>;
  moveFile(kind: BacklogKind, name: string, sourcePath: string, destinationPath: string): Promise<BacklogFileOperationResult>;
  copyFile(kind: BacklogKind, name: string, sourcePath: string, destinationPath: string): Promise<BacklogFileOperationResult>;
  deleteFile(kind: BacklogKind, name: string, sourcePath: string): Promise<BacklogFileOperationResult>;
  queue(
    kind: BacklogKind,
    name: string,
    options?: {
      operation?: "generator" | "improver";
      mode?: "manual" | "scheduled" | "yolo";
      delaySeconds?: number;
      startedBy?: string;
      confirm?: boolean;
      force?: boolean;
    }
  ): Promise<QueueResponse>;
  research(
    kind: BacklogKind,
    name: string,
    payload?: {
      prompt?: string;
      scopePath?: string;
      projectRoot?: string;
      mode?: string;
      targetKind?: BacklogResearchTarget;
      contextPaths?: string[];
      contextTargetIds?: string[];
      contextRequirementIds?: string[];
    }
  ): Promise<ResearchResponse>;
  convert(
    kind: BacklogKind,
    name: string,
    payload: { targetKind: BacklogKind; targetName?: string }
  ): Promise<BacklogItem>;
  getArchiveTargets(kind: BacklogKind, name: string): Promise<ArchiveTargetsResponse>;
  createArchiveTarget(kind: string, name: string, target: ArchiveTargetFormValues): Promise<void>;
  updateArchiveTarget(kind: string, name: string, targetId: string, target: ArchiveTargetFormValues): Promise<void>;
  deleteArchiveTarget(kind: string, name: string, targetId: string): Promise<void>;
  updateModuleRequirements(kind: string, name: string, moduleId: string, requirements: ArchiveRequirementRecord[]): Promise<void>;
  createModule(kind: string, name: string, payload: ModuleFormValues & { position?: number }): Promise<void>;
  updateModuleMeta(kind: string, name: string, moduleId: string, payload: { title: string; description: string }): Promise<void>;
  deleteModule(kind: string, name: string, moduleId: string): Promise<void>;
  batchReview(kind: string, name: string, items: ReviewUpdate[]): Promise<void>;
  exportItems(params?: {
    kinds?: string[];
    statuses?: string[];
    names?: string[];
    priorityMax?: number;
    tags?: string[];
    includePrd?: boolean;
    includeRequirements?: boolean;
    includeClarifyQuestions?: boolean;
    includeSuggestions?: boolean;
    includeNotes?: boolean;
    includeTemplate?: boolean;
  }): Promise<Blob>;
  importItems(file: File, apply?: boolean): Promise<ImportBacklogResponse>;
  getFeedbackSummary(): Promise<FeedbackSummaryResponse>;
  getMaturitySummary(): Promise<MaturitySummaryResponse>;
}

export interface ImportBacklogResponse {
  dryRun: boolean;
  changes: Array<{ item: string; action: string; details: string[] }>;
  errors: string[];
  summary: string;
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

  const operateFile = async (
    kind: BacklogKind,
    name: string,
    operation: "rename" | "move" | "copy" | "delete",
    sourcePath: string,
    destinationPath?: string
  ): Promise<BacklogFileOperationResult> => {
    const message = buildMessage(BacklogFileOperationRequestSchema, {
      operation,
      sourcePath,
      ...(destinationPath ? { destinationPath } : {}),
    });
    const payload = toProtoJson(BacklogFileOperationRequestSchema, message);
    const data = await apiClient.patch<unknown>(API_ENDPOINTS.backlogFileOperations(kind, name), payload);
    const parsed = parseProtoResponse(backlogFileOperationResponseSchema, data, "backlog file operation");
    return {
      ...(parsed.file ? { file: mapProtoBacklogFile(parsed.file) } : {}),
      ...(parsed.deletedPath ? { deletedPath: parsed.deletedPath } : {}),
    };
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

    async renameFile(kind: BacklogKind, name: string, sourcePath: string, destinationPath: string): Promise<BacklogFileOperationResult> {
      return operateFile(kind, name, "rename", sourcePath, destinationPath);
    },

    async moveFile(kind: BacklogKind, name: string, sourcePath: string, destinationPath: string): Promise<BacklogFileOperationResult> {
      return operateFile(kind, name, "move", sourcePath, destinationPath);
    },

    async copyFile(kind: BacklogKind, name: string, sourcePath: string, destinationPath: string): Promise<BacklogFileOperationResult> {
      return operateFile(kind, name, "copy", sourcePath, destinationPath);
    },

    async deleteFile(kind: BacklogKind, name: string, sourcePath: string): Promise<BacklogFileOperationResult> {
      return operateFile(kind, name, "delete", sourcePath);
    },

    async queue(
      kind: BacklogKind,
      name: string,
    options?: {
      operation?: "generator" | "improver";
      mode?: "manual" | "scheduled" | "yolo";
      delaySeconds?: number;
      startedBy?: string;
      confirm?: boolean;
      force?: boolean;
    }
  ): Promise<QueueResponse> {
      const msg = buildMessage(QueueBacklogItemRequestSchema, {
        operation: options?.operation ?? "generator",
        mode: options?.mode ?? "yolo",
        ...(options?.delaySeconds !== undefined ? { delaySeconds: BigInt(options.delaySeconds) } : {}),
        ...(options?.startedBy ? { startedBy: options.startedBy } : {}),
        ...(options?.confirm !== undefined ? { confirm: options.confirm } : {}),
        ...(options?.force !== undefined ? { force: options.force } : {}),
      });
      const data = await apiClient.post<unknown>(
        API_ENDPOINTS.backlogQueue(kind, name),
        toProtoJson(QueueBacklogItemRequestSchema, msg),
      );
      const parsed = parseProtoResponse(queueBacklogResponseSchema, data, "backlog queue");
      return {
        item: parsed.item ? mapProtoBacklogItem(parsed.item) : undefined,
        taskId: parsed.taskId ?? "",
        runId: parsed.runId ?? "",
        baseUrl: parsed.baseUrl ?? "",
        created: parsed.created ?? "",
        dryRun: parsed.dryRun ?? false,
        queued: parsed.queued ?? false,
        message: parsed.message ?? "",
        blockingReasons: parsed.blockingReasons ?? [],
        pendingDecisions: parsed.unansweredQuestions ?? 0,
        pendingSuggestions: parsed.pendingSuggestions ?? 0,
      };
    },

    async research(
      kind: BacklogKind,
      name: string,
      payload?: {
        prompt?: string;
        scopePath?: string;
        projectRoot?: string;
        mode?: string;
        targetKind?: BacklogResearchTarget;
        contextPaths?: string[];
        contextTargetIds?: string[];
        contextRequirementIds?: string[];
      }
    ): Promise<ResearchResponse> {
      const message = buildMessage(BacklogResearchRequestSchema, {
        prompt: payload?.prompt,
        scopePath: payload?.scopePath,
        projectRoot: payload?.projectRoot,
        mode: payload?.mode,
        targetKind: payload?.targetKind,
        contextPaths: payload?.contextPaths ?? [],
        contextTargetIds: payload?.contextTargetIds ?? [],
        contextRequirementIds: payload?.contextRequirementIds ?? [],
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

    async getArchiveTargets(kind: BacklogKind, name: string): Promise<ArchiveTargetsResponse> {
      return apiClient.get<ArchiveTargetsResponse>(API_ENDPOINTS.backlogArchiveTargets(kind, name));
    },

    async createArchiveTarget(kind: string, name: string, target: ArchiveTargetFormValues): Promise<void> {
      await apiClient.post<void>(API_ENDPOINTS.backlogArchiveTargets(kind, name), target);
    },

    async updateArchiveTarget(kind: string, name: string, targetId: string, target: ArchiveTargetFormValues): Promise<void> {
      await apiClient.put<void>(API_ENDPOINTS.backlogArchiveTarget(kind, name, targetId), target);
    },

    async deleteArchiveTarget(kind: string, name: string, targetId: string): Promise<void> {
      await apiClient.delete<void>(API_ENDPOINTS.backlogArchiveTarget(kind, name, targetId));
    },

    async updateModuleRequirements(kind: string, name: string, moduleId: string, requirements: ArchiveRequirementRecord[]): Promise<void> {
      await apiClient.put<void>(API_ENDPOINTS.backlogArchiveRequirementsModule(kind, name, moduleId), { requirements });
    },

    async createModule(kind: string, name: string, payload: ModuleFormValues & { position?: number }): Promise<void> {
      await apiClient.post<void>(API_ENDPOINTS.backlogArchiveRequirements(kind, name), payload);
    },

    async updateModuleMeta(kind: string, name: string, moduleId: string, payload: { title: string; description: string }): Promise<void> {
      await apiClient.put<void>(API_ENDPOINTS.backlogArchiveRequirementsModuleMeta(kind, name, moduleId), payload);
    },

    async deleteModule(kind: string, name: string, moduleId: string): Promise<void> {
      await apiClient.delete<void>(API_ENDPOINTS.backlogArchiveRequirementsModule(kind, name, moduleId));
    },

    async batchReview(kind: string, name: string, items: ReviewUpdate[]): Promise<void> {
      await apiClient.put<void>(API_ENDPOINTS.backlogArchiveReview(kind, name), { items });
    },

    async exportItems(params?: {
      kinds?: string[];
      statuses?: string[];
      names?: string[];
      priorityMax?: number;
      tags?: string[];
      includePrd?: boolean;
      includeRequirements?: boolean;
      includeClarifyQuestions?: boolean;
      includeSuggestions?: boolean;
      includeNotes?: boolean;
      includeTemplate?: boolean;
    }): Promise<Blob> {
      const response = await apiClient.post<Blob>(API_ENDPOINTS.backlogExport, params ?? {}, {
        responseType: "blob",
      });
      return response;
    },

    async importItems(file: File, apply = false): Promise<ImportBacklogResponse> {
      const formData = new FormData();
      formData.append("file", file);
      if (apply) {
        formData.append("apply", "true");
      }
      return apiClient.post<ImportBacklogResponse>(API_ENDPOINTS.backlogImport, formData, {
        headers: {},
      });
    },

    async getFeedbackSummary(): Promise<FeedbackSummaryResponse> {
      return apiClient.get<FeedbackSummaryResponse>(API_ENDPOINTS.backlogFeedbackSummary);
    },

    async getMaturitySummary(): Promise<MaturitySummaryResponse> {
      return apiClient.get<MaturitySummaryResponse>(API_ENDPOINTS.backlogMaturitySummary);
    },
  };
}

/**
 * Default backlog service instance.
 * Uses the default API client for production use.
 */
export const backlogService = createBacklogService();
