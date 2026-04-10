/**
 * Backlog File Service Adapter
 *
 * Wraps the existing backlog service's file methods into the generic
 * IFileService interface. Entity identifiers (kind, name) are captured
 * at construction time.
 */

import type { IFileService } from "../file-service-types";
import type { BacklogKind } from "../../types";
import { createBacklogService } from ".";
import type { IApiClient } from "../../lib/api-client";
import { defaultApiClient } from "../../lib/api-client";

export function createBacklogFileServiceAdapter(
  kind: BacklogKind,
  name: string,
  apiClient: IApiClient = defaultApiClient,
): IFileService {
  const svc = createBacklogService(apiClient);

  return {
    entityLabel: "backlog item",
    protectedFile: "spec.json",
    fileContentBaseUrl: `/api/v1/backlog/${kind}/${name}/files`,
    queryKeyPrefix: ["backlog", kind, name],

    getFiles: () => svc.getFiles(kind, name),
    getFileContent: (filePath) => svc.getFileContent(kind, name, filePath),
    uploadFile: (file, path) => svc.uploadFile(kind, name, file, path),
    saveFileContent: (filePath, content, contentType) =>
      svc.saveFileContent(kind, name, filePath, content, contentType),
    renameFile: (src, dst) => svc.renameFile(kind, name, src, dst),
    moveFile: (src, dst) => svc.moveFile(kind, name, src, dst),
    copyFile: (src, dst) => svc.copyFile(kind, name, src, dst),
    deleteFile: (src) => svc.deleteFile(kind, name, src),
  };
}
