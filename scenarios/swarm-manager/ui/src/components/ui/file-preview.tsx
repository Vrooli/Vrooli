/**
 * FilePreview Component
 *
 * Renders file content with appropriate formatting based on file type.
 * Supports:
 * - Markdown: Rendered as HTML (basic styling)
 * - Code files: Syntax highlighted with line numbers
 * - Images: Displayed inline
 * - Text: Plain text display
 *
 * [REQ:REQ-P0-004] File preview for backlog details page
 */

import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Editor, { DiffEditor } from "@monaco-editor/react";
import {
  Check,
  Code,
  Copy,
  Diff,
  Eye,
  FileCode,
  FileImage,
  FileText,
  Info,
  Loader2,
  RotateCcw,
  Save,
} from "lucide-react";
import { cn } from "../../lib/utils";
import { defaultQueryOptions, getFileExtension, useResolvedTheme } from "../../lib";
import { backlogService } from "../../services";
import type { BacklogKind } from "../../types";
import { ErrorState } from "./error-state";

export interface FilePreviewProps {
  /** Backlog kind containing the file */
  backlogKind: BacklogKind;
  /** Backlog item name containing the file */
  backlogName: string;
  /** Path to the file within the backlog folder */
  filePath: string;
  /** File name for display */
  fileName: string;
  /** Optional className for styling */
  className?: string;
  /** Optional className for the content area */
  contentClassName?: string;
  /** Optional header actions aligned to the right */
  headerActions?: ReactNode;
  /** Compact header layout (optimized for mobile) */
  compactHeader?: boolean;
  /** Make header sticky within the preview */
  stickyHeader?: boolean;
  /** data-testid attribute */
  "data-testid"?: string;
}

/**
 * Determines the file type category from the file extension.
 */
function getFileType(fileName: string): "markdown" | "code" | "image" | "text" {
  const ext = getFileExtension(fileName);

  if (["md", "markdown"].includes(ext)) {
    return "markdown";
  }

  if (["png", "jpg", "jpeg", "gif", "svg", "webp", "bmp", "ico"].includes(ext)) {
    return "image";
  }

  if ([
    "js", "jsx", "ts", "tsx", "json", "go", "py", "rs", "java", "c", "cpp", "h",
    "html", "css", "scss", "yaml", "yml", "toml", "xml", "sh", "bash", "zsh",
    "sql", "graphql", "proto", "dockerfile",
  ].includes(ext)) {
    return "code";
  }

  return "text";
}

/**
 * Returns the appropriate icon for a file type.
 */
function FileIcon({ type }: { type: ReturnType<typeof getFileType> }) {
  switch (type) {
    case "markdown":
    case "text":
      return <FileText className="h-5 w-5 text-slate-400" />;
    case "code":
      return <FileCode className="h-5 w-5 text-cyan-400" />;
    case "image":
      return <FileImage className="h-5 w-5 text-purple-400" />;
  }
}

/**
 * Simple markdown to HTML converter for basic rendering.
 * Handles headers, bold, italic, code blocks, and links.
 */
function renderMarkdown(content: string): string {
  return content
    // Headers
    .replace(/^### (.+)$/gm, '<h3 class="text-lg font-semibold text-slate-200 mt-4 mb-2">$1</h3>')
    .replace(/^## (.+)$/gm, '<h2 class="text-xl font-semibold text-slate-200 mt-6 mb-3">$1</h2>')
    .replace(/^# (.+)$/gm, '<h1 class="text-2xl font-bold text-slate-100 mt-6 mb-4">$1</h1>')
    // Code blocks (multiline)
    .replace(/```(\w*)\n([\s\S]*?)```/g, '<pre class="bg-slate-900 rounded p-3 overflow-x-auto my-3"><code class="text-sm text-slate-300">$2</code></pre>')
    // Inline code
    .replace(/`([^`]+)`/g, '<code class="bg-slate-900 px-1.5 py-0.5 rounded text-cyan-300 text-sm">$1</code>')
    // Bold
    .replace(/\*\*([^*]+)\*\*/g, '<strong class="font-semibold text-slate-200">$1</strong>')
    // Italic
    .replace(/\*([^*]+)\*/g, '<em class="italic">$1</em>')
    // Links
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" class="text-cyan-400 hover:underline" target="_blank" rel="noopener noreferrer">$1</a>')
    // Line breaks (paragraphs)
    .replace(/\n\n/g, '</p><p class="text-slate-300 mb-3">')
    // Wrap in paragraph
    .replace(/^(.+)$/gm, (match) => {
      if (match.startsWith('<')) return match;
      return `<p class="text-slate-300 mb-3">${match}</p>`;
    });
}

/**
 * Returns the Monaco language identifier for a file.
 */
function getMonacoLanguage(fileName: string): string {
  const ext = getFileExtension(fileName);
  const languageMap: Record<string, string> = {
    js: "javascript",
    jsx: "javascript",
    ts: "typescript",
    tsx: "typescript",
    json: "json",
    md: "markdown",
    markdown: "markdown",
    yaml: "yaml",
    yml: "yaml",
    toml: "toml",
    html: "html",
    css: "css",
    scss: "scss",
    go: "go",
    py: "python",
    rs: "rust",
    java: "java",
    c: "c",
    cpp: "cpp",
    h: "cpp",
    sh: "shell",
    bash: "shell",
    zsh: "shell",
    sql: "sql",
    graphql: "graphql",
    proto: "protobuf",
    xml: "xml",
  };
  return languageMap[ext] ?? "plaintext";
}

/**
 * Maps file extensions to reasonable content types for saving.
 */
function getContentTypeForFile(fileName: string): string {
  const ext = getFileExtension(fileName);
  switch (ext) {
    case "json":
      return "application/json";
    case "md":
    case "markdown":
      return "text/markdown";
    case "yaml":
    case "yml":
      return "text/yaml";
    case "toml":
      return "text/plain";
    case "html":
      return "text/html";
    case "css":
    case "scss":
      return "text/css";
    case "xml":
      return "text/xml";
    case "sql":
      return "text/plain";
    default:
      return "text/plain";
  }
}

const DEFAULT_EDITOR_OPTIONS = {
  minimap: { enabled: false },
  wordWrap: "on",
  lineNumbers: "on",
  fontSize: 13,
  fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
  tabSize: 2,
  scrollBeyondLastLine: false,
  padding: { top: 12, bottom: 12 },
  renderLineHighlight: "line",
  cursorBlinking: "smooth",
  smoothScrolling: true,
  scrollbar: {
    vertical: "auto",
    horizontal: "auto",
    verticalScrollbarSize: 8,
    horizontalScrollbarSize: 8,
  },
  overviewRulerBorder: false,
  hideCursorInOverviewRuler: true,
  folding: true,
  foldingStrategy: "indentation",
  automaticLayout: true,
} as const;

type FileDraftState = {
  original: string;
  draft: string;
};

/**
 * FilePreview component that fetches and renders file content.
 */
// DOC: docs/internal/INTENT.md#git-tracked-backlog
export function FilePreview({
  backlogKind,
  backlogName,
  filePath,
  fileName,
  className,
  contentClassName,
  headerActions,
  compactHeader = false,
  stickyHeader = false,
  "data-testid": testId,
}: FilePreviewProps) {
  const queryClient = useQueryClient();
  const resolvedTheme = useResolvedTheme();
  const isLight = resolvedTheme === "light";
  const editorTheme = isLight ? "vs" : "vs-dark";
  const fileType = getFileType(fileName);
  const isImage = fileType === "image";
  const isEditable = fileType !== "image";
  const editorLanguage = useMemo(() => getMonacoLanguage(fileName), [fileName]);
  const contentQueryKey = useMemo(
    () => ["backlog", backlogKind, backlogName, "files", filePath, "content"],
    [backlogKind, backlogName, filePath]
  );
  const [fileStateByPath, setFileStateByPath] = useState<Record<string, FileDraftState>>({});
  const [showMobilePath, setShowMobilePath] = useState(false);
  const [copied, setCopied] = useState(false);
  const [markdownView, setMarkdownView] = useState<"rendered" | "raw">("raw");
  const [isDiffMode, setIsDiffMode] = useState(false);
  const fileState = fileStateByPath[filePath];
  const isDirty = Boolean(fileState && fileState.draft !== fileState.original);

  useEffect(() => {
    setShowMobilePath(false);
    setMarkdownView("raw");
    setIsDiffMode(false);
  }, [filePath]);

  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 1600);
    return () => clearTimeout(timeout);
  }, [copied]);

  const {
    data: content,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: contentQueryKey,
    queryFn: () => backlogService.getFileContent(backlogKind, backlogName, filePath),
    enabled: !isImage,
    ...defaultQueryOptions,
    refetchOnWindowFocus: defaultQueryOptions.refetchOnWindowFocus && !isDirty,
  });

  useEffect(() => {
    if (typeof content !== "string") return;
    setFileStateByPath((prev) => {
      const existing = prev[filePath];
      if (!existing) {
        return { ...prev, [filePath]: { original: content, draft: content } };
      }
      if (existing.original === content) return prev;
      if (existing.draft !== existing.original) return prev;
      return { ...prev, [filePath]: { original: content, draft: content } };
    });
  }, [content, filePath]);

  const draftContent = fileState?.draft ?? content ?? "";
  const originalContent = fileState?.original ?? content ?? "";

  useEffect(() => {
    if (!isDirty) {
      setIsDiffMode(false);
    }
  }, [isDirty]);

  const saveMutation = useMutation({
    mutationFn: async (nextContent: string) =>
      backlogService.saveFileContent(
        backlogKind,
        backlogName,
        filePath,
        nextContent,
        getContentTypeForFile(fileName)
      ),
    onSuccess: (_result, nextContent) => {
      setFileStateByPath((prev) => {
        return {
          ...prev,
          [filePath]: {
            original: nextContent,
            draft: nextContent,
          },
        };
      });
      queryClient.setQueryData(contentQueryKey, nextContent);
      queryClient.invalidateQueries({
        queryKey: ["backlog", backlogKind, backlogName, "files"],
      });
      if (filePath.endsWith("spec.json")) {
        queryClient.invalidateQueries({
          queryKey: ["backlog", backlogKind, backlogName],
        });
      }
    },
  });

  useEffect(() => {
    saveMutation.reset();
  }, [filePath, saveMutation]);

  const isSaving = saveMutation.isPending;
  const saveErrorMessage =
    saveMutation.error instanceof Error
      ? saveMutation.error.message
      : saveMutation.error
        ? "Unable to save file."
        : "";

  const handleDraftChange = useCallback(
    (nextValue?: string) => {
      const normalized = nextValue ?? "";
      setFileStateByPath((prev) => {
        const existing =
          prev[filePath] ??
          (typeof content === "string"
            ? { original: content, draft: content }
            : { original: "", draft: "" });
        if (existing.draft === normalized) {
          return prev;
        }
        return {
          ...prev,
          [filePath]: {
            original: existing.original,
            draft: normalized,
          },
        };
      });
    },
    [content, filePath]
  );

  const handleDiscard = useCallback(() => {
    setFileStateByPath((prev) => {
      const existing = prev[filePath];
      if (existing) {
        return {
          ...prev,
          [filePath]: {
            original: existing.original,
            draft: existing.original,
          },
        };
      }
      if (typeof content === "string") {
        return {
          ...prev,
          [filePath]: {
            original: content,
            draft: content,
          },
        };
      }
      return prev;
    });
    setIsDiffMode(false);
  }, [content, filePath]);

  const handleSave = useCallback(() => {
    if (!isEditable || !isDirty || isSaving) return;
    saveMutation.mutate(draftContent);
  }, [draftContent, isDirty, isEditable, isSaving, saveMutation]);

  const handleCopyPath = async () => {
    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(filePath);
        setCopied(true);
        return;
      } catch {
        // Fall through to showing the path if copy fails.
      }
    }
    setShowMobilePath(true);
  };

  const showMarkdownToggle = fileType === "markdown" && !isDiffMode;
  const markdownToggleLabel =
    markdownView === "rendered" ? "Show raw markdown" : "Show rendered markdown";
  const showEditor = isEditable && !isDiffMode && (fileType !== "markdown" || markdownView === "raw");
  const showRenderedMarkdown =
    fileType === "markdown" && markdownView === "rendered" && !isDiffMode;
  const showDiff = isEditable && isDiffMode && !isLoading && !error;
  const canSave = isEditable && isDirty && !isSaving && !isLoading && !error;
  const canDiscard = isEditable && isDirty && !isSaving;
  const canToggleDiff = isEditable && isDirty && !isSaving;

  const actionButtonClass = (enabled: boolean, active = false) =>
    cn(
      "rounded-full p-1 transition-colors",
      enabled
        ? "text-slate-300 hover:bg-slate-700/60 hover:text-white"
        : "text-slate-600 cursor-not-allowed",
      active && enabled && "bg-slate-700/60 text-white"
    );

  return (
    <div
      className={cn(
        "rounded-lg border border-white/10 bg-slate-800/30 overflow-hidden flex flex-col",
        className
      )}
      data-testid={testId ?? "file-preview"}
    >
      {/* Header */}
      <div
        className={cn(
          "flex items-center gap-2 border-b border-white/10 bg-slate-800/50",
          compactHeader ? "px-3 py-2 sm:px-4 sm:py-3" : "px-4 py-3",
          stickyHeader && "sticky top-0 z-20 backdrop-blur"
        )}
      >
        <FileIcon type={fileType} />
        <span className="font-medium text-slate-200 truncate" data-testid="file-preview-name">
          {fileName}
        </span>
        <div className="ml-auto flex items-center gap-2">
          <span
            className={cn(
              "text-xs text-slate-500 max-w-[220px] truncate",
              compactHeader && "hidden sm:inline"
            )}
          >
            {filePath}
          </span>
          {isEditable && (
            <>
              <button
                type="button"
                onClick={handleSave}
                disabled={!canSave}
                className={cn(
                  "rounded-full p-1 transition-colors",
                  canSave
                    ? "bg-cyan-500/20 text-cyan-200 hover:bg-cyan-500/30"
                    : "text-slate-600 cursor-not-allowed"
                )}
                aria-label="Save changes"
                title={canSave ? "Save changes" : "No changes to save"}
                data-testid="file-preview-save"
              >
                {isSaving ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Save className="h-4 w-4" />
                )}
              </button>
              <button
                type="button"
                onClick={handleDiscard}
                disabled={!canDiscard}
                className={actionButtonClass(canDiscard)}
                aria-label="Discard changes"
                title={canDiscard ? "Discard changes" : "No changes to discard"}
                data-testid="file-preview-discard"
              >
                <RotateCcw className="h-4 w-4" />
              </button>
              {isDirty && (
                <button
                  type="button"
                  onClick={() => setIsDiffMode((prev) => !prev)}
                  disabled={!canToggleDiff}
                  className={actionButtonClass(canToggleDiff, isDiffMode)}
                  aria-label="Toggle diff"
                  aria-pressed={isDiffMode}
                  title={isDiffMode ? "Hide diff" : "Show diff"}
                  data-testid="file-preview-diff-toggle"
                >
                  <Diff className="h-4 w-4" />
                </button>
              )}
            </>
          )}
          {showMarkdownToggle && (
            <button
              type="button"
              onClick={() =>
                setMarkdownView((prev) => (prev === "rendered" ? "raw" : "rendered"))
              }
              className={actionButtonClass(true)}
              aria-label={markdownToggleLabel}
              title={markdownToggleLabel}
            >
              {markdownView === "rendered" ? (
                <Code className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          )}
          {compactHeader && (
            <button
              type="button"
              onClick={() => setShowMobilePath((prev) => !prev)}
              className="sm:hidden rounded-full p-1 text-slate-400 hover:bg-slate-700/60 hover:text-slate-200"
              aria-label="Toggle file path"
              title="Show file path"
            >
              <Info className="h-4 w-4" />
            </button>
          )}
          {headerActions}
        </div>
      </div>
      {saveErrorMessage && (
        <div className="border-b border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-200">
          {saveErrorMessage}
        </div>
      )}
      {compactHeader && showMobilePath && (
        <div className="sm:hidden flex items-center justify-between gap-2 border-b border-white/10 bg-slate-900/60 px-3 py-2 text-xs text-slate-400">
          <span className="truncate">{filePath}</span>
          <button
            type="button"
            onClick={handleCopyPath}
            className="rounded-full p-1 text-slate-300 hover:bg-slate-800/70 hover:text-white"
            aria-label="Copy file path"
            title={copied ? "Copied" : "Copy file path"}
          >
            {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
          </button>
        </div>
      )}

      {/* Content */}
      <div
        className={cn(
          "relative flex-1 min-h-0 overflow-hidden",
          contentClassName
        )}
      >
        {isLoading && (
          <div className="absolute inset-0 flex items-center justify-center bg-slate-800/50">
            <Loader2 className="h-6 w-6 animate-spin text-cyan-400" />
          </div>
        )}

        {error && (
          <div className="p-4">
            <ErrorState
              error={error}
              title="Unable to load file"
              onRetry={() => refetch()}
            />
          </div>
        )}

        {/* Image preview */}
        {isImage && (
          <div className="flex items-center justify-center p-4 bg-slate-900/50">
            <img
              src={`/api/v1/backlog/${backlogKind}/${backlogName}/files/${filePath}`}
              alt={fileName}
              className="max-w-full max-h-full object-contain"
              data-testid="file-preview-image"
            />
          </div>
        )}

        {!isImage && !isLoading && !error && showDiff && (
          <DiffEditor
            height="100%"
            language={editorLanguage}
            original={originalContent}
            modified={draftContent}
            theme={editorTheme}
            data-testid="file-preview-diff"
            options={{
              ...DEFAULT_EDITOR_OPTIONS,
              readOnly: true,
              renderSideBySide: !compactHeader,
            }}
          />
        )}

        {!isImage && !isLoading && !error && showRenderedMarkdown && (
          <div
            className={cn(
              "h-full overflow-auto p-4 prose max-w-none",
              isLight ? "prose-slate" : "prose-invert"
            )}
            data-testid="file-preview-markdown"
            dangerouslySetInnerHTML={{ __html: renderMarkdown(draftContent) }}
          />
        )}

        {!isImage && !isLoading && !error && showEditor && (
          <Editor
            height="100%"
            language={editorLanguage}
            value={draftContent}
            onChange={handleDraftChange}
            path={filePath}
            theme={editorTheme}
            data-testid="file-preview-editor"
            options={{
              ...DEFAULT_EDITOR_OPTIONS,
              readOnly: !isEditable,
            }}
          />
        )}
      </div>
    </div>
  );
}
