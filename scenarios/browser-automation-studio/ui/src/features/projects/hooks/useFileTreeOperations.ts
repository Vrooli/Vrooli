import { useCallback } from "react";
import { getConfig } from "@/config";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import type { FileTreeDragPayload } from "../FileTree";
import { useProjectDetailStore } from "./useProjectDetailStore";

/**
 * Hook providing file tree CRUD operations for a project
 */
export function useFileTreeOperations(projectId: string) {
  const fetchProjectEntries = useProjectDetailStore((s) => s.fetchProjectEntries);
  const fetchWorkflows = useProjectDetailStore((s) => s.fetchWorkflows);
  const setDragSourcePath = useProjectDetailStore((s) => s.setDragSourcePath);
  const setDropTargetFolder = useProjectDetailStore((s) => s.setDropTargetFolder);

  /**
   * Validates and normalizes a project-relative path
   */
  const normalizeProjectRelPath = useCallback((raw: string): { ok: true; path: string } | { ok: false; error: string } => {
    const trimmed = raw.trim().replace(/\\/g, "/");
    const withoutLeading = trimmed.replace(/^\/+/, "").replace(/\/+$/, "");
    if (!withoutLeading) {
      return { ok: false, error: "Path is required." };
    }
    const parts: string[] = withoutLeading.split("/");
    for (const part of parts) {
      const segment = part.trim();
      if (!segment || segment === "." || segment === "..") {
        return { ok: false, error: "Path contains invalid segments." };
      }
    }
    return {
      ok: true,
      path: parts.map((segment: string) => segment.trim()).join("/"),
    };
  }, []);

  /**
   * Extracts the filename from a path
   */
  const fileBasename = useCallback((relPath: string): string => {
    const normalized = relPath.replace(/\\/g, "/").replace(/^\/+/, "").replace(/\/+$/, "");
    const parts = normalized.split("/").filter(Boolean);
    return parts.length > 0 ? parts[parts.length - 1] : "";
  }, []);

  /**
   * Parses drag payload from a DataTransfer object
   */
  const parseDragPayload = useCallback(
    (dt: DataTransfer | null): FileTreeDragPayload | null => {
      if (!dt) return null;
      try {
        const raw = dt.getData("application/x-bas-project-entry");
        if (!raw) return null;
        const parsed = JSON.parse(raw) as Partial<FileTreeDragPayload>;
        if (!parsed || typeof parsed.path !== "string" || typeof parsed.kind !== "string") {
          return null;
        }
        if (
          parsed.kind !== "folder" &&
          parsed.kind !== "workflow_file" &&
          parsed.kind !== "asset_file"
        ) {
          return null;
        }
        return { path: parsed.path, kind: parsed.kind };
      } catch {
        return null;
      }
    },
    [],
  );

  /**
   * Creates a folder at the given path
   */
  const createFolder = useCallback(
    async (relPath: string): Promise<void> => {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${projectId}/files/mkdir`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ path: relPath }),
        },
      );
      if (!response.ok) {
        const msg = await response.text();
        throw new Error(msg || `Failed to create folder: ${response.status}`);
      }
    },
    [projectId],
  );

  /**
   * Creates a workflow file at the given path
   */
  const createWorkflowFile = useCallback(
    async (relPath: string, type: "action" | "flow" | "case", name?: string): Promise<string | undefined> => {
      const config = await getConfig();
      const inferredName =
        name?.trim() ||
        relPath
          .split("/")
          .pop()
          ?.replace(/\.action\.json$/i, "")
          .replace(/\.flow\.json$/i, "")
          .replace(/\.case\.json$/i, "") ||
        "workflow";
      const response = await fetch(
        `${config.API_URL}/projects/${projectId}/files/write`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            path: relPath,
            workflow: {
              name: inferredName,
              type,
              flow_definition: { nodes: [], edges: [] },
            },
          }),
        },
      );
      if (!response.ok) {
        const msg = await response.text();
        throw new Error(msg || `Failed to create workflow file: ${response.status}`);
      }
      const payload = (await response.json()) as {
        workflowId?: string;
        warnings?: string[];
      };
      if (payload.warnings && payload.warnings.length > 0) {
        toast(payload.warnings[0] ?? "Created with warnings");
      }
      return payload.workflowId;
    },
    [projectId],
  );

  /**
   * Moves a file or folder from one path to another
   */
  const moveFile = useCallback(
    async (fromPath: string, toPath: string): Promise<void> => {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${projectId}/files/move`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ from_path: fromPath, to_path: toPath }),
        },
      );
      if (!response.ok) {
        const msg = await response.text();
        throw new Error(msg || `Failed to move: ${response.status}`);
      }
    },
    [projectId],
  );

  /**
   * Deletes a file or folder at the given path
   */
  const deleteFile = useCallback(
    async (relPath: string): Promise<void> => {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${projectId}/files/delete`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ path: relPath }),
        },
      );
      if (!response.ok) {
        const msg = await response.text();
        throw new Error(msg || `Failed to delete: ${response.status}`);
      }
    },
    [projectId],
  );

  /**
   * Resyncs the project files from disk
   */
  const resyncFiles = useCallback(async (): Promise<void> => {
    try {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${projectId}/files/resync`,
        { method: "POST" },
      );
      if (!response.ok) {
        throw new Error(`Resync failed: ${response.status}`);
      }
      toast.success("Project files resynced");
      await fetchProjectEntries(projectId);
      await fetchWorkflows(projectId);
    } catch (error) {
      logger.error(
        "Failed to resync project files",
        { component: "useFileTreeOperations", action: "resyncFiles", projectId },
        error,
      );
      toast.error(error instanceof Error ? error.message : "Failed to resync");
      throw error;
    }
  }, [projectId, fetchProjectEntries, fetchWorkflows]);

  /**
   * Handles drop move operation for drag and drop
   */
  const handleDropMove = useCallback(
    async (e: React.DragEvent, targetFolderPath: string): Promise<void> => {
      e.preventDefault();
      e.stopPropagation();
      const payload = parseDragPayload(e.dataTransfer);
      setDropTargetFolder(null);
      setDragSourcePath(null);
      if (!payload) return;

      const sourcePath = payload.path;
      const baseName = fileBasename(sourcePath);
      if (!baseName) return;

      const destPath = targetFolderPath ? `${targetFolderPath}/${baseName}` : baseName;
      if (destPath === sourcePath) return;

      if (payload.kind === "folder") {
        if (targetFolderPath === sourcePath || targetFolderPath.startsWith(`${sourcePath}/`)) {
          toast.error("Cannot move a folder into itself.");
          return;
        }
      }

      try {
        await moveFile(sourcePath, destPath);
        toast.success("Moved");
        await fetchProjectEntries(projectId);
        await fetchWorkflows(projectId);
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Move failed");
      }
    },
    [fetchProjectEntries, fetchWorkflows, fileBasename, moveFile, parseDragPayload, projectId, setDragSourcePath, setDropTargetFolder],
  );

  /**
   * Refreshes both project entries and workflows
   */
  const refreshAll = useCallback(async (): Promise<void> => {
    await Promise.all([
      fetchProjectEntries(projectId),
      fetchWorkflows(projectId),
    ]);
  }, [projectId, fetchProjectEntries, fetchWorkflows]);

  return {
    // Path utilities
    normalizeProjectRelPath,
    fileBasename,
    parseDragPayload,

    // File operations
    createFolder,
    createWorkflowFile,
    moveFile,
    deleteFile,
    resyncFiles,

    // Drag and drop
    handleDropMove,

    // Refresh
    refreshAll,
  };
}
