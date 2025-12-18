/**
 * FileTreeItem Component
 *
 * Renders a single item in the file tree (folder or file) with:
 * - Drag and drop support
 * - Expand/collapse for folders
 * - Actions (run, rename, delete) for files
 * - Recursive rendering of children
 *
 * Extracted from ProjectDetail.tsx for better separation of concerns.
 */
import { ReactNode } from "react";
import {
  Play,
  ChevronRight,
  ChevronDown,
  Trash2,
  PencilLine,
  ListChecks,
  FolderOpen,
  FileCode,
  FileText,
} from "lucide-react";
import toast from "react-hot-toast";

// ============================================================================
// Types
// ============================================================================

export type FileTreeNodeKind = "folder" | "workflow_file" | "asset_file";

export interface FileTreeNode {
  kind: FileTreeNodeKind;
  path: string;
  name: string;
  children?: FileTreeNode[];
  workflowId?: string;
  metadata?: Record<string, unknown>;
}

export type FileTreeDragPayload = {
  path: string;
  kind: FileTreeNodeKind;
};

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Renders the tree prefix characters (├──, └──, │) for visual hierarchy
 */
export function renderTreePrefix(prefixParts: boolean[]): ReactNode {
  if (prefixParts.length === 0) {
    return null;
  }

  const prefix = prefixParts
    .map((hasSibling, index) => {
      const isLast = index === prefixParts.length - 1;
      if (isLast) {
        return hasSibling ? "├── " : "└── ";
      }
      return hasSibling ? "│   " : "    ";
    })
    .join("");

  return (
    <span
      aria-hidden="true"
      className="font-mono text-[11px] leading-4 text-gray-500 whitespace-pre pointer-events-none select-none"
    >
      {prefix}
    </span>
  );
}

/**
 * Gets a human-readable label for workflow file types based on extension
 */
export function fileTypeLabelFromPath(relPath: string): string | null {
  const normalized = relPath.toLowerCase();
  if (normalized.endsWith(".action.json")) return "Action";
  if (normalized.endsWith(".flow.json")) return "Flow";
  if (normalized.endsWith(".case.json")) return "Case";
  return null;
}

/**
 * Returns the appropriate icon component for a file tree node
 */
export function fileKindIcon(node: FileTreeNode): ReactNode {
  if (node.kind === "folder") {
    return <FolderOpen size={14} className="text-yellow-500" />;
  }
  if (node.kind === "workflow_file") {
    return <FileCode size={14} className="text-green-400" />;
  }
  return <FileText size={14} className="text-gray-400" />;
}

// ============================================================================
// Component Props
// ============================================================================

export interface FileTreeItemProps {
  /** The node to render */
  node: FileTreeNode;
  /** Prefix parts for rendering tree hierarchy lines */
  prefixParts?: boolean[];

  // State from parent
  /** Set of expanded folder paths */
  expandedFolders: Set<string>;
  /** Currently selected folder path */
  selectedTreeFolder: string | null;
  /** Path of node being dragged (for drop target highlighting) */
  dragSourcePath: string | null;
  /** Path of current drop target folder */
  dropTargetFolder: string | null;

  // Callbacks
  /** Toggle folder expansion */
  onToggleFolder: (path: string) => void;
  /** Set the selected folder */
  onSelectFolder: (path: string) => void;
  /** Set drag source path */
  onSetDragSourcePath: (path: string | null) => void;
  /** Set drop target folder */
  onSetDropTargetFolder: (path: string | null) => void;
  /** Handle drop for move operation */
  onDropMove: (e: React.DragEvent, targetFolderPath: string) => void;
  /** Open a workflow file */
  onOpenWorkflowFile: (workflowId: string) => Promise<void>;
  /** Execute a workflow */
  onExecuteWorkflow: (e: React.MouseEvent, workflowId: string) => Promise<void>;
  /** Delete a node (file or folder) */
  onDeleteNode: (path: string) => Promise<void>;
  /** Rename/move a node */
  onRenameNode: (path: string) => Promise<void>;
}

// ============================================================================
// Component
// ============================================================================

/**
 * FileTreeItem - renders a single file tree node with actions and recursion
 */
export function FileTreeItem({
  node,
  prefixParts = [],
  expandedFolders,
  selectedTreeFolder,
  dragSourcePath,
  dropTargetFolder,
  onToggleFolder,
  onSelectFolder,
  onSetDragSourcePath,
  onSetDropTargetFolder,
  onDropMove,
  onOpenWorkflowFile,
  onExecuteWorkflow,
  onDeleteNode,
  onRenameNode,
}: FileTreeItemProps) {
  const isFolder = node.kind === "folder";
  const isExpanded = isFolder && expandedFolders.has(node.path);
  const children = node.children ?? [];
  const hasChildren = isFolder && children.length > 0;
  const typeLabel = node.kind === "workflow_file" ? fileTypeLabelFromPath(node.path) : null;
  const isDropTarget = Boolean(
    dragSourcePath !== null && dropTargetFolder !== null && dropTargetFolder === node.path
  );

  const handleToggle = () => {
    if (!isFolder) return;
    onToggleFolder(node.path);
  };

  const handleOpenWorkflowFile = async () => {
    if (node.kind !== "workflow_file" || !node.workflowId) {
      return;
    }
    try {
      await onOpenWorkflowFile(node.workflowId);
    } catch {
      toast.error("Failed to open workflow");
    }
  };

  const handleDeleteNode = async (e: React.MouseEvent) => {
    e.stopPropagation();
    await onDeleteNode(node.path);
  };

  const handleRenameNode = async (e: React.MouseEvent) => {
    e.stopPropagation();
    await onRenameNode(node.path);
  };

  const handleExecuteFromTree = async (e: React.MouseEvent) => {
    if (node.kind !== "workflow_file" || !node.workflowId) {
      e.stopPropagation();
      return;
    }
    await onExecuteWorkflow(e, node.workflowId);
  };

  const handleDragStart = (e: React.DragEvent) => {
    e.stopPropagation();
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData(
      "application/x-bas-project-entry",
      JSON.stringify({ path: node.path, kind: node.kind } as FileTreeDragPayload)
    );
    onSetDragSourcePath(node.path);
  };

  const handleDragEnd = () => {
    onSetDragSourcePath(null);
    onSetDropTargetFolder(null);
  };

  const handleDragOver = (e: React.DragEvent) => {
    if (!isFolder) return;
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = "move";
    onSetDropTargetFolder(node.path);
  };

  const handleDrop = (e: React.DragEvent) => {
    if (!isFolder) return;
    onDropMove(e, node.path);
  };

  const handleClick = () => {
    if (isFolder) {
      onSelectFolder(node.path);
      handleToggle();
      return;
    }
    if (node.kind === "workflow_file") {
      void handleOpenWorkflowFile();
      return;
    }
    toast("Assets are read-only in v1");
  };

  return (
    <div className="select-none">
      <div
        draggable={node.path !== ""}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
        className={`flex items-center gap-1.5 px-2 py-1.5 rounded cursor-pointer transition-colors group ${
          isFolder && selectedTreeFolder === node.path ? "bg-flow-node" : "hover:bg-flow-node"
        } ${isDropTarget ? "ring-1 ring-flow-accent bg-flow-node" : ""}`}
        onClick={handleClick}
      >
        {renderTreePrefix(prefixParts)}
        {isFolder ? (
          hasChildren ? (
            isExpanded ? (
              <ChevronDown size={14} className="text-gray-400" />
            ) : (
              <ChevronRight size={14} className="text-gray-400" />
            )
          ) : (
            <span className="inline-block w-3.5" aria-hidden="true" />
          )
        ) : (
          <span className="inline-block w-3.5" aria-hidden="true" />
        )}
        {fileKindIcon(node)}
        <span className="text-sm text-gray-300">
          {node.name}
          {typeLabel ? (
            <span className="ml-2 text-[11px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-400">
              {typeLabel}
            </span>
          ) : null}
        </span>

        <div className="ml-auto flex items-center gap-2 opacity-100 md:opacity-0 md:group-hover:opacity-100 transition-opacity">
          {node.kind === "workflow_file" && node.workflowId ? (
            <button
              onClick={handleExecuteFromTree}
              className="text-subtle hover:text-surface"
              title={typeLabel === "Case" ? "Test" : "Run"}
              aria-label={typeLabel === "Case" ? "Test workflow" : "Run workflow"}
            >
              {typeLabel === "Case" ? <ListChecks size={14} /> : <Play size={14} />}
            </button>
          ) : null}
          <button
            onClick={handleRenameNode}
            className="text-subtle hover:text-surface"
            title="Rename / Move"
            aria-label="Rename / Move"
          >
            <PencilLine size={14} />
          </button>
          <button
            onClick={handleDeleteNode}
            className="text-subtle hover:text-red-300"
            title="Delete"
            aria-label="Delete"
          >
            <Trash2 size={14} />
          </button>
        </div>
      </div>

      {isFolder &&
        isExpanded &&
        children.map((child, index) => {
          const isLastChild = index === children.length - 1;
          const childPrefix = [...prefixParts, !isLastChild];
          return (
            <FileTreeItem
              key={`${child.kind}:${child.path}`}
              node={child}
              prefixParts={childPrefix}
              expandedFolders={expandedFolders}
              selectedTreeFolder={selectedTreeFolder}
              dragSourcePath={dragSourcePath}
              dropTargetFolder={dropTargetFolder}
              onToggleFolder={onToggleFolder}
              onSelectFolder={onSelectFolder}
              onSetDragSourcePath={onSetDragSourcePath}
              onSetDropTargetFolder={onSetDropTargetFolder}
              onDropMove={onDropMove}
              onOpenWorkflowFile={onOpenWorkflowFile}
              onExecuteWorkflow={onExecuteWorkflow}
              onDeleteNode={onDeleteNode}
              onRenameNode={onRenameNode}
            />
          );
        })}
    </div>
  );
}

export default FileTreeItem;
