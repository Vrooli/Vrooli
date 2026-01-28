/**
 * FileTree Component
 *
 * Displays a hierarchical file tree with expandable directories.
 * Used in the idea details page to show the contents of an idea folder.
 *
 * [REQ:REQ-P0-004] File tree component for idea details page
 */

import { useState } from "react";
import { ChevronRight, ChevronDown, File, Folder, FolderOpen } from "lucide-react";
import { cn, formatFileSize } from "../../lib";
import type { IdeaFile } from "../../types";

export interface FileTreeProps {
  files: IdeaFile[];
  onFileSelect?: (file: IdeaFile) => void;
  selectedPath?: string;
  "data-testid"?: string;
}

export interface FileTreeItemProps {
  file: IdeaFile;
  depth?: number;
  onFileSelect?: (file: IdeaFile) => void;
  selectedPath?: string;
}

/**
 * Single item in the file tree (file or directory)
 */
function FileTreeItem({ file, depth = 0, onFileSelect, selectedPath }: FileTreeItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const isDirectory = file.type === "directory";
  const isSelected = selectedPath === file.path;
  const hasChildren = isDirectory && file.children && file.children.length > 0;

  const handleClick = () => {
    if (isDirectory) {
      setIsExpanded(!isExpanded);
    }
    onFileSelect?.(file);
  };

  return (
    <div data-testid={`file-tree-item-${file.path}`}>
      <button
        type="button"
        onClick={handleClick}
        className={cn(
          "flex w-full items-center gap-2 rounded px-2 py-1 text-left text-sm transition-colors",
          "hover:bg-slate-700/50",
          isSelected && "bg-cyan-500/20 text-cyan-300",
          !isSelected && "text-slate-300"
        )}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        data-testid={`file-tree-button-${file.path}`}
      >
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
export function FileTree({ files, onFileSelect, selectedPath, "data-testid": testId }: FileTreeProps) {
  if (files.length === 0) {
    return (
      <div
        className="rounded-lg border border-white/10 bg-slate-800/30 p-6 text-center"
        data-testid={testId ?? "file-tree"}
      >
        <Folder className="mx-auto h-8 w-8 text-slate-600" />
        <p className="mt-2 text-sm text-slate-400">No files yet</p>
      </div>
    );
  }

  return (
    <div
      className="rounded-lg border border-white/10 bg-slate-800/30 py-2"
      data-testid={testId ?? "file-tree"}
    >
      {files.map((file) => (
        <FileTreeItem
          key={file.path}
          file={file}
          onFileSelect={onFileSelect}
          selectedPath={selectedPath}
        />
      ))}
    </div>
  );
}
