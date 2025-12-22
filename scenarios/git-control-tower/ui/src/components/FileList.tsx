import { useState, useEffect, useRef, useMemo, useCallback, memo } from "react";
import {
  File,
  FilePlus,
  FileX,
  AlertTriangle,
  Plus,
  Minus,
  Trash2,
  Binary,
  ChevronDown,
  ChevronRight,
  Loader2,
  ShieldCheck
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import type { RepoFilesStatus } from "../lib/api";

type FileCategory = "staged" | "unstaged" | "untracked" | "conflicts";

type SelectedFileEntry = { path: string; staged: boolean };

interface FileListProps {
  files?: RepoFilesStatus;
  selectedFiles?: SelectedFileEntry[];
  selectedKeySet?: Set<string>;
  selectionKey: (entry: SelectedFileEntry) => string;
  syncStatus?: { ahead: number; behind: number; canPush: boolean; canPull: boolean; warning?: string };
  approvedChanges?: { available: boolean; committableFiles: number; warning?: string };
  approvedPaths?: Set<string>;
  onStageApproved?: () => void;
  isStagingApproved?: boolean;
  onPush?: () => void;
  onPull?: () => void;
  isPushing?: boolean;
  isPulling?: boolean;
  onSelectFile: (path: string, staged: boolean, event: React.MouseEvent<HTMLLIElement>) => void;
  onStageFile: (path: string) => void;
  onUnstageFile: (path: string) => void;
  onDiscardFile: (path: string, untracked: boolean) => void;
  onStageAll: () => void;
  onUnstageAll: () => void;
  isStaging: boolean;
  isDiscarding: boolean;
  confirmingDiscard: string | null;
  onConfirmDiscard: (path: string | null) => void;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
  fillHeight?: boolean;
}

interface FileSectionProps {
  title: string;
  category: FileCategory;
  files: string[];
  fileStatuses?: Record<string, string>;
  binaryFiles?: Set<string>;
  maxPathChars: number;
  icon: React.ReactNode;
  approvedFiles?: Set<string>;
  selectedFiles?: SelectedFileEntry[];
  selectedKeySet?: Set<string>;
  selectionKey: (entry: SelectedFileEntry) => string;
  onSelectFile: (path: string, staged: boolean, event: React.MouseEvent<HTMLLIElement>) => void;
  onAction: (path: string) => void;
  actionIcon: React.ReactNode;
  actionLabel: string;
  isLoading: boolean;
  defaultExpanded?: boolean;
  onDiscard?: (path: string) => void;
  isDiscarding?: boolean;
  confirmingDiscard?: string | null;
  onConfirmDiscard?: (path: string | null) => void;
}

const statusStyleMap = {
  D: "text-red-400 border-red-500/40 bg-red-500/10",
  M: "text-amber-300 border-amber-500/40 bg-amber-500/10",
  A: "text-emerald-300 border-emerald-500/40 bg-emerald-500/10",
  R: "text-cyan-300 border-cyan-500/40 bg-cyan-500/10",
  U: "text-red-300 border-red-500/40 bg-red-500/10",
  "?": "text-slate-300 border-slate-500/40 bg-slate-500/10"
};

function formatPath(path: string, maxChars: number) {
  if (path.length <= maxChars) return path;

  const segments = path.split("/");
  const last = segments[segments.length - 1] || path;
  const lastTwo = segments.length > 1 ? segments.slice(-2).join("/") : last;
  const first = segments[0] || path;

  const candidateMiddle = `${first}/.../${lastTwo}`;
  if (candidateMiddle.length <= maxChars) return candidateMiddle;

  const candidateMiddleShort = `${first}/.../${last}`;
  if (candidateMiddleShort.length <= maxChars) return candidateMiddleShort;

  const candidateEnd = `.../${lastTwo}`;
  if (candidateEnd.length <= maxChars) return candidateEnd;

  const candidateEndShort = `.../${last}`;
  if (candidateEndShort.length <= maxChars) return candidateEndShort;

  const tailMax = Math.max(1, maxChars - 4);
  return `.../${last.slice(-tailMax)}`;
}

function getStatusBadge(code: string | undefined, category: FileCategory) {
  if (!code) {
    if (category === "untracked") return { label: "?", style: statusStyleMap["?"] };
    if (category === "conflicts") return { label: "U", style: statusStyleMap.U };
    return { label: "M", style: statusStyleMap.M };
  }

  const normalized = code.toUpperCase();
  if (normalized.includes("D")) return { label: "D", style: statusStyleMap.D };
  if (normalized.includes("M")) return { label: "M", style: statusStyleMap.M };
  if (normalized.includes("A")) return { label: "A", style: statusStyleMap.A };
  if (normalized.includes("R")) return { label: "R", style: statusStyleMap.R };
  if (normalized.includes("U")) return { label: "U", style: statusStyleMap.U };
  if (normalized.includes("?")) return { label: "?", style: statusStyleMap["?"] };

  return { label: "M", style: statusStyleMap.M };
}

interface FileRowProps {
  file: string;
  displayPath: string;
  badge: { label: string; style: string };
  isSelected: boolean;
  isStaged: boolean;
  isConfirming: boolean;
  canDiscard: boolean;
  isLoading: boolean;
  isDiscarding: boolean;
  isBinary: boolean;
  isApproved: boolean;
  itemTestId: string;
  actionTestId: string;
  discardTestId: string;
  actionIcon: React.ReactNode;
  actionLabel: string;
  onSelectFile: (path: string, staged: boolean, event: React.MouseEvent<HTMLLIElement>) => void;
  onAction: (path: string) => void;
  onDiscard?: (path: string) => void;
  onConfirmDiscard?: (path: string | null) => void;
}

const FileRow = memo(function FileRow({
  file,
  displayPath,
  badge,
  isSelected,
  isStaged,
  isConfirming,
  canDiscard,
  isLoading,
  isDiscarding,
  isBinary,
  isApproved,
  itemTestId,
  actionTestId,
  discardTestId,
  actionIcon,
  actionLabel,
  onSelectFile,
  onAction,
  onDiscard,
  onConfirmDiscard
}: FileRowProps) {
  return (
    <li
      className={`group w-full flex items-center gap-2 px-2 py-1 rounded text-sm cursor-pointer transition-colors min-w-0 overflow-hidden select-none ${
        isSelected
          ? "bg-slate-700/50 text-slate-100"
          : "hover:bg-slate-800/50 text-slate-300"
      }`}
      data-testid={itemTestId}
      data-file-path={file}
      onClick={(event) => onSelectFile(file, isStaged, event)}
    >
      <span
        className={`h-5 w-5 flex items-center justify-center rounded border text-[10px] font-bold ${badge.style}`}
        aria-label={`Status ${badge.label}`}
        title={`Status ${badge.label}`}
      >
        {badge.label}
      </span>
      <File className="h-3.5 w-3.5 text-slate-500 flex-shrink-0" />
      <div className="flex-1 min-w-0 overflow-hidden">
        <span className="font-mono text-xs truncate block w-full" title={file}>
          {displayPath}
        </span>
      </div>

      {isBinary && (
        <span
          className="flex items-center gap-1 rounded border border-slate-700/60 bg-slate-900/60 px-1.5 py-0.5 text-[10px] text-slate-400"
          title="Binary file"
        >
          <Binary className="h-3 w-3" />
          bin
        </span>
      )}

      {isApproved && (
        <span
          className="flex items-center gap-1 rounded border border-emerald-500/30 bg-emerald-500/10 px-1.5 py-0.5 text-[10px] text-emerald-300"
          title="Sandbox-approved change"
        >
          <ShieldCheck className="h-3 w-3" />
          approved
        </span>
      )}

      {isConfirming && onConfirmDiscard && onDiscard && (
        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
          <span className="text-xs text-red-400 mr-1">Discard?</span>
          <button
            className="px-1.5 py-0.5 text-xs bg-red-600 hover:bg-red-500 text-white rounded transition-colors"
            onClick={() => {
              onDiscard(file);
              onConfirmDiscard(null);
            }}
            disabled={isDiscarding}
            data-testid="confirm-discard-yes"
          >
            Yes
          </button>
          <button
            className="px-1.5 py-0.5 text-xs bg-slate-600 hover:bg-slate-500 text-white rounded transition-colors"
            onClick={() => onConfirmDiscard(null)}
            data-testid="confirm-discard-no"
          >
            No
          </button>
        </div>
      )}

      {!isConfirming && (
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-all">
          <button
            className="p-1 rounded hover:bg-slate-700 transition-all"
            onClick={(e) => {
              e.stopPropagation();
              onAction(file);
            }}
            disabled={isLoading}
            title={actionLabel}
            data-testid={actionTestId}
          >
            {isLoading ? (
              <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
            ) : (
              actionIcon
            )}
          </button>
          {canDiscard && onConfirmDiscard && (
            <button
              className="p-1 rounded hover:bg-red-900/50 transition-all"
              onClick={(e) => {
                e.stopPropagation();
                onConfirmDiscard(file);
              }}
              disabled={isDiscarding}
              title="Discard changes"
              data-testid={discardTestId}
            >
              {isDiscarding ? (
                <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
              ) : (
                <Trash2 className="h-3 w-3 text-red-400" />
              )}
            </button>
          )}
        </div>
      )}
    </li>
  );
});

function FileSection({
  title,
  category,
  files,
  fileStatuses,
  binaryFiles,
  approvedFiles,
  maxPathChars,
  icon,
  selectedFiles,
  selectedKeySet,
  selectionKey,
  onSelectFile,
  onAction,
  actionIcon,
  actionLabel,
  isLoading,
  defaultExpanded = true,
  onDiscard,
  isDiscarding,
  confirmingDiscard,
  onConfirmDiscard
}: FileSectionProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  const isStaged = category === "staged";
  const canDiscard = category === "unstaged" || category === "untracked";
  const selectedKeys = selectedKeySet ?? new Set(selectedFiles?.map(selectionKey));

  const entries = useMemo(
    () =>
      files.map((file) => {
        const badge = getStatusBadge(fileStatuses?.[file], category);
        return {
          file,
          key: selectionKey({ path: file, staged: isStaged }),
          badge,
          displayPath: formatPath(file, maxPathChars),
          isBinary: binaryFiles?.has(file) ?? false,
          isApproved: approvedFiles?.has(file) ?? false
        };
      }),
    [files, fileStatuses, category, maxPathChars, selectionKey, isStaged, binaryFiles, approvedFiles]
  );

  if (files.length === 0) return null;

  return (
    <div className="mb-4" data-testid={`file-section-${category}`}>
      <button
        className="flex items-center gap-2 w-full text-left px-2 py-1.5 hover:bg-slate-800/50 rounded transition-colors"
        onClick={() => setExpanded(!expanded)}
        data-testid={`file-section-toggle-${category}`}
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 text-slate-500" />
        ) : (
          <ChevronRight className="h-3 w-3 text-slate-500" />
        )}
        {icon}
        <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">
          {title}
        </span>
        <span className="text-xs text-slate-600 ml-auto">{files.length}</span>
      </button>

      {expanded && (
        <ul className="mt-1 space-y-0.5 min-w-0">
          {entries.map((entry) => (
            <FileRow
              key={entry.file}
              file={entry.file}
              displayPath={entry.displayPath}
              badge={entry.badge}
              isSelected={selectedKeys?.has(entry.key) ?? false}
              isStaged={isStaged}
              isConfirming={confirmingDiscard === entry.file}
              canDiscard={canDiscard}
              isLoading={isLoading}
              isDiscarding={isDiscarding ?? false}
              isBinary={entry.isBinary}
              isApproved={entry.isApproved}
              itemTestId={`file-item-${category}`}
              actionTestId={`file-action-${category}`}
              discardTestId={`file-discard-${category}`}
              actionIcon={actionIcon}
              actionLabel={actionLabel}
              onSelectFile={onSelectFile}
              onAction={onAction}
              onDiscard={onDiscard}
              onConfirmDiscard={onConfirmDiscard}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

export function FileList({
  files,
  selectedFiles,
  selectedKeySet,
  selectionKey,
  syncStatus,
  approvedChanges,
  approvedPaths,
  onStageApproved,
  isStagingApproved = false,
  onPush,
  onPull,
  isPushing = false,
  isPulling = false,
  onSelectFile,
  onStageFile,
  onUnstageFile,
  onDiscardFile,
  onStageAll,
  onUnstageAll,
  isStaging,
  isDiscarding,
  confirmingDiscard,
  onConfirmDiscard,
  collapsed = false,
  onToggleCollapse,
  fillHeight = true
}: FileListProps) {
  const hasStaged = (files?.staged?.length ?? 0) > 0;
  const hasUnstaged = (files?.unstaged?.length ?? 0) > 0 || (files?.untracked?.length ?? 0) > 0;
  const handleToggleCollapse = onToggleCollapse ?? (() => {});
  const scrollAreaRef = useRef<HTMLDivElement | null>(null);
  const [maxPathChars, setMaxPathChars] = useState(72);
  const binarySet = useMemo(() => new Set(files?.binary ?? []), [files?.binary]);
  const handleDiscardUnstaged = useCallback(
    (path: string) => onDiscardFile(path, false),
    [onDiscardFile]
  );
  const showApprovedBanner = Boolean(
    approvedChanges?.available && (approvedChanges.committableFiles ?? 0) > 0
  );
  const showSync = Boolean(syncStatus && (syncStatus.ahead > 0 || syncStatus.behind > 0));
  const handleDiscardUntracked = useCallback(
    (path: string) => onDiscardFile(path, true),
    [onDiscardFile]
  );

  useEffect(() => {
    if (!scrollAreaRef.current || typeof ResizeObserver === "undefined") return;

    const update = () => {
      const width = scrollAreaRef.current?.clientWidth ?? 0;
      const usable = Math.max(0, width - 64);
      const nextMax = Math.max(12, Math.min(140, Math.floor(usable / 7)));
      setMaxPathChars(nextMax);
    };

    update();
    const observer = new ResizeObserver(update);
    observer.observe(scrollAreaRef.current);
    return () => observer.disconnect();
  }, []);

  return (
    <Card
      className={`flex flex-col min-w-0 ${fillHeight ? "h-full" : "h-auto"}`}
      data-testid="file-list-panel"
    >
      <CardHeader className="flex-row items-center justify-between space-y-0 py-3 gap-2 min-w-0">
        <CardTitle className="flex items-center gap-2 min-w-0">
          <button
            className="p-1 rounded hover:bg-slate-800/70 transition-colors"
            onClick={handleToggleCollapse}
            aria-label={collapsed ? "Expand changes" : "Collapse changes"}
            type="button"
          >
            {collapsed ? (
              <ChevronRight className="h-3 w-3 text-slate-400" />
            ) : (
              <ChevronDown className="h-3 w-3 text-slate-400" />
            )}
          </button>
          <span className="truncate">Changes</span>
        </CardTitle>
        <div className="flex flex-wrap gap-2 justify-end min-w-0">
          {hasUnstaged && (
            <Button
              variant="outline"
              size="sm"
              onClick={onStageAll}
              disabled={isStaging}
              className="min-w-0 whitespace-normal px-3"
              data-testid="stage-all-button"
            >
              <Plus className="h-3 w-3 mr-1" />
              Stage All
            </Button>
          )}
          {hasStaged && (
            <Button
              variant="outline"
              size="sm"
              onClick={onUnstageAll}
              disabled={isStaging}
              className="min-w-0 whitespace-normal px-3"
              data-testid="unstage-all-button"
            >
              <Minus className="h-3 w-3 mr-1" />
              Unstage All
            </Button>
          )}
        </div>
      </CardHeader>

      {!collapsed && (
        <CardContent className="flex-1 min-w-0 p-0 overflow-hidden">
        {showApprovedBanner && (
          <div className="mx-2 mt-2 mb-1 rounded-md border border-emerald-800/50 bg-emerald-950/20 p-2 text-xs text-emerald-200">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <ShieldCheck className="h-3.5 w-3.5 text-emerald-300" />
                <span>Approved changes ready</span>
                <span className="text-emerald-300">
                  {approvedChanges?.committableFiles ?? 0} file
                  {(approvedChanges?.committableFiles ?? 0) !== 1 ? "s" : ""}
                </span>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={onStageApproved}
                disabled={isStagingApproved}
                className="h-7 px-2"
                data-testid="stage-approved-button"
              >
                Stage approved
              </Button>
            </div>
            {approvedChanges?.warning && (
              <div className="mt-1 text-[11px] text-emerald-300/80">
                {approvedChanges.warning}
              </div>
            )}
          </div>
        )}
        {showSync && (
          <div className="mx-2 mt-2 mb-1 rounded-md border border-slate-800/60 bg-slate-900/50 p-2 text-xs text-slate-300">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span className="text-slate-400">Sync</span>
                {syncStatus?.ahead ? (
                  <span className="text-emerald-300">{syncStatus.ahead} ahead</span>
                ) : null}
                {syncStatus?.behind ? (
                  <span className="text-amber-300">{syncStatus.behind} behind</span>
                ) : null}
              </div>
              <div className="flex items-center gap-1">
                {syncStatus?.behind ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onPull}
                    disabled={isPulling || !syncStatus?.canPull}
                    title={syncStatus?.warning || "Pull from remote"}
                    className="h-7 px-2"
                  >
                    Pull
                  </Button>
                ) : null}
                {syncStatus?.ahead ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onPush}
                    disabled={isPushing || !syncStatus?.canPush}
                    title={syncStatus?.warning || "Push to remote"}
                    className="h-7 px-2"
                  >
                    Push
                  </Button>
                ) : null}
              </div>
            </div>
            {syncStatus?.warning && (
              <div className="mt-1 text-[11px] text-amber-400">{syncStatus.warning}</div>
            )}
          </div>
        )}
        <ScrollArea className="h-full min-w-0 px-2 py-2 select-none" ref={scrollAreaRef}>
            {/* Conflicts - Always show first if any */}
            <FileSection
              title="Conflicts"
              category="conflicts"
              files={files?.conflicts ?? []}
              fileStatuses={files?.statuses}
              binaryFiles={binarySet}
              approvedFiles={approvedPaths}
              maxPathChars={maxPathChars}
              icon={<AlertTriangle className="h-3.5 w-3.5 text-red-500" />}
              selectedFiles={selectedFiles}
              selectedKeySet={selectedKeySet}
              selectionKey={selectionKey}
              onSelectFile={onSelectFile}
              onAction={onStageFile}
              actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
              actionLabel="Stage file"
              isLoading={isStaging}
            />

          {/* Staged Changes */}
            <FileSection
              title="Staged"
              category="staged"
              files={files?.staged ?? []}
              fileStatuses={files?.statuses}
              binaryFiles={binarySet}
              approvedFiles={approvedPaths}
              maxPathChars={maxPathChars}
              icon={<FilePlus className="h-3.5 w-3.5 text-emerald-500" />}
              selectedFiles={selectedFiles}
              selectedKeySet={selectedKeySet}
              selectionKey={selectionKey}
              onSelectFile={onSelectFile}
              onAction={onUnstageFile}
              actionIcon={<Minus className="h-3 w-3 text-slate-400" />}
              actionLabel="Unstage file"
              isLoading={isStaging}
            />

          {/* Unstaged Changes */}
            <FileSection
              title="Modified"
              category="unstaged"
              files={files?.unstaged ?? []}
              fileStatuses={files?.statuses}
              binaryFiles={binarySet}
              approvedFiles={approvedPaths}
              maxPathChars={maxPathChars}
              icon={<FileX className="h-3.5 w-3.5 text-amber-500" />}
              selectedFiles={selectedFiles}
              selectedKeySet={selectedKeySet}
              selectionKey={selectionKey}
              onSelectFile={onSelectFile}
              onAction={onStageFile}
              actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
              actionLabel="Stage file"
              isLoading={isStaging}
              onDiscard={handleDiscardUnstaged}
              isDiscarding={isDiscarding}
              confirmingDiscard={confirmingDiscard}
              onConfirmDiscard={onConfirmDiscard}
            />

          {/* Untracked Files */}
            <FileSection
              title="Untracked"
              category="untracked"
              files={files?.untracked ?? []}
              fileStatuses={files?.statuses}
              binaryFiles={binarySet}
              approvedFiles={approvedPaths}
              maxPathChars={maxPathChars}
              icon={<File className="h-3.5 w-3.5 text-slate-500" />}
              selectedFiles={selectedFiles}
              selectedKeySet={selectedKeySet}
              selectionKey={selectionKey}
              onSelectFile={onSelectFile}
              onAction={onStageFile}
              actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
              actionLabel="Stage file"
              isLoading={isStaging}
              defaultExpanded={false}
              onDiscard={handleDiscardUntracked}
              isDiscarding={isDiscarding}
              confirmingDiscard={confirmingDiscard}
              onConfirmDiscard={onConfirmDiscard}
            />

          {/* Empty State */}
            {files &&
             (files.staged?.length ?? 0) === 0 &&
             (files.unstaged?.length ?? 0) === 0 &&
             (files.untracked?.length ?? 0) === 0 &&
             (files.conflicts?.length ?? 0) === 0 && (
              <div className="flex flex-col items-center justify-center py-12 text-center" data-testid="empty-state">
                <File className="h-8 w-8 text-slate-700 mb-3" />
                <p className="text-sm text-slate-500">No changes detected</p>
                <p className="text-xs text-slate-600 mt-1">
                  Working directory is clean
                </p>
              </div>
            )}
        </ScrollArea>
        </CardContent>
      )}
    </Card>
  );
}
