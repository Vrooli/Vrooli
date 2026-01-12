import { useState, useMemo, useRef, useEffect, useCallback } from "react";
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
import { ViewModeSelector } from "./ViewModeSelector";
import type { DiffResult, FileChange, ChangeType, ViewMode, AnnotatedLine } from "../lib/api";
import { SELECTORS } from "../consts/selectors";
import {
  highlightCode,
  getLanguageFromPath,
  type HighlightToken,
  type HighlightedLine,
} from "../lib/highlighter";

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
  // File-level selection props for partial approval
  showFileSelection?: boolean;
  selectedFiles?: string[];
  onFileSelectionChange?: (fileIds: string[]) => void;
  // Hunk-level selection props [OT-P1-001]
  showHunkSelection?: boolean;
  selectedHunks?: HunkSelection[];
  onHunkSelectionChange?: (hunks: HunkSelection[]) => void;
  // View mode props
  viewMode?: ViewMode;
  onViewModeChange?: (mode: ViewMode) => void;
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

// Hook for async syntax highlighting
function useHighlighting(content: string | undefined, filePath: string | undefined) {
  const [highlightedLines, setHighlightedLines] = useState<HighlightedLine[] | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (!content || !filePath) {
      setHighlightedLines(null);
      return;
    }

    let cancelled = false;
    setIsLoading(true);

    const language = getLanguageFromPath(filePath);
    highlightCode(content, language)
      .then((lines) => {
        if (!cancelled) {
          setHighlightedLines(lines);
          setIsLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setHighlightedLines(null);
          setIsLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [content, filePath]);

  return { highlightedLines, isLoading };
}

// Renders syntax-highlighted tokens
function HighlightedTokens({ tokens }: { tokens: HighlightToken[] }) {
  return (
    <>
      {tokens.map((token, i) => (
        <span
          key={i}
          style={{ color: token.color }}
          className={token.fontStyle === "italic" ? "italic" : token.fontStyle === "bold" ? "font-bold" : ""}
        >
          {token.content}
        </span>
      ))}
    </>
  );
}

// Renders a single line with syntax highlighting and optional change marker
interface HighlightedCodeLineProps {
  lineNumber: number;
  tokens?: HighlightToken[];
  content: string;
  change?: "" | "added" | "deleted";
  showChangeMarker?: boolean;
}

function HighlightedCodeLine({
  lineNumber,
  tokens,
  content,
  change,
  showChangeMarker = false,
}: HighlightedCodeLineProps) {
  let bgColor = "";
  let markerColor = "text-slate-600";

  if (change === "added") {
    bgColor = "bg-emerald-950/30";
    markerColor = "text-emerald-400";
  } else if (change === "deleted") {
    bgColor = "bg-red-950/30";
    markerColor = "text-red-400";
  }

  return (
    <div className={`flex font-mono text-xs ${bgColor}`} data-testid={SELECTORS.diffLine}>
      {showChangeMarker && (
        <span className={`w-6 flex-shrink-0 text-center select-none ${markerColor}`}>
          {change === "added" ? "+" : change === "deleted" ? "-" : " "}
        </span>
      )}
      <span className="w-10 flex-shrink-0 text-right pr-2 text-slate-600 select-none">
        {lineNumber > 0 ? lineNumber : ""}
      </span>
      <pre className="flex-1 px-2 py-0.5 whitespace-pre overflow-x-auto text-slate-300">
        {tokens ? <HighlightedTokens tokens={tokens} /> : content || " "}
      </pre>
    </div>
  );
}

// Full file view with change annotations (full_diff mode)
interface FullFileViewProps {
  annotatedLines: AnnotatedLine[];
  highlightedLines: HighlightedLine[] | null;
  filePath: string;
}

function FullFileView({ annotatedLines, highlightedLines, filePath }: FullFileViewProps) {
  // Create a map from line number to highlighted tokens
  const highlightMap = useMemo(() => {
    if (!highlightedLines) return new Map<number, HighlightToken[]>();
    const map = new Map<number, HighlightToken[]>();
    highlightedLines.forEach((line) => {
      map.set(line.lineNumber, line.tokens);
    });
    return map;
  }, [highlightedLines]);

  return (
    <div data-testid="full-file-content">
      {annotatedLines.map((line, index) => (
        <HighlightedCodeLine
          key={index}
          lineNumber={line.number}
          tokens={line.number > 0 ? highlightMap.get(line.number) : undefined}
          content={line.content}
          change={line.change || ""}
          showChangeMarker={true}
        />
      ))}
    </div>
  );
}

// Source view - clean file content without change markers (source mode)
interface SourceViewProps {
  content: string;
  highlightedLines: HighlightedLine[] | null;
  filePath: string;
}

function SourceView({ content, highlightedLines, filePath }: SourceViewProps) {
  const lines = useMemo(() => content.split("\n"), [content]);

  // Create a map from line number to highlighted tokens
  const highlightMap = useMemo(() => {
    if (!highlightedLines) return new Map<number, HighlightToken[]>();
    const map = new Map<number, HighlightToken[]>();
    highlightedLines.forEach((line) => {
      map.set(line.lineNumber, line.tokens);
    });
    return map;
  }, [highlightedLines]);

  return (
    <div data-testid="source-content">
      {lines.map((line, index) => (
        <HighlightedCodeLine
          key={index}
          lineNumber={index + 1}
          tokens={highlightMap.get(index + 1)}
          content={line}
          showChangeMarker={false}
        />
      ))}
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
  // File-level selection props
  showFileSelection?: boolean;
  isFileSelected?: boolean;
  isFileIndeterminate?: boolean;
  onToggleFileSelection?: () => void;
  // Hunk selection props [OT-P1-001]
  showHunkSelection?: boolean;
  selectedHunkIndices?: Set<number>;
  onToggleHunkSelection?: (hunkIndex: number, hunk: ParsedHunk) => void;
  // View mode props
  viewMode?: ViewMode;
  fullContent?: string;
  annotatedLines?: AnnotatedLine[];
}

function FileDiffSection({
  file,
  fileChange,
  onApprove,
  onReject,
  showActions,
  showFileSelection,
  isFileSelected,
  isFileIndeterminate,
  onToggleFileSelection,
  showHunkSelection,
  selectedHunkIndices,
  onToggleHunkSelection,
  viewMode = "diff",
  fullContent,
  annotatedLines,
}: FileDiffSectionProps) {
  const [expanded, setExpanded] = useState(true);
  const fileCheckboxRef = useRef<HTMLInputElement>(null);

  // Get syntax highlighting for full_diff and source modes
  const { highlightedLines } = useHighlighting(
    viewMode !== "diff" ? fullContent : undefined,
    viewMode !== "diff" ? file.path : undefined
  );

  // Set indeterminate state on checkbox (can't be set via attribute)
  useEffect(() => {
    if (fileCheckboxRef.current) {
      fileCheckboxRef.current.indeterminate = isFileIndeterminate ?? false;
    }
  }, [isFileIndeterminate]);

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

  const acceptanceBadge = (() => {
    const status = fileChange?.acceptance?.status;
    if (!status) return null;
    switch (status) {
      case "accepted":
        return { label: "accepted", variant: "success" as const };
      case "ignored":
        return { label: "ignored", variant: "warning" as const };
      case "denied":
        return { label: "denied", variant: "error" as const };
      case "binary_ignored":
        return { label: "binary", variant: "warning" as const };
      default:
        return null;
    }
  })();

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
        {/* File-level selection checkbox */}
        {showFileSelection && (
          <label
            className="flex items-center cursor-pointer"
            onClick={(e) => e.stopPropagation()}
            data-testid={`file-checkbox-${fileChange?.id || file.path}`}
          >
            <input
              ref={fileCheckboxRef}
              type="checkbox"
              checked={isFileSelected}
              onChange={onToggleFileSelection}
              className="w-4 h-4 rounded border-slate-600 bg-slate-800 text-emerald-500 focus:ring-emerald-500 focus:ring-offset-0 cursor-pointer"
            />
          </label>
        )}
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
        {acceptanceBadge && (
          <Badge variant={acceptanceBadge.variant} className="text-[10px]">
            {acceptanceBadge.label}
          </Badge>
        )}

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
          {/* Diff mode - show hunks */}
          {viewMode === "diff" && (
            <>
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
            </>
          )}

          {/* Full diff mode - show full file with change annotations */}
          {viewMode === "full_diff" && annotatedLines && (
            <FullFileView
              annotatedLines={annotatedLines}
              highlightedLines={highlightedLines}
              filePath={file.path}
            />
          )}

          {/* Source mode - show clean file content */}
          {viewMode === "source" && fullContent && (
            <SourceView
              content={fullContent}
              highlightedLines={highlightedLines}
              filePath={file.path}
            />
          )}

          {/* No content available for non-diff modes */}
          {viewMode !== "diff" && !fullContent && !annotatedLines && (
            <div className="px-3 py-4 text-xs text-slate-500 text-center">
              {file.changeType === "deleted"
                ? "File was deleted"
                : "Content not available (binary file?)"}
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
  showFileSelection = false,
  selectedFiles = [],
  onFileSelectionChange,
  showHunkSelection = false,
  selectedHunks = [],
  onHunkSelectionChange,
  viewMode = "diff",
  onViewModeChange,
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

  // Build a set of selected file IDs for quick lookup
  const selectedFileSet = useMemo(() => new Set(selectedFiles), [selectedFiles]);

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

  // Handle toggling file selection - also selects/deselects all hunks
  const handleToggleFile = (filePath: string, fileId: string, hunks: ParsedHunk[]) => {
    if (!onFileSelectionChange || !onHunkSelectionChange) return;

    const currentHunkSelection = selectedHunksByFile.get(filePath);
    const selectedCount = currentHunkSelection?.size ?? 0;
    const totalHunks = hunks.length;

    // If all hunks selected (or file is selected), deselect all
    // If no hunks or partial, select all
    const shouldSelectAll = selectedCount < totalHunks;

    if (shouldSelectAll) {
      // Add file to selection
      if (!selectedFileSet.has(fileId)) {
        onFileSelectionChange([...selectedFiles, fileId]);
      }
      // Add all hunks that aren't already selected
      const hunksToAdd = hunks
        .map((hunk, index) => ({
          fileId,
          filePath,
          startLine: hunk.newStart,
          endLine: hunk.newStart + hunk.newCount - 1,
          hunkIndex: index,
        }))
        .filter((h) => !currentHunkSelection?.has(h.hunkIndex));
      onHunkSelectionChange([...selectedHunks, ...hunksToAdd]);
    } else {
      // Remove file from selection
      onFileSelectionChange(selectedFiles.filter((id) => id !== fileId));
      // Remove all hunks for this file
      onHunkSelectionChange(selectedHunks.filter((h) => h.filePath !== filePath));
    }
  };

  // Handle toggling hunk selection - also updates file selection when all hunks selected/deselected
  const handleToggleHunk = (filePath: string, fileId: string, hunkIndex: number, hunk: ParsedHunk, totalHunks: number) => {
    if (!onHunkSelectionChange || !onFileSelectionChange) return;

    const isCurrentlySelected = selectedHunks.some(
      (s) => s.filePath === filePath && s.hunkIndex === hunkIndex
    );

    if (isCurrentlySelected) {
      // Remove this hunk from selection
      const newHunkSelection = selectedHunks.filter(
        (s) => !(s.filePath === filePath && s.hunkIndex === hunkIndex)
      );
      onHunkSelectionChange(newHunkSelection);

      // If no more hunks selected for this file, also remove file from selection
      const remainingHunksForFile = newHunkSelection.filter((h) => h.filePath === filePath);
      if (remainingHunksForFile.length === 0 && selectedFileSet.has(fileId)) {
        onFileSelectionChange(selectedFiles.filter((id) => id !== fileId));
      }
    } else {
      // Add this hunk to selection
      const newHunkSelection = [
        ...selectedHunks,
        {
          fileId,
          filePath,
          startLine: hunk.newStart,
          endLine: hunk.newStart + hunk.newCount - 1,
          hunkIndex,
        },
      ];
      onHunkSelectionChange(newHunkSelection);

      // If all hunks now selected, also add file to selection
      const newHunksForFile = newHunkSelection.filter((h) => h.filePath === filePath);
      if (newHunksForFile.length === totalHunks && !selectedFileSet.has(fileId)) {
        onFileSelectionChange([...selectedFiles, fileId]);
      }
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

        <div className="flex items-center gap-3">
          {/* View mode selector */}
          {onViewModeChange && diff && hasChanges && (
            <ViewModeSelector
              mode={viewMode}
              onChange={onViewModeChange}
              disabled={showHunkSelection}
              compact={true}
            />
          )}

        {diff && hasChanges && (
          <div className="flex items-center gap-3" data-testid={SELECTORS.diffStats}>
            {/* Bulk select controls */}
            {showFileSelection && diff.files && diff.files.length > 0 && (
              <>
                <button
                  className="text-xs text-slate-400 hover:text-slate-200 transition-colors px-2 py-1 rounded hover:bg-slate-800"
                  onClick={() => {
                    if (!onFileSelectionChange || !onHunkSelectionChange || !diff.files) return;
                    const allFileIds = diff.files.map((f) => f.id);
                    // Check if all hunks are selected (fully selected state)
                    const totalHunks = parsedFiles.reduce((sum, f) => sum + f.hunks.length, 0);
                    const isAllSelected = selectedHunks.length === totalHunks && totalHunks > 0;

                    if (isAllSelected) {
                      // Deselect all files and hunks
                      onFileSelectionChange([]);
                      onHunkSelectionChange([]);
                    } else {
                      // Select all files and all hunks
                      onFileSelectionChange(allFileIds);
                      const allHunks: HunkSelection[] = [];
                      parsedFiles.forEach((file) => {
                        const fileChange = fileChangeMap.get(file.path);
                        const fileId = fileChange?.id ?? "";
                        file.hunks.forEach((hunk, index) => {
                          allHunks.push({
                            fileId,
                            filePath: file.path,
                            startLine: hunk.newStart,
                            endLine: hunk.newStart + hunk.newCount - 1,
                            hunkIndex: index,
                          });
                        });
                      });
                      onHunkSelectionChange(allHunks);
                    }
                  }}
                  data-testid="select-all-button"
                >
                  {(() => {
                    const totalHunks = parsedFiles.reduce((sum, f) => sum + f.hunks.length, 0);
                    return selectedHunks.length === totalHunks && totalHunks > 0
                      ? "Deselect All"
                      : "Select All";
                  })()}
                </button>
                {selectedHunks.length > 0 && (
                  <span className="text-xs text-emerald-500">
                    {selectedHunks.length} hunks selected
                  </span>
                )}
                <span className="text-slate-700">|</span>
              </>
            )}
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
        </div>
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
                const totalHunks = file.hunks.length;
                const selectedHunkCount = selectedHunksByFile.get(file.path)?.size ?? 0;
                // File is fully selected if all hunks are selected
                const isFileFullySelected = totalHunks > 0 && selectedHunkCount === totalHunks;
                // File is indeterminate if some (but not all) hunks are selected
                const isFileIndeterminate = selectedHunkCount > 0 && selectedHunkCount < totalHunks;
                // Get file content for non-diff view modes
                const fileViewData = diff.fileContents?.[file.path];
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
                    // File selection props
                    showFileSelection={showFileSelection}
                    isFileSelected={isFileFullySelected}
                    isFileIndeterminate={isFileIndeterminate}
                    onToggleFileSelection={
                      onFileSelectionChange && onHunkSelectionChange
                        ? () => handleToggleFile(file.path, fileId, file.hunks)
                        : undefined
                    }
                    // Hunk selection props [OT-P1-001]
                    showHunkSelection={showHunkSelection}
                    selectedHunkIndices={selectedHunksByFile.get(file.path)}
                    onToggleHunkSelection={
                      onHunkSelectionChange && onFileSelectionChange
                        ? (hunkIndex, hunk) => handleToggleHunk(file.path, fileId, hunkIndex, hunk, totalHunks)
                        : undefined
                    }
                    // View mode props
                    viewMode={viewMode}
                    fullContent={fileViewData?.fullContent}
                    annotatedLines={fileViewData?.annotatedLines}
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
