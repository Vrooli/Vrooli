/**
 * EntityFileWorkspace
 *
 * Composes EntityFileBrowser + FilePreview into a resizable split pane on
 * desktop and a preview-plus-files-sheet on mobile. Entity-agnostic: the
 * active file service arrives through FileServiceContext, so the backlog and
 * goal detail pages share this workspace rather than forking it.
 *
 * Grows to fill its container, which must be a flex column with a bounded
 * height (the detail pages' full-bleed body). It must never guess at the
 * detail header's height.
 */

import { useState, useCallback, useRef } from "react";
import {
  Files,
  Lock,
  MoreHorizontal,
} from "lucide-react";
import { useResizablePanel } from "../../hooks/useResizablePanel";
import { EntityFileBrowser, type FileActionType, type HeaderSlotProps } from "./entity-file-browser";
import { ErrorBoundary } from "../ui/error-boundary";
import { ErrorState } from "../ui/error-state";
import { FilePreview } from "../ui/file-preview";
import { Button } from "../ui/button";
import { BottomSheet } from "../ui/bottom-sheet";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import { useFileService } from "../../contexts/FileServiceContext";
import type { BacklogFile } from "../../types";

const MIN_FILES_PANEL_WIDTH = 240;
const MAX_FILES_PANEL_WIDTH = 520;
const MIN_PREVIEW_WIDTH = 320;
const RESIZE_HANDLE_WIDTH = 8;

export interface EntityFileWorkspaceProps {
  files: BacklogFile[] | undefined;
  isLoadingFiles: boolean;
  filesError: Error | null;
  selectedFile: BacklogFile | null;
  isLocked: boolean;
  onFileSelect: (file: BacklogFile) => void;
  onRefetchFiles: () => void;
  onUploadComplete: () => void;
  fileActionPending: boolean;
  onFileAction: (action: FileActionType, target: BacklogFile, destinationPath?: string) => void;
}

export function EntityFileWorkspace({
  files,
  isLoadingFiles,
  filesError,
  selectedFile,
  isLocked,
  onFileSelect,
  onRefetchFiles,
  onUploadComplete,
  fileActionPending,
  onFileAction,
}: EntityFileWorkspaceProps) {
  const workspaceRef = useRef<HTMLDivElement | null>(null);
  const filesPanelRef = useRef<HTMLDivElement | null>(null);
  const [showFilesSheet, setShowFilesSheet] = useState(false);
  const [previewResetKey, setPreviewResetKey] = useState(0);

  // Store header slot props from the render prop so we can use them in
  // the file preview header actions. Uses state (not ref) so changes trigger re-render.
  const [headerSlotProps, setHeaderSlotProps] = useState<HeaderSlotProps | null>(null);

  const { size: filesPanelWidth, isResizing, resizeHandleProps } = useResizablePanel({
    containerRef: workspaceRef,
    targetRef: filesPanelRef,
    minSize: MIN_FILES_PANEL_WIDTH,
    maxSize: MAX_FILES_PANEL_WIDTH,
    defaultSize: 320,
    adjacentMinSize: MIN_PREVIEW_WIDTH,
    handleWidth: RESIZE_HANDLE_WIDTH,
  });

  const handlePreviewRetry = useCallback(() => {
    setPreviewResetKey((prev) => prev + 1);
  }, []);

  const handleFileSelect = useCallback((file: BacklogFile) => {
    onFileSelect(file);
    setShowFilesSheet(false);
  }, [onFileSelect]);

  const filesButton = (
    <Button
      variant="outline"
      size="sm"
      className="lg:hidden"
      onClick={() => setShowFilesSheet(true)}
    >
      <Files className="mr-2 h-4 w-4" />
      Files
    </Button>
  );

  const handleHeaderSlotChange = useCallback((props: HeaderSlotProps) => {
    setHeaderSlotProps(props);
  }, []);

  const fileService = useFileService();
  const isProtectedSelectedFile = selectedFile?.path === fileService.protectedFile;

  const fileHeaderActions = (
    <>
      {filesButton}
      {selectedFile && headerSlotProps && (
        <div className="relative" ref={headerSlotProps.headerFileActionsRef as React.RefObject<HTMLDivElement>}>
          <Button
            variant="outline"
            size="sm"
            onClick={headerSlotProps.handleOpenHeaderMenu}
            aria-label="File actions"
            title="File actions"
            className="h-8 w-8 p-0"
            data-testid="file-header-actions-trigger"
          >
            <MoreHorizontal className="h-4 w-4" />
          </Button>
          {headerSlotProps.showFileActionsMenu && (
            <div
              className="absolute right-0 top-10 z-30 min-w-[180px] overflow-visible rounded-md border border-white/10 bg-slate-900 shadow-lg"
              data-testid="file-header-actions-popover"
            >
              {headerSlotProps.renderFileActionItems(selectedFile, () => {
                // The menu close is handled inside renderFileActionItems callback
              })}
            </div>
          )}
        </div>
      )}
      {selectedFile && isProtectedSelectedFile && (
        // Read-only, not hidden: the file still renders in full. Say what the
        // restriction actually is rather than leaving "Protected" to be guessed.
        <span
          className="inline-flex shrink-0 items-center gap-1 rounded-full border border-amber-400/30 bg-amber-400/10 px-2 py-0.5 text-xs text-amber-200"
          title={`${fileService.protectedFile} is this item's canonical specification. You can read it here, but it cannot be edited, renamed, moved, or deleted — edit the item's fields instead.`}
          data-testid="file-read-only-badge"
        >
          <Lock className="h-3 w-3" aria-hidden />
          Read-only
        </span>
      )}
    </>
  );

  const fileBrowserProps = {
    files,
    isLoadingFiles,
    filesError,
    selectedFile,
    isLocked,
    onFileSelect: handleFileSelect,
    onRefetchFiles,
    onUploadComplete,
    fileActionPending,
    onFileAction,
    onHeaderSlotChange: handleHeaderSlotChange,
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div
        ref={workspaceRef}
        className={cn(
          "flex min-h-0 flex-1 flex-col lg:flex-row",
          isResizing && "select-none",
        )}
      >
        <div ref={filesPanelRef} className="hidden min-h-0 lg:flex lg:flex-col" style={{ width: filesPanelWidth }}>
          <EntityFileBrowser {...fileBrowserProps} />
        </div>
        <div
          className="hidden lg:flex w-2 shrink-0 items-center justify-center bg-slate-900/40 border-x border-white/10 cursor-col-resize"
          {...resizeHandleProps}
        >
          <div className="h-10 w-1 rounded-full bg-slate-700/80" />
        </div>
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          {selectedFile ? (
            <ErrorBoundary
              key={`${selectedFile.path}-${previewResetKey}`}
              fallback={
                <div className="flex flex-1 items-center justify-center p-6">
                  <ErrorState
                    title="Unable to render file preview"
                    message="Try reloading the preview or choose another file."
                    onRetry={handlePreviewRetry}
                  />
                </div>
              }
            >
              <FilePreview
                filePath={selectedFile.path}
                fileName={selectedFile.name}
                compactHeader
                stickyHeader
                readOnly={isProtectedSelectedFile}
                headerActions={fileHeaderActions}
                className="min-h-0 flex-1 border-0 rounded-none bg-transparent"
                contentClassName="min-h-0 flex-1 max-h-none"
                data-testid={selectors.backlogDetails.filePreview}
              />
            </ErrorBoundary>
          ) : (
            <>
              <div className="flex shrink-0 items-center justify-between border-b border-white/10 bg-slate-800/50 px-3 py-2 sm:px-4 sm:py-3">
                <span className="text-sm font-medium text-slate-300">No file selected</span>
                {filesButton}
              </div>
              <div className="flex flex-1 items-center justify-center p-8 text-center text-slate-500">
                Select a file to preview its contents
              </div>
            </>
          )}
        </div>
      </div>

      <BottomSheet
        isOpen={showFilesSheet}
        onClose={() => setShowFilesSheet(false)}
        title="Files"
        // The browser owns its own padding, and a file list is the kind of
        // content that wants the full sheet rather than a short strip.
        className="h-[85dvh] lg:h-auto"
        contentClassName="p-0"
      >
        <EntityFileBrowser {...fileBrowserProps} />
      </BottomSheet>
    </div>
  );
}
