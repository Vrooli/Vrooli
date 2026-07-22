/**
 * Generic File Service Interface
 *
 * Entity-agnostic contract for file operations. Both backlog items and
 * goals implement this interface so that shared UI components
 * (FilePreview, FileUpload, FileBrowser, etc.) work with either entity
 * type without code duplication.
 *
 * Entity identifiers (kind, name) are captured at construction time,
 * so callers never need to know which entity type they're working with.
 */

import type { BacklogFile } from "../types";

export interface FileOperationResult {
  file?: BacklogFile;
  deletedPath?: string;
}

export interface IFileService {
  /** Human-readable entity label for UI strings (e.g., "backlog item", "goal") */
  readonly entityLabel: string;
  /** File path that cannot be operated on (e.g., "spec.json", "goal.json") */
  readonly protectedFile: string;
  /** Base URL for raw file content — used for image src attributes */
  readonly fileContentBaseUrl: string;
  /** React Query cache key prefix (e.g., ["backlog", kind, name] or ["goal", name]) */
  readonly queryKeyPrefix: readonly string[];

  getFiles(): Promise<BacklogFile[]>;
  getFileContent(filePath: string): Promise<string>;
  uploadFile(file: File, path?: string): Promise<BacklogFile>;
  saveFileContent(filePath: string, content: string, contentType?: string): Promise<BacklogFile>;
  renameFile(sourcePath: string, destinationPath: string): Promise<FileOperationResult>;
  moveFile(sourcePath: string, destinationPath: string): Promise<FileOperationResult>;
  copyFile(sourcePath: string, destinationPath: string): Promise<FileOperationResult>;
  deleteFile(sourcePath: string): Promise<FileOperationResult>;
}
