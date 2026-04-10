/**
 * useBacklogFileHandlers
 *
 * File selection, upload, rename, move, copy, and delete handlers for
 * BacklogDetailsPage. Extracted from useBacklogHandlers for modularity.
 */

import { useCallback } from "react";
import { type SetURLSearchParams } from "react-router-dom";
import type { useBacklogDetailData } from "./useBacklogDetailData";
import type { BacklogFile } from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type DetailsTab = "info" | "prompt" | "files";
type FileActionType = "rename" | "move" | "copy" | "delete";

/** The subset of useBacklogDetailData return used by file handlers. */
type BacklogDetailData = ReturnType<typeof useBacklogDetailData>;

export interface UseBacklogFileHandlersOptions {
  data: BacklogDetailData;
  setSelectedFile: (v: BacklogFile | null) => void;
  setActiveTab: (v: DetailsTab) => void;
  setSearchParams: SetURLSearchParams;
  selectedFile: BacklogFile | null;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const remapSelectedPath = (
  currentPath: string,
  target: BacklogFile,
  destinationPath: string,
): string | null => {
  if (target.type === "file") {
    return currentPath === target.path ? destinationPath : currentPath;
  }
  const prefix = `${target.path}/`;
  if (currentPath === target.path) return destinationPath;
  if (currentPath.startsWith(prefix)) {
    return `${destinationPath}/${currentPath.slice(prefix.length)}`;
  }
  return currentPath;
};

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useBacklogFileHandlers(opts: UseBacklogFileHandlersOptions) {
  const { data, setSelectedFile, setActiveTab, setSearchParams, selectedFile } = opts;
  const { _mutations } = data;

  const handleFileSelect = useCallback(
    (file: BacklogFile) => {
      if (file.type === "file") {
        setSelectedFile(file);
        setActiveTab("files");
        setSearchParams(
          (prev) => {
            const next = new URLSearchParams(prev);
            next.set("file", file.path);
            return next;
          },
          { replace: true },
        );
      }
    },
    [setSelectedFile, setActiveTab, setSearchParams],
  );

  const handleUploadComplete = useCallback(() => {
    data.invalidateFiles();
  }, [data]);

  const handleFileAction = useCallback(
    (action: FileActionType, target: BacklogFile, destinationPath?: string) => {
      _mutations.fileAction.mutate(
        { action, target, destinationPath },
        {
          onSuccess: (_result, variables) => {
            const currentSelectedPath = selectedFile?.path;
            if (!currentSelectedPath) return;

            if (variables.action === "delete") {
              const affectedPath =
                currentSelectedPath === variables.target.path ||
                (variables.target.type === "directory" &&
                  currentSelectedPath.startsWith(`${variables.target.path}/`));
              if (affectedPath) {
                setSelectedFile(null);
                setSearchParams(
                  (prev) => {
                    const next = new URLSearchParams(prev);
                    next.delete("file");
                    return next;
                  },
                  { replace: true },
                );
              }
              return;
            }

            if (!variables.destinationPath) return;
            const remapped = remapSelectedPath(
              currentSelectedPath,
              variables.target,
              variables.destinationPath,
            );
            if (!remapped || remapped === currentSelectedPath) return;
            setSearchParams(
              (prev) => {
                const next = new URLSearchParams(prev);
                next.set("file", remapped);
                return next;
              },
              { replace: true },
            );
          },
        },
      );
    },
    [_mutations.fileAction, selectedFile?.path, setSelectedFile, setSearchParams],
  );

  return {
    handleFileSelect,
    handleUploadComplete,
    handleFileAction,
  } as const;
}
