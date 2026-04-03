/**
 * FilePreviewHeader Component
 *
 * Header bar for FilePreview with file info, save/discard/diff/markdown toggle buttons.
 *
 * [REQ:REQ-P0-004] File preview header controls
 */

import type { ReactNode } from "react";
import {
  Check,
  Code,
  Copy,
  Diff,
  Eye,
  Info,
  Loader2,
  RotateCcw,
  Save,
} from "lucide-react";
import { cn } from "../../lib/utils";
import { getFileTypeIcon, getFileTypeIconClass, type FileType } from "../../lib/file-type-utils";

export interface FilePreviewHeaderProps {
  /** The detected file type (e.g. "markdown", "code", "image") */
  fileType: FileType;
  /** Display name for the file */
  fileName: string;
  /** Full path to the file */
  filePath: string;
  /** Compact header layout (optimized for mobile) */
  compactHeader: boolean;
  /** Make header sticky */
  stickyHeader: boolean;
  /** Whether the file is editable */
  isEditable: boolean;
  /** Whether the content has unsaved changes */
  isDirty: boolean;
  /** Whether a save is in progress */
  isSaving: boolean;
  /** Error message from save attempt, or empty string */
  saveErrorMessage: string;
  /** Current markdown view mode */
  markdownView: "rendered" | "raw";
  /** Whether diff mode is active */
  isDiffMode: boolean;
  /** Whether the mobile path row is visible */
  showMobilePath: boolean;
  /** Whether the path was recently copied */
  copied: boolean;
  /** Whether the markdown toggle should be shown */
  showMarkdownToggle: boolean;
  /** Derived: can the user save */
  canSave: boolean;
  /** Derived: can the user discard */
  canDiscard: boolean;
  /** Derived: can the user toggle diff */
  canToggleDiff: boolean;
  /** Handlers */
  onSave: () => void;
  onDiscard: () => void;
  onToggleDiff: () => void;
  onToggleMarkdownView: () => void;
  onToggleMobilePath: () => void;
  onCopyPath: () => void;
  /** Optional header actions aligned to the right */
  headerActions?: ReactNode;
}

const actionButtonClass = (
  enabled: boolean,
  compactHeader: boolean,
  active = false,
  tone: "default" | "accent" = "default",
) =>
  cn(
    "inline-flex h-8 w-8 items-center justify-center transition-colors",
    compactHeader ? "rounded-md bg-transparent" : "rounded-full",
    enabled
      ? tone === "accent"
        ? "text-cyan-200 hover:bg-cyan-500/20 hover:text-cyan-100"
        : "text-slate-300 hover:bg-slate-700/60 hover:text-white"
      : "text-slate-600 cursor-not-allowed",
    active &&
      enabled &&
      (tone === "accent"
        ? "bg-cyan-500/20 text-cyan-100"
        : "bg-slate-700/60 text-white"),
  );

export function FilePreviewHeader({
  fileType,
  fileName,
  filePath,
  compactHeader,
  stickyHeader,
  isEditable,
  isDirty,
  isSaving,
  saveErrorMessage,
  markdownView,
  isDiffMode,
  showMobilePath,
  copied,
  showMarkdownToggle,
  canSave,
  canDiscard,
  canToggleDiff,
  onSave,
  onDiscard,
  onToggleDiff,
  onToggleMarkdownView,
  onToggleMobilePath,
  onCopyPath,
  headerActions,
}: FilePreviewHeaderProps) {
  const markdownToggleLabel =
    markdownView === "rendered" ? "Show raw markdown" : "Show rendered markdown";

  return (
    <>
      <div
        className={cn(
          "flex items-center gap-2 border-b border-white/10 bg-slate-800/50",
          compactHeader ? "px-3 py-2 sm:px-4 sm:py-3" : "px-4 py-3",
          stickyHeader && "sticky top-0 z-20 backdrop-blur",
        )}
      >
        {(() => {
          const Icon = getFileTypeIcon(fileType);
          return <Icon className={getFileTypeIconClass(fileType)} />;
        })()}
        <span
          className="font-medium text-slate-200 truncate"
          data-testid="file-preview-name"
        >
          {fileName}
        </span>
        <div className="ml-auto flex items-center gap-2">
          <span
            className={cn(
              "text-xs text-slate-500 max-w-[220px] truncate",
              compactHeader && "hidden sm:inline",
            )}
          >
            {filePath}
          </span>
          {isEditable && (
            <>
              <button
                type="button"
                onClick={onSave}
                disabled={!canSave}
                className={actionButtonClass(canSave, compactHeader, false, "accent")}
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
                onClick={onDiscard}
                disabled={!canDiscard}
                className={actionButtonClass(canDiscard, compactHeader)}
                aria-label="Discard changes"
                title={canDiscard ? "Discard changes" : "No changes to discard"}
                data-testid="file-preview-discard"
              >
                <RotateCcw className="h-4 w-4" />
              </button>
              {isDirty && (
                <button
                  type="button"
                  onClick={onToggleDiff}
                  disabled={!canToggleDiff}
                  className={actionButtonClass(canToggleDiff, compactHeader, isDiffMode)}
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
              onClick={onToggleMarkdownView}
              className={actionButtonClass(true, compactHeader)}
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
              onClick={onToggleMobilePath}
              className="sm:hidden inline-flex h-8 w-8 items-center justify-center rounded-md bg-transparent text-slate-400 transition-colors hover:bg-slate-700/60 hover:text-slate-200"
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
            onClick={onCopyPath}
            className="inline-flex h-8 w-8 items-center justify-center rounded-md bg-transparent text-slate-300 transition-colors hover:bg-slate-800/70 hover:text-white"
            aria-label="Copy file path"
            title={copied ? "Copied" : "Copy file path"}
          >
            {copied ? (
              <Check className="h-4 w-4" />
            ) : (
              <Copy className="h-4 w-4" />
            )}
          </button>
        </div>
      )}
    </>
  );
}
