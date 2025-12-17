import { useState, useMemo } from "react";
import {
  FileDiff,
  FilePlus,
  FileX,
  FileCode,
  Plus,
  Minus,
  Loader2,
  ChevronDown,
  ChevronRight,
  Check,
  X,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Badge } from "./ui/badge";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import type { DiffResult, FileChange, ChangeType } from "../lib/api";
import { SELECTORS } from "../consts/selectors";

// Hunk selection type for approval workflow
export interface HunkSelection {
  fileId: string;
  filePath: string;
  startLine: number;
  endLine: number;
  hunkIndex: number;
}

interface DiffViewerProps {
  diff?: DiffResult;
  isLoading: boolean;
  error?: Error | null;
  onApproveFile?: (fileId: string) => void;
  onRejectFile?: (fileId: string) => void;
  showFileActions?: boolean;
  // Hunk-level selection props [OT-P1-001]
  showHunkSelection?: boolean;
  selectedHunks?: HunkSelection[];
  onHunkSelectionChange?: (hunks: HunkSelection[]) => void;
}

interface ParsedHunk {
  header: string;
  oldStart: number;
  oldCount: number;
  newStart: number;
  newCount: number;
  lines: string[];
}

interface ParsedFileDiff {
  path: string;
  changeType: ChangeType;
  hunks: ParsedHunk[];
  oldPath?: string;
}

// Parse unified diff into structured format
function parseUnifiedDiff(diff: string): ParsedFileDiff[] {
  const files: ParsedFileDiff[] = [];
  const lines = diff.split("\n");
  let currentFile: ParsedFileDiff | null = null;
  let currentHunk: ParsedHunk | null = null;

  for (const line of lines) {
    // File header: diff --git a/path b/path
    if (line.startsWith("diff --git")) {
      if (currentFile) {
        if (currentHunk) {
          currentFile.hunks.push(currentHunk);
        }
        files.push(currentFile);
      }
      // Extract path from: diff --git a/path b/path
      const match = line.match(/diff --git a\/(.*) b\/(.*)/);
      const path = match ? match[2] : "";
      currentFile = {
        path,
        changeType: "modified",
        hunks: [],
      };
      currentHunk = null;
      continue;
    }

    if (!currentFile) continue;

    // New file indicator
    if (line.startsWith("new file mode")) {
      currentFile.changeType = "added";
      continue;
    }

    // Deleted file indicator
    if (line.startsWith("deleted file mode")) {
      currentFile.changeType = "deleted";
      continue;
    }

    // Rename indicator
    if (line.startsWith("rename from")) {
      const match = line.match(/rename from (.*)/);
      if (match) currentFile.oldPath = match[1];
      continue;
    }

    // Hunk header: @@ -old,count +new,count @@
    if (line.startsWith("@@")) {
      if (currentHunk) {
        currentFile.hunks.push(currentHunk);
      }
      const match = line.match(/@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)?/);
      if (match) {
        currentHunk = {
          header: line,
          oldStart: parseInt(match[1], 10),
          oldCount: match[2] ? parseInt(match[2], 10) : 1,
          newStart: parseInt(match[3], 10),
          newCount: match[4] ? parseInt(match[4], 10) : 1,
          lines: [],
        };
      }
      continue;
    }

    // Skip file path lines
    if (line.startsWith("---") || line.startsWith("+++")) continue;
    if (line.startsWith("index ")) continue;

    // Content lines
    if (currentHunk) {
      currentHunk.lines.push(line);
    }
  }

  // Push final file and hunk
  if (currentFile) {
    if (currentHunk) {
      currentFile.hunks.push(currentHunk);
    }
    files.push(currentFile);
  }

  return files;
}

function DiffLine({ line }: { line: string }) {
  const isAddition = line.startsWith("+");
  const isDeletion = line.startsWith("-");
  const isContext = !isAddition && !isDeletion;

  let bgColor = "";
  let textColor = "text-slate-300";

  if (isAddition) {
    bgColor = "bg-emerald-950/30";
    textColor = "text-emerald-300";
  } else if (isDeletion) {
    bgColor = "bg-red-950/30";
    textColor = "text-red-300";
  }

  return (
    <div className={`flex font-mono text-xs ${bgColor}`} data-testid={SELECTORS.diffLine}>
      <span className="w-6 flex-shrink-0 text-center text-slate-600 select-none">
        {isAddition ? "+" : isDeletion ? "-" : " "}
      </span>
      <pre className={`flex-1 px-2 py-0.5 whitespace-pre overflow-x-auto ${textColor}`}>
        {line.slice(1) || " "}
      </pre>
    </div>
  );
}

interface HunkDisplayProps {
  hunk: ParsedHunk;
  index: number;
  showSelection?: boolean;
  isSelected?: boolean;
  onToggleSelection?: () => void;
}

function HunkDisplay({ hunk, index, showSelection, isSelected, onToggleSelection }: HunkDisplayProps) {
  return (
    <div className="border-b border-slate-800 last:border-b-0" data-testid={SELECTORS.diffHunk(index)}>
      {/* Hunk header with optional selection checkbox */}
      <div className="bg-slate-800/50 px-3 py-1.5 font-mono text-xs text-blue-400 flex items-center gap-2">
        {showSelection && (
          <label
            className="flex items-center cursor-pointer"
            onClick={(e) => e.stopPropagation()}
            data-testid={`hunk-checkbox-${index}`}
          >
            <input
              type="checkbox"
              checked={isSelected}
              onChange={onToggleSelection}
              className="w-4 h-4 rounded border-slate-600 bg-slate-800 text-emerald-500 focus:ring-emerald-500 focus:ring-offset-0 cursor-pointer"
            />
          </label>
        )}
        <span className="flex-1">{hunk.header}</span>
        {isSelected && (
          <Check className="h-3.5 w-3.5 text-emerald-400" />
        )}
      </div>

      {/* Hunk lines */}
      <div className={showSelection && !isSelected ? "opacity-50" : ""}>
        {hunk.lines.map((line, lineIdx) => (
          <DiffLine key={lineIdx} line={line} />
        ))}
      </div>
    </div>
  );
}

interface FileDiffSectionProps {
  file: ParsedFileDiff;
  fileChange?: FileChange;
  onApprove?: () => void;
  onReject?: () => void;
  showActions?: boolean;
  // Hunk selection props [OT-P1-001]
  showHunkSelection?: boolean;
  selectedHunkIndices?: Set<number>;
  onToggleHunkSelection?: (hunkIndex: number, hunk: ParsedHunk) => void;
}

function FileDiffSection({
  file,
  fileChange,
  onApprove,
  onReject,
  showActions,
  showHunkSelection,
  selectedHunkIndices,
  onToggleHunkSelection,
}: FileDiffSectionProps) {
  const [expanded, setExpanded] = useState(true);

  const changeIcon = {
    added: <FilePlus className="h-4 w-4 text-emerald-400" />,
    modified: <FileCode className="h-4 w-4 text-blue-400" />,
    deleted: <FileX className="h-4 w-4 text-red-400" />,
  };

  const changeBadge = {
    added: "added" as const,
    modified: "modified" as const,
    deleted: "removed" as const,
  };

  // Count selected hunks for this file
  const selectedCount = selectedHunkIndices?.size ?? 0;
  const totalHunks = file.hunks.length;

  return (
    <div
      className="border border-slate-800 rounded-lg mb-3 overflow-hidden"
      data-testid={SELECTORS.diffFileItem}
      data-file-path={file.path}
    >
      {/* File header */}
      <div
        className="flex items-center gap-2 px-3 py-2 bg-slate-800/30 cursor-pointer hover:bg-slate-800/50 transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-slate-500" />
        ) : (
          <ChevronRight className="h-4 w-4 text-slate-500" />
        )}
        {changeIcon[file.changeType]}
        <span className="font-mono text-xs text-slate-200 flex-1 truncate">
          {file.path}
        </span>

        {/* Show hunk selection count when in selection mode */}
        {showHunkSelection && totalHunks > 0 && (
          <span className="text-xs text-slate-500">
            {selectedCount}/{totalHunks} hunks
          </span>
        )}

        <Badge variant={changeBadge[file.changeType]} className="text-[10px]">
          {file.changeType}
        </Badge>

        {/* File-level approval status */}
        {fileChange?.approvalStatus === "approved" && (
          <Check className="h-4 w-4 text-emerald-400" />
        )}
        {fileChange?.approvalStatus === "rejected" && (
          <X className="h-4 w-4 text-red-400" />
        )}

        {/* File actions */}
        {showActions && fileChange?.approvalStatus === "pending" && (
          <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
            {onApprove && (
              <button
                className="p-1 rounded hover:bg-emerald-900/50 transition-colors"
                onClick={onApprove}
                title="Approve file"
              >
                <Check className="h-3.5 w-3.5 text-emerald-400" />
              </button>
            )}
            {onReject && (
              <button
                className="p-1 rounded hover:bg-red-900/50 transition-colors"
                onClick={onReject}
                title="Reject file"
              >
                <X className="h-3.5 w-3.5 text-red-400" />
              </button>
            )}
          </div>
        )}
      </div>

      {/* File content */}
      {expanded && (
        <div data-testid={SELECTORS.diffContent}>
          {file.hunks.map((hunk, index) => (
            <HunkDisplay
              key={index}
              hunk={hunk}
              index={index}
              showSelection={showHunkSelection}
              isSelected={selectedHunkIndices?.has(index)}
              onToggleSelection={
                onToggleHunkSelection
                  ? () => onToggleHunkSelection(index, hunk)
                  : undefined
              }
            />
          ))}
          {file.hunks.length === 0 && (
            <div className="px-3 py-4 text-xs text-slate-500 text-center">
              {file.changeType === "added"
                ? "New empty file"
                : file.changeType === "deleted"
                ? "File deleted"
                : "No changes in this file"}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function DiffViewer({
  diff,
  isLoading,
  error,
  onApproveFile,
  onRejectFile,
  showFileActions = false,
  showHunkSelection = false,
  selectedHunks = [],
  onHunkSelectionChange,
}: DiffViewerProps) {
  // Parse the unified diff
  const parsedFiles = useMemo(() => {
    if (!diff?.unifiedDiff) return [];
    return parseUnifiedDiff(diff.unifiedDiff);
  }, [diff?.unifiedDiff]);

  // Create a map of file changes by path for quick lookup
  const fileChangeMap = useMemo(() => {
    const map = new Map<string, FileChange>();
    if (diff?.files) {
      for (const fc of diff.files) {
        map.set(fc.filePath, fc);
      }
    }
    return map;
  }, [diff?.files]);

  // Build a map of file path -> set of selected hunk indices [OT-P1-001]
  const selectedHunksByFile = useMemo(() => {
    const map = new Map<string, Set<number>>();
    for (const sel of selectedHunks) {
      const existing = map.get(sel.filePath) ?? new Set<number>();
      existing.add(sel.hunkIndex);
      map.set(sel.filePath, existing);
    }
    return map;
  }, [selectedHunks]);

  // Handle toggling hunk selection
  const handleToggleHunk = (filePath: string, fileId: string, hunkIndex: number, hunk: ParsedHunk) => {
    if (!onHunkSelectionChange) return;

    const isCurrentlySelected = selectedHunks.some(
      (s) => s.filePath === filePath && s.hunkIndex === hunkIndex
    );

    if (isCurrentlySelected) {
      // Remove this hunk from selection
      onHunkSelectionChange(
        selectedHunks.filter(
          (s) => !(s.filePath === filePath && s.hunkIndex === hunkIndex)
        )
      );
    } else {
      // Add this hunk to selection
      onHunkSelectionChange([
        ...selectedHunks,
        {
          fileId,
          filePath,
          startLine: hunk.newStart,
          endLine: hunk.newStart + hunk.newCount - 1,
          hunkIndex,
        },
      ]);
    }
  };

  const hasChanges = parsedFiles.length > 0 || (diff?.files && diff.files.length > 0);

  return (
    <Card className="h-full flex flex-col" data-testid={SELECTORS.diffViewer}>
      <CardHeader className="flex-row items-center justify-between space-y-0 py-3">
        <CardTitle className="flex items-center gap-2">
          <FileDiff className="h-4 w-4 text-slate-500" />
          Changes
        </CardTitle>

        {diff && hasChanges && (
          <div className="flex items-center gap-3" data-testid={SELECTORS.diffStats}>
            <span className="flex items-center gap-1 text-xs text-emerald-500">
              <Plus className="h-3 w-3" />
              {diff.totalAdded}
            </span>
            <span className="flex items-center gap-1 text-xs text-red-500">
              <Minus className="h-3 w-3" />
              {diff.totalDeleted}
            </span>
            <span className="text-xs text-slate-500">
              {diff.files?.length || parsedFiles.length} file(s)
            </span>
          </div>
        )}
      </CardHeader>

      <CardContent className="flex-1 p-0 overflow-hidden">
        <ScrollArea className="h-full px-3 py-3">
          {/* Loading State */}
          {isLoading && (
            <div className="flex items-center justify-center py-12" data-testid={SELECTORS.diffLoading}>
              <Loader2 className="h-6 w-6 animate-spin text-slate-500" />
            </div>
          )}

          {/* Error State */}
          {error && !isLoading && (
            <div className="flex flex-col items-center justify-center py-12 text-center px-4" data-testid={SELECTORS.diffError}>
              <p className="text-sm text-red-400">{error.message}</p>
            </div>
          )}

          {/* Empty State - No changes */}
          {!isLoading && !error && diff && !hasChanges && (
            <div className="flex flex-col items-center justify-center py-12 text-center" data-testid={SELECTORS.diffEmpty}>
              <FileDiff className="h-10 w-10 text-slate-700 mb-4" />
              <p className="text-sm text-slate-400">No changes detected</p>
              <p className="text-xs text-slate-500 mt-1">
                The sandbox has no file modifications
              </p>
            </div>
          )}

          {/* Empty State - No diff loaded */}
          {!isLoading && !error && !diff && (
            <div className="flex flex-col items-center justify-center py-12 text-center" data-testid={SELECTORS.diffEmpty}>
              <FileDiff className="h-10 w-10 text-slate-700 mb-4" />
              <p className="text-sm text-slate-400">Select a sandbox to view changes</p>
              <p className="text-xs text-slate-500 mt-1">
                Click on a sandbox from the list
              </p>
            </div>
          )}

          {/* Diff content */}
          {!isLoading && !error && diff && hasChanges && (
            <div data-testid={SELECTORS.diffFileList}>
              {parsedFiles.map((file) => {
                const fileChange = fileChangeMap.get(file.path);
                const fileId = fileChange?.id ?? "";
                return (
                  <FileDiffSection
                    key={file.path}
                    file={file}
                    fileChange={fileChange}
                    showActions={showFileActions}
                    onApprove={
                      fileChange && onApproveFile
                        ? () => onApproveFile(fileChange.id)
                        : undefined
                    }
                    onReject={
                      fileChange && onRejectFile
                        ? () => onRejectFile(fileChange.id)
                        : undefined
                    }
                    // Hunk selection props [OT-P1-001]
                    showHunkSelection={showHunkSelection}
                    selectedHunkIndices={selectedHunksByFile.get(file.path)}
                    onToggleHunkSelection={
                      onHunkSelectionChange
                        ? (hunkIndex, hunk) => handleToggleHunk(file.path, fileId, hunkIndex, hunk)
                        : undefined
                    }
                  />
                );
              })}
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
