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

import { useEffect, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, Code, Copy, Eye, FileCode, FileImage, FileText, Info, Loader2 } from "lucide-react";
import { cn } from "../../lib/utils";
import { defaultQueryOptions, getFileExtension } from "../../lib";
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
 * Renders code with line numbers.
 */
function CodeView({ content, fileName }: { content: string; fileName: string }) {
  const lines = content.split("\n");
  const ext = getFileExtension(fileName);

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <tbody>
          {lines.map((line, i) => (
            <tr key={i} className="hover:bg-slate-700/30">
              <td className="select-none px-3 py-0.5 text-right text-slate-600 border-r border-slate-700 w-10">
                {i + 1}
              </td>
              <td className="px-3 py-0.5 whitespace-pre font-mono text-slate-300">
                {line || " "}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="absolute top-2 right-2 text-xs text-slate-600 uppercase">
        {ext}
      </div>
    </div>
  );
}

/**
 * FilePreview component that fetches and renders file content.
 */
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
  const fileType = getFileType(fileName);
  const [showMobilePath, setShowMobilePath] = useState(false);
  const [copied, setCopied] = useState(false);
  const [markdownView, setMarkdownView] = useState<"rendered" | "raw">("rendered");

  useEffect(() => {
    setShowMobilePath(false);
  }, [filePath]);

  useEffect(() => {
    setMarkdownView("rendered");
  }, [filePath]);

  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 1600);
    return () => clearTimeout(timeout);
  }, [copied]);

  // For images, we build the URL directly instead of fetching content
  const isImage = fileType === "image";

  // Fetch file content for non-image files
  const {
    data: content,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["backlog", backlogKind, backlogName, "files", filePath, "content"],
    queryFn: () => backlogService.getFileContent(backlogKind, backlogName, filePath),
    enabled: !isImage,
    ...defaultQueryOptions,
  });

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

  const showMarkdownToggle = fileType === "markdown";
  const markdownToggleLabel =
    markdownView === "rendered" ? "Show raw markdown" : "Show rendered markdown";

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
          {showMarkdownToggle && (
            <button
              type="button"
              onClick={() =>
                setMarkdownView((prev) => (prev === "rendered" ? "raw" : "rendered"))
              }
              className="rounded-full p-1 text-slate-400 hover:bg-slate-700/60 hover:text-slate-200"
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
          "relative min-h-[200px] max-h-[600px] overflow-auto",
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

        {/* Markdown preview */}
        {!isImage && !isLoading && !error && content && fileType === "markdown" && markdownView === "rendered" && (
          <div
            className="p-4 prose prose-invert max-w-none"
            data-testid="file-preview-markdown"
            dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
          />
        )}

        {!isImage && !isLoading && !error && content && fileType === "markdown" && markdownView === "raw" && (
          <pre
            className="p-4 font-mono text-sm text-slate-300 whitespace-pre-wrap"
            data-testid="file-preview-markdown-raw"
          >
            {content}
          </pre>
        )}

        {/* Code preview */}
        {!isImage && !isLoading && !error && content && fileType === "code" && (
          <div className="relative bg-slate-900" data-testid="file-preview-code">
            <CodeView content={content} fileName={fileName} />
          </div>
        )}

        {/* Text preview */}
        {!isImage && !isLoading && !error && content && fileType === "text" && (
          <pre
            className="p-4 font-mono text-sm text-slate-300 whitespace-pre-wrap"
            data-testid="file-preview-text"
          >
            {content}
          </pre>
        )}

        {/* Empty content */}
        {!isImage && !isLoading && !error && content === "" && (
          <div className="flex items-center justify-center p-8 text-slate-500">
            This file is empty
          </div>
        )}
      </div>
    </div>
  );
}
