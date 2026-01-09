import { useCallback, useMemo, useEffect, useRef, useState } from "react";
import {
  FolderOpen,
  FolderTree,
  WifiOff,
  RefreshCw,
  Loader,
  Plus,
  X,
  Workflow as WorkflowIcon,
  Upload,
} from "lucide-react";
import {
  FileTreeItem,
  type FileTreeNode,
} from "./FileTree";
import type { Project } from "./store";
import { useWorkflowStore, type Workflow } from "@stores/workflowStore";
import { useStartWorkflow } from "@/domains/executions";
import { useConfirmDialog } from "@hooks/useConfirmDialog";
import { usePromptDialog } from "@hooks/usePromptDialog";
import { ConfirmDialog, PromptDialog } from "@shared/ui";
import { useModals } from "@shared/modals/ModalContext";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import {
  useProjectDetailStore,
  type ProjectEntry,
} from "./hooks/useProjectDetailStore";
import { useFileTreeOperations } from "./hooks/useFileTreeOperations";
import { selectors } from "@constants/selectors";
import { WorkflowPreviewPane } from "./WorkflowPreviewPane";
import { EmptyWorkflowState } from "./EmptyWorkflowState";

interface ProjectFileTreeProps {
  project: Project;
  onWorkflowSelect: (workflow: Workflow) => Promise<void>;
  onCreateWorkflow: () => void;
  onCreateWorkflowDirect?: () => void;
  onStartRecording?: () => void;
}

/**
 * Tree view of project files with drag-drop, expand/collapse, and file operations
 */
export function ProjectFileTree({
  project,
  onWorkflowSelect,
  onCreateWorkflow,
  onCreateWorkflowDirect,
  onStartRecording,
}: ProjectFileTreeProps) {
  // Local state for inline add menu
  const [inlineAddMenuFolder, setInlineAddMenuFolder] = useState<string | null>(null);
  const [inlineAddMenuAnchor, setInlineAddMenuAnchor] = useState<{ x: number; y: number } | null>(null);
  const inlineAddMenuRef = useRef<HTMLDivElement>(null);
  const treeContainerRef = useRef<HTMLDivElement>(null);

  // Resizable divider state
  const TREE_MIN_WIDTH = 280;
  const TREE_MAX_WIDTH_PERCENT = 70;
  const TREE_DEFAULT_WIDTH_PERCENT = 50;
  const STORAGE_KEY_TREE_WIDTH = "project-file-tree-width-percent";

  const [treeWidthPercent, setTreeWidthPercent] = useState(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY_TREE_WIDTH);
      if (stored) {
        const parsed = parseFloat(stored);
        if (!isNaN(parsed) && parsed >= 30 && parsed <= TREE_MAX_WIDTH_PERCENT) {
          return parsed;
        }
      }
    } catch {
      // Ignore storage errors
    }
    return TREE_DEFAULT_WIDTH_PERCENT;
  });
  const [isResizingDivider, setIsResizingDivider] = useState(false);
  const resizeStartXRef = useRef(0);
  const resizeStartWidthRef = useRef(0);
  const containerRef = useRef<HTMLDivElement>(null);

  // Store state
  const projectEntries = useProjectDetailStore((s) => s.projectEntries);
  const projectEntriesLoading = useProjectDetailStore((s) => s.projectEntriesLoading);
  const projectEntriesError = useProjectDetailStore((s) => s.projectEntriesError);
  const searchTerm = useProjectDetailStore((s) => s.searchTerm);
  const expandedFolders = useProjectDetailStore((s) => s.expandedFolders);
  const selectedTreeFolder = useProjectDetailStore((s) => s.selectedTreeFolder);
  const dragSourcePath = useProjectDetailStore((s) => s.dragSourcePath);
  const dropTargetFolder = useProjectDetailStore((s) => s.dropTargetFolder);
  const focusedTreePath = useProjectDetailStore((s) => s.focusedTreePath);
  const previewWorkflowId = useProjectDetailStore((s) => s.previewWorkflowId);
  const workflows = useProjectDetailStore((s) => s.workflows);

  // Store actions
  const setSearchTerm = useProjectDetailStore((s) => s.setSearchTerm);
  const toggleFolder = useProjectDetailStore((s) => s.toggleFolder);
  const setSelectedTreeFolder = useProjectDetailStore((s) => s.setSelectedTreeFolder);
  const setDragSourcePath = useProjectDetailStore((s) => s.setDragSourcePath);
  const setDropTargetFolder = useProjectDetailStore((s) => s.setDropTargetFolder);
  const setFocusedTreePath = useProjectDetailStore((s) => s.setFocusedTreePath);
  const setPreviewWorkflowId = useProjectDetailStore((s) => s.setPreviewWorkflowId);
  const setExecutionInProgress = useProjectDetailStore((s) => s.setExecutionInProgress);
  const setActiveTab = useProjectDetailStore((s) => s.setActiveTab);
  const fetchProjectEntries = useProjectDetailStore((s) => s.fetchProjectEntries);
  const fetchWorkflows = useProjectDetailStore((s) => s.fetchWorkflows);

  // File operations
  const fileOps = useFileTreeOperations(project.id);

  // Modal context
  const { openWorkflowCreationModal, openAssetUploadModal } = useModals();

  // External stores
  const loadWorkflow = useWorkflowStore((state) => state.loadWorkflow);

  // Execution hook with start URL prompt
  const { startWorkflow, promptDialogProps: startUrlPromptProps } = useStartWorkflow({
    onSuccess: () => {
      setActiveTab("executions");
    },
  });

  // Get the previewed workflow details
  const previewedWorkflow = useMemo(() => {
    if (!previewWorkflowId) return null;
    return workflows.find((w) => w.id === previewWorkflowId) || null;
  }, [previewWorkflowId, workflows]);

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

  // Auto-expand first 2 levels of folders on initial load
  const hasAutoExpandedRef = useRef(false);
  useEffect(() => {
    if (hasAutoExpandedRef.current || memoizedFileTree.length === 0) return;
    hasAutoExpandedRef.current = true;

    const foldersToExpand: string[] = [""];  // Always expand root
    const collectFolders = (nodes: FileTreeNode[], depth: number) => {
      if (depth >= 2) return;  // Only expand depth 0 and 1
      for (const node of nodes) {
        if (node.kind === "folder") {
          foldersToExpand.push(node.path);
          if (node.children) {
            collectFolders(node.children, depth + 1);
          }
        }
      }
    };
    collectFolders(memoizedFileTree, 0);

    // Expand all collected folders
    for (const path of foldersToExpand) {
      if (!expandedFolders.has(path)) {
        toggleFolder(path);
      }
    }
  }, [memoizedFileTree, expandedFolders, toggleFolder]);

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
        const executionId = await startWorkflow({ workflowId });
        if (executionId) {
          logger.info("Workflow execution started", {
            component: "ProjectFileTree",
            action: "handleExecuteWorkflow",
            workflowId,
            executionId,
          });
        }
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
        toast.error("Failed to execute workflow. Please try again.");
      } finally {
        setExecutionInProgress(workflowId, false);
      }
    },
    [startWorkflow, setExecutionInProgress],
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

  // Handler for preview workflow (single click)
  const handlePreviewWorkflow = useCallback(
    (workflowId: string) => {
      setPreviewWorkflowId(workflowId);
    },
    [setPreviewWorkflowId],
  );

  // Handler for inline add menu - takes event to anchor popover
  const handleShowInlineAddMenu = useCallback(
    (folderPath: string, e?: React.MouseEvent) => {
      setInlineAddMenuFolder(folderPath);
      setSelectedTreeFolder(folderPath);
      // Capture anchor position from click event
      if (e) {
        const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
        setInlineAddMenuAnchor({ x: rect.left, y: rect.bottom + 4 });
      }
    },
    [setSelectedTreeFolder],
  );

  // Handler for creating folders from inline menu
  const handleCreateFolder = useCallback(
    async () => {
      const folderPath = inlineAddMenuFolder ?? "";
      setInlineAddMenuFolder(null);
      setInlineAddMenuAnchor(null);

      const suggested = folderPath === "" ? "new-folder" : `${folderPath}/new-folder`;
      const relPath = await requestPrompt(
        {
          title: "New Folder",
          label: "Folder path (relative to project root)",
          defaultValue: suggested,
          submitLabel: "Create Folder",
          cancelLabel: "Cancel",
        },
        {
          validate: (value) => {
            const normalized = fileOps.normalizeProjectRelPath(value);
            if (!normalized.ok) return normalized.error;
            return null;
          },
          normalize: (value) => {
            const normalized = fileOps.normalizeProjectRelPath(value);
            return normalized.ok ? normalized.path : value.trim();
          },
        },
      );
      if (!relPath) return;
      try {
        await fileOps.createFolder(relPath);
        toast.success("Folder created");
        await fetchProjectEntries(project.id);
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Failed to create folder");
      }
    },
    [inlineAddMenuFolder, requestPrompt, fileOps, fetchProjectEntries, project.id],
  );

  // Handler for creating workflow from inline menu - opens professional modal
  const handleCreateWorkflow = useCallback(
    () => {
      setInlineAddMenuFolder(null);
      setInlineAddMenuAnchor(null);
      openWorkflowCreationModal();
    },
    [openWorkflowCreationModal],
  );

  // Handler for uploading assets from inline menu
  const handleUploadAsset = useCallback(
    () => {
      const folder = inlineAddMenuFolder ?? "";
      setInlineAddMenuFolder(null);
      setInlineAddMenuAnchor(null);
      openAssetUploadModal({ folder, projectId: project.id });
    },
    [inlineAddMenuFolder, openAssetUploadModal, project.id],
  );

  // Build flat list of visible paths for keyboard navigation
  const visiblePaths = useMemo(() => {
    const paths: { path: string; workflowId?: string; kind: string }[] = [];

    const collectPaths = (nodes: FileTreeNode[]) => {
      for (const node of nodes) {
        paths.push({ path: node.path, workflowId: node.workflowId, kind: node.kind });
        if (node.kind === "folder" && expandedFolders.has(node.path) && node.children) {
          collectPaths(node.children);
        }
      }
    };

    collectPaths(filteredFileTree);
    return paths;
  }, [filteredFileTree, expandedFolders]);

  // Keyboard navigation handler
  useEffect(() => {
    const container = treeContainerRef.current;
    if (!container) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      // Only handle if tree container or its children are focused
      if (!container.contains(document.activeElement) && document.activeElement !== container) {
        return;
      }

      const currentIndex = focusedTreePath
        ? visiblePaths.findIndex((p) => p.path === focusedTreePath)
        : -1;

      switch (e.key) {
        case "ArrowDown": {
          e.preventDefault();
          const nextIndex = currentIndex < visiblePaths.length - 1 ? currentIndex + 1 : 0;
          const nextNode = visiblePaths[nextIndex];
          if (nextNode) {
            setFocusedTreePath(nextNode.path);
            // Auto-update preview pane for workflow files
            if (nextNode.kind === "workflow_file" && nextNode.workflowId) {
              setPreviewWorkflowId(nextNode.workflowId);
            }
          }
          break;
        }
        case "ArrowUp": {
          e.preventDefault();
          const prevIndex = currentIndex > 0 ? currentIndex - 1 : visiblePaths.length - 1;
          const prevNode = visiblePaths[prevIndex];
          if (prevNode) {
            setFocusedTreePath(prevNode.path);
            // Auto-update preview pane for workflow files
            if (prevNode.kind === "workflow_file" && prevNode.workflowId) {
              setPreviewWorkflowId(prevNode.workflowId);
            }
          }
          break;
        }
        case "ArrowRight": {
          e.preventDefault();
          if (focusedTreePath) {
            const node = visiblePaths.find((p) => p.path === focusedTreePath);
            if (node?.kind === "folder" && !expandedFolders.has(focusedTreePath)) {
              toggleFolder(focusedTreePath);
            }
          }
          break;
        }
        case "ArrowLeft": {
          e.preventDefault();
          if (focusedTreePath) {
            const node = visiblePaths.find((p) => p.path === focusedTreePath);
            if (node?.kind === "folder" && expandedFolders.has(focusedTreePath)) {
              toggleFolder(focusedTreePath);
            }
          }
          break;
        }
        case "Enter": {
          e.preventDefault();
          if (focusedTreePath) {
            const node = visiblePaths.find((p) => p.path === focusedTreePath);
            if (node?.kind === "folder") {
              toggleFolder(focusedTreePath);
              setSelectedTreeFolder(focusedTreePath);
            } else if (node?.workflowId) {
              void handleOpenWorkflowFile(node.workflowId);
            }
          }
          break;
        }
        case " ": {
          e.preventDefault();
          if (focusedTreePath) {
            const node = visiblePaths.find((p) => p.path === focusedTreePath);
            if (node?.workflowId) {
              setPreviewWorkflowId(node.workflowId);
            }
          }
          break;
        }
        case "Escape": {
          e.preventDefault();
          setFocusedTreePath(null);
          setPreviewWorkflowId(null);
          setInlineAddMenuFolder(null);
          setInlineAddMenuAnchor(null);
          break;
        }
      }
    };

    container.addEventListener("keydown", handleKeyDown);
    return () => container.removeEventListener("keydown", handleKeyDown);
  }, [
    focusedTreePath,
    visiblePaths,
    expandedFolders,
    toggleFolder,
    setFocusedTreePath,
    setSelectedTreeFolder,
    setPreviewWorkflowId,
    handleOpenWorkflowFile,
  ]);

  // Close inline add menu when clicking outside
  useEffect(() => {
    if (!inlineAddMenuFolder) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (inlineAddMenuRef.current && !inlineAddMenuRef.current.contains(e.target as Node)) {
        setInlineAddMenuFolder(null);
        setInlineAddMenuAnchor(null);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [inlineAddMenuFolder]);

  // Resize divider start handler
  const handleDividerResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizingDivider(true);
    resizeStartXRef.current = e.clientX;
    resizeStartWidthRef.current = treeWidthPercent;
  }, [treeWidthPercent]);

  // Resize divider mouse move and up effect
  useEffect(() => {
    if (!isResizingDivider) return;

    const handleMouseMove = (e: MouseEvent) => {
      const container = containerRef.current;
      if (!container) return;

      const containerRect = container.getBoundingClientRect();
      const containerWidth = containerRect.width;
      const deltaX = e.clientX - resizeStartXRef.current;
      const deltaPercent = (deltaX / containerWidth) * 100;
      const newPercent = Math.min(
        TREE_MAX_WIDTH_PERCENT,
        Math.max(30, resizeStartWidthRef.current + deltaPercent)
      );

      // Enforce minimum pixel width
      const pixelWidth = (newPercent / 100) * containerWidth;
      if (pixelWidth >= TREE_MIN_WIDTH) {
        setTreeWidthPercent(newPercent);
      }
    };

    const handleMouseUp = () => {
      setIsResizingDivider(false);
      // Persist width
      try {
        localStorage.setItem(STORAGE_KEY_TREE_WIDTH, String(treeWidthPercent));
      } catch {
        // Ignore storage errors
      }
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizingDivider, treeWidthPercent]);

  // Render loading state
  if (projectEntriesLoading) {
    return (
      <div className="flex-1 overflow-auto p-6" data-testid={selectors.projects.fileTree.container}>
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
      <div className="flex-1 overflow-auto p-6" data-testid={selectors.projects.fileTree.container}>
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
      <EmptyWorkflowState
        error={null}
        onCreateWorkflow={onCreateWorkflow}
        onCreateWorkflowDirect={onCreateWorkflowDirect}
        onStartRecording={onStartRecording}
      />
    );
  }

  // Render no results state
  if (filteredFileTree.length === 0) {
    return (
      <div className="flex-1 overflow-auto p-6" data-testid={selectors.projects.fileTree.container}>
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

  // Render file tree with preview pane
  return (
    <div
      ref={containerRef}
      className="flex-1 flex overflow-hidden relative"
      data-testid={selectors.projects.fileTree.container}
    >
      {/* Left: File Tree */}
      <div
        ref={treeContainerRef}
        className="overflow-auto p-4 flex-shrink-0"
        style={{ width: previewedWorkflow ? `${treeWidthPercent}%` : "100%" }}
        tabIndex={0}
      >
        <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
          <div className="space-y-0.5">
            {/* Root folder */}
            <div
              data-testid={selectors.projects.fileTree.root}
              className={`flex items-center gap-2 px-2 py-2 rounded cursor-pointer transition-colors group ${
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
              onDragLeave={(e) => {
                e.preventDefault();
                e.stopPropagation();
                const relatedTarget = e.relatedTarget as Node | null;
                const currentTarget = e.currentTarget as HTMLElement;
                if (!relatedTarget || !currentTarget.contains(relatedTarget)) {
                  setDropTargetFolder(null);
                }
              }}
              onDrop={(e) => {
                void fileOps.handleDropMove(e, "");
              }}
            >
              <FolderOpen size={16} className="text-yellow-500 flex-shrink-0" />
              <span className="text-base text-gray-300">{project.name}</span>
              <span className="ml-1 text-xs px-1.5 py-0.5 rounded bg-gray-800 text-gray-400 flex-shrink-0">
                root
              </span>
              {/* Add button for root */}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleShowInlineAddMenu("", e);
                }}
                className="ml-3 p-1 rounded text-gray-500 hover:text-gray-300 hover:bg-gray-700 opacity-0 group-hover:opacity-100 transition-all"
                title="Add new item"
              >
                <Plus size={14} />
              </button>
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
                  depth={0}
                  expandedFolders={expandedFolders}
                  selectedTreeFolder={selectedTreeFolder}
                  dragSourcePath={dragSourcePath}
                  dropTargetFolder={dropTargetFolder}
                  focusedTreePath={focusedTreePath}
                  onToggleFolder={toggleFolder}
                  onSelectFolder={setSelectedTreeFolder}
                  onSetDragSourcePath={setDragSourcePath}
                  onSetDropTargetFolder={setDropTargetFolder}
                  onDropMove={fileOps.handleDropMove}
                  onPreviewWorkflow={handlePreviewWorkflow}
                  onOpenWorkflowFile={handleOpenWorkflowFile}
                  onExecuteWorkflow={handleExecuteWorkflow}
                  onDeleteNode={handleDeleteTreeNode}
                  onRenameNode={handleRenameTreeNode}
                  onShowInlineAddMenu={handleShowInlineAddMenu}
                />
              );
            })}
          </div>
        </div>

        {/* Inline add menu popup */}
        {inlineAddMenuFolder !== null && inlineAddMenuAnchor && (
          <div
            ref={inlineAddMenuRef}
            className="fixed z-50 bg-flow-node border border-gray-700 rounded-lg shadow-xl overflow-hidden"
            style={{
              top: inlineAddMenuAnchor.y,
              left: inlineAddMenuAnchor.x,
            }}
          >
            <div className="px-4 py-2 border-b border-gray-700 flex items-center justify-between">
              <span className="text-sm font-medium text-gray-300">
                Add to {inlineAddMenuFolder || "root"}
              </span>
              <button
                onClick={() => {
                  setInlineAddMenuFolder(null);
                  setInlineAddMenuAnchor(null);
                }}
                className="p-1 text-gray-500 hover:text-gray-300 rounded transition-colors"
              >
                <X size={14} />
              </button>
            </div>
            <div className="p-2">
              <button
                onClick={handleCreateFolder}
                className="w-full flex items-center gap-3 px-3 py-2 text-left text-gray-300 hover:bg-gray-700 rounded transition-colors"
              >
                <FolderOpen size={16} className="text-yellow-500" />
                <span className="text-sm">New Folder</span>
              </button>
              <button
                onClick={handleCreateWorkflow}
                className="w-full flex items-center gap-3 px-3 py-2 text-left text-gray-300 hover:bg-gray-700 rounded transition-colors"
              >
                <WorkflowIcon size={16} className="text-blue-400" />
                <span className="text-sm">New Workflow</span>
              </button>
              <button
                onClick={handleUploadAsset}
                className="w-full flex items-center gap-3 px-3 py-2 text-left text-gray-300 hover:bg-gray-700 rounded transition-colors"
              >
                <Upload size={16} className="text-green-400" />
                <span className="text-sm">Upload Asset</span>
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Resize Handle */}
      {previewedWorkflow && (
        <div
          className={`w-1 flex-shrink-0 cursor-col-resize hover:bg-flow-accent/50 active:bg-flow-accent transition-colors ${
            isResizingDivider ? "bg-flow-accent" : "bg-gray-700"
          }`}
          onMouseDown={handleDividerResizeStart}
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize preview pane"
        />
      )}

      {/* Right: Preview Pane */}
      {previewedWorkflow && (
        <div className="flex-1 overflow-hidden p-4">
          <WorkflowPreviewPane
            workflow={previewedWorkflow}
            onClose={() => setPreviewWorkflowId(null)}
            onOpenEditor={handleOpenWorkflowFile}
          />
        </div>
      )}

      <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
      <PromptDialog
        state={promptDialogState}
        onValueChange={setPromptValue}
        onClose={closePromptDialog}
        onSubmit={submitPrompt}
      />
      {/* Start URL prompt for workflows without navigate step */}
      <PromptDialog {...startUrlPromptProps} />
    </div>
  );
}
