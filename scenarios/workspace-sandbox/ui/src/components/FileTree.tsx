import { useMemo } from "react";
import {
  FilePlus,
  FileX,
  FileCode,
  Check,
  Minus as MinusIcon,
  FolderOpen,
  ArrowLeft,
} from "lucide-react";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import type { DiffResult, FileChange } from "../lib/api";
import type { HunkSelection } from "./DiffViewer";

interface FileTreeProps {
  diff?: DiffResult;
  sandboxPath: string;
  selectedHunks: HunkSelection[];
  onFileClick: (filePath: string) => void;
  onExitReview: () => void;
}

// Group files by directory for tree structure
interface FileNode {
  name: string;
  path: string;
  isDirectory: boolean;
  children: FileNode[];
  file?: FileChange;
  selectedHunkCount?: number;
  totalHunkCount?: number;
}

function buildFileTree(
  files: FileChange[],
  selectedHunks: HunkSelection[],
  hunkCountsByFile: Map<string, number>
): FileNode {
  const root: FileNode = {
    name: "",
    path: "",
    isDirectory: true,
    children: [],
  };

  // Count selected hunks by file path
  const selectedByPath = new Map<string, number>();
  for (const hunk of selectedHunks) {
    const count = selectedByPath.get(hunk.filePath) ?? 0;
    selectedByPath.set(hunk.filePath, count + 1);
  }

  for (const file of files) {
    const parts = file.filePath.split("/");
    let current = root;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const isLast = i === parts.length - 1;
      const currentPath = parts.slice(0, i + 1).join("/");

      let child = current.children.find((c) => c.name === part);
      if (!child) {
        child = {
          name: part,
          path: currentPath,
          isDirectory: !isLast,
          children: [],
          file: isLast ? file : undefined,
          selectedHunkCount: isLast ? selectedByPath.get(file.filePath) ?? 0 : undefined,
          totalHunkCount: isLast ? hunkCountsByFile.get(file.filePath) ?? 0 : undefined,
        };
        current.children.push(child);
      }
      current = child;
    }
  }

  // Sort: directories first, then alphabetically
  const sortNodes = (nodes: FileNode[]): FileNode[] => {
    return nodes
      .map((node) => ({
        ...node,
        children: sortNodes(node.children),
      }))
      .sort((a, b) => {
        if (a.isDirectory !== b.isDirectory) {
          return a.isDirectory ? -1 : 1;
        }
        return a.name.localeCompare(b.name);
      });
  };

  root.children = sortNodes(root.children);
  return root;
}

function FileIcon({ changeType }: { changeType: string }) {
  switch (changeType) {
    case "added":
      return <FilePlus className="h-4 w-4 text-emerald-400 flex-shrink-0" />;
    case "deleted":
      return <FileX className="h-4 w-4 text-red-400 flex-shrink-0" />;
    default:
      return <FileCode className="h-4 w-4 text-blue-400 flex-shrink-0" />;
  }
}

function SelectionIndicator({
  selectedCount,
  totalCount,
}: {
  selectedCount: number;
  totalCount: number;
}) {
  if (totalCount === 0) return null;

  if (selectedCount === totalCount) {
    return <Check className="h-3.5 w-3.5 text-emerald-400 flex-shrink-0" />;
  }
  if (selectedCount > 0) {
    return <MinusIcon className="h-3.5 w-3.5 text-amber-400 flex-shrink-0" />;
  }
  return null;
}

interface FileNodeComponentProps {
  node: FileNode;
  depth: number;
  onFileClick: (filePath: string) => void;
}

function FileNodeComponent({ node, depth, onFileClick }: FileNodeComponentProps) {
  if (node.isDirectory) {
    return (
      <div>
        <div
          className="flex items-center gap-2 py-1.5 px-2 text-slate-400 text-xs"
          style={{ paddingLeft: `${depth * 12 + 8}px` }}
        >
          <FolderOpen className="h-3.5 w-3.5 flex-shrink-0" />
          <span className="truncate">{node.name}</span>
        </div>
        {node.children.map((child) => (
          <FileNodeComponent
            key={child.path}
            node={child}
            depth={depth + 1}
            onFileClick={onFileClick}
          />
        ))}
      </div>
    );
  }

  const file = node.file!;
  return (
    <button
      className="w-full flex items-center gap-2 py-1.5 px-2 text-left hover:bg-slate-800/50 transition-colors rounded-sm"
      style={{ paddingLeft: `${depth * 12 + 8}px` }}
      onClick={() => onFileClick(file.filePath)}
      title={file.filePath}
    >
      <FileIcon changeType={file.changeType} />
      <span className="flex-1 truncate text-xs text-slate-200">{node.name}</span>
      <SelectionIndicator
        selectedCount={node.selectedHunkCount ?? 0}
        totalCount={node.totalHunkCount ?? 0}
      />
    </button>
  );
}

export function FileTree({
  diff,
  sandboxPath,
  selectedHunks,
  onFileClick,
  onExitReview,
}: FileTreeProps) {
  // Count total hunks per file (we need to parse the diff to get this)
  const hunkCountsByFile = useMemo(() => {
    const counts = new Map<string, number>();
    if (!diff?.unifiedDiff) return counts;

    // Simple hunk counting from unified diff
    const lines = diff.unifiedDiff.split("\n");
    let currentFile = "";
    for (const line of lines) {
      if (line.startsWith("diff --git")) {
        const match = line.match(/diff --git a\/(.*) b\/(.*)/);
        currentFile = match ? match[2] : "";
        counts.set(currentFile, 0);
      } else if (line.startsWith("@@") && currentFile) {
        counts.set(currentFile, (counts.get(currentFile) ?? 0) + 1);
      }
    }
    return counts;
  }, [diff?.unifiedDiff]);

  const tree = useMemo(() => {
    if (!diff?.files) return null;
    return buildFileTree(diff.files, selectedHunks, hunkCountsByFile);
  }, [diff?.files, selectedHunks, hunkCountsByFile]);

  const fileCount = diff?.files?.length ?? 0;
  const totalSelected = selectedHunks.length;

  return (
    <div className="h-full flex flex-col bg-slate-900/50">
      {/* Header */}
      <div className="flex-shrink-0 p-3 border-b border-slate-800">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-sm font-medium text-slate-200">Review Changes</h3>
          <Button
            variant="ghost"
            size="sm"
            onClick={onExitReview}
            className="h-7 px-2 text-xs"
          >
            <ArrowLeft className="h-3.5 w-3.5 mr-1" />
            Exit
          </Button>
        </div>
        <p className="text-xs text-slate-500 truncate" title={sandboxPath}>
          {sandboxPath}
        </p>
        <div className="flex items-center gap-3 mt-2 text-xs">
          <span className="text-slate-400">
            {fileCount} {fileCount === 1 ? "file" : "files"}
          </span>
          {totalSelected > 0 && (
            <span className="text-emerald-400">
              {totalSelected} {totalSelected === 1 ? "hunk" : "hunks"} selected
            </span>
          )}
        </div>
      </div>

      {/* File tree */}
      <ScrollArea className="flex-1">
        <div className="py-2">
          {tree?.children.map((node) => (
            <FileNodeComponent
              key={node.path}
              node={node}
              depth={0}
              onFileClick={onFileClick}
            />
          ))}
          {(!tree || tree.children.length === 0) && (
            <p className="text-xs text-slate-500 text-center py-8">
              No changes to review
            </p>
          )}
        </div>
      </ScrollArea>

      {/* Legend */}
      <div className="flex-shrink-0 p-3 border-t border-slate-800">
        <div className="flex items-center gap-4 text-[10px] text-slate-500">
          <span className="flex items-center gap-1">
            <FilePlus className="h-3 w-3 text-emerald-400" /> Added
          </span>
          <span className="flex items-center gap-1">
            <FileCode className="h-3 w-3 text-blue-400" /> Modified
          </span>
          <span className="flex items-center gap-1">
            <FileX className="h-3 w-3 text-red-400" /> Deleted
          </span>
        </div>
      </div>
    </div>
  );
}
