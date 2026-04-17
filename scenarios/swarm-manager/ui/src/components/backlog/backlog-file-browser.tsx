/**
 * BacklogFileBrowser
 *
 * Extracted from BacklogDetailsPage — encapsulates the file tree browser
 * with search, upload toggle, context menu, and file action dialogs
 * (rename/move/copy/delete).
 */

import { useState, useCallback, useEffect, useMemo, useRef, type MouseEvent, type ReactNode } from "react";
import { Upload } from "lucide-react";
import { Popover } from "../ui/popover";
import { Button } from "../ui/button";
import { ErrorState } from "../ui/error-state";
import { InlineLoadingIndicator } from "../ui/loading-states";
import { FileTree, type TreeFile } from "../ui/file-tree";
import { FileUpload } from "../ui/file-upload";
import { collectMatchingFiles, getBaseName, getParentPath, joinPath, normalizeDestinationPath } from "../../lib/file-path-utils";
import { selectors } from "../../consts/selectors";
import type { BacklogFile } from "../../types";
import { FileActionDialogs } from "./file-action-dialogs";
import { useFileActionMenuRenderer } from "./file-action-menu";
import { FileSearchResults, FileSearchResultsList } from "./file-search-results";

export type FileActionType = "rename" | "move" | "copy" | "delete";

interface FileActionTarget {
  action: FileActionType;
  target: BacklogFile;
}

interface FileActionMenuState {
  x: number;
  y: number;
  target: BacklogFile;
}

const RECENT_FILES_LIMIT = 5;

export interface BacklogFileBrowserProps {
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
  /**
   * Callback that receives internal state needed to build header action
   * buttons (file actions menu, context menu items).
   * Called via useEffect when the relevant state changes.
   */
  onHeaderSlotChange?: (props: HeaderSlotProps) => void;
}

export interface HeaderSlotProps {
  showFileActionsMenu: boolean;
  handleOpenHeaderMenu: (event: MouseEvent<HTMLButtonElement>) => void;
  renderFileActionItems: (target: BacklogFile, closeMenu: () => void) => ReactNode;
  headerFileActionsRef: React.RefObject<HTMLDivElement | null>;
}

export function BacklogFileBrowser({
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
  onHeaderSlotChange,
}: BacklogFileBrowserProps) {
  const [fileSearch, setFileSearch] = useState("");
  const [recentFiles, setRecentFiles] = useState<BacklogFile[]>([]);
  const [showUpload, setShowUpload] = useState(false);
  const [showFileActionsMenu, setShowFileActionsMenu] = useState(false);
  const [fileContextMenu, setFileContextMenu] = useState<FileActionMenuState | null>(null);
  const [activeFileAction, setActiveFileAction] = useState<FileActionTarget | null>(null);
  const [fileActionInput, setFileActionInput] = useState("");
  const [fileActionError, setFileActionError] = useState<string | null>(null);
  const headerFileActionsRef = useRef<HTMLDivElement | null>(null);

  const searchResults = useMemo(
    () => collectMatchingFiles(files ?? [], fileSearch),
    [files, fileSearch],
  );

  // Close header actions menu on outside click
  useEffect(() => {
    if (!showFileActionsMenu) return;
    const onMouseDown = (event: globalThis.MouseEvent) => {
      if (headerFileActionsRef.current && !headerFileActionsRef.current.contains(event.target as Node)) {
        setShowFileActionsMenu(false);
      }
    };
    document.addEventListener("mousedown", onMouseDown);
    return () => document.removeEventListener("mousedown", onMouseDown);
  }, [showFileActionsMenu]);

  // Reset action state when file selection changes
  useEffect(() => {
    setShowFileActionsMenu(false);
    setFileContextMenu(null);
    setActiveFileAction(null);
    setFileActionInput("");
    setFileActionError(null);
  }, [selectedFile?.path]);

  const openFileActionDialog = useCallback((action: FileActionType, target: BacklogFile) => {
    setActiveFileAction({ action, target });
    setFileActionError(null);
    if (action === "rename") {
      setFileActionInput(getBaseName(target.path));
      return;
    }
    if (action === "move") {
      setFileActionInput(target.path);
      return;
    }
    if (action === "copy") {
      const suffix = "-copy";
      setFileActionInput(joinPath(getParentPath(target.path), `${getBaseName(target.path)}${suffix}`));
      return;
    }
    setFileActionInput("");
  }, []);

  const handleFileContextMenu = useCallback((file: TreeFile, event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    setShowFileActionsMenu(false);
    setFileContextMenu({
      x: event.clientX,
      y: event.clientY,
      target: file as BacklogFile,
    });
  }, []);

  const handleOpenHeaderMenu = useCallback((event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    setFileContextMenu(null);
    setShowFileActionsMenu((prev) => !prev);
  }, []);

  const handleFileActionConfirm = useCallback(() => {
    if (!activeFileAction) return;
    const { action, target } = activeFileAction;
    if (action === "delete") {
      onFileAction(action, target);
      setActiveFileAction(null);
      setFileActionError(null);
      return;
    }

    if (action === "rename") {
      const nextName = fileActionInput.trim();
      if (!nextName || nextName.includes("/")) {
        setFileActionError("Rename requires a file or folder name without slashes.");
        return;
      }
      const destinationPath = joinPath(getParentPath(target.path), nextName);
      onFileAction(action, target, destinationPath);
      setActiveFileAction(null);
      setFileActionError(null);
      return;
    }

    const destinationPath = normalizeDestinationPath(fileActionInput);
    if (!destinationPath) {
      setFileActionError("Destination path is required.");
      return;
    }
    onFileAction(action, target, destinationPath);
    setActiveFileAction(null);
    setFileActionError(null);
  }, [activeFileAction, fileActionInput, onFileAction]);

  const handleFileSelect = useCallback((file: BacklogFile) => {
    if (file.type === "file") {
      onFileSelect(file);
      setRecentFiles((prev) => {
        const next = [file, ...prev.filter((entry) => entry.path !== file.path)];
        return next.slice(0, RECENT_FILES_LIMIT);
      });
    }
  }, [onFileSelect]);

  const renderFileActionItems = useFileActionMenuRenderer({
    onOpenActionDialog: openFileActionDialog,
  });

  // Notify the parent when header-relevant state changes so the workspace
  // can compose the header bar with the file actions menu.
  useEffect(() => {
    onHeaderSlotChange?.({
      showFileActionsMenu,
      handleOpenHeaderMenu,
      renderFileActionItems,
      headerFileActionsRef,
    });
  }, [showFileActionsMenu, onHeaderSlotChange, handleOpenHeaderMenu, renderFileActionItems, headerFileActionsRef]);

  const handleDialogClose = useCallback(() => {
    setActiveFileAction(null);
    setFileActionError(null);
  }, []);

  return (
    <>
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-end border-b border-white/10 px-3 py-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowUpload(!showUpload)}
            disabled={isLocked}
            data-testid="toggle-upload"
          >
            <Upload className="mr-2 h-4 w-4" />
            {showUpload ? "Hide Upload" : "Upload Files"}
          </Button>
        </div>
        <div className="flex-1 space-y-4 overflow-y-auto px-3 pb-4 pt-4">
          <FileSearchResults
            fileSearch={fileSearch}
            onFileSearchChange={setFileSearch}
            searchResults={searchResults}
            recentFiles={recentFiles}
            onFileSelect={handleFileSelect}
          />

          {showUpload && (
            <FileUpload
              onUploadComplete={onUploadComplete}
              data-testid={selectors.backlogDetails.fileUpload}
            />
          )}

          {isLoadingFiles ? (
            <div className="rounded-lg border border-white/10 bg-slate-800/30 p-6 text-center">
              <InlineLoadingIndicator
                label="Loading files..."
                className="border-transparent bg-transparent px-0 text-slate-400"
                testId="backlog-files-loading"
              />
            </div>
          ) : filesError ? (
            <ErrorState
              error={filesError}
              title="Unable to load files"
              message="Try again to reload the file tree."
              onRetry={() => {
                onRefetchFiles();
              }}
            />
          ) : fileSearch.trim().length > 0 ? (
            <FileSearchResultsList
              searchResults={searchResults}
              fileSearch={fileSearch}
              onFileSelect={handleFileSelect}
            />
          ) : (
            <FileTree
              files={files ?? []}
              onFileSelect={handleFileSelect}
              onItemContextMenu={handleFileContextMenu}
              selectedPath={selectedFile?.path}
              className="lg:rounded-none lg:border-0 lg:bg-transparent lg:py-0"
              data-testid={selectors.backlogDetails.fileTree}
            />
          )}
          <Popover
            isOpen={Boolean(fileContextMenu)}
            onClose={() => setFileContextMenu(null)}
            x={fileContextMenu?.x}
            y={fileContextMenu?.y}
            delayClickOutside
            testId="file-tree-context-popover"
          >
            {fileContextMenu
              ? renderFileActionItems(fileContextMenu.target, () => setFileContextMenu(null))
              : null}
          </Popover>
        </div>
      </div>

      <FileActionDialogs
        activeAction={activeFileAction}
        fileActionInput={fileActionInput}
        fileActionError={fileActionError}
        fileActionPending={fileActionPending}
        onInputChange={setFileActionInput}
        onConfirm={handleFileActionConfirm}
        onClose={handleDialogClose}
      />
    </>
  );
}
