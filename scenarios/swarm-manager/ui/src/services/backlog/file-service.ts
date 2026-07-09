/**
 * Backlog File Service — file tree operations (list, get, upload, operate)
 */

import {
  BacklogFileOperationRequestSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import {
  backlogFileResponseSchema,
  backlogFileOperationResponseSchema,
  backlogFilesResponseSchema,
  mapProtoBacklogFile,
  parseProtoResponse,
  requireProtoField,
  buildMessage,
  toProtoJson,
} from "../proto-contracts";
import type { IApiClient } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import type { BacklogFile, BacklogKind, PlanRef } from "../../types";
import type { BacklogFileOperationResult, RenderedBacklogPlan } from "./types";

function createUploadFile(apiClient: IApiClient) {
  return async (
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
}

function createOperateFile(apiClient: IApiClient) {
  return async (
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
}

export function createFileMethods(apiClient: IApiClient) {
  const uploadFile = createUploadFile(apiClient);
  const operateFile = createOperateFile(apiClient);

  return {
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

    async getRenderedPlan(kind: BacklogKind, name: string): Promise<RenderedBacklogPlan> {
      const data = await apiClient.get<{
        path?: string;
        markdown?: string;
        quality_status?: string;
        quality_findings?: string[];
        plan_ref?: PlanRef;
      }>(API_ENDPOINTS.backlogPlanRender(kind, name));
      return {
        path: data.path ?? "",
        markdown: data.markdown ?? "",
        qualityStatus: data.quality_status,
        qualityFindings: data.quality_findings ?? [],
        planRef: data.plan_ref,
      };
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
  };
}
