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

interface DiffViewerProps {
  diff?: DiffResult;
  isLoading: boolean;
  error?: Error | null;
  onApproveFile?: (fileId: string) => void;
  onRejectFile?: (fileId: string) => void;
  showFileActions?: boolean;
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

function HunkDisplay({ hunk, index }: { hunk: ParsedHunk; index: number }) {
  return (
    <div className="border-b border-slate-800 last:border-b-0" data-testid={SELECTORS.diffHunk(index)}>
      {/* Hunk header */}
      <div className="bg-slate-800/50 px-3 py-1.5 font-mono text-xs text-blue-400">
        {hunk.header}
      </div>

      {/* Hunk lines */}
      <div>
        {hunk.lines.map((line, lineIdx) => (
          <DiffLine key={lineIdx} line={line} />
        ))}
      </div>
    </div>
  );
}

function FileDiffSection({
  file,
  fileChange,
  onApprove,
  onReject,
  showActions,
}: {
  file: ParsedFileDiff;
  fileChange?: FileChange;
  onApprove?: () => void;
  onReject?: () => void;
  showActions?: boolean;
}) {
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
            <HunkDisplay key={index} hunk={hunk} index={index} />
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
