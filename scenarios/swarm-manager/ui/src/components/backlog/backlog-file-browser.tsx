/**
 * BacklogFileBrowser
 *
 * Extracted from BacklogDetailsPage — encapsulates the file tree browser
 * with search, upload toggle, context menu, and file action dialogs
 * (rename/move/copy/delete).
 */

import { useState, useCallback, useEffect, useMemo, useRef, type MouseEvent, type ReactNode } from "react";
import {
  ArrowRightLeft,
  Copy,
  Edit,
  FileText,
  Lock,
  Loader2,
  Search,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import { Popover } from "../ui/popover";
import { Button } from "../ui/button";
import { ErrorState } from "../ui/error-state";
import { Input } from "../ui/input";
import { InlineLoadingIndicator } from "../ui/loading-states";
import { FileTree, type TreeFile } from "../ui/file-tree";
import { FileUpload } from "../ui/file-upload";
import { Dialog } from "../ui/dialog";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { cn } from "../../lib";
import { collectMatchingFiles, getBaseName, getParentPath, joinPath, normalizeDestinationPath } from "../../lib/file-path-utils";
import { selectors } from "../../consts/selectors";
import type { BacklogFile, BacklogKind } from "../../types";

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
  backlogKind: BacklogKind;
  backlogName: string;
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
  backlogKind,
  backlogName,
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
  }, [selectedFile?.path, backlogKind, backlogName]);

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

  const renderFileActionItems = useCallback((target: BacklogFile, closeMenu: () => void) => {
    const isProtected = target.path === "spec.json";
    const rowClass = "flex w-full items-center justify-start gap-2 px-3 py-2 text-sm text-slate-100 hover:bg-slate-800/80";
    return (
      <div className="py-1" data-testid="backlog-file-actions-menu">
        <button
          type="button"
          className={rowClass}
          disabled={isProtected}
          onClick={() => {
            closeMenu();
            openFileActionDialog("rename", target);
          }}
        >
          <Edit className="h-4 w-4 text-slate-300" />
          Rename
        </button>
        <button
          type="button"
          className={rowClass}
          disabled={isProtected}
          onClick={() => {
            closeMenu();
            openFileActionDialog("move", target);
          }}
        >
          <ArrowRightLeft className="h-4 w-4 text-slate-300" />
          Move
        </button>
        <button
          type="button"
          className={rowClass}
          disabled={isProtected}
          onClick={() => {
            closeMenu();
            openFileActionDialog("copy", target);
          }}
        >
          <Copy className="h-4 w-4 text-slate-300" />
          Copy
        </button>
        <button
          type="button"
          className={cn(rowClass, "text-red-300 hover:bg-red-500/20")}
          disabled={isProtected}
          onClick={() => {
            closeMenu();
            openFileActionDialog("delete", target);
          }}
        >
          <Trash2 className="h-4 w-4 text-red-300" />
          Delete
        </button>
        {isProtected && (
          <p className="flex items-center gap-2 px-3 py-2 text-xs text-slate-400">
            <Lock className="h-3.5 w-3.5" />
            `spec.json` is protected.
          </p>
        )}
      </div>
    );
  }, [openFileActionDialog]);

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
          <div className="space-y-3 lg:hidden">
            <Input
              type="text"
              value={fileSearch}
              onChange={(event) => setFileSearch(event.target.value)}
              placeholder="Search files"
              leftIcon={<Search className="h-4 w-4" />}
              rightSlot={
                fileSearch.trim().length > 0 ? (
                  <button
                    type="button"
                    onClick={() => setFileSearch("")}
                    className="rounded-full p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
                    aria-label="Clear search"
                  >
                    <X className="h-4 w-4" />
                  </button>
                ) : null
              }
            />
            {recentFiles.length > 0 && fileSearch.trim().length === 0 && (
              <div className="space-y-2">
                <p className="text-xs uppercase tracking-wider text-slate-500">Recent files</p>
                <div className="space-y-1">
                  {recentFiles.map((file) => (
                    <button
                      key={file.path}
                      type="button"
                      onClick={() => handleFileSelect(file)}
                      className="flex w-full items-center gap-2 rounded-lg border border-white/10 bg-slate-800/40 px-3 py-2 text-left text-sm text-slate-200 hover:border-cyan-500/50 hover:bg-slate-800/70"
                    >
                      <FileText className="h-4 w-4 text-slate-400" />
                      <span className="truncate">{file.name}</span>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>

          {showUpload && (
            <FileUpload
              backlogKind={backlogKind}
              backlogName={backlogName}
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
                void onRefetchFiles();
              }}
            />
          ) : fileSearch.trim().length > 0 ? (
            searchResults.length > 0 ? (
              <div className="space-y-1">
                {searchResults.map((file) => (
                  <button
                    key={file.path}
                    type="button"
                    onClick={() => handleFileSelect(file)}
                    className="flex w-full items-center gap-2 rounded-lg border border-white/10 bg-slate-800/40 px-3 py-2 text-left text-sm text-slate-200 hover:border-cyan-500/50 hover:bg-slate-800/70"
                  >
                    <FileText className="h-4 w-4 text-slate-400" />
                    <div className="flex min-w-0 flex-1 flex-col">
                      <span className="truncate">{file.name}</span>
                      <span className="truncate text-xs text-slate-500">{file.path}</span>
                    </div>
                  </button>
                ))}
              </div>
            ) : (
              <div className="rounded-lg border border-white/10 bg-slate-800/30 p-6 text-center text-sm text-slate-500">
                No files match "{fileSearch.trim()}".
              </div>
            )
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

      {/* File action dialogs (rename/move/copy) */}
      <Dialog
        isOpen={Boolean(activeFileAction && activeFileAction.action !== "delete")}
        onClose={() => {
          setActiveFileAction(null);
          setFileActionError(null);
        }}
        title={
          activeFileAction?.action === "rename"
            ? `Rename ${activeFileAction.target.type}`
            : activeFileAction?.action === "move"
              ? `Move ${activeFileAction.target.type}`
              : activeFileAction?.action === "copy"
                ? `Copy ${activeFileAction.target.type}`
                : "File Action"
        }
        maxWidth="max-w-md"
      >
        {activeFileAction && activeFileAction.action !== "delete" && (
          <div className="space-y-4">
            <div className="text-sm text-slate-300">
              <p className="text-xs uppercase tracking-wide text-slate-500">Source</p>
              <p className="mt-1 break-all rounded-lg bg-slate-800/60 px-3 py-2">{activeFileAction.target.path}</p>
            </div>
            <div className="space-y-2">
              <label className="text-xs uppercase tracking-wide text-slate-500">
                {activeFileAction.action === "rename" ? "New name" : "Destination path"}
              </label>
              <Input
                value={fileActionInput}
                onChange={(event) => setFileActionInput(event.target.value)}
                placeholder={activeFileAction.action === "rename" ? "new-name.ext" : "path/to/target"}
              />
            </div>
            {fileActionError && (
              <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">
                {fileActionError}
              </div>
            )}
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => {
                  setActiveFileAction(null);
                  setFileActionError(null);
                }}
                disabled={fileActionPending}
              >
                Cancel
              </Button>
              <Button
                variant="default"
                onClick={handleFileActionConfirm}
                disabled={fileActionPending}
                data-testid="confirm-file-action"
              >
                {fileActionPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : null}
                Apply
              </Button>
            </div>
          </div>
        )}
      </Dialog>

      {/* File delete confirmation dialog */}
      <ConfirmDialog
        isOpen={Boolean(activeFileAction && activeFileAction.action === "delete")}
        onClose={() => {
          setActiveFileAction(null);
          setFileActionError(null);
        }}
        onConfirm={handleFileActionConfirm}
        title={`Delete ${activeFileAction?.target.type ?? "file"}`}
        description={`Delete "${activeFileAction?.target.path ?? ""}" from this backlog item? This cannot be undone.`}
        confirmLabel="Delete"
        isLoading={fileActionPending}
      />
    </>
  );
}
