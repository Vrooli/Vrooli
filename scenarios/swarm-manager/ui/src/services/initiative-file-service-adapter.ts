/**
 * Initiative File Service Adapter
 *
 * Implements IFileService for initiative files. The backend already
 * supports full CRUD operations for initiative files — this adapter
 * exposes them through the generic interface.
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
} from "./proto-contracts";
import type { IFileService, FileOperationResult } from "./file-service-types";
import type { BacklogFile } from "../types";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";

export function createInitiativeFileServiceAdapter(
  name: string,
  apiClient: IApiClient = defaultApiClient,
): IFileService {
  async function uploadFile(file: File, path?: string): Promise<BacklogFile> {
    const formData = new FormData();
    formData.append("file", file);
    if (path) {
      formData.append("path", path);
    }
    const data = await apiClient.post<unknown>(API_ENDPOINTS.initiativeFiles(name), formData, {
      headers: {},
    });
    const parsed = parseProtoResponse(backlogFileResponseSchema, data, "initiative file");
    return mapProtoBacklogFile(requireProtoField(parsed.file, "initiative file"));
  }

  async function operateFile(
    operation: "rename" | "move" | "copy" | "delete",
    sourcePath: string,
    destinationPath?: string,
  ): Promise<FileOperationResult> {
    const message = buildMessage(BacklogFileOperationRequestSchema, {
      operation,
      sourcePath,
      ...(destinationPath ? { destinationPath } : {}),
    });
    const payload = toProtoJson(BacklogFileOperationRequestSchema, message);
    const data = await apiClient.patch<unknown>(API_ENDPOINTS.initiativeFileOperations(name), payload);
    const parsed = parseProtoResponse(backlogFileOperationResponseSchema, data, "initiative file operation");
    return {
      ...(parsed.file ? { file: mapProtoBacklogFile(parsed.file) } : {}),
      ...(parsed.deletedPath ? { deletedPath: parsed.deletedPath } : {}),
    };
  }

  return {
    entityLabel: "initiative",
    protectedFile: "initiative.json",
    fileContentBaseUrl: `/api/v1/initiatives/${name}/files`,
    queryKeyPrefix: ["initiative", name],

    async getFiles(): Promise<BacklogFile[]> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.initiativeFiles(name));
      const parsed = parseProtoResponse(backlogFilesResponseSchema, data, "initiative files");
      return parsed.files.map(mapProtoBacklogFile);
    },

    async getFileContent(filePath: string): Promise<string> {
      return apiClient.get<string>(API_ENDPOINTS.initiativeFileContent(name, filePath), {
        responseType: "text",
      });
    },

    uploadFile,

    async saveFileContent(filePath: string, content: string, contentType = "text/plain"): Promise<BacklogFile> {
      const normalizedPath = filePath.replace(/^[\\/]+/, "");
      const segments = normalizedPath.split("/");
      const fileName = segments.pop() || "notes.txt";
      const directory = segments.length > 0 ? segments.join("/") : undefined;
      const file = new File([content], fileName, { type: contentType });
      return uploadFile(file, directory);
    },

    renameFile: (src, dst) => operateFile("rename", src, dst),
    moveFile: (src, dst) => operateFile("move", src, dst),
    copyFile: (src, dst) => operateFile("copy", src, dst),
    deleteFile: (src) => operateFile("delete", src),
  };
}
