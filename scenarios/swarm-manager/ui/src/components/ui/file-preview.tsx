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

import { useCallback, type ReactNode } from "react";
import { cn } from "../../lib/utils";
import type { BacklogKind } from "../../types";
import { useFilePreviewState } from "./useFilePreviewState";
import { FilePreviewHeader } from "./FilePreviewHeader";
import { FilePreviewContent } from "./FilePreviewContent";

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
  /** Pre-fetched content string. When provided, skips the internal query. */
  content?: string;
  /** When true, disables editing (save/discard/diff). Defaults to false. */
  readOnly?: boolean;
  /** data-testid attribute */
  "data-testid"?: string;
}

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
  content: externalContent,
  readOnly = false,
  "data-testid": testId,
}: FilePreviewProps) {
  const state = useFilePreviewState({
    backlogKind,
    backlogName,
    filePath,
    fileName,
    externalContent,
    readOnly,
  });

  const { setIsDiffMode, setMarkdownView, setShowMobilePath } = state;

  const handleToggleDiff = useCallback(
    () => setIsDiffMode((prev) => !prev),
    [setIsDiffMode],
  );
  const handleToggleMarkdownView = useCallback(
    () => setMarkdownView((prev) => (prev === "rendered" ? "raw" : "rendered")),
    [setMarkdownView],
  );
  const handleToggleMobilePath = useCallback(
    () => setShowMobilePath((prev) => !prev),
    [setShowMobilePath],
  );

  return (
    <div
      className={cn(
        "rounded-lg border border-white/10 bg-slate-800/30 overflow-hidden flex flex-col",
        className,
      )}
      data-testid={testId ?? "file-preview"}
    >
      {/* Header */}
      <FilePreviewHeader
        fileType={state.fileType}
        fileName={fileName}
        filePath={filePath}
        compactHeader={compactHeader}
        stickyHeader={stickyHeader}
        isEditable={state.isEditable}
        isDirty={state.isDirty}
        isSaving={state.isSaving}
        saveErrorMessage={state.saveErrorMessage}
        markdownView={state.markdownView}
        isDiffMode={state.isDiffMode}
        showMobilePath={state.showMobilePath}
        copied={state.copied}
        showMarkdownToggle={state.showMarkdownToggle}
        canSave={state.canSave}
        canDiscard={state.canDiscard}
        canToggleDiff={state.canToggleDiff}
        onSave={state.handleSave}
        onDiscard={state.handleDiscard}
        onToggleDiff={handleToggleDiff}
        onToggleMarkdownView={handleToggleMarkdownView}
        onToggleMobilePath={handleToggleMobilePath}
        onCopyPath={state.handleCopyPath}
        headerActions={headerActions}
      />

      {/* Content */}
      <FilePreviewContent
        backlogKind={backlogKind}
        backlogName={backlogName}
        filePath={filePath}
        fileName={fileName}
        contentClassName={contentClassName}
        compactHeader={compactHeader}
        isImage={state.isImage}
        isEditable={state.isEditable}
        isLoading={state.isLoading}
        error={state.error}
        onRetry={() => state.refetch()}
        draftContent={state.draftContent}
        originalContent={state.originalContent}
        showEditor={state.showEditor}
        showRenderedMarkdown={state.showRenderedMarkdown}
        showDiff={state.showDiff}
        onDraftChange={state.handleDraftChange}
      />
    </div>
  );
}
