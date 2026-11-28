/**
 * Goal File Service Adapter
 *
 * Gives the shared editable file workspace the same entity-scoped service
 * contract used by backlog items. `goal.json` is deliberately protected: it
 * is canonical goal graph state and must change through validated goal APIs.
 */

import { defaultApiClient, type IApiClient } from "../lib/api-client";
import type { IFileService } from "./file-service-types";
import { createGoalsService } from "./goals-service";

export function createGoalFileServiceAdapter(name: string, apiClient: IApiClient = defaultApiClient): IFileService {
  const goals = createGoalsService(apiClient);
  return {
    entityLabel: "goal",
    protectedFile: "goal.json",
    fileContentBaseUrl: `/api/v1/goals/${name}/files`,
    queryKeyPrefix: ["goal", name],
    getFiles: () => goals.getFiles(name),
    getFileContent: (filePath) => goals.getFileContent(name, filePath),
    uploadFile: (file, path) => goals.uploadFile(name, file, path),
    saveFileContent: (filePath, content, contentType) => goals.saveFileContent(name, filePath, content, contentType),
    renameFile: (sourcePath, destinationPath) => goals.renameFile(name, sourcePath, destinationPath),
    moveFile: (sourcePath, destinationPath) => goals.moveFile(name, sourcePath, destinationPath),
    copyFile: (sourcePath, destinationPath) => goals.copyFile(name, sourcePath, destinationPath),
    deleteFile: (sourcePath) => goals.deleteFile(name, sourcePath),
  };
}
