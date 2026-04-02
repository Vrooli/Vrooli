/**
 * BacklogFileWorkspace
 *
 * Extracted from BacklogDetailsPage — composes BacklogFileBrowser + FilePreview
 * into a resizable split-pane layout with a mobile files dialog.
 */

import { useState, useCallback, useRef } from "react";
import {
  Files,
  MoreHorizontal,
} from "lucide-react";
import { useResizablePanel } from "../../hooks/useResizablePanel";
import { BacklogFileBrowser, type FileActionType, type HeaderSlotProps } from "./backlog-file-browser";
import { ErrorBoundary } from "../ui/error-boundary";
import { ErrorState } from "../ui/error-state";
import { FilePreview } from "../ui/file-preview";
import { Button } from "../ui/button";
import { BottomSheet } from "../ui/bottom-sheet";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import type { BacklogFile, BacklogKind } from "../../types";

const MIN_FILES_PANEL_WIDTH = 240;
const MAX_FILES_PANEL_WIDTH = 520;
const MIN_PREVIEW_WIDTH = 320;
const RESIZE_HANDLE_WIDTH = 8;

export interface BacklogFileWorkspaceProps {
  files: BacklogFile[] | undefined;
  isLoadingFiles: boolean;
  filesError: Error | null;
  selectedFile: BacklogFile | null;
  isLocked: boolean;
  backlogKind: BacklogKind;
  backlogName: string;
  onFileSelect: (file: BacklogFile) => void;
  onRefetchFiles: () => void;
  onUploadComplete: () => void;
  fileActionPending: boolean;
  onFileAction: (action: FileActionType, target: BacklogFile, destinationPath?: string) => void;
}

export function BacklogFileWorkspace({
  files,
  isLoadingFiles,
  filesError,
  selectedFile,
  isLocked,
  backlogKind,
  backlogName,
  onFileSelect,
  onRefetchFiles,
  onUploadComplete,
  fileActionPending,
  onFileAction,
}: BacklogFileWorkspaceProps) {
  const workspaceRef = useRef<HTMLDivElement | null>(null);
  const [showFilesSheet, setShowFilesSheet] = useState(false);
  const [previewResetKey, setPreviewResetKey] = useState(0);

  // Store header slot props from the render prop so we can use them in
  // the file preview header actions. Uses state (not ref) so changes trigger re-render.
  const [headerSlotProps, setHeaderSlotProps] = useState<HeaderSlotProps | null>(null);

  const { size: filesPanelWidth, isResizing, resizeHandleProps } = useResizablePanel({
    containerRef: workspaceRef,
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

  const isProtectedSelectedFile = selectedFile?.path === "spec.json";

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
        <span
          className="text-xs text-amber-300"
          title="This file is essential and cannot be renamed, moved, or deleted"
        >
          Protected file
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
    backlogKind,
    backlogName,
    onFileSelect: handleFileSelect,
    onRefetchFiles,
    onUploadComplete,
    fileActionPending,
    onFileAction,
    onHeaderSlotChange: handleHeaderSlotChange,
  };

  return (
    <div className="h-[calc(100dvh-6rem)] lg:h-auto lg:flex-1">
      <div className="h-full overflow-hidden lg:rounded-xl lg:border lg:border-white/10 lg:bg-slate-900/30">
        <div
          ref={workspaceRef}
          className={cn(
            "flex h-full flex-1 flex-col lg:flex-row min-h-[calc(100dvh-6rem)] lg:min-h-[calc(100dvh-16rem)]",
            isResizing && "select-none"
          )}
        >
          <div className="hidden lg:flex flex-col" style={{ width: filesPanelWidth }}>
            <BacklogFileBrowser {...fileBrowserProps} />
          </div>
          <div
            className="hidden lg:flex w-2 items-center justify-center bg-slate-900/40 border-x border-white/10 cursor-col-resize"
            {...resizeHandleProps}
          >
            <div className="h-10 w-1 rounded-full bg-slate-700/80" />
          </div>
          <div className="flex flex-1 flex-col min-w-0">
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
                  backlogKind={backlogKind}
                  backlogName={backlogName}
                  filePath={selectedFile.path}
                  fileName={selectedFile.name}
                  compactHeader
                  stickyHeader
                  headerActions={fileHeaderActions}
                  className="flex-1 border-0 rounded-none bg-transparent"
                  contentClassName="flex-1 max-h-none"
                  data-testid={selectors.backlogDetails.filePreview}
                />
              </ErrorBoundary>
            ) : (
              <>
                <div className="flex items-center justify-between border-b border-white/10 bg-slate-800/50 px-3 py-2 sm:px-4 sm:py-3">
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
      </div>

      <BottomSheet
        isOpen={showFilesSheet}
        onClose={() => setShowFilesSheet(false)}
        title="Files"
      >
        <BacklogFileBrowser {...fileBrowserProps} />
      </BottomSheet>
    </div>
  );
}
