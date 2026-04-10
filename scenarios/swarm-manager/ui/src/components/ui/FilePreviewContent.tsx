/**
 * FilePreviewContent Component
 *
 * Content area that dispatches to the appropriate renderer:
 * Editor, DiffEditor, rendered Markdown, or Image preview.
 *
 * [REQ:REQ-P0-004] File preview content rendering
 */

import { useMemo } from "react";
import Editor, { DiffEditor } from "@monaco-editor/react";
import { Loader2 } from "lucide-react";
import { cn } from "../../lib/utils";
import { useResolvedTheme } from "../../lib";
import { getMonacoLanguage } from "../../lib/file-type-utils";
import { renderMarkdown } from "../../lib/render-markdown";
import { ErrorState } from "./error-state";

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

export interface FilePreviewContentProps {
  /** Base URL for raw file content (used for image src) */
  fileContentBaseUrl: string;
  /** File path within the entity folder */
  filePath: string;
  /** File name for display / language detection */
  fileName: string;
  /** Optional className for the content wrapper */
  contentClassName?: string;
  /** Whether the file is compact (used for side-by-side diff) */
  compactHeader: boolean;
  /** Whether the file is an image */
  isImage: boolean;
  /** Whether the file is editable */
  isEditable: boolean;
  /** Whether content is loading */
  isLoading: boolean;
  /** Query error, if any */
  error: Error | null;
  /** Retry callback for error state */
  onRetry: () => void;
  /** Current draft content */
  draftContent: string;
  /** Original content (for diff) */
  originalContent: string;
  /** Rendering flags */
  showEditor: boolean;
  showRenderedMarkdown: boolean;
  showDiff: boolean;
  /** Callback when editor content changes */
  onDraftChange: (value?: string) => void;
}

export function FilePreviewContent({
  fileContentBaseUrl,
  filePath,
  fileName,
  contentClassName,
  compactHeader,
  isImage,
  isEditable,
  isLoading,
  error,
  onRetry,
  draftContent,
  originalContent,
  showEditor,
  showRenderedMarkdown,
  showDiff,
  onDraftChange,
}: FilePreviewContentProps) {
  const resolvedTheme = useResolvedTheme();
  const isLight = resolvedTheme === "light";
  const editorTheme = isLight ? "vs" : "vs-dark";
  const editorLanguage = useMemo(() => getMonacoLanguage(fileName), [fileName]);

  return (
    <div
      className={cn(
        "relative flex-1 min-h-[200px] sm:min-h-[300px] overflow-hidden",
        contentClassName,
      )}
    >
      {isLoading && (
        <div
          className="absolute inset-0 flex flex-col justify-center gap-3 bg-slate-900/70 p-4 backdrop-blur-[1px]"
          data-testid="file-preview-loading-state"
          role="status"
          aria-live="polite"
          aria-busy="true"
        >
          <div className="inline-flex items-center gap-2 text-sm text-slate-300">
            <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
            <span>Loading file preview...</span>
          </div>
          <div className="space-y-2">
            <div className="h-3 w-full animate-pulse rounded bg-slate-700/60" />
            <div className="h-3 w-11/12 animate-pulse rounded bg-slate-700/60" />
            <div className="h-3 w-9/12 animate-pulse rounded bg-slate-700/60" />
          </div>
        </div>
      )}

      {error && (
        <div className="p-4">
          <ErrorState
            error={error}
            title="Unable to load file"
            onRetry={onRetry}
          />
        </div>
      )}

      {/* Image preview */}
      {isImage && (
        <div className="flex items-center justify-center p-4 bg-slate-900/50">
          <img
            src={`${fileContentBaseUrl}/${filePath}`}
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
            isLight ? "prose-slate" : "prose-invert",
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
          onChange={onDraftChange}
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
  );
}
