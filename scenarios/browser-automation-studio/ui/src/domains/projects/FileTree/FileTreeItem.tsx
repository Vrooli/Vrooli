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
  GripVertical,
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
      className="font-mono text-xs leading-5 text-gray-500 whitespace-pre pointer-events-none select-none"
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
    return <FolderOpen size={16} className="text-yellow-500 flex-shrink-0" />;
  }
  if (node.kind === "workflow_file") {
    return <FileCode size={16} className="text-green-400 flex-shrink-0" />;
  }
  return <FileText size={16} className="text-gray-400 flex-shrink-0" />;
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
  /** Currently focused path for keyboard navigation */
  focusedTreePath: string | null;

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
  /** Preview a workflow file (single click) */
  onPreviewWorkflow: (workflowId: string) => void;
  /** Open a workflow file (double click) */
  onOpenWorkflowFile: (workflowId: string) => Promise<void>;
  /** Execute a workflow */
  onExecuteWorkflow: (e: React.MouseEvent, workflowId: string) => Promise<void>;
  /** Delete a node (file or folder) */
  onDeleteNode: (path: string) => Promise<void>;
  /** Rename/move a node */
  onRenameNode: (path: string) => Promise<void>;
  /** Show inline add menu for a folder */
  onShowInlineAddMenu?: (folderPath: string, e?: React.MouseEvent) => void;
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
  focusedTreePath,
  onToggleFolder,
  onSelectFolder,
  onSetDragSourcePath,
  onSetDropTargetFolder,
  onDropMove,
  onPreviewWorkflow,
  onOpenWorkflowFile,
  onExecuteWorkflow,
  onDeleteNode,
  onRenameNode,
  onShowInlineAddMenu,
}: FileTreeItemProps) {
  const isFolder = node.kind === "folder";
  const isExpanded = isFolder && expandedFolders.has(node.path);
  const children = node.children ?? [];
  const hasChildren = isFolder && children.length > 0;
  const typeLabel = node.kind === "workflow_file" ? fileTypeLabelFromPath(node.path) : null;
  const isDropTarget = Boolean(
    dragSourcePath !== null && dropTargetFolder !== null && dropTargetFolder === node.path
  );
  const isFocused = focusedTreePath === node.path;

  const handleToggle = () => {
    if (!isFolder) return;
    onToggleFolder(node.path);
  };

  const handlePreviewWorkflow = () => {
    if (node.kind !== "workflow_file" || !node.workflowId) {
      return;
    }
    onPreviewWorkflow(node.workflowId);
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

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    // Only clear if leaving the actual folder element (not entering a child)
    const relatedTarget = e.relatedTarget as Node | null;
    const currentTarget = e.currentTarget as HTMLElement;
    if (!relatedTarget || !currentTarget.contains(relatedTarget)) {
      onSetDropTargetFolder(null);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
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
      // Single click shows preview
      handlePreviewWorkflow();
      return;
    }
    toast("Assets are read-only in v1");
  };

  const handleDoubleClick = () => {
    if (node.kind === "workflow_file") {
      // Double click opens for editing
      void handleOpenWorkflowFile();
    }
  };

  return (
    <div className="select-none">
      <div
        draggable={node.path !== ""}
        data-testid={isFolder ? "file-tree-folder" : "file-tree-file"}
        data-path={node.path}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={`flex items-center gap-2 px-2 py-2 rounded cursor-pointer transition-colors group ${
          isFolder && selectedTreeFolder === node.path
            ? "bg-flow-node"
            : "hover:bg-flow-node"
        } ${isDropTarget ? "ring-1 ring-flow-accent bg-flow-node" : ""} ${
          isFocused ? "ring-2 ring-flow-accent/50 bg-flow-node" : ""
        }`}
        onClick={handleClick}
        onDoubleClick={handleDoubleClick}
      >
        {/* Drag handle - visible on hover */}
        <div className="opacity-0 group-hover:opacity-100 transition-opacity cursor-grab active:cursor-grabbing flex-shrink-0">
          <GripVertical size={14} className="text-gray-500" />
        </div>

        {renderTreePrefix(prefixParts)}

        {/* Expand/collapse chevron for folders */}
        {isFolder ? (
          hasChildren ? (
            isExpanded ? (
              <ChevronDown size={16} className="text-gray-400 flex-shrink-0" />
            ) : (
              <ChevronRight size={16} className="text-gray-400 flex-shrink-0" />
            )
          ) : (
            <span className="inline-block w-4" aria-hidden="true" />
          )
        ) : (
          <span className="inline-block w-4" aria-hidden="true" />
        )}

        {/* File/folder icon */}
        {fileKindIcon(node)}

        {/* Name and type badge */}
        <span className="text-base text-gray-300 truncate">
          {node.name}
        </span>
        {typeLabel ? (
          <span className="ml-1 text-xs px-1.5 py-0.5 rounded bg-gray-800 text-gray-400 flex-shrink-0">
            {typeLabel}
          </span>
        ) : null}

        {/* Action buttons - positioned close to name, visible on hover */}
        <div className="ml-3 flex items-center gap-1.5 opacity-100 md:opacity-0 md:group-hover:opacity-100 transition-opacity flex-shrink-0">
          {node.kind === "workflow_file" && node.workflowId ? (
            <button
              onClick={handleExecuteFromTree}
              className="p-1 rounded text-subtle hover:text-surface hover:bg-gray-700 transition-colors"
              title={typeLabel === "Case" ? "Test" : "Run"}
              aria-label={typeLabel === "Case" ? "Test workflow" : "Run workflow"}
            >
              {typeLabel === "Case" ? <ListChecks size={16} /> : <Play size={16} />}
            </button>
          ) : null}
          <button
            onClick={handleRenameNode}
            className="p-1 rounded text-subtle hover:text-surface hover:bg-gray-700 transition-colors"
            title="Rename / Move"
            aria-label="Rename / Move"
          >
            <PencilLine size={16} />
          </button>
          <button
            onClick={handleDeleteNode}
            className="p-1 rounded text-subtle hover:text-red-300 hover:bg-red-500/10 transition-colors"
            title="Delete"
            aria-label="Delete"
          >
            <Trash2 size={16} />
          </button>
        </div>
      </div>

      {/* Render children when folder is expanded */}
      {isFolder && isExpanded && (
        <>
          {children.map((child, index) => {
            const isLastChild = index === children.length - 1 && !onShowInlineAddMenu;
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
                focusedTreePath={focusedTreePath}
                onToggleFolder={onToggleFolder}
                onSelectFolder={onSelectFolder}
                onSetDragSourcePath={onSetDragSourcePath}
                onSetDropTargetFolder={onSetDropTargetFolder}
                onDropMove={onDropMove}
                onPreviewWorkflow={onPreviewWorkflow}
                onOpenWorkflowFile={onOpenWorkflowFile}
                onExecuteWorkflow={onExecuteWorkflow}
                onDeleteNode={onDeleteNode}
                onRenameNode={onRenameNode}
                onShowInlineAddMenu={onShowInlineAddMenu}
              />
            );
          })}

          {/* Inline "Add" button at the bottom of expanded folders */}
          {onShowInlineAddMenu && (
            <div
              className="flex items-center gap-2 px-2 py-1.5 ml-6 rounded cursor-pointer text-gray-500 hover:text-gray-300 hover:bg-flow-node transition-colors group"
              onClick={(e) => {
                e.stopPropagation();
                onShowInlineAddMenu(node.path, e);
              }}
            >
              {renderTreePrefix([...prefixParts, false])}
              <span className="inline-block w-4" aria-hidden="true" />
              <span className="text-sm italic">+ Add new...</span>
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default FileTreeItem;
