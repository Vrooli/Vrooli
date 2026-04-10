/* eslint-disable react-refresh/only-export-components */
/**
 * FileServiceContext
 *
 * Provides an IFileService instance to all nested file components
 * (FilePreview, FileUpload, FileBrowser, etc.) so they can perform
 * file operations without knowing whether they're working with a
 * backlog item or an initiative.
 */

import { createContext, useContext } from "react";
import type { IFileService } from "../services/file-service-types";

const FileServiceContext = createContext<IFileService | null>(null);

export const FileServiceProvider = FileServiceContext.Provider;

export function useFileService(): IFileService {
  const ctx = useContext(FileServiceContext);
  if (!ctx) {
    throw new Error("useFileService must be used within a FileServiceProvider");
  }
  return ctx;
}
