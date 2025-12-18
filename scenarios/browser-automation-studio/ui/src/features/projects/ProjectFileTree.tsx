import { useCallback, useMemo } from "react";
import { FolderOpen, FolderTree, WifiOff, RefreshCw, Loader } from "lucide-react";
import {
  FileTreeItem,
  type FileTreeNode,
} from "./FileTree";
import type { Project } from "@stores/projectStore";
import { useWorkflowStore, type Workflow } from "@stores/workflowStore";
import { useExecutionStore } from "@stores/executionStore";
import { useConfirmDialog } from "@hooks/useConfirmDialog";
import { usePromptDialog } from "@hooks/usePromptDialog";
import { ConfirmDialog, PromptDialog } from "@shared/ui";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import {
  useProjectDetailStore,
  type ProjectEntry,
} from "./hooks/useProjectDetailStore";
import { useFileTreeOperations } from "./hooks/useFileTreeOperations";

interface ProjectFileTreeProps {
  project: Project;
  onWorkflowSelect: (workflow: Workflow) => Promise<void>;
}

/**
 * Tree view of project files with drag-drop, expand/collapse, and file operations
 */
export function ProjectFileTree({
  project,
  onWorkflowSelect,
}: ProjectFileTreeProps) {
  // Store state
  const projectEntries = useProjectDetailStore((s) => s.projectEntries);
  const projectEntriesLoading = useProjectDetailStore((s) => s.projectEntriesLoading);
  const projectEntriesError = useProjectDetailStore((s) => s.projectEntriesError);
  const searchTerm = useProjectDetailStore((s) => s.searchTerm);
  const expandedFolders = useProjectDetailStore((s) => s.expandedFolders);
  const selectedTreeFolder = useProjectDetailStore((s) => s.selectedTreeFolder);
  const dragSourcePath = useProjectDetailStore((s) => s.dragSourcePath);
  const dropTargetFolder = useProjectDetailStore((s) => s.dropTargetFolder);

  // Store actions
  const setSearchTerm = useProjectDetailStore((s) => s.setSearchTerm);
  const toggleFolder = useProjectDetailStore((s) => s.toggleFolder);
  const setSelectedTreeFolder = useProjectDetailStore((s) => s.setSelectedTreeFolder);
  const setDragSourcePath = useProjectDetailStore((s) => s.setDragSourcePath);
  const setDropTargetFolder = useProjectDetailStore((s) => s.setDropTargetFolder);
  const setExecutionInProgress = useProjectDetailStore((s) => s.setExecutionInProgress);
  const setActiveTab = useProjectDetailStore((s) => s.setActiveTab);
  const fetchProjectEntries = useProjectDetailStore((s) => s.fetchProjectEntries);
  const fetchWorkflows = useProjectDetailStore((s) => s.fetchWorkflows);

  // File operations
  const fileOps = useFileTreeOperations(project.id);

  // External stores
  const loadWorkflow = useWorkflowStore((state) => state.loadWorkflow);
  const startExecution = useExecutionStore((state) => state.startExecution);

  // Dialog hooks
  const {
    dialogState: confirmDialogState,
    confirm: requestConfirm,
    close: closeConfirmDialog,
  } = useConfirmDialog();
  const {
    dialogState: promptDialogState,
    prompt: requestPrompt,
    setValue: setPromptValue,
    close: closePromptDialog,
    submit: submitPrompt,
  } = usePromptDialog();

  // Build file tree from project entries
  const buildFileTree = useCallback((entries: ProjectEntry[]): FileTreeNode[] => {
    const folderMap = new Map<string, FileTreeNode>();

    const ensureFolder = (folderPath: string): FileTreeNode => {
      if (folderMap.has(folderPath)) {
        return folderMap.get(folderPath)!;
      }
      const name = folderPath === "" ? "root" : folderPath.split("/").pop()!;
      const node: FileTreeNode = {
        kind: "folder",
        path: folderPath,
        name,
        children: [],
      };
      folderMap.set(folderPath, node);
      return node;
    };

    // Always create root
    ensureFolder("");

    for (const entry of entries) {
      if (!entry?.path || typeof entry.path !== "string") continue;
      const relPath = entry.path.replace(/^\/+/, "");
      const parts = relPath.split("/").filter(Boolean);
      if (parts.length === 0) continue;
      const isDir = entry.kind === "folder";

      let current = "";
      for (let i = 0; i < parts.length - (isDir ? 0 : 1); i++) {
        current = current ? `${current}/${parts[i]}` : parts[i];
        ensureFolder(current);
      }

      if (isDir) {
        ensureFolder(relPath);
      } else {
        const parentPath = parts.length > 1 ? parts.slice(0, -1).join("/") : "";
        const parent = ensureFolder(parentPath);
        parent.children = parent.children ?? [];
        parent.children.push({
          kind: entry.kind,
          path: relPath,
          name: parts[parts.length - 1],
          workflowId: entry.workflow_id,
          metadata: entry.metadata,
        });
      }
    }

    // Attach folder children to parents
    const folders = Array.from(folderMap.values());
    folders.sort((a, b) => a.path.localeCompare(b.path));
    for (const folder of folders) {
      if (folder.path === "") continue;
      const parentPath = folder.path.includes("/")
        ? folder.path.split("/").slice(0, -1).join("/")
        : "";
      const parent = ensureFolder(parentPath);
      parent.children = parent.children ?? [];
      if (!parent.children.find((c) => c.kind === "folder" && c.path === folder.path)) {
        parent.children.push(folder);
      }
    }

    const sortChildren = (node: FileTreeNode) => {
      if (!node.children) return;
      node.children.sort((a, b) => {
        if (a.kind !== b.kind) {
          if (a.kind === "folder") return -1;
          if (b.kind === "folder") return 1;
          if (a.kind === "workflow_file") return -1;
          if (b.kind === "workflow_file") return 1;
        }
        return a.name.localeCompare(b.name);
      });
      node.children.forEach(sortChildren);
    };

    const root = ensureFolder("");
    sortChildren(root);
    return root.children ?? [];
  }, []);

  const memoizedFileTree = useMemo(() => {
    return buildFileTree(projectEntries);
  }, [buildFileTree, projectEntries]);

  const filteredFileTree = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    if (!term) return memoizedFileTree;

    const filterNodes = (nodes: FileTreeNode[]): FileTreeNode[] => {
      const out: FileTreeNode[] = [];
      for (const node of nodes) {
        const matches =
          node.name.toLowerCase().includes(term) ||
          node.path.toLowerCase().includes(term);
        if (node.kind === "folder") {
          const children = filterNodes(node.children ?? []);
          if (matches || children.length > 0) {
            out.push({ ...node, children });
          }
        } else if (matches) {
          out.push(node);
        }
      }
      return out;
    };

    return filterNodes(memoizedFileTree);
  }, [memoizedFileTree, searchTerm]);

  // Handlers
  const handleOpenWorkflowFile = useCallback(
    async (workflowId: string) => {
      await loadWorkflow(workflowId);
      const wf = useWorkflowStore.getState().currentWorkflow;
      if (wf) {
        await onWorkflowSelect(wf);
      }
    },
    [loadWorkflow, onWorkflowSelect],
  );

  const handleExecuteWorkflow = useCallback(
    async (e: React.MouseEvent, workflowId: string) => {
      e.stopPropagation();
      setExecutionInProgress(workflowId, true);

      try {
        await startExecution(workflowId);
        setActiveTab("executions");
        logger.info("Workflow execution started", {
          component: "ProjectFileTree",
          action: "handleExecuteWorkflow",
          workflowId,
        });
      } catch (error) {
        logger.error(
          "Failed to execute workflow",
          {
            component: "ProjectFileTree",
            action: "handleExecuteWorkflow",
            workflowId,
          },
          error,
        );
        alert("Failed to execute workflow. Please try again.");
      } finally {
        setExecutionInProgress(workflowId, false);
      }
    },
    [startExecution, setExecutionInProgress, setActiveTab],
  );

  const handleDeleteTreeNode = useCallback(
    async (path: string) => {
      const confirmed = await requestConfirm({
        title: "Delete?",
        message: `Delete "${path}"? This cannot be undone.`,
        confirmLabel: "Delete",
        cancelLabel: "Cancel",
        danger: true,
      });
      if (!confirmed) return;
      try {
        await fileOps.deleteFile(path);
        toast.success("Deleted");
        await fetchProjectEntries(project.id);
        await fetchWorkflows(project.id);
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Delete failed");
      }
    },
    [requestConfirm, fileOps, fetchProjectEntries, fetchWorkflows, project.id],
  );

  const handleRenameTreeNode = useCallback(
    async (path: string) => {
      const next = await requestPrompt(
        {
          title: "Move / Rename",
          label: "New path (relative to project root)",
          defaultValue: path,
          submitLabel: "Move",
          cancelLabel: "Cancel",
        },
        {
          validate: (value) => {
            const normalized = fileOps.normalizeProjectRelPath(value);
            if (!normalized.ok) return normalized.error;
            if (normalized.path === path) return "Path must be different.";
            return null;
          },
          normalize: (value) => {
            const normalized = fileOps.normalizeProjectRelPath(value);
            return normalized.ok ? normalized.path : value.trim();
          },
        },
      );
      if (!next) return;
      try {
        await fileOps.moveFile(path, next);
        toast.success("Moved");
        await fetchProjectEntries(project.id);
        await fetchWorkflows(project.id);
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Move failed");
      }
    },
    [requestPrompt, fileOps, fetchProjectEntries, fetchWorkflows, project.id],
  );

  // Render loading state
  if (projectEntriesLoading) {
    return (
      <div className="flex-1 overflow-auto p-6">
        <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
          <div className="text-center text-gray-400 py-8">
            <Loader size={24} className="mx-auto mb-3 animate-spin" />
            <p>Loading project files…</p>
          </div>
        </div>
      </div>
    );
  }

  // Render error state
  if (projectEntriesError) {
    return (
      <div className="flex-1 overflow-auto p-6">
        <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
          <div className="text-center text-gray-400 py-8">
            <WifiOff size={40} className="mx-auto mb-3 opacity-50" />
            <p className="mb-3">{projectEntriesError}</p>
            <button
              onClick={() => fetchProjectEntries(project.id)}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-flow-accent text-white hover:bg-blue-600 transition-colors"
            >
              <RefreshCw size={16} />
              <span>Retry</span>
            </button>
          </div>
        </div>
      </div>
    );
  }

  // Render empty state
  if (memoizedFileTree.length === 0) {
    return (
      <div className="flex-1 overflow-auto p-6">
        <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
          <div className="text-center text-gray-400 py-8">
            <FolderTree size={48} className="mx-auto mb-4 opacity-50" />
            <p>No files indexed yet.</p>
            <p className="text-sm text-gray-500 mt-2">
              Use "New" to create folders/files, or resync from disk.
            </p>
          </div>
        </div>
      </div>
    );
  }

  // Render no results state
  if (filteredFileTree.length === 0) {
    return (
      <div className="flex-1 overflow-auto p-6">
        <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
          <div className="text-center text-gray-400 py-8">
            <FolderTree size={48} className="mx-auto mb-4 opacity-50" />
            <p>No files match &ldquo;{searchTerm}&rdquo;.</p>
            <button
              onClick={() => setSearchTerm("")}
              className="mt-3 text-flow-accent hover:text-blue-400 text-sm font-medium transition-colors"
            >
              Clear search
            </button>
          </div>
        </div>
      </div>
    );
  }

  // Render file tree
  return (
    <div className="flex-1 overflow-auto p-6">
      <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
        <div className="space-y-1">
          {/* Root folder */}
          <div
            className={`flex items-center gap-1.5 px-2 py-1.5 rounded cursor-pointer transition-colors ${
              selectedTreeFolder === "" ? "bg-flow-node" : "hover:bg-flow-node"
            } ${
              dragSourcePath !== null && dropTargetFolder === ""
                ? "ring-1 ring-flow-accent bg-flow-node"
                : ""
            }`}
            onClick={() => setSelectedTreeFolder("")}
            onDragOver={(e) => {
              e.preventDefault();
              e.stopPropagation();
              e.dataTransfer.dropEffect = "move";
              setDropTargetFolder("");
            }}
            onDrop={(e) => {
              void fileOps.handleDropMove(e, "");
            }}
          >
            <FolderOpen size={14} className="text-yellow-500" />
            <span className="text-sm text-gray-300">{project.name}</span>
            <span className="ml-2 text-[11px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-400">
              root
            </span>
          </div>

          {/* Tree nodes */}
          {filteredFileTree.map((node, index) => {
            const hasNextRoot = index < filteredFileTree.length - 1;
            const rootPrefix = filteredFileTree.length > 1 ? [hasNextRoot] : [];
            return (
              <FileTreeItem
                key={`${node.kind}:${node.path}`}
                node={node}
                prefixParts={rootPrefix}
                expandedFolders={expandedFolders}
                selectedTreeFolder={selectedTreeFolder}
                dragSourcePath={dragSourcePath}
                dropTargetFolder={dropTargetFolder}
                onToggleFolder={toggleFolder}
                onSelectFolder={setSelectedTreeFolder}
                onSetDragSourcePath={setDragSourcePath}
                onSetDropTargetFolder={setDropTargetFolder}
                onDropMove={fileOps.handleDropMove}
                onOpenWorkflowFile={handleOpenWorkflowFile}
                onExecuteWorkflow={handleExecuteWorkflow}
                onDeleteNode={handleDeleteTreeNode}
                onRenameNode={handleRenameTreeNode}
              />
            );
          })}
        </div>
      </div>

      <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
      <PromptDialog
        state={promptDialogState}
        onValueChange={setPromptValue}
        onClose={closePromptDialog}
        onSubmit={submitPrompt}
      />
    </div>
  );
}
