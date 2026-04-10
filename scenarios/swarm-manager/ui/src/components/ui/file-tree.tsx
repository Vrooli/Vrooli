/**
 * FileTree Component
 *
 * Displays a hierarchical file tree with expandable directories.
 * Supports two selection modes:
 * - "single": Click to select a single file (for file viewing)
 * - "checkbox": Checkboxes for multi-select (for file preservation)
 *
 * [REQ:REQ-P0-004] File tree component for backlog details page
 */

import { useState, useCallback } from "react";
import {
  ChevronRight,
  ChevronDown,
  File,
  Folder,
  FolderOpen,
  Check,
  Minus,
} from "lucide-react";
import { cn, formatFileSize } from "../../lib";
import type { BacklogFile, ScenarioFile } from "../../types";

/** Generic file type that works with both BacklogFile and ScenarioFile */
export type TreeFile = BacklogFile | ScenarioFile;

export type SelectionMode = "single" | "checkbox";

export interface FileTreeProps {
  files: TreeFile[];
  /** Single file selection handler (for single mode) */
  onFileSelect?: (file: TreeFile) => void;
  /** Currently selected path (for single mode) */
  selectedPath?: string;
  /** Selection mode: "single" for click-to-select, "checkbox" for multi-select */
  selectionMode?: SelectionMode;
  /** Set of selected file paths (for checkbox mode) */
  selectedPaths?: Set<string>;
  /** Callback when selection changes (for checkbox mode) */
  onSelectionChange?: (selectedPaths: Set<string>) => void;
  /** Right-click handler for files/directories */
  onItemContextMenu?: (file: TreeFile, event: React.MouseEvent<HTMLButtonElement>) => void;
  className?: string;
  "data-testid"?: string;
}

export interface FileTreeItemProps {
  file: TreeFile;
  depth?: number;
  onFileSelect?: (file: TreeFile) => void;
  selectedPath?: string;
  selectionMode?: SelectionMode;
  selectedPaths?: Set<string>;
  onCheckboxChange?: (path: string, checked: boolean) => void;
  getChildPaths?: (file: TreeFile) => string[];
  onItemContextMenu?: (file: TreeFile, event: React.MouseEvent<HTMLButtonElement>) => void;
}

/**
 * Get all descendant file paths (files only, not directories) from a file node
 */
function getAllFilePaths(file: TreeFile): string[] {
  if (file.type === "file") {
    return [file.path];
  }
  if (!file.children) {
    return [];
  }
  return file.children.flatMap(getAllFilePaths);
}

/**
 * Determine checkbox state for a directory based on children selection
 */
function getDirectoryCheckState(
  file: TreeFile,
  selectedPaths: Set<string>
): "checked" | "unchecked" | "indeterminate" {
  const childPaths = getAllFilePaths(file);
  if (childPaths.length === 0) return "unchecked";

  const selectedCount = childPaths.filter((p) => selectedPaths.has(p)).length;
  if (selectedCount === 0) return "unchecked";
  if (selectedCount === childPaths.length) return "checked";
  return "indeterminate";
}

/**
 * Single item in the file tree (file or directory)
 */
function FileTreeItem({
  file,
  depth = 0,
  onFileSelect,
  selectedPath,
  selectionMode = "single",
  selectedPaths,
  onCheckboxChange,
  onItemContextMenu,
}: FileTreeItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const isDirectory = file.type === "directory";
  const isSelected = selectedPath === file.path;
  const hasChildren = isDirectory && file.children && file.children.length > 0;
  const isCheckboxMode = selectionMode === "checkbox";

  // For checkbox mode
  const checkState =
    isCheckboxMode && selectedPaths
      ? isDirectory
        ? getDirectoryCheckState(file, selectedPaths)
        : selectedPaths.has(file.path)
          ? "checked"
          : "unchecked"
      : "unchecked";

  const handleClick = () => {
    if (isDirectory) {
      setIsExpanded(!isExpanded);
    }
    if (!isCheckboxMode) {
      onFileSelect?.(file);
    }
  };

  const handleCheckboxClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!onCheckboxChange) return;

    if (isDirectory) {
      // Toggle all children
      const childPaths = getAllFilePaths(file);
      const shouldCheck = checkState !== "checked";
      childPaths.forEach((path) => onCheckboxChange(path, shouldCheck));
    } else {
      onCheckboxChange(file.path, !selectedPaths?.has(file.path));
    }
  };

  return (
    <div data-testid={`file-tree-item-${file.path}`}>
      <button
        type="button"
        onClick={handleClick}
        onContextMenu={(event) => onItemContextMenu?.(file, event)}
        className={cn(
          "flex w-full items-center gap-2 rounded px-2 py-1 text-left text-sm transition-colors",
          "hover:bg-slate-700/50",
          !isCheckboxMode && isSelected && "bg-cyan-500/20 text-cyan-300",
          !isCheckboxMode && !isSelected && "text-slate-300",
          isCheckboxMode && "text-slate-300"
        )}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        data-testid={`file-tree-button-${file.path}`}
      >
        {/* Checkbox (for checkbox mode) */}
        {isCheckboxMode && (
          <span
            role="checkbox"
            aria-checked={checkState === "checked"}
            onClick={handleCheckboxClick}
            onKeyDown={(e) => {
              if (e.key === " " || e.key === "Enter") {
                e.preventDefault();
                handleCheckboxClick(e as unknown as React.MouseEvent);
              }
            }}
            tabIndex={0}
            className={cn(
              "flex h-4 w-4 shrink-0 items-center justify-center rounded border transition-colors cursor-pointer",
              checkState === "checked" && "border-cyan-500 bg-cyan-500 text-white",
              checkState === "indeterminate" && "border-cyan-500 bg-cyan-500/50 text-white",
              checkState === "unchecked" && "border-slate-500 bg-transparent hover:border-cyan-400"
            )}
            data-testid={`file-tree-checkbox-${file.path}`}
          >
            {checkState === "checked" && <Check className="h-3 w-3" />}
            {checkState === "indeterminate" && <Minus className="h-3 w-3" />}
          </span>
        )}

        {/* Expand/collapse chevron for directories */}
        {isDirectory ? (
          <span className="flex h-4 w-4 items-center justify-center text-slate-400">
            {hasChildren ? (
              isExpanded ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronRight className="h-3 w-3" />
              )
            ) : null}
          </span>
        ) : (
          <span className="w-4" />
        )}

        {/* Icon */}
        {isDirectory ? (
          isExpanded ? (
            <FolderOpen className="h-4 w-4 text-cyan-400" />
          ) : (
            <Folder className="h-4 w-4 text-cyan-400" />
          )
        ) : (
          <File className="h-4 w-4 text-slate-400" />
        )}

        {/* Name */}
        <span className="flex-1 truncate">{file.name}</span>

        {/* Size (for files only) */}
        {!isDirectory && file.size !== undefined && (
          <span className="text-xs text-slate-400">{formatFileSize(file.size)}</span>
        )}
      </button>

      {/* Children (for expanded directories) */}
      {isDirectory && isExpanded && file.children && (
        <div>
          {file.children.map((child) => (
            <FileTreeItem
              key={child.path}
              file={child}
              depth={depth + 1}
              onFileSelect={onFileSelect}
              selectedPath={selectedPath}
              selectionMode={selectionMode}
              selectedPaths={selectedPaths}
              onCheckboxChange={onCheckboxChange}
              onItemContextMenu={onItemContextMenu}
            />
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * File tree component displaying a hierarchical list of files and directories
 */
export function FileTree({
  files = [],
  onFileSelect,
  selectedPath,
  selectionMode = "single",
  selectedPaths,
  onSelectionChange,
  onItemContextMenu,
  className,
  "data-testid": testId,
}: FileTreeProps) {
  // Internal handler for checkbox changes
  const handleCheckboxChange = useCallback(
    (path: string, checked: boolean) => {
      if (!onSelectionChange) return;
      const newSelection = new Set(selectedPaths);
      if (checked) {
        newSelection.add(path);
      } else {
        newSelection.delete(path);
      }
      onSelectionChange(newSelection);
    },
    [selectedPaths, onSelectionChange]
  );

  if (files.length === 0) {
    return (
      <div
        className={cn(
          "rounded-lg border border-white/10 bg-slate-800/30 p-6 text-center",
          className
        )}
        data-testid={testId ?? "file-tree"}
      >
        <Folder className="mx-auto h-8 w-8 text-slate-600" />
        <p className="mt-2 text-sm text-slate-400">No files yet</p>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "rounded-lg border border-white/10 bg-slate-800/30 py-2",
        className
      )}
      data-testid={testId ?? "file-tree"}
    >
      {files.map((file) => (
        <FileTreeItem
          key={file.path}
          file={file}
          onFileSelect={onFileSelect}
          selectedPath={selectedPath}
          selectionMode={selectionMode}
          selectedPaths={selectedPaths}
          onCheckboxChange={handleCheckboxChange}
          onItemContextMenu={onItemContextMenu}
        />
      ))}
    </div>
  );
}
