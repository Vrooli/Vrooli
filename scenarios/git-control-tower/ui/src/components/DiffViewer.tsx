import { useEffect, useState, useRef, useCallback, useMemo } from "react";
import { FileDiff, Plus, Minus, Loader2, AlertTriangle, Copy, Check, ChevronLeft, ChevronRight, Upload, Download, Trash2, X } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Badge } from "./ui/badge";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import { ViewModeSelector } from "./ViewModeSelector";
import { useIsMobile } from "../hooks";
import type { DiffResponse, DiffHunk, ViewMode, AnnotatedLine, LineChange } from "../lib/api";
import { highlightCode, getLanguageFromPath, type HighlightToken, type HighlightedLine } from "../lib/highlighter";

interface DiffViewerProps {
  diff?: DiffResponse;
  selectedFile?: string;
  isStaged: boolean;
  isUntracked: boolean;
  isLoading: boolean;
  error?: Error | null;
  repoDir?: string;
  // View mode control
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  // Mobile action callbacks
  onStage?: (path: string) => void;
  onUnstage?: (path: string) => void;
  onDiscard?: (path: string, untracked: boolean) => void;
  isStaging?: boolean;
  isDiscarding?: boolean;
  // History mode props
  isHistoryMode?: boolean;
  commitHash?: string;
}

// Hook to detect horizontal scroll state
function useScrollHints(ref: React.RefObject<HTMLElement | null>) {
  const [canScrollLeft, setCanScrollLeft] = useState(false);
  const [canScrollRight, setCanScrollRight] = useState(false);

  const checkScroll = useCallback(() => {
    const el = ref.current;
    if (!el) return;
    setCanScrollLeft(el.scrollLeft > 0);
    setCanScrollRight(el.scrollLeft < el.scrollWidth - el.clientWidth - 1);
  }, [ref]);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    checkScroll();
    el.addEventListener("scroll", checkScroll, { passive: true });
    window.addEventListener("resize", checkScroll);

    return () => {
      el.removeEventListener("scroll", checkScroll);
      window.removeEventListener("resize", checkScroll);
    };
  }, [checkScroll, ref]);

  return { canScrollLeft, canScrollRight };
}

// Hook for syntax highlighting
function useHighlighting(content: string | undefined, filePath: string | undefined) {
  const [highlighted, setHighlighted] = useState<HighlightedLine[] | null>(null);
  const [isHighlighting, setIsHighlighting] = useState(false);

  useEffect(() => {
    if (!content || !filePath) {
      setHighlighted(null);
      return;
    }

    let cancelled = false;
    setIsHighlighting(true);

    const language = getLanguageFromPath(filePath);
    highlightCode(content, language)
      .then((result) => {
        if (!cancelled) {
          setHighlighted(result);
          setIsHighlighting(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setHighlighted(null);
          setIsHighlighting(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [content, filePath]);

  return { highlighted, isHighlighting };
}

// Render highlighted tokens
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

// Get background color for line change
function getLineBackground(change?: LineChange): string {
  switch (change) {
    case "added":
      return "bg-emerald-950/30";
    case "deleted":
      return "bg-red-950/30";
    case "modified":
      return "bg-amber-950/30";
    default:
      return "";
  }
}

// Get line number color for change type
function getLineNumberColor(change?: LineChange): string {
  switch (change) {
    case "added":
      return "text-emerald-700";
    case "deleted":
      return "text-red-700";
    case "modified":
      return "text-amber-700";
    default:
      return "text-slate-600";
  }
}

// Diff line for classic diff mode (no syntax highlighting on diff lines)
function DiffLine({ line, lineNumber }: { line: string; lineNumber?: number }) {
  const isAddition = line.startsWith("+") && !line.startsWith("+++");
  const isDeletion = line.startsWith("-") && !line.startsWith("---");
  const isHeader = line.startsWith("@@");
  const isContext = !isAddition && !isDeletion && !isHeader;

  let bgColor = "";
  let textColor = "text-slate-300";
  let lineNumColor = "text-slate-600";

  if (isAddition) {
    bgColor = "bg-emerald-950/30";
    textColor = "text-emerald-300";
    lineNumColor = "text-emerald-700";
  } else if (isDeletion) {
    bgColor = "bg-red-950/30";
    textColor = "text-red-300";
    lineNumColor = "text-red-700";
  } else if (isHeader) {
    bgColor = "bg-blue-950/30";
    textColor = "text-blue-400";
    lineNumColor = "text-blue-700";
  }

  return (
    <div className={`flex font-mono text-xs ${bgColor}`} data-testid="diff-line">
      {lineNumber !== undefined && (
        <span
          className={`w-12 flex-shrink-0 px-2 py-0.5 text-right select-none border-r border-slate-800 ${lineNumColor}`}
        >
          {isContext ? lineNumber : ""}
        </span>
      )}
      <pre className={`flex-1 px-3 py-0.5 whitespace-pre overflow-x-auto ${textColor}`}>
        {line || " "}
      </pre>
    </div>
  );
}

// Syntax-highlighted line for full_diff and source modes
function HighlightedCodeLine({
  lineNumber,
  tokens,
  change,
  oldNumber
}: {
  lineNumber: number;
  tokens?: HighlightToken[];
  change?: LineChange;
  oldNumber?: number;
}) {
  const bgColor = getLineBackground(change);
  const lineNumColor = getLineNumberColor(change);
  const isDeleted = change === "deleted";

  return (
    <div className={`flex font-mono text-xs ${bgColor}`} data-testid="code-line">
      {/* Line number gutter */}
      <span
        className={`w-12 flex-shrink-0 px-2 py-0.5 text-right select-none border-r border-slate-800 ${lineNumColor}`}
      >
        {isDeleted ? (oldNumber || "") : lineNumber}
      </span>
      {/* Change indicator */}
      <span
        className={`w-5 flex-shrink-0 px-1 py-0.5 text-center select-none ${lineNumColor}`}
      >
        {change === "added" && "+"}
        {change === "deleted" && "-"}
      </span>
      {/* Code content */}
      <pre className="flex-1 px-2 py-0.5 whitespace-pre overflow-x-auto text-slate-300">
        {tokens ? <HighlightedTokens tokens={tokens} /> : " "}
      </pre>
    </div>
  );
}

// Hunk display for diff mode
function HunkDisplay({ hunk, index }: { hunk: DiffHunk; index: number }) {
  let currentLine = hunk.new_start;

  return (
    <div className="border-b border-slate-800 last:border-b-0" data-testid={`diff-hunk-${index}`}>
      {/* Hunk header */}
      <div className="bg-slate-800/50 px-3 py-1.5 font-mono text-xs text-slate-500">
        {hunk.header}
      </div>

      {/* Hunk lines */}
      <div className="divide-y divide-slate-800/30">
        {hunk.lines.map((line, lineIdx) => {
          const isAddition = line.startsWith("+") && !line.startsWith("+++");
          const isDeletion = line.startsWith("-") && !line.startsWith("---");
          const lineNum = isDeletion ? undefined : currentLine;

          if (!isDeletion && !line.startsWith("@@")) {
            currentLine++;
          }

          return (
            <DiffLine
              key={`${index}-${lineIdx}`}
              line={line}
              lineNumber={lineNum}
            />
          );
        })}
      </div>
    </div>
  );
}

// Full file view with annotations
function FullFileView({
  annotatedLines,
  highlightedLines,
  showChangeMarkers
}: {
  annotatedLines: AnnotatedLine[];
  highlightedLines: HighlightedLine[] | null;
  showChangeMarkers: boolean;
}) {
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
    <div className="divide-y divide-slate-800/30" data-testid="full-file-content">
      {annotatedLines.map((line, index) => {
        // For deleted lines, we need special handling since they don't exist in current file
        const tokens = line.number > 0 ? highlightMap.get(line.number) : undefined;
        const fallbackTokens: HighlightToken[] = [{ content: line.content }];

        return (
          <HighlightedCodeLine
            key={index}
            lineNumber={line.number}
            tokens={tokens || fallbackTokens}
            change={showChangeMarkers ? line.change : undefined}
            oldNumber={line.old_number}
          />
        );
      })}
    </div>
  );
}

// Simple source view (no change markers)
function SourceView({
  content,
  highlightedLines
}: {
  content: string;
  highlightedLines: HighlightedLine[] | null;
}) {
  const lines = useMemo(() => content.split("\n"), [content]);

  return (
    <div className="divide-y divide-slate-800/30" data-testid="source-content">
      {lines.map((line, index) => {
        const lineNum = index + 1;
        const highlighted = highlightedLines?.find((h) => h.lineNumber === lineNum);
        const tokens = highlighted?.tokens || [{ content: line }];

        return (
          <HighlightedCodeLine
            key={index}
            lineNumber={lineNum}
            tokens={tokens}
          />
        );
      })}
    </div>
  );
}

export function DiffViewer({
  diff,
  selectedFile,
  isStaged,
  isUntracked,
  isLoading,
  error,
  repoDir,
  viewMode,
  onViewModeChange,
  onStage,
  onUnstage,
  onDiscard,
  isStaging = false,
  isDiscarding = false,
  isHistoryMode = false,
  commitHash
}: DiffViewerProps) {
  const isMobile = useIsMobile();
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const { canScrollLeft, canScrollRight } = useScrollHints(scrollContainerRef);
  const [showBinary, setShowBinary] = useState(false);
  const [copied, setCopied] = useState(false);
  const [confirmingDiscard, setConfirmingDiscard] = useState(false);

  const isBinaryDiff = Boolean(
    diff?.raw && (diff.raw.includes("Binary files") || diff.raw.includes("GIT binary patch"))
  );
  const absolutePath =
    selectedFile && repoDir ? `${repoDir.replace(/\/$/, "")}/${selectedFile}` : selectedFile;

  // Get content for syntax highlighting
  const contentForHighlighting = useMemo(() => {
    if (!diff) return undefined;
    if (viewMode === "source" || viewMode === "full_diff") {
      return diff.full_content;
    }
    return undefined;
  }, [diff, viewMode]);

  const { highlighted: highlightedLines, isHighlighting } = useHighlighting(
    contentForHighlighting,
    selectedFile
  );

  // Scroll helpers for mobile
  const scrollLeft = useCallback(() => {
    scrollContainerRef.current?.scrollBy({ left: -150, behavior: "smooth" });
  }, []);
  const scrollRight = useCallback(() => {
    scrollContainerRef.current?.scrollBy({ left: 150, behavior: "smooth" });
  }, []);

  useEffect(() => {
    setShowBinary(false);
  }, [selectedFile, diff?.raw]);

  useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => setCopied(false), 1500);
    return () => window.clearTimeout(timer);
  }, [copied]);

  const handleCopyPath = async () => {
    if (!absolutePath) return;

    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(absolutePath);
        setCopied(true);
        return;
      } catch {
        // Fallback to legacy copy.
      }
    }

    try {
      const textarea = document.createElement("textarea");
      textarea.value = absolutePath;
      textarea.style.position = "fixed";
      textarea.style.opacity = "0";
      document.body.appendChild(textarea);
      textarea.focus();
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setCopied(true);
    } catch {
      // Ignore copy errors.
    }
  };

  const showBinaryNotice =
    selectedFile && !isLoading && !error && diff?.has_diff && isBinaryDiff && !showBinary;

  // Determine what content to show
  const hasAnnotatedLines = diff?.annotated_lines && diff.annotated_lines.length > 0;
  const hasFullContent = diff?.full_content !== undefined;
  const hasHunks = diff?.hunks && diff.hunks.length > 0;

  return (
    <Card className="h-full flex flex-col" data-testid="diff-viewer-panel">
      <CardHeader className={`flex-row items-center justify-between space-y-0 ${isMobile ? "py-4 px-4" : "py-3"}`}>
        <div className={`flex items-center min-w-0 ${isMobile ? "gap-2 flex-1" : "gap-3"}`}>
          <CardTitle className={`flex items-center gap-2 min-w-0 ${isMobile ? "flex-1" : ""}`}>
            <FileDiff className={`flex-shrink-0 text-slate-500 ${isMobile ? "h-5 w-5" : "h-4 w-4"}`} />
            {selectedFile ? (
              <span className={`font-mono truncate ${isMobile ? "text-sm" : "text-xs"}`}>{selectedFile}</span>
            ) : (
              <span className={isMobile ? "text-sm" : "text-xs"}>Diff Viewer</span>
            )}
          </CardTitle>
          {selectedFile && (
            <button
              type="button"
              className={`inline-flex items-center justify-center rounded-full border border-white/20 text-slate-300 transition-colors hover:bg-white/10 active:bg-white/20 flex-shrink-0 ${
                isMobile ? "h-10 w-10 touch-target" : "h-7 w-7"
              }`}
              onClick={handleCopyPath}
              title={copied ? "Copied" : "Copy absolute path"}
              aria-label="Copy absolute path"
              data-testid="copy-absolute-path"
            >
              {copied ? (
                <Check className={`text-emerald-300 ${isMobile ? "h-4 w-4" : "h-3.5 w-3.5"}`} />
              ) : (
                <Copy className={isMobile ? "h-4 w-4" : "h-3.5 w-3.5"} />
              )}
            </button>
          )}
          {selectedFile && !isMobile && (
            isHistoryMode ? (
              <Badge variant="warning">
                {commitHash ? commitHash.substring(0, 7) : "history"}
              </Badge>
            ) : (
              <Badge variant={isUntracked ? "untracked" : isStaged ? "staged" : "unstaged"}>
                {isUntracked ? "untracked" : isStaged ? "staged" : "unstaged"}
              </Badge>
            )
          )}
        </div>

        <div className={`flex items-center ${isMobile ? "gap-2" : "gap-3"}`}>
          {/* View mode selector - only show when file is selected */}
          {selectedFile && !isLoading && !error && (
            <ViewModeSelector
              mode={viewMode}
              onChange={onViewModeChange}
              compact={isMobile}
              disabled={isLoading}
            />
          )}

          {/* Mobile: show badge in header right */}
          {selectedFile && isMobile && (
            isHistoryMode ? (
              <Badge variant="warning">
                {commitHash ? commitHash.substring(0, 7) : "hist"}
              </Badge>
            ) : (
              <Badge variant={isUntracked ? "untracked" : isStaged ? "staged" : "unstaged"}>
                {isUntracked ? "new" : isStaged ? "staged" : "mod"}
              </Badge>
            )
          )}
          {diff?.stats && diff.has_diff && viewMode !== "source" && (
            <div className="flex items-center gap-2" data-testid="diff-stats">
              <span className={`flex items-center gap-1 text-emerald-500 ${isMobile ? "text-sm" : "text-xs"}`}>
                <Plus className={isMobile ? "h-4 w-4" : "h-3 w-3"} />
                {diff.stats.additions}
              </span>
              <span className={`flex items-center gap-1 text-red-500 ${isMobile ? "text-sm" : "text-xs"}`}>
                <Minus className={isMobile ? "h-4 w-4" : "h-3 w-3"} />
                {diff.stats.deletions}
              </span>
            </div>
          )}
        </div>
      </CardHeader>

      <CardContent className="flex-1 p-0 overflow-hidden relative">
        {/* Mobile horizontal scroll hints */}
        {isMobile && (canScrollLeft || canScrollRight) && (
          <>
            {canScrollLeft && (
              <button
                type="button"
                onClick={scrollLeft}
                className="absolute left-0 top-1/2 -translate-y-1/2 z-10 h-12 w-8 flex items-center justify-center bg-gradient-to-r from-slate-950 to-transparent touch-target"
                aria-label="Scroll left"
              >
                <ChevronLeft className="h-5 w-5 text-slate-400" />
              </button>
            )}
            {canScrollRight && (
              <button
                type="button"
                onClick={scrollRight}
                className="absolute right-0 top-1/2 -translate-y-1/2 z-10 h-12 w-8 flex items-center justify-center bg-gradient-to-l from-slate-950 to-transparent touch-target"
                aria-label="Scroll right"
              >
                <ChevronRight className="h-5 w-5 text-slate-400" />
              </button>
            )}
          </>
        )}

        <ScrollArea className="h-full" ref={scrollContainerRef}>
          {/* Loading State */}
          {(isLoading || isHighlighting) && (
            <div className="flex items-center justify-center py-12" data-testid="diff-loading">
              <Loader2 className="h-6 w-6 animate-spin text-slate-500" />
            </div>
          )}

          {/* Error State */}
          {error && !isLoading && (
            <div className="flex flex-col items-center justify-center py-12 text-center px-4" data-testid="diff-error">
              <p className={`text-red-400 ${isMobile ? "text-base" : "text-sm"}`}>{error.message}</p>
            </div>
          )}

          {/* Empty State - No file selected */}
          {!selectedFile && !isLoading && !error && (
            <div className="flex flex-col items-center justify-center py-12 text-center px-4" data-testid="diff-empty">
              <FileDiff className={`text-slate-700 mb-4 ${isMobile ? "h-12 w-12" : "h-10 w-10"}`} />
              <p className={`text-slate-500 ${isMobile ? "text-base" : "text-sm"}`}>Select a file to view changes</p>
              <p className={`text-slate-600 mt-1 ${isMobile ? "text-sm" : "text-xs"}`}>
                {isMobile ? "Tap a file from the Changes tab" : "Click on a file from the list on the left"}
              </p>
            </div>
          )}

          {/* No diff content */}
          {selectedFile && !isLoading && !isHighlighting && !error && diff && !diff.has_diff && !hasFullContent && (
            <div className="flex flex-col items-center justify-center py-12 text-center" data-testid="diff-no-changes">
              <FileDiff className="h-10 w-10 text-slate-700 mb-4" />
              <p className="text-sm text-slate-500">No changes detected</p>
              <p className="text-xs text-slate-600 mt-1">
                This file has no differences from HEAD
              </p>
            </div>
          )}

          {/* Source mode - just the file content */}
          {selectedFile && !isLoading && !isHighlighting && !error && viewMode === "source" && hasFullContent && (
            <SourceView
              content={diff!.full_content!}
              highlightedLines={highlightedLines}
            />
          )}

          {/* Full + Diff mode - full file with change annotations */}
          {selectedFile && !isLoading && !isHighlighting && !error && viewMode === "full_diff" && hasAnnotatedLines && (
            <FullFileView
              annotatedLines={diff!.annotated_lines!}
              highlightedLines={highlightedLines}
              showChangeMarkers={true}
            />
          )}

          {/* Diff mode - traditional hunk view */}
          {selectedFile && !isLoading && !isHighlighting && !error && viewMode === "diff" && hasHunks && (
            <div data-testid="diff-content">
              {diff!.hunks!.map((hunk, index) => (
                <HunkDisplay key={index} hunk={hunk} index={index} />
              ))}
            </div>
          )}

          {/* Fallback for untracked files in diff mode (show full content) */}
          {selectedFile && !isLoading && !isHighlighting && !error && viewMode === "diff" && !hasHunks && hasFullContent && isUntracked && (
            <SourceView
              content={diff!.full_content!}
              highlightedLines={highlightedLines}
            />
          )}

          {/* Binary diff notice */}
          {showBinaryNotice && (
            <div className="flex flex-col items-center justify-center py-16 text-center px-6">
              <AlertTriangle className="h-10 w-10 text-amber-400 mb-4" />
              <p className="text-sm text-slate-300">
                The file is not displayed in the text editor because it is either binary or uses an unsupported text encoding.
              </p>
              <Button
                variant="outline"
                size="sm"
                className="mt-4"
                onClick={() => setShowBinary(true)}
              >
                Show Anyway
              </Button>
            </div>
          )}

          {/* Raw diff fallback */}
          {selectedFile &&
            !isLoading &&
            !error &&
            diff?.has_diff &&
            !hasHunks &&
            !hasFullContent &&
            diff.raw &&
            (!isBinaryDiff || showBinary) && (
            <pre
              className="p-4 font-mono text-xs text-slate-300 whitespace-pre overflow-x-auto"
              data-testid="diff-raw"
            >
              {diff.raw}
            </pre>
          )}

          {/* Mobile spacer to account for fixed action bar */}
          {isMobile && selectedFile && !isLoading && !isHistoryMode && <div className="h-16" aria-hidden="true" />}
        </ScrollArea>

        {/* Mobile Action Bar - hidden in history mode */}
        {isMobile && selectedFile && !isLoading && !isHistoryMode && (
          <div className="absolute bottom-0 left-0 right-0 p-3 bg-slate-900/95 backdrop-blur-sm border-t border-slate-800" data-testid="diff-mobile-actions">
            {confirmingDiscard ? (
              <div className="flex items-center gap-2">
                <span className="text-sm text-amber-400 flex-1">
                  {isUntracked ? "Delete this file?" : "Discard changes?"}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-10 px-4 touch-target"
                  onClick={() => setConfirmingDiscard(false)}
                  disabled={isDiscarding}
                >
                  <X className="h-4 w-4 mr-1" />
                  Cancel
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-10 px-4 touch-target bg-red-600 border-red-600 hover:bg-red-700 text-white"
                  onClick={() => {
                    onDiscard?.(selectedFile, isUntracked);
                    setConfirmingDiscard(false);
                  }}
                  disabled={isDiscarding}
                >
                  <Trash2 className="h-4 w-4 mr-1" />
                  {isDiscarding ? "..." : isUntracked ? "Delete" : "Discard"}
                </Button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                {/* Stage/Unstage button */}
                {isStaged ? (
                  <Button
                    variant="outline"
                    size="sm"
                    className="flex-1 h-10 touch-target"
                    onClick={() => onUnstage?.(selectedFile)}
                    disabled={isStaging}
                  >
                    <Download className="h-4 w-4 mr-2" />
                    {isStaging ? "Unstaging..." : "Unstage"}
                  </Button>
                ) : (
                  <Button
                    variant="default"
                    size="sm"
                    className="flex-1 h-10 touch-target bg-emerald-600 hover:bg-emerald-700"
                    onClick={() => onStage?.(selectedFile)}
                    disabled={isStaging}
                  >
                    <Upload className="h-4 w-4 mr-2" />
                    {isStaging ? "Staging..." : "Stage"}
                  </Button>
                )}

                {/* Discard button - only for unstaged/untracked files */}
                {!isStaged && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-10 px-4 touch-target text-red-400 border-red-400/50 hover:bg-red-950/50"
                    onClick={() => setConfirmingDiscard(true)}
                    disabled={isDiscarding}
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    {isUntracked ? "Delete" : "Discard"}
                  </Button>
                )}
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
