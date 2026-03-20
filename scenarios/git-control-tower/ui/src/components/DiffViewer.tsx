import { useEffect, useState, useRef, useCallback, useMemo } from "react";
import Editor, { type Monaco as MonacoInstance } from "@monaco-editor/react";
import type * as Monaco from "monaco-editor";
import { FileDiff, Plus, Minus, Loader2, AlertTriangle, Copy, Check, ChevronLeft, ChevronRight, Upload, Download, Trash2, X, Link2, Pencil, Save, RotateCcw, MoreVertical } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { Badge } from "./ui/badge";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import { ViewModeSelector } from "./ViewModeSelector";
import { MarkdownPreview } from "./MarkdownPreview";
import { ImagePreview } from "./ImagePreview";
import { useIsMobile } from "../hooks";
import {
  FileContentConflictError,
  type DiffResponse,
  type DiffHunk,
  type SaveFileContentResponse,
  type ViewMode,
  type AnnotatedLine,
  type LineChange
} from "../lib/api";
import { highlightCode, getLanguageFromPath, type HighlightToken, type HighlightedLine } from "../lib/highlighter";
import { getFileTypeInfo } from "../lib/fileTypes";
import { ChangeMetricsModal } from "./ChangeMetricsModal";
import { BottomSheet, BottomSheetAction } from "./ui/bottom-sheet";
import { formatPath } from "../lib/utils";

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
  // Related files
  onShowRelatedFiles?: (path: string) => void;
  // Read-only mode (viewing any file from search)
  isReadOnly?: boolean;
  onSaveFileContent?: (path: string, content: string, expectedHash?: string) => Promise<SaveFileContentResponse>;
  isSavingFile?: boolean;
  onDeletePath?: (path: string, isDir: boolean) => void;
  isDeleting?: boolean;
}

const maxHighlightChars = 200000;
const minimapMinLines = 80;
const minimapMaxMarkers = 180;
const monacoThemeName = "git-control-tower-dark";

interface MinimapMarker {
  topPercent: number;
  change: Exclude<LineChange, "">;
}

interface MinimapTextureRow {
  topPercent: number;
  widthPercent: number;
  opacity: number;
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function getChangedLineNumber(line: AnnotatedLine, fallbackLine: number): number {
  if (line.number > 0) return line.number;
  if (line.old_number && line.old_number > 0) return line.old_number;
  return fallbackLine;
}

function markerPriority(change: Exclude<LineChange, "">): number {
  switch (change) {
    case "deleted":
      return 3;
    case "modified":
      return 2;
    case "added":
      return 1;
    default:
      return 0;
  }
}

function buildMinimapMarkers(annotatedLines: AnnotatedLine[]): MinimapMarker[] {
  const changedLines = annotatedLines
    .map((line, index) => ({ line, index }))
    .filter(({ line }) => line.change === "added" || line.change === "deleted" || line.change === "modified");

  if (changedLines.length === 0) return [];

  const maxLineNumber = Math.max(
    ...annotatedLines.map((line, index) => getChangedLineNumber(line, index + 1)),
    annotatedLines.length
  );
  const bucketCount = Math.min(minimapMaxMarkers, Math.max(maxLineNumber, 1));
  const buckets = new Map<number, Exclude<LineChange, "">>();

  changedLines.forEach(({ line, index }) => {
    const change = line.change as Exclude<LineChange, "">;
    const lineNumber = getChangedLineNumber(line, index + 1);
    const ratio = maxLineNumber <= 1 ? 0 : (lineNumber - 1) / (maxLineNumber - 1);
    const bucket = clamp(Math.round(ratio * (bucketCount - 1)), 0, bucketCount - 1);
    const existing = buckets.get(bucket);
    if (!existing || markerPriority(change) >= markerPriority(existing)) {
      buckets.set(bucket, change);
    }
  });

  return Array.from(buckets.entries())
    .sort((a, b) => a[0] - b[0])
    .map(([bucket, change]) => ({
      topPercent: bucketCount <= 1 ? 0 : (bucket / (bucketCount - 1)) * 100,
      change
    }));
}

function scrollTopFromMinimapPointer(
  pointerOffsetY: number,
  railHeight: number,
  scrollHeight: number,
  clientHeight: number
): number {
  if (railHeight <= 0) return 0;
  const ratio = clamp(pointerOffsetY / railHeight, 0, 1);
  const maxScrollable = Math.max(scrollHeight - clientHeight, 0);
  return ratio * maxScrollable;
}

function getMinimapMarkerClass(change: Exclude<LineChange, "">): string {
  switch (change) {
    case "added":
      return "bg-emerald-400/80";
    case "deleted":
      return "bg-red-400/80";
    case "modified":
      return "bg-amber-400/80";
    default:
      return "bg-slate-400/80";
  }
}

function indentationDepth(line: string): number {
  let depth = 0;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (ch === " ") {
      depth += 1;
    } else if (ch === "\t") {
      depth += 2;
    } else {
      break;
    }
  }
  return depth;
}

function textureFromLine(rawLine: string): { widthPercent: number; opacity: number } {
  const line = rawLine.replace(/\s+$/g, "");
  if (!line.trim()) {
    return { widthPercent: 18, opacity: 0.2 };
  }

  const indent = indentationDepth(line);
  const indentPenalty = Math.min(indent * 1.3, 32);
  const lengthFactor = Math.min(line.length, 120);
  const widthPercent = clamp(30 + (lengthFactor / 120) * 68 - indentPenalty, 22, 96);

  const trimmed = line.trim();
  let opacity = 0.35;
  if (/^(import|export|class|interface|type|function|const|let|var)\b/.test(trimmed)) {
    opacity = 0.72;
  } else if (/^(#|##|###|####)/.test(trimmed)) {
    opacity = 0.7;
  } else if (/^(\}|\]|\)|return\b)/.test(trimmed)) {
    opacity = 0.42;
  } else if (trimmed.length < 10) {
    opacity = 0.3;
  } else if (trimmed.length > 80) {
    opacity = 0.5;
  }

  return { widthPercent, opacity };
}

function getMonacoLanguage(filePath?: string): string {
  if (!filePath) return "plaintext";
  const detected = getLanguageFromPath(filePath);
  if (!detected) return "plaintext";
  const languageMap: Record<string, string> = {
    jsx: "javascript",
    tsx: "typescript",
    bash: "shell",
    sh: "shell",
    zsh: "shell",
    fish: "shell",
    "objective-c": "cpp",
    markdown: "markdown",
    jsonc: "json",
    yml: "yaml"
  };
  return languageMap[detected] ?? detected;
}

function defineMonacoTheme(monaco: MonacoInstance): void {
  monaco.editor.defineTheme(monacoThemeName, {
    base: "vs-dark",
    inherit: true,
    rules: [],
    colors: {
      "editor.background": "#020617",
      "editor.foreground": "#d4d4d4",
      "editorLineNumber.foreground": "#444d56",
      "editorLineNumber.activeForeground": "#e1e4e8",
      "editorCursor.foreground": "#c8e1ff",
      "editor.selectionBackground": "#3392FF44",
      "editor.selectionHighlightBackground": "#17E5E633",
      "editor.lineHighlightBackground": "#2b303655",
      "editorLineNumber.background": "#020617",
      "editorGutter.background": "#020617",
      "editorWhitespace.foreground": "#444d56",
      "editorIndentGuide.background1": "#2f363d",
      "editorIndentGuide.activeBackground1": "#444d56",
      "scrollbarSlider.background": "#6a737d33",
      "scrollbarSlider.hoverBackground": "#6a737d44",
      "scrollbarSlider.activeBackground": "#6a737d88",
      "editorOverviewRuler.border": "#1b1f23"
    }
  });
}

function buildMinimapTextureRows(lines: string[], maxRows = 220): MinimapTextureRow[] {
  if (lines.length === 0) return [];

  const bucketCount = Math.min(maxRows, lines.length);
  const rows: MinimapTextureRow[] = [];
  const linesPerBucket = lines.length / bucketCount;

  for (let bucket = 0; bucket < bucketCount; bucket++) {
    const start = Math.floor(bucket * linesPerBucket);
    const end = Math.min(lines.length, Math.floor((bucket + 1) * linesPerBucket));
    const bucketLines = lines.slice(start, Math.max(end, start + 1));

    let widthSum = 0;
    let opacitySum = 0;
    bucketLines.forEach((line) => {
      const metrics = textureFromLine(line);
      widthSum += metrics.widthPercent;
      opacitySum += metrics.opacity;
    });

    const count = Math.max(bucketLines.length, 1);
    rows.push({
      topPercent: bucketCount <= 1 ? 0 : (bucket / (bucketCount - 1)) * 100,
      widthPercent: widthSum / count,
      opacity: opacitySum / count
    });
  }

  return rows;
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
      setIsHighlighting(false);
      return;
    }
    if (content.length > maxHighlightChars) {
      setHighlighted(null);
      setIsHighlighting(false);
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
      setIsHighlighting(false);
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
      <pre className={`flex-1 px-3 py-0.5 whitespace-pre ${textColor}`}>
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
      <pre className="flex-1 px-2 py-0.5 whitespace-pre text-slate-300">
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
  commitHash,
  onShowRelatedFiles,
  isReadOnly = false,
  onSaveFileContent,
  isSavingFile = false,
  onDeletePath,
  isDeleting = false
}: DiffViewerProps) {
  const isMobile = useIsMobile();
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const minimapRailRef = useRef<HTMLDivElement>(null);
  const titleRowRef = useRef<HTMLDivElement>(null);
  const [maxPathChars, setMaxPathChars] = useState(60);
  const { canScrollLeft, canScrollRight } = useScrollHints(scrollContainerRef);
  const [showBinary, setShowBinary] = useState(false);
  const [copied, setCopied] = useState(false);
  const [confirmingDiscard, setConfirmingDiscard] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [draftContent, setDraftContent] = useState("");
  const [expectedHash, setExpectedHash] = useState<string | undefined>();
  const [saveError, setSaveError] = useState<string | null>(null);
  const [conflictHash, setConflictHash] = useState<string | null>(null);
  const monacoEditorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
  const monacoRef = useRef<MonacoInstance | null>(null);
  const monacoDecorationIdsRef = useRef<string[]>([]);
  const [scrollMetrics, setScrollMetrics] = useState({
    scrollTop: 0,
    scrollHeight: 1,
    clientHeight: 1
  });
  const [metricsOpen, setMetricsOpen] = useState(false);
  const [mobileActionsOpen, setMobileActionsOpen] = useState(false);

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
  const annotatedLines = useMemo(() => diff?.annotated_lines ?? [], [diff?.annotated_lines]);
  const hunks = useMemo(() => diff?.hunks ?? [], [diff?.hunks]);
  const fullContent = diff?.full_content ?? "";
  const hasAnnotatedLines = annotatedLines.length > 0;
  const hasFullContent = diff?.full_content !== undefined;
  const hasHunks = hunks.length > 0;
  const fullContentLineCount = useMemo(() => {
    if (!fullContent) return 0;
    return fullContent.split("\n").length;
  }, [fullContent]);
  const isPreviewable = selectedFile ? getFileTypeInfo(selectedFile) : null;
  const canEditMode = viewMode === "source" || viewMode === "full_diff";
  const canEditTextFile =
    isPreviewable?.category === "code" || isPreviewable?.category === "markdown";
  const canEdit =
    Boolean(selectedFile) &&
    !isHistoryMode &&
    canEditMode &&
    hasFullContent &&
    canEditTextFile &&
    Boolean(onSaveFileContent);
  const isDirty = isEditing && draftContent !== fullContent;
  const monacoLanguage = getMonacoLanguage(selectedFile);
  const showMarkdownPreview =
    selectedFile && !isLoading && !error && viewMode === "preview" && hasFullContent && isPreviewable?.category === "markdown";
  const showImagePreview =
    selectedFile && !isLoading && !error && viewMode === "preview" && hasFullContent && isPreviewable?.category === "image" && isPreviewable.mimeType;
  const minimapSourceLines = useMemo(() => {
    if (viewMode === "source") {
      return fullContent.split("\n");
    }
    if (viewMode === "full_diff") {
      return annotatedLines.map((line) => line.content);
    }
    return [] as string[];
  }, [annotatedLines, fullContent, viewMode]);
  const minimapLineCount = viewMode === "source" ? fullContentLineCount : viewMode === "full_diff" ? annotatedLines.length : 0;
  const minimapMarkers = useMemo(
    () => (viewMode === "full_diff" ? buildMinimapMarkers(annotatedLines) : []),
    [annotatedLines, viewMode]
  );
  const minimapTextureRows = useMemo(
    () => buildMinimapTextureRows(minimapSourceLines),
    [minimapSourceLines]
  );
  const showMinimap =
    !isMobile &&
    selectedFile &&
    !isLoading &&
    !isHighlighting &&
    !error &&
    !isEditing &&
    minimapLineCount >= minimapMinLines &&
    ((viewMode === "source" && hasFullContent) || (viewMode === "full_diff" && hasAnnotatedLines));

  useEffect(() => {
    if (!isEditing) {
      setDraftContent(fullContent);
      setExpectedHash(diff?.content_hash);
    }
  }, [diff?.content_hash, fullContent, isEditing]);

  useEffect(() => {
    setIsEditing(false);
    setSaveError(null);
    setConflictHash(null);
  }, [selectedFile, viewMode, isHistoryMode]);

  const handleStartEditing = useCallback(() => {
    if (!canEdit) return;
    setDraftContent(fullContent);
    setExpectedHash(diff?.content_hash);
    setSaveError(null);
    setConflictHash(null);
    setIsEditing(true);
  }, [canEdit, diff?.content_hash, fullContent]);

  const handleCancelEditing = useCallback(() => {
    setDraftContent(fullContent);
    setExpectedHash(diff?.content_hash);
    setSaveError(null);
    setConflictHash(null);
    setIsEditing(false);
  }, [diff?.content_hash, fullContent]);

  const handleSaveContent = useCallback(async () => {
    if (!selectedFile || !onSaveFileContent) return;
    try {
      const result = await onSaveFileContent(selectedFile, draftContent, expectedHash);
      setExpectedHash(result.content_hash);
      setSaveError(null);
      setConflictHash(null);
      setIsEditing(false);
    } catch (err) {
      if (err instanceof FileContentConflictError) {
        setConflictHash(err.currentHash);
        setExpectedHash(err.currentHash);
        setSaveError("File changed on disk. Review latest content and save again.");
        return;
      }
      setSaveError(err instanceof Error ? err.message : "Failed to save file");
    }
  }, [draftContent, expectedHash, onSaveFileContent, selectedFile]);
  const handleMonacoBeforeMount = useCallback((monaco: MonacoInstance) => {
    monacoRef.current = monaco;
    defineMonacoTheme(monaco);
  }, []);
  const handleMonacoMount = useCallback((editor: Monaco.editor.IStandaloneCodeEditor) => {
    monacoEditorRef.current = editor;
  }, []);

  useEffect(() => {
    const editor = monacoEditorRef.current;
    const monaco = monacoRef.current;
    if (!editor || !monaco) return;

    const hasEditableFullDiff = isEditing && viewMode === "full_diff" && hasAnnotatedLines;
    if (!hasEditableFullDiff) {
      if (monacoDecorationIdsRef.current.length > 0) {
        monacoDecorationIdsRef.current = editor.deltaDecorations(monacoDecorationIdsRef.current, []);
      }
      return;
    }

    const lineDecorations = annotatedLines
      .filter(
        (line) =>
          line.number > 0 &&
          (line.change === "added" || line.change === "modified")
      )
      .map((line) => {
        const isAdded = line.change === "added";
        return {
          range: new monaco.Range(line.number, 1, line.number, 1),
          options: {
            isWholeLine: true,
            className: isAdded ? "monaco-diff-line-added" : "monaco-diff-line-modified",
            linesDecorationsClassName: isAdded
              ? "monaco-diff-line-gutter-added"
              : "monaco-diff-line-gutter-modified",
            overviewRuler: {
              color: isAdded ? "#34d399aa" : "#fbbf24aa",
              position: monaco.editor.OverviewRulerLane.Left
            }
          }
        };
      });

    monacoDecorationIdsRef.current = editor.deltaDecorations(
      monacoDecorationIdsRef.current,
      lineDecorations
    );
  }, [annotatedLines, hasAnnotatedLines, isEditing, viewMode]);

  useEffect(() => {
    return () => {
      const editor = monacoEditorRef.current;
      if (!editor || monacoDecorationIdsRef.current.length === 0) return;
      editor.deltaDecorations(monacoDecorationIdsRef.current, []);
      monacoDecorationIdsRef.current = [];
    };
  }, []);
  const maxScrollable = Math.max(scrollMetrics.scrollHeight - scrollMetrics.clientHeight, 0);
  const viewportHeightPercent = clamp(
    (scrollMetrics.clientHeight / Math.max(scrollMetrics.scrollHeight, 1)) * 100,
    8,
    100
  );
  const viewportTopPercent = maxScrollable <= 0
    ? 0
    : (scrollMetrics.scrollTop / maxScrollable) * Math.max(100 - viewportHeightPercent, 0);

  useEffect(() => {
    if (!showMinimap) {
      setScrollMetrics({ scrollTop: 0, scrollHeight: 1, clientHeight: 1 });
      return;
    }

    const scroller = scrollContainerRef.current;
    if (!scroller) return;

    const updateMetrics = () => {
      setScrollMetrics({
        scrollTop: scroller.scrollTop,
        scrollHeight: scroller.scrollHeight,
        clientHeight: scroller.clientHeight
      });
    };

    updateMetrics();
    scroller.addEventListener("scroll", updateMetrics, { passive: true });
    window.addEventListener("resize", updateMetrics);

    return () => {
      scroller.removeEventListener("scroll", updateMetrics);
      window.removeEventListener("resize", updateMetrics);
    };
  }, [showMinimap, selectedFile, viewMode, fullContent, annotatedLines.length]);

  const jumpToMinimapPosition = useCallback((clientY: number) => {
    const rail = minimapRailRef.current;
    const scroller = scrollContainerRef.current;
    if (!rail || !scroller) return;

    const rect = rail.getBoundingClientRect();
    const pointerOffsetY = clamp(clientY - rect.top, 0, rect.height);
    const nextScrollTop = scrollTopFromMinimapPointer(
      pointerOffsetY,
      rect.height,
      scroller.scrollHeight,
      scroller.clientHeight
    );
    scroller.scrollTo({ top: nextScrollTop, behavior: "auto" });
  }, []);

  const handleMinimapPointerDown = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
    event.preventDefault();
    jumpToMinimapPosition(event.clientY);

    const handleMove = (moveEvent: PointerEvent) => {
      jumpToMinimapPosition(moveEvent.clientY);
    };

    const handleUp = () => {
      window.removeEventListener("pointermove", handleMove);
      window.removeEventListener("pointerup", handleUp);
    };

    window.addEventListener("pointermove", handleMove);
    window.addEventListener("pointerup", handleUp);
  }, [jumpToMinimapPosition]);

  const handleMinimapKeyDown = useCallback((event: React.KeyboardEvent<HTMLDivElement>) => {
    const scroller = scrollContainerRef.current;
    if (!scroller) return;

    if (event.key === "ArrowDown" || event.key === "PageDown") {
      event.preventDefault();
      const step = event.key === "PageDown" ? scroller.clientHeight : 80;
      scroller.scrollTo({ top: scroller.scrollTop + step, behavior: "auto" });
    } else if (event.key === "ArrowUp" || event.key === "PageUp") {
      event.preventDefault();
      const step = event.key === "PageUp" ? scroller.clientHeight : 80;
      scroller.scrollTo({ top: scroller.scrollTop - step, behavior: "auto" });
    } else if (event.key === "Home") {
      event.preventDefault();
      scroller.scrollTo({ top: 0, behavior: "auto" });
    } else if (event.key === "End") {
      event.preventDefault();
      scroller.scrollTo({ top: scroller.scrollHeight, behavior: "auto" });
    }
  }, []);

  // Dynamically compute max path chars based on available header width
  useEffect(() => {
    if (!titleRowRef.current || typeof ResizeObserver === "undefined") return;
    const update = () => {
      const width = titleRowRef.current?.clientWidth ?? 0;
      // Account for: dot/badge (~30px), stats (~80px), overflow menu (~44px), gaps (~24px)
      const usable = Math.max(0, width - 180);
      const nextMax = Math.max(12, Math.min(100, Math.floor(usable / 7)));
      setMaxPathChars(nextMax);
    };
    const rafId = requestAnimationFrame(update);
    const observer = new ResizeObserver(update);
    observer.observe(titleRowRef.current);
    return () => {
      cancelAnimationFrame(rafId);
      observer.disconnect();
    };
  }, []);

  const displayPath = selectedFile ? formatPath(selectedFile, maxPathChars) : null;

  return (
    <Card className="h-full flex flex-col" data-testid="diff-viewer-panel">
      <CardHeader className={`space-y-0 ${isMobile ? "py-3 px-4" : "py-3 flex-row items-center justify-between"}`}>
        {/* Row 1: Title + primary indicators */}
        <div ref={titleRowRef} className={`flex items-center min-w-0 ${isMobile ? "gap-2" : "gap-3"}`}>
          <div className={`flex items-center min-w-0 flex-1 ${isMobile ? "gap-2" : "gap-3"}`}>
            <CardTitle className={`flex items-center gap-2 min-w-0 ${isMobile ? "flex-1" : ""}`}>
              {!isMobile && (
                <FileDiff className="flex-shrink-0 text-slate-500 h-4 w-4" />
              )}
              {selectedFile ? (
                <span className="font-mono text-xs truncate" title={selectedFile}>{displayPath}</span>
              ) : (
                <span className="text-xs">Diff Viewer</span>
              )}
            </CardTitle>
            {/* Desktop-only: inline copy/related buttons */}
            {!isMobile && selectedFile && (
              <button
                type="button"
                className="inline-flex items-center justify-center rounded-full border border-white/20 text-slate-300 transition-colors hover:bg-white/10 active:bg-white/20 flex-shrink-0 h-7 w-7"
                onClick={handleCopyPath}
                title={copied ? "Copied" : "Copy absolute path"}
                aria-label="Copy absolute path"
                data-testid="copy-absolute-path"
              >
                {copied ? (
                  <Check className="text-emerald-300 h-3.5 w-3.5" />
                ) : (
                  <Copy className="h-3.5 w-3.5" />
                )}
              </button>
            )}
            {!isMobile && selectedFile && onShowRelatedFiles && (
              <button
                type="button"
                className="inline-flex items-center justify-center rounded-full border border-white/20 text-slate-300 transition-colors hover:bg-white/10 active:bg-white/20 flex-shrink-0 h-7 w-7"
                onClick={() => onShowRelatedFiles(selectedFile)}
                title="Related files"
                aria-label="Show related files"
                data-testid="related-files-button"
              >
                <Link2 className="h-3.5 w-3.5" />
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

          {/* Right side of row 1 */}
          <div className={`flex items-center flex-shrink-0 ${isMobile ? "gap-2" : "gap-3"}`}>
            {/* Mobile: colored dot status indicator */}
            {selectedFile && isMobile && !isHistoryMode && (
              <span
                className={`flex-shrink-0 rounded-full h-2.5 w-2.5 ${
                  isUntracked ? "bg-slate-400" :
                  isStaged ? "bg-emerald-400" :
                  "bg-amber-300"
                }`}
                title={isUntracked ? "Untracked" : isStaged ? "Staged" : "Modified"}
              />
            )}
            {selectedFile && isMobile && isHistoryMode && (
              <Badge variant="warning">
                {commitHash ? commitHash.substring(0, 7) : "hist"}
              </Badge>
            )}
            {diff?.stats && diff.has_diff && viewMode !== "source" && (
              <button
                type="button"
                className="flex items-center gap-2 hover:underline decoration-slate-600 cursor-pointer"
                data-testid="diff-stats"
                onClick={() => setMetricsOpen(true)}
                aria-label="View change metrics"
              >
                <span className="flex items-center gap-1 text-emerald-500 text-xs">
                  <Plus className="h-3 w-3" />
                  {diff.stats.additions}
                </span>
                <span className="flex items-center gap-1 text-red-500 text-xs">
                  <Minus className="h-3 w-3" />
                  {diff.stats.deletions}
                </span>
              </button>
            )}
            {/* Mobile overflow menu trigger */}
            {isMobile && selectedFile && (
              <button
                type="button"
                className="h-10 w-10 inline-flex items-center justify-center rounded-full text-slate-400 hover:bg-slate-800/70 active:bg-slate-700 touch-target"
                onClick={() => setMobileActionsOpen(true)}
                title="More actions"
                aria-label="More actions"
              >
                <MoreVertical className="h-5 w-5" />
              </button>
            )}
            {/* Desktop: view mode + edit/save buttons */}
            {!isMobile && (
              <>
                {selectedFile && !isLoading && !error && (
                  <ViewModeSelector
                    mode={viewMode}
                    onChange={onViewModeChange}
                    compact={false}
                    disabled={isLoading}
                    filePath={selectedFile}
                    hasDiff={!isReadOnly && diff?.has_diff}
                  />
                )}
                {selectedFile && !isLoading && !error && canEdit && !isEditing && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleStartEditing}
                    className="h-7 px-2"
                    data-testid="start-editing-button"
                  >
                    <Pencil className="h-3.5 w-3.5 mr-1" />
                    Edit
                  </Button>
                )}
                {selectedFile && !isLoading && !error && isEditing && (
                  <>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleCancelEditing}
                      className="h-7 px-2"
                      disabled={isSavingFile}
                      data-testid="cancel-editing-button"
                    >
                      <RotateCcw className="h-3.5 w-3.5 mr-1" />
                      Cancel
                    </Button>
                    <Button
                      variant="default"
                      size="sm"
                      onClick={handleSaveContent}
                      className="h-7 px-2 bg-emerald-600 hover:bg-emerald-700"
                      disabled={isSavingFile || !isDirty}
                      data-testid="save-file-button"
                    >
                      {isSavingFile ? <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" /> : <Save className="h-3.5 w-3.5 mr-1" />}
                      Save
                    </Button>
                  </>
                )}
                {selectedFile && !isLoading && !error && !isEditing && onDeletePath && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onDeletePath(selectedFile, false)}
                    className="h-7 px-2 text-red-400 border-red-400/50 hover:bg-red-950/50 hover:text-red-300"
                    disabled={isDeleting}
                    data-testid="delete-file-button"
                  >
                    <Trash2 className="h-3.5 w-3.5 mr-1" />
                    Delete
                  </Button>
                )}
              </>
            )}
          </div>
        </div>

        {/* Row 2 (mobile only): View mode + edit actions */}
        {isMobile && selectedFile && !isLoading && !error && (
          <div className="flex items-center gap-2 mt-2 pt-2 border-t border-slate-800/50">
            <ViewModeSelector
              mode={viewMode}
              onChange={onViewModeChange}
              compact={true}
              disabled={isLoading}
              filePath={selectedFile}
              hasDiff={!isReadOnly && diff?.has_diff}
            />
            <div className="flex-1" />
            {canEdit && !isEditing && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleStartEditing}
                className="h-9 px-3"
                data-testid="start-editing-button"
              >
                <Pencil className="h-3.5 w-3.5 mr-1" />
                Edit
              </Button>
            )}
            {isEditing && (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCancelEditing}
                  className="h-9 px-3"
                  disabled={isSavingFile}
                  data-testid="cancel-editing-button"
                >
                  <RotateCcw className="h-3.5 w-3.5 mr-1" />
                  Cancel
                </Button>
                <Button
                  variant="default"
                  size="sm"
                  onClick={handleSaveContent}
                  className="h-9 px-3 bg-emerald-600 hover:bg-emerald-700"
                  disabled={isSavingFile || !isDirty}
                  data-testid="save-file-button"
                >
                  {isSavingFile ? <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" /> : <Save className="h-3.5 w-3.5 mr-1" />}
                  Save
                </Button>
              </>
            )}
          </div>
        )}
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

          {saveError && !isLoading && (
            <div className="mx-3 mt-3 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-200" data-testid="save-error">
              <p>{saveError}</p>
              {conflictHash && (
                <p className="mt-1 font-mono text-[11px] text-amber-300">Current hash: {conflictHash}</p>
              )}
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

          {/* Monaco edit mode */}
          {selectedFile && !isLoading && !error && isEditing && canEditMode && hasFullContent && (
            <div className="monaco-diff-editor h-full min-h-[360px] border-y border-slate-800 bg-slate-950" data-testid="monaco-editor-container">
              <Editor
                height="100%"
                defaultLanguage={monacoLanguage}
                language={monacoLanguage}
                value={draftContent}
                onChange={(value) => setDraftContent(value ?? "")}
                beforeMount={handleMonacoBeforeMount}
                onMount={handleMonacoMount}
                theme={monacoThemeName}
                options={{
                  automaticLayout: true,
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  wordWrap: "off",
                  fontSize: 12,
                  lineHeight: 20,
                  lineNumbersMinChars: 3,
                  fontFamily: "JetBrains Mono, Fira Code, SF Mono, Consolas, Liberation Mono, Menlo, monospace",
                  padding: { top: 2, bottom: 2 },
                  renderLineHighlight: "line"
                }}
              />
            </div>
          )}

          {/* Source mode - just the file content */}
          {selectedFile && !isLoading && !isHighlighting && !error && !isEditing && viewMode === "source" && hasFullContent && (
            <SourceView
              content={fullContent}
              highlightedLines={highlightedLines}
            />
          )}

          {/* Full + Diff mode - full file with change annotations */}
          {selectedFile && !isLoading && !isHighlighting && !error && !isEditing && viewMode === "full_diff" && hasAnnotatedLines && (
            <FullFileView
              annotatedLines={annotatedLines}
              highlightedLines={highlightedLines}
              showChangeMarkers={true}
            />
          )}

          {/* Diff mode - traditional hunk view */}
          {selectedFile && !isLoading && !isHighlighting && !error && viewMode === "diff" && hasHunks && (
            <div data-testid="diff-content">
              {hunks.map((hunk, index) => (
                <HunkDisplay key={index} hunk={hunk} index={index} />
              ))}
            </div>
          )}

          {/* Fallback for untracked files in diff mode (show full content) */}
          {selectedFile && !isLoading && !isHighlighting && !error && viewMode === "diff" && !hasHunks && hasFullContent && isUntracked && (
            <SourceView
              content={fullContent}
              highlightedLines={highlightedLines}
            />
          )}

          {/* Preview mode - render markdown */}
          {showMarkdownPreview && (
            <MarkdownPreview content={fullContent} />
          )}

          {/* Preview mode - render images */}
          {showImagePreview && isPreviewable?.mimeType && (
            <ImagePreview
              src={`data:${isPreviewable.mimeType};base64,${fullContent}`}
              alt={selectedFile}
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
          {isMobile && selectedFile && !isLoading && !isEditing && (!isHistoryMode || onDeletePath) && <div className="h-16" aria-hidden="true" />}
        </ScrollArea>

        {showMinimap && (
          <aside
            className="absolute right-2 top-2 bottom-2 w-10 rounded-md border border-slate-700/70 bg-slate-900/90 shadow-lg"
            data-testid="diff-minimap"
          >
            <div
              ref={minimapRailRef}
              className="relative h-full w-full cursor-pointer rounded-md"
              role="slider"
              tabIndex={0}
              aria-label="Diff minimap"
              aria-valuemin={0}
              aria-valuemax={Math.round(maxScrollable)}
              aria-valuenow={Math.round(scrollMetrics.scrollTop)}
              data-testid="diff-minimap-rail"
              onPointerDown={handleMinimapPointerDown}
              onKeyDown={handleMinimapKeyDown}
            >
              <div
                className="absolute inset-0"
                data-testid="diff-minimap-texture"
                aria-hidden="true"
              >
                {minimapTextureRows.map((row, index) => (
                  <div
                    key={`texture-${index}`}
                    className="absolute right-0 h-[1px] bg-slate-300/90"
                    style={{
                      top: `${row.topPercent}%`,
                      width: `${row.widthPercent}%`,
                      opacity: row.opacity
                    }}
                    data-testid="diff-minimap-texture-line"
                  />
                ))}
              </div>
              {minimapMarkers.map((marker, index) => (
                <div
                  key={`${marker.change}-${index}`}
                  className={`absolute left-0 right-0 h-[2px] ${getMinimapMarkerClass(marker.change)}`}
                  style={{ top: `${marker.topPercent}%` }}
                  data-testid="diff-minimap-marker"
                />
              ))}
              <div
                className="pointer-events-none absolute left-0 right-0 rounded-sm border border-sky-300/40 bg-sky-400/20"
                style={{
                  top: `${viewportTopPercent}%`,
                  height: `${viewportHeightPercent}%`
                }}
                data-testid="diff-minimap-viewport"
              />
            </div>
          </aside>
        )}

        {/* Mobile Action Bar - history mode: delete only */}
        {isMobile && selectedFile && !isLoading && isHistoryMode && !isEditing && onDeletePath && (
          <div className="absolute bottom-0 left-0 right-0 p-3 bg-slate-900/95 backdrop-blur-sm border-t border-slate-800" data-testid="diff-mobile-actions-history">
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                className="flex-1 h-10 touch-target text-red-400 border-red-400/50 hover:bg-red-950/50"
                onClick={() => onDeletePath(selectedFile, false)}
                disabled={isDeleting}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                {isDeleting ? "Deleting..." : "Delete"}
              </Button>
            </div>
          </div>
        )}

        {/* Mobile Action Bar - normal mode */}
        {isMobile && selectedFile && !isLoading && !isHistoryMode && !isEditing && (
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

      <ChangeMetricsModal
        isOpen={metricsOpen}
        onClose={() => setMetricsOpen(false)}
        mode="file"
        stats={diff?.stats}
        filePath={selectedFile}
      />
      {/* Mobile overflow actions */}
      {isMobile && (
        <BottomSheet
          isOpen={mobileActionsOpen}
          onClose={() => setMobileActionsOpen(false)}
          title="File Actions"
        >
          <div className="flex flex-col gap-1">
            <BottomSheetAction
              icon={copied ? <Check className="h-5 w-5 text-emerald-300" /> : <Copy className="h-5 w-5" />}
              label={copied ? "Copied!" : "Copy absolute path"}
              onClick={() => {
                handleCopyPath();
                setMobileActionsOpen(false);
              }}
            />
            {onShowRelatedFiles && selectedFile && (
              <BottomSheetAction
                icon={<Link2 className="h-5 w-5" />}
                label="Related files"
                description="Find files related to this one"
                onClick={() => {
                  onShowRelatedFiles(selectedFile);
                  setMobileActionsOpen(false);
                }}
              />
            )}
            {onDeletePath && selectedFile && (
              <BottomSheetAction
                icon={<Trash2 className="h-5 w-5 text-red-400" />}
                label="Delete file"
                description="Delete this file from the working tree"
                onClick={() => {
                  onDeletePath(selectedFile, false);
                  setMobileActionsOpen(false);
                }}
              />
            )}
          </div>
        </BottomSheet>
      )}
    </Card>
  );
}
