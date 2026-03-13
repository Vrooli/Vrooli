import {
  useState,
  useEffect,
  useRef,
  useMemo,
  useCallback,
  memo,
  createContext,
  useContext,
} from "react";
import {
  File,
  FilePlus,
  FileX,
  AlertTriangle,
  Plus,
  Minus,
  Trash2,
  EyeOff,
  Binary,
  ChevronDown,
  ChevronRight,
  Loader2,
  ShieldCheck,
  MoreVertical,
  History,
  Square,
  CheckSquare,
  X,
  BarChart3,
  ClipboardCheck,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import { BottomSheet, BottomSheetAction } from "./ui/bottom-sheet";
import { useIsMobile, useLongPress } from "../hooks";
import type { DiffStats, RepoFilesStatus, RepoFileStats, FileViewMode } from "../lib/api";
import { resolveGroupForFile } from "../lib/grouping";
import { ViewModeCycleButton } from "./ViewModeCycleButton";
import { ProjectTreeView } from "./ProjectTreeView";
import { ContextMenu, type ContextMenuItem } from "./ContextMenu";
import { ChangeMetricsModal } from "./ChangeMetricsModal";
import { getFileStats, filterFileStats, filterCategoryStats } from "../lib/metrics";
import { useDiffStats } from "../lib/hooks";

// Context to pass mobile state down without prop drilling
const MobileContext = createContext(false);

type FileCategory = "staged" | "unstaged" | "untracked" | "conflicts";

type SelectedFileEntry = { path: string; staged: boolean };

export type GroupingRule = {
  id: string;
  label: string;
  prefix?: string;
  prefixes?: string[];
  mode?: "prefix" | "segment";
};

interface FileListProps {
  files?: RepoFilesStatus;
  fileStats?: RepoFileStats;
  selectedFiles?: SelectedFileEntry[];
  selectedKeySet?: Set<string>;
  selectionKey: (entry: SelectedFileEntry) => string;
  approvedChanges?: {
    available: boolean;
    committableFiles: number;
    warning?: string;
  };
  approvedPaths?: Set<string>;
  onStageApproved?: () => void;
  isStagingApproved?: boolean;
  onSelectFile: (
    path: string,
    staged: boolean,
    event: React.MouseEvent<HTMLLIElement>,
  ) => void;
  onStageFile: (path: string) => void;
  onUnstageFile: (path: string) => void;
  onDiscardFile: (path: string, untracked: boolean) => void;
  onIgnoreFile: (path: string, level?: "project" | "group", groupDir?: string) => void;
  onStageAll: () => void;
  onUnstageAll: () => void;
  isStaging: boolean;
  isDiscarding: boolean;
  isIgnoring: boolean;
  confirmingDiscard: string | null;
  onConfirmDiscard: (path: string | null) => void;
  confirmingIgnore: string | null;
  onConfirmIgnore: (path: string | null) => void;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
  fillHeight?: boolean;
  fileViewMode?: FileViewMode;
  groupingRules?: GroupingRule[];
  groupingAvailable?: boolean;
  onCycleViewMode?: () => void;
  onStagePaths?: (paths: string[]) => void;
  onDiscardPaths?: (paths: string[], untracked: boolean) => void;
  onSelectAnyFile?: (path: string) => void;
  scrollToFile?: string;
  onScrollComplete?: () => void;
  onDeletePath?: (path: string, isDir: boolean) => void;
  onBlameFile?: (path: string) => void;
  repoId?: string | null;
  mobileSelectionMode?: boolean;
  onOpenReview?: (scenarioSlug: string) => void;
  onEnterSelectionMode?: (path: string, staged: boolean) => void;
  onExitSelectionMode?: () => void;
  onMobileSelectFile?: (path: string, staged: boolean, mode: "toggle" | "range") => void;
  fileHotspots?: Record<string, number>;
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
  onSelectFile: (
    path: string,
    staged: boolean,
    event: React.MouseEvent<HTMLLIElement>,
  ) => void;
  onAction: (path: string) => void;
  actionIcon: React.ReactNode;
  actionLabel: string;
  isLoading: boolean;
  changeStats?: DiffStats;
  defaultExpanded?: boolean;
  onDiscard?: (path: string) => void;
  isDiscarding?: boolean;
  confirmingDiscard?: string | null;
  onConfirmDiscard?: (path: string | null) => void;
  onIgnore?: (path: string, level?: "project" | "group", groupDir?: string) => void;
  isIgnoring?: boolean;
  confirmingIgnore?: string | null;
  onConfirmIgnore?: (path: string | null) => void;
  groupingRules?: GroupingRule[];
  onOpenMobileActions?: (file: string) => void;
  onContextMenu?: (file: string, event: React.MouseEvent) => void;
  mobileSelectionMode?: boolean;
  onLongPress?: (file: string, staged: boolean) => void;
  onMobileTap?: (file: string, staged: boolean, mode: "toggle" | "range") => void;
  onStatsClick?: () => void;
  onViewMetrics?: (file: string, category: FileCategory) => void;
}

const statusStyleMap = {
  D: "text-red-400 border-red-500/40 bg-red-500/10",
  M: "text-amber-300 border-amber-500/40 bg-amber-500/10",
  A: "text-emerald-300 border-emerald-500/40 bg-emerald-500/10",
  R: "text-cyan-300 border-cyan-500/40 bg-cyan-500/10",
  U: "text-red-300 border-red-500/40 bg-red-500/10",
  "?": "text-slate-300 border-slate-500/40 bg-slate-500/10",
};

function summarizeFileStats(
  paths: string[],
  stats?: Record<string, DiffStats>,
) {
  if (!stats) return undefined;
  const summary = { additions: 0, deletions: 0, files: 0 };
  let hasStats = false;
  for (const path of paths) {
    const entry = stats[path];
    if (!entry) continue;
    hasStats = true;
    summary.additions += entry.additions;
    summary.deletions += entry.deletions;
    summary.files += entry.files || 1;
  }
  return hasStats ? summary : undefined;
}

function hasLineStats(stats?: DiffStats) {
  return Boolean(stats && (stats.additions > 0 || stats.deletions > 0));
}

function LineStats({
  stats,
  compact = false,
  onClick,
}: {
  stats?: DiffStats;
  compact?: boolean;
  onClick?: (e: React.MouseEvent) => void;
}) {
  if (!hasLineStats(stats)) return null;
  const textSize = compact ? "text-[11px]" : "text-xs";
  const iconSize = compact ? "h-3 w-3" : "h-3.5 w-3.5";
  const inner = (
    <>
      <span className="flex items-center gap-1 text-emerald-500">
        <Plus className={iconSize} />
        {stats?.additions ?? 0}
      </span>
      <span className="flex items-center gap-1 text-red-500">
        <Minus className={iconSize} />
        {stats?.deletions ?? 0}
      </span>
    </>
  );
  if (onClick) {
    return (
      <button
        type="button"
        className={`flex items-center gap-2 ${textSize} hover:underline decoration-slate-600 cursor-pointer`}
        onClick={(e) => {
          e.stopPropagation();
          onClick(e);
        }}
        aria-label="View change metrics"
      >
        {inner}
      </button>
    );
  }
  return (
    <div className={`flex items-center gap-2 ${textSize}`}>
      {inner}
    </div>
  );
}

function normalizePrefix(prefix: string) {
  const trimmed = prefix.trim();
  if (!trimmed) return "";
  if (trimmed === "/") return "/";
  return trimmed.endsWith("/") ? trimmed : `${trimmed}/`;
}

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
    if (category === "untracked")
      return { label: "?", style: statusStyleMap["?"] };
    if (category === "conflicts")
      return { label: "U", style: statusStyleMap.U };
    return { label: "M", style: statusStyleMap.M };
  }

  const normalized = code.toUpperCase();
  if (normalized.includes("D")) return { label: "D", style: statusStyleMap.D };
  if (normalized.includes("M")) return { label: "M", style: statusStyleMap.M };
  if (normalized.includes("A")) return { label: "A", style: statusStyleMap.A };
  if (normalized.includes("R")) return { label: "R", style: statusStyleMap.R };
  if (normalized.includes("U")) return { label: "U", style: statusStyleMap.U };
  if (normalized.includes("?"))
    return { label: "?", style: statusStyleMap["?"] };

  return { label: "M", style: statusStyleMap.M };
}

interface FileRowProps {
  file: string;
  displayPath: string;
  badge: { label: string; style: string };
  isSelected: boolean;
  isStaged: boolean;
  canDiscard: boolean;
  isLoading: boolean;
  isDiscarding: boolean;
  isIgnoring: boolean;
  isBinary: boolean;
  isApproved: boolean;
  itemTestId: string;
  actionTestId: string;
  discardTestId: string;
  ignoreTestId: string;
  actionIcon: React.ReactNode;
  actionLabel: string;
  onSelectFile: (
    path: string,
    staged: boolean,
    event: React.MouseEvent<HTMLLIElement>,
  ) => void;
  onAction: (path: string) => void;
  onDiscard?: (path: string) => void;
  onConfirmDiscard?: (path: string | null) => void;
  onIgnore?: (path: string, level?: "project" | "group", groupDir?: string) => void;
  onConfirmIgnore?: (path: string | null) => void;
  confirmingDiscard?: string | null;
  confirmingIgnore?: string | null;
  groupingRules?: GroupingRule[];
  onOpenMobileActions?: (file: string) => void;
  onContextMenu?: (file: string, event: React.MouseEvent) => void;
  mobileSelectionMode?: boolean;
  onLongPress?: (file: string, staged: boolean) => void;
  onMobileTap?: (file: string, staged: boolean, mode: "toggle" | "range") => void;
  onViewMetrics?: (file: string, category: FileCategory) => void;
  category: FileCategory;
}

const FileRow = memo(function FileRow({
  file,
  displayPath,
  badge,
  isSelected,
  isStaged,
  canDiscard,
  isLoading,
  isDiscarding,
  isIgnoring,
  isBinary,
  isApproved,
  itemTestId,
  actionTestId,
  discardTestId,
  ignoreTestId,
  actionIcon,
  actionLabel,
  onSelectFile,
  onAction,
  onDiscard,
  onConfirmDiscard,
  onIgnore,
  onConfirmIgnore,
  confirmingDiscard,
  confirmingIgnore,
  groupingRules,
  onOpenMobileActions,
  onContextMenu,
  mobileSelectionMode,
  onLongPress,
  onMobileTap,
  onViewMetrics,
  category,
}: FileRowProps) {
  const isMobile = useContext(MobileContext);
  const isConfirmingIgnore = confirmingIgnore === file;
  const isConfirmingDiscard = confirmingDiscard === file;
  const showActionButtons = !(isConfirmingIgnore || isConfirmingDiscard) && !mobileSelectionMode;

  const longPressHandlers = useLongPress({
    onLongPress: () => {
      if (mobileSelectionMode) {
        onMobileTap?.(file, isStaged, "range");
      } else {
        onLongPress?.(file, isStaged);
      }
    },
    onTap: () => {
      if (mobileSelectionMode) {
        onMobileTap?.(file, isStaged, "toggle");
      }
      // When not in selection mode, tap is handled by onClick
    },
  });

  const touchProps = isMobile && onLongPress ? longPressHandlers : {};

  return (
    <li
      className={`group w-full flex items-center gap-2 rounded cursor-pointer transition-colors min-w-0 overflow-hidden select-none ${
        isMobile ? "px-3 py-3" : "px-2 py-1"
      } ${
        mobileSelectionMode && isSelected
          ? "bg-blue-900/30 ring-1 ring-blue-500/30 text-slate-100"
          : isSelected
            ? "bg-slate-700/50 text-slate-100"
            : "hover:bg-slate-800/50 active:bg-slate-700/50 text-slate-300"
      }`}
      data-testid={itemTestId}
      data-file-path={file}
      onClick={(event) => {
        // In mobile selection mode, taps are handled by useLongPress onTap
        if (isMobile && mobileSelectionMode) return;
        onSelectFile(file, isStaged, event);
      }}
      onContextMenu={onContextMenu ? (e) => {
        e.preventDefault();
        onContextMenu(file, e);
      } : undefined}
      {...touchProps}
    >
      {mobileSelectionMode && (
        <span className="flex-shrink-0 checkbox-appear">
          {isSelected ? (
            <CheckSquare className="h-5 w-5 text-blue-400" />
          ) : (
            <Square className="h-5 w-5 text-slate-500" />
          )}
        </span>
      )}
      <span
        className={`flex items-center justify-center rounded border font-bold ${badge.style} ${
          isMobile ? "h-7 w-7 text-xs" : "h-5 w-5 text-[10px]"
        }`}
        aria-label={`Status ${badge.label}`}
        title={`Status ${badge.label}`}
      >
        {badge.label}
      </span>
      <File
        className={`text-slate-500 flex-shrink-0 ${isMobile ? "h-4 w-4" : "h-3.5 w-3.5"}`}
      />
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

      {isConfirmingIgnore && onConfirmIgnore && onIgnore && (() => {
        const group = groupingRules ? resolveGroupForFile(file, groupingRules) : null;
        return (
          <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
            {group ? (
              <>
                <span className="text-xs text-amber-300 mr-1">Ignore in:</span>
                <button
                  className="px-1.5 py-0.5 text-xs bg-blue-600 hover:bg-blue-500 text-white rounded transition-colors"
                  onClick={() => { onIgnore(file, "project"); onConfirmIgnore(null); }}
                  disabled={isIgnoring}
                  data-testid="confirm-ignore-project"
                >
                  Project
                </button>
                <button
                  className="px-1.5 py-0.5 text-xs bg-amber-500 hover:bg-amber-400 text-slate-900 rounded transition-colors"
                  onClick={() => { onIgnore(file, "group", group.groupDir); onConfirmIgnore(null); }}
                  disabled={isIgnoring}
                  data-testid="confirm-ignore-group"
                >
                  {group.groupLabel}
                </button>
                <button
                  className="px-1.5 py-0.5 text-xs bg-slate-600 hover:bg-slate-500 text-white rounded transition-colors"
                  onClick={() => onConfirmIgnore(null)}
                  data-testid="confirm-ignore-cancel"
                >
                  Cancel
                </button>
              </>
            ) : (
              <>
                <span className="text-xs text-amber-300 mr-1">Ignore?</span>
                <button
                  className="px-1.5 py-0.5 text-xs bg-amber-500 hover:bg-amber-400 text-slate-900 rounded transition-colors"
                  onClick={() => { onIgnore(file); onConfirmIgnore(null); }}
                  disabled={isIgnoring}
                  data-testid="confirm-ignore-yes"
                >
                  Yes
                </button>
                <button
                  className="px-1.5 py-0.5 text-xs bg-slate-600 hover:bg-slate-500 text-white rounded transition-colors"
                  onClick={() => onConfirmIgnore(null)}
                  data-testid="confirm-ignore-no"
                >
                  No
                </button>
              </>
            )}
          </div>
        );
      })()}

      {isConfirmingDiscard && onConfirmDiscard && onDiscard && (
        <div
          className="flex items-center gap-1"
          onClick={(e) => e.stopPropagation()}
        >
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

      {showActionButtons && (
        <>
          {/* Desktop: hover-to-reveal actions */}
          {!isMobile && (
            <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-all">
              <button
                className="p-1 rounded hover:bg-slate-700 transition-all"
                onClick={(e) => {
                  e.stopPropagation();
                  onAction(file);
                }}
                disabled={isLoading || isIgnoring}
                title={actionLabel}
                data-testid={actionTestId}
              >
                {isLoading ? (
                  <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
                ) : (
                  actionIcon
                )}
              </button>
              {onViewMetrics && (
                <button
                  className="p-1 rounded hover:bg-slate-700 transition-all"
                  onClick={(e) => { e.stopPropagation(); onViewMetrics(file, category); }}
                  title="View file metrics"
                >
                  <BarChart3 className="h-3 w-3 text-slate-400" />
                </button>
              )}
              {onConfirmIgnore && onIgnore && (
                <button
                  className="p-1 rounded hover:bg-amber-900/40 transition-all"
                  onClick={(e) => {
                    e.stopPropagation();
                    onConfirmIgnore(file);
                  }}
                  disabled={isIgnoring}
                  title="Ignore file"
                  data-testid={ignoreTestId}
                >
                  {isIgnoring ? (
                    <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
                  ) : (
                    <EyeOff className="h-3 w-3 text-amber-300" />
                  )}
                </button>
              )}
              {canDiscard && onConfirmDiscard && (
                <button
                  className="p-1 rounded hover:bg-red-900/50 transition-all"
                  onClick={(e) => {
                    e.stopPropagation();
                    onConfirmDiscard(file);
                  }}
                  disabled={isDiscarding || isIgnoring}
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

          {/* Mobile: show primary action always, menu for secondary actions */}
          {isMobile && (
            <div className="flex items-center gap-1">
              {/* Primary action (stage/unstage) - always visible */}
              <button
                className="p-2 rounded-lg hover:bg-slate-700 active:bg-slate-600 transition-all touch-target"
                onClick={(e) => {
                  e.stopPropagation();
                  onAction(file);
                }}
                disabled={isLoading || isIgnoring}
                title={actionLabel}
                data-testid={actionTestId}
              >
                {isLoading ? (
                  <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
                ) : (
                  <span className="[&>svg]:h-5 [&>svg]:w-5">{actionIcon}</span>
                )}
              </button>

              {/* More actions button - opens bottom sheet */}
              {(onConfirmIgnore || canDiscard || onViewMetrics) && onOpenMobileActions && (
                <button
                  className="p-2 rounded-lg hover:bg-slate-700 active:bg-slate-600 transition-all touch-target"
                  onClick={(e) => {
                    e.stopPropagation();
                    onOpenMobileActions(file);
                  }}
                  title="More actions"
                  data-testid={`${itemTestId}-more-actions`}
                >
                  <MoreVertical className="h-5 w-5 text-slate-400" />
                </button>
              )}
            </div>
          )}
        </>
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
  changeStats,
  defaultExpanded = true,
  onDiscard,
  isDiscarding,
  confirmingDiscard,
  onConfirmDiscard,
  onIgnore,
  isIgnoring,
  confirmingIgnore,
  onConfirmIgnore,
  groupingRules,
  onOpenMobileActions,
  onContextMenu,
  mobileSelectionMode,
  onLongPress,
  onMobileTap,
  onStatsClick,
  onViewMetrics,
}: FileSectionProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  const isStaged = category === "staged";
  const canDiscard = category === "unstaged" || category === "untracked";
  const selectedKeys =
    selectedKeySet ?? new Set(selectedFiles?.map(selectionKey));

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
          isApproved: approvedFiles?.has(file) ?? false,
        };
      }),
    [
      files,
      fileStatuses,
      category,
      maxPathChars,
      selectionKey,
      isStaged,
      binaryFiles,
      approvedFiles,
    ],
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
        <div className="ml-auto flex items-center gap-2">
          <LineStats stats={changeStats} compact onClick={onStatsClick} />
          <span className="text-xs text-slate-600">{files.length}</span>
        </div>
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
              canDiscard={canDiscard}
              isLoading={isLoading}
              isDiscarding={isDiscarding ?? false}
              isIgnoring={isIgnoring ?? false}
              isBinary={entry.isBinary}
              isApproved={entry.isApproved}
              itemTestId={`file-item-${category}`}
              actionTestId={`file-action-${category}`}
              discardTestId={`file-discard-${category}`}
              ignoreTestId={`file-ignore-${category}`}
              actionIcon={actionIcon}
              actionLabel={actionLabel}
              onSelectFile={onSelectFile}
              onAction={onAction}
              onDiscard={onDiscard}
              onConfirmDiscard={onConfirmDiscard}
              onIgnore={onIgnore}
              onConfirmIgnore={onConfirmIgnore}
              confirmingDiscard={confirmingDiscard}
              confirmingIgnore={confirmingIgnore}
              groupingRules={groupingRules}
              onOpenMobileActions={onOpenMobileActions}
              onContextMenu={onContextMenu}
              mobileSelectionMode={mobileSelectionMode}
              onLongPress={onLongPress}
              onMobileTap={onMobileTap}
              onViewMetrics={onViewMetrics}
              category={category}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

export function FileList({
  files,
  fileStats,
  selectedFiles,
  selectedKeySet,
  selectionKey,
  approvedChanges,
  approvedPaths,
  onStageApproved,
  isStagingApproved = false,
  onSelectFile,
  onStageFile,
  onUnstageFile,
  onDiscardFile,
  onIgnoreFile,
  onStageAll,
  onUnstageAll,
  isStaging,
  isDiscarding,
  isIgnoring,
  confirmingDiscard,
  onConfirmDiscard,
  confirmingIgnore,
  onConfirmIgnore,
  collapsed = false,
  onToggleCollapse,
  fillHeight = true,
  fileViewMode = "flat",
  groupingRules = [],
  groupingAvailable = false,
  onCycleViewMode,
  onStagePaths,
  onDiscardPaths,
  onSelectAnyFile,
  scrollToFile,
  onScrollComplete,
  onDeletePath,
  onBlameFile,
  repoId,
  onOpenReview,
  mobileSelectionMode = false,
  onEnterSelectionMode,
  onExitSelectionMode,
  onMobileSelectFile,
  fileHotspots,
}: FileListProps) {
  const isMobile = useIsMobile();
  const hasStaged = (files?.staged?.length ?? 0) > 0;
  const hasUnstaged =
    (files?.unstaged?.length ?? 0) > 0 || (files?.untracked?.length ?? 0) > 0;
  const handleToggleCollapse = onToggleCollapse ?? (() => {});
  const scrollAreaRef = useRef<HTMLDivElement | null>(null);
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [maxPathChars, setMaxPathChars] = useState(72);
  const [compactHeader, setCompactHeader] = useState(false);
  const [confirmingGroup, setConfirmingGroup] = useState<string | null>(null);
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(
    new Set(),
  );
  const binarySet = useMemo(
    () => new Set(files?.binary ?? []),
    [files?.binary],
  );

  // Metrics modal state
  const [metricsModal, setMetricsModal] = useState<{
    mode: "file" | "aggregate";
    stats?: DiffStats;
    path?: string;
    filteredFileStats?: RepoFileStats;
    title?: string;
    category?: FileCategory;
  } | null>(null);
  const openAggregateMetrics = useCallback(
    () => setMetricsModal({ mode: "aggregate" }),
    [],
  );

  const openFileMetrics = useCallback(
    (path: string, category?: FileCategory) => {
      const stats = getFileStats(path, fileStats);
      if (!stats) return;
      setMetricsModal({ mode: "file", stats, path, category });
    },
    [fileStats],
  );

  const openGroupMetrics = useCallback(
    (groupFiles: Record<string, string[]>, groupLabel: string) => {
      const allPaths = [
        ...(groupFiles.conflicts ?? []),
        ...(groupFiles.staged ?? []),
        ...(groupFiles.unstaged ?? []),
        ...(groupFiles.untracked ?? []),
      ];
      const filtered = filterFileStats(allPaths, fileStats);
      setMetricsModal({
        mode: "aggregate",
        filteredFileStats: filtered,
        title: `${groupLabel} — Change Metrics`,
      });
    },
    [fileStats],
  );

  const openGroupCategoryMetrics = useCallback(
    (paths: string[], category: "staged" | "unstaged" | "untracked", groupLabel: string) => {
      const catLabel = category.charAt(0).toUpperCase() + category.slice(1);
      const filtered = filterCategoryStats(paths, category, fileStats);
      setMetricsModal({
        mode: "aggregate",
        filteredFileStats: filtered,
        title: `${catLabel} (${groupLabel})`,
      });
    },
    [fileStats],
  );

  // Fetch enhanced stats (hunks, density, comments) on-demand when file metrics modal is open
  const enhancedQuery = useDiffStats(
    metricsModal?.path,
    metricsModal?.category === "staged",
    metricsModal?.category === "untracked",
    metricsModal !== null && metricsModal.mode === "file",
    repoId,
  );

  // Mobile file actions state
  const [mobileActionFile, setMobileActionFile] = useState<string | null>(null);
  const mobileActionFileInfo = useMemo(() => {
    if (!mobileActionFile) return null;
    const isStaged = files?.staged?.includes(mobileActionFile) ?? false;
    const isUnstaged = files?.unstaged?.includes(mobileActionFile) ?? false;
    const isUntracked = files?.untracked?.includes(mobileActionFile) ?? false;
    const isConflict = files?.conflicts?.includes(mobileActionFile) ?? false;
    return {
      path: mobileActionFile,
      isStaged,
      isUnstaged,
      isUntracked,
      isConflict,
    };
  }, [mobileActionFile, files]);

  // Context menu state for right-click blame action
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    file: string;
  } | null>(null);

  const handleFileContextMenu = useCallback((file: string, event: React.MouseEvent) => {
    setContextMenu({
      x: event.clientX,
      y: event.clientY,
      file,
    });
  }, []);

  const handleCloseContextMenu = useCallback(() => {
    setContextMenu(null);
  }, []);

  const contextMenuItems = useMemo<ContextMenuItem[]>(() => {
    if (!contextMenu || !onBlameFile) return [];
    return [
      {
        label: "View File History",
        icon: <History className="h-4 w-4" />,
        onClick: () => onBlameFile(contextMenu.file),
      },
    ];
  }, [contextMenu, onBlameFile]);

  // Mobile long-press enters selection mode; tap in selection mode toggles
  const handleLongPress = useCallback(
    (file: string, staged: boolean) => {
      if (mobileSelectionMode) return;
      onEnterSelectionMode?.(file, staged);
    },
    [mobileSelectionMode, onEnterSelectionMode],
  );

  const handleMobileTap = useCallback(
    (file: string, staged: boolean, mode: "toggle" | "range") => {
      onMobileSelectFile?.(file, staged, mode);
    },
    [onMobileSelectFile],
  );

  // Auto-exit selection mode when all files deselected
  useEffect(() => {
    if (mobileSelectionMode && (selectedFiles?.length ?? 0) === 0) {
      onExitSelectionMode?.();
    }
  }, [mobileSelectionMode, selectedFiles?.length, onExitSelectionMode]);

  // Suppress bottom sheet in selection mode
  const handleOpenMobileActions = useCallback(
    (file: string) => {
      if (mobileSelectionMode) return;
      setMobileActionFile(file);
    },
    [mobileSelectionMode],
  );

  // Selection toolbar: compute action availability
  const selectionHasUnstaged = useMemo(() => {
    if (!mobileSelectionMode || !selectedFiles) return false;
    return selectedFiles.some((entry) => !entry.staged);
  }, [mobileSelectionMode, selectedFiles]);
  const selectionHasStaged = useMemo(() => {
    if (!mobileSelectionMode || !selectedFiles) return false;
    return selectedFiles.some((entry) => entry.staged);
  }, [mobileSelectionMode, selectedFiles]);
  const selectionHasDiscardable = useMemo(() => {
    if (!mobileSelectionMode || !selectedFiles || !files) return false;
    const unstaged = new Set(files.unstaged ?? []);
    const untracked = new Set(files.untracked ?? []);
    return selectedFiles.some((entry) => !entry.staged && (unstaged.has(entry.path) || untracked.has(entry.path)));
  }, [mobileSelectionMode, selectedFiles, files]);

  const normalizedRules = useMemo(
    () =>
      groupingRules
        .map((rule) => {
          const rawPrefixes = Array.isArray(rule.prefixes)
            ? rule.prefixes
            : typeof rule.prefix === "string"
              ? [rule.prefix]
              : [];
          const normalizedPrefixes = rawPrefixes
            .map((prefix) => normalizePrefix(prefix))
            .filter((prefix) => prefix);
          if (normalizedPrefixes.length === 0) return null;
          const fallbackLabel =
            rawPrefixes.find((prefix) => prefix.trim()) ?? "";
          return {
            ...rule,
            mode: rule.mode ?? "prefix",
            normalizedPrefixes,
            label: rule.label.trim() || fallbackLabel.trim(),
          };
        })
        .filter((rule): rule is NonNullable<typeof rule> => Boolean(rule)),
    [groupingRules],
  );
  // Use local normalizedRules check for actual grouping computation
  const hasValidRules = normalizedRules.length > 0;
  const groupingActive = fileViewMode === "grouped" && hasValidRules;
  const totalStats = useMemo(() => {
    if (!files) return undefined;
    const stagedStats = summarizeFileStats(
      files.staged ?? [],
      fileStats?.staged,
    );
    const unstagedStats = summarizeFileStats(
      files.unstaged ?? [],
      fileStats?.unstaged,
    );
    const untrackedStats = summarizeFileStats(
      files.untracked ?? [],
      fileStats?.untracked,
    );
    const summary = { additions: 0, deletions: 0, files: 0 };
    const sources = [stagedStats, unstagedStats, untrackedStats];
    let hasStats = false;
    sources.forEach((source) => {
      if (!source) return;
      hasStats = true;
      summary.additions += source.additions;
      summary.deletions += source.deletions;
      summary.files += source.files;
    });
    return hasStats ? summary : undefined;
  }, [files, fileStats]);
  const handleDiscardUnstaged = useCallback(
    (path: string) => onDiscardFile(path, false),
    [onDiscardFile],
  );
  const showApprovedBanner = Boolean(
    approvedChanges?.available && (approvedChanges.committableFiles ?? 0) > 0,
  );
  const handleDiscardUntracked = useCallback(
    (path: string) => onDiscardFile(path, true),
    [onDiscardFile],
  );
  const handleIgnoreFile = useCallback(
    (path: string, level?: "project" | "group", groupDir?: string) => onIgnoreFile(path, level, groupDir),
    [onIgnoreFile],
  );
  const toggleGroupCollapse = useCallback((groupId: string) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) {
        next.delete(groupId);
      } else {
        next.add(groupId);
      }
      return next;
    });
  }, []);

  useEffect(() => {
    if (!scrollAreaRef.current || typeof ResizeObserver === "undefined") return;

    const update = () => {
      const width = scrollAreaRef.current?.clientWidth ?? 0;
      // Account for: status badge (~28px), file icon (~22px), action buttons (~80px),
      // badges (binary/approved ~100px), padding (~16px) = ~250px total
      const usable = Math.max(0, width - 180);
      const nextMax = Math.max(12, Math.min(100, Math.floor(usable / 7.5)));
      setMaxPathChars(nextMax);
    };

    // Defer initial measurement to ensure layout is complete after expanding
    const rafId = requestAnimationFrame(update);
    const observer = new ResizeObserver(update);
    observer.observe(scrollAreaRef.current);
    return () => {
      cancelAnimationFrame(rafId);
      observer.disconnect();
    };
    // Re-run when collapsed changes to re-observe the new ScrollArea element
  }, [collapsed]);

  // Track card width to swap +/- stats for a compact icon in the header
  useEffect(() => {
    if (!cardRef.current || typeof ResizeObserver === "undefined") return;
    const update = () => {
      const width = cardRef.current?.clientWidth ?? 0;
      setCompactHeader(width < 500);
    };
    update();
    const observer = new ResizeObserver(update);
    observer.observe(cardRef.current);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    if (!groupingActive) {
      setConfirmingGroup(null);
    }
  }, [groupingActive]);

  // Scroll to file when returning from Related Files panel
  useEffect(() => {
    if (!scrollToFile || fileViewMode === "tree") return; // Tree view handles its own scrolling

    const timeoutId = setTimeout(() => {
      const element = document.querySelector(`[data-file-path="${CSS.escape(scrollToFile)}"]`);
      if (element) {
        element.scrollIntoView({ behavior: "smooth", block: "center" });
      }
      onScrollComplete?.();
    }, 100);

    return () => clearTimeout(timeoutId);
  }, [scrollToFile, onScrollComplete, fileViewMode]);

  const groupedSections = useMemo(() => {
    if (!groupingActive) return [];
    const groupMap = new Map<
      string,
      {
        id: string;
        label: string;
        displayPrefixes: Set<string>;
        files: Record<FileCategory, string[]>;
      }
    >();
    const groupOrder: string[] = [];
    const ensureGroup = (id: string, label: string, displayPrefix: string) => {
      const existing = groupMap.get(id);
      const group = existing ?? {
        id,
        label,
        displayPrefixes: new Set<string>(),
        files: {
          conflicts: [],
          staged: [],
          unstaged: [],
          untracked: [],
        },
      };
      if (!existing) {
        groupMap.set(id, group);
        groupOrder.push(id);
      }
      if (displayPrefix) {
        group.displayPrefixes.add(displayPrefix);
      }
      return group;
    };
    const otherGroup = {
      id: "other",
      label: "Other",
      displayPrefixes: new Set<string>(),
      files: {
        conflicts: [] as string[],
        staged: [] as string[],
        unstaged: [] as string[],
        untracked: [] as string[],
      },
    };

    const addFile = (file: string, category: FileCategory) => {
      for (const rule of normalizedRules) {
        for (const normalizedPrefix of rule.normalizedPrefixes) {
          if (!file.startsWith(normalizedPrefix)) continue;
          if (rule.mode === "segment") {
            const rest = file.slice(normalizedPrefix.length);
            const segment = rest.split("/")[0];
            const segmentLabel = segment || rule.label;
            const segmentPrefix = segment
              ? `${normalizedPrefix}${segment}/`
              : normalizedPrefix;
            const groupId = segment ? `${rule.id}:${segment}` : rule.id;
            const group = ensureGroup(groupId, segmentLabel, segmentPrefix);
            group.files[category].push(file);
          } else {
            const group = ensureGroup(rule.id, rule.label, normalizedPrefix);
            group.files[category].push(file);
          }
          return;
        }
      }
      otherGroup.files[category].push(file);
    };

    (files?.conflicts ?? []).forEach((file) => addFile(file, "conflicts"));
    (files?.staged ?? []).forEach((file) => addFile(file, "staged"));
    (files?.unstaged ?? []).forEach((file) => addFile(file, "unstaged"));
    (files?.untracked ?? []).forEach((file) => addFile(file, "untracked"));

    const filledGroups = groupOrder
      .map((id) => groupMap.get(id))
      .filter((group): group is NonNullable<typeof group> => {
        if (!group) return false;
        return (
          group.files.conflicts.length +
            group.files.staged.length +
            group.files.unstaged.length +
            group.files.untracked.length >
          0
        );
      });
    const hasOther =
      otherGroup.files.conflicts.length +
        otherGroup.files.staged.length +
        otherGroup.files.unstaged.length +
        otherGroup.files.untracked.length >
      0;

    const formattedGroups = filledGroups.map((group) => {
      const prefixes = Array.from(group.displayPrefixes).filter(
        (prefix) => prefix,
      );
      let displayPrefix = "";
      if (prefixes.length === 1) {
        const [firstPrefix] = prefixes;
        displayPrefix = firstPrefix ?? "";
      } else if (prefixes.length > 1) {
        displayPrefix = `${prefixes.length} prefixes`;
      }
      return { ...group, displayPrefix };
    });

    if (!hasOther) return formattedGroups;
    return [
      ...formattedGroups,
      {
        ...otherGroup,
        displayPrefix: "",
      },
    ];
  }, [files, groupingActive, normalizedRules]);

  const handleCycleViewMode = onCycleViewMode ?? (() => {});
  const totalFilesCount =
    (files?.conflicts?.length ?? 0) +
    (files?.staged?.length ?? 0) +
    (files?.unstaged?.length ?? 0) +
    (files?.untracked?.length ?? 0);

  return (
    <MobileContext.Provider value={isMobile}>
      <Card
        ref={cardRef}
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
            {compactHeader ? (
              <button
                type="button"
                className="p-1 rounded hover:bg-slate-800/70 transition-colors"
                onClick={(e) => {
                  e.stopPropagation();
                  setMetricsModal({ mode: "aggregate" });
                }}
                aria-label="View change metrics"
              >
                <BarChart3 className="h-3.5 w-3.5 text-slate-400" />
              </button>
            ) : (
              <LineStats
                stats={totalStats}
                onClick={() => setMetricsModal({ mode: "aggregate" })}
              />
            )}
          </CardTitle>
          <div className="flex flex-wrap gap-2 justify-end min-w-0">
            {hasUnstaged && (
              <Button
                variant="outline"
                size="sm"
                onClick={onStageAll}
                disabled={isStaging}
                className={compactHeader ? "h-7 w-7 p-0" : "min-w-0 whitespace-normal px-3"}
                data-testid="stage-all-button"
                title="Stage All"
              >
                <Plus className="h-3 w-3" />
                {!compactHeader && <span className="ml-1">Stage All</span>}
              </Button>
            )}
            {hasStaged && (
              <Button
                variant="outline"
                size="sm"
                onClick={onUnstageAll}
                disabled={isStaging}
                className={compactHeader ? "h-7 w-7 p-0" : "min-w-0 whitespace-normal px-3"}
                data-testid="unstage-all-button"
                title="Unstage All"
              >
                <Minus className="h-3 w-3" />
                {!compactHeader && <span className="ml-1">Unstage All</span>}
              </Button>
            )}
            <ViewModeCycleButton
              mode={fileViewMode}
              onCycle={handleCycleViewMode}
              groupingAvailable={groupingAvailable}
              compact={compactHeader}
            />
          </div>
        </CardHeader>

        {!collapsed && (
          <CardContent className="flex-1 min-w-0 p-0 overflow-hidden">
            {/* Mobile multi-select toolbar */}
            {isMobile && mobileSelectionMode && (selectedFiles?.length ?? 0) > 0 && (
              <div className="sticky top-0 z-10 flex items-center justify-between gap-2 px-3 py-2 bg-slate-900/95 backdrop-blur-sm border-b border-slate-700">
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    className="p-1 rounded hover:bg-slate-700 transition-colors"
                    onClick={onExitSelectionMode}
                    aria-label="Exit selection mode"
                  >
                    <X className="h-4 w-4 text-slate-400" />
                  </button>
                  <span className="text-sm text-slate-200" aria-live="polite">
                    {selectedFiles?.length ?? 0} file{(selectedFiles?.length ?? 0) !== 1 ? "s" : ""} selected
                  </span>
                </div>
                <div className="flex items-center gap-1">
                  {selectionHasUnstaged && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 px-2 border-emerald-500/40 text-emerald-200 hover:bg-emerald-900/20"
                      onClick={() => {
                        const unstaged = selectedFiles?.filter((e) => !e.staged).map((e) => e.path) ?? [];
                        if (unstaged.length > 0) onStagePaths?.(unstaged);
                      }}
                      disabled={isStaging}
                    >
                      Stage
                    </Button>
                  )}
                  {selectionHasStaged && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 px-2"
                      onClick={() => {
                        const staged = selectedFiles?.filter((e) => e.staged).map((e) => e.path) ?? [];
                        staged.forEach((p) => onUnstageFile(p));
                      }}
                      disabled={isStaging}
                    >
                      Unstage
                    </Button>
                  )}
                  {selectionHasDiscardable && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 px-2 border-red-400/40 text-red-200 hover:bg-red-900/20"
                      onClick={() => {
                        const discardable = selectedFiles?.filter((e) => !e.staged) ?? [];
                        const tracked = discardable
                          .filter((e) => !(files?.untracked ?? []).includes(e.path))
                          .map((e) => e.path);
                        const untracked = discardable
                          .filter((e) => (files?.untracked ?? []).includes(e.path))
                          .map((e) => e.path);
                        if (tracked.length > 0) onDiscardPaths?.(tracked, false);
                        if (untracked.length > 0) onDiscardPaths?.(untracked, true);
                      }}
                      disabled={isDiscarding}
                    >
                      Discard
                    </Button>
                  )}
                </div>
              </div>
            )}
            {showApprovedBanner && (
              <div className="mx-2 mt-2 mb-1 rounded-md border border-emerald-800/50 bg-emerald-950/20 p-2 text-xs text-emerald-200">
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <ShieldCheck className="h-3.5 w-3.5 text-emerald-300" />
                    <span>Approved changes ready</span>
                    <span className="text-emerald-300">
                      {approvedChanges?.committableFiles ?? 0} file
                      {(approvedChanges?.committableFiles ?? 0) !== 1
                        ? "s"
                        : ""}
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
            <ScrollArea
              className="h-full min-w-0 px-2 pt-2 select-none"
              ref={scrollAreaRef}
            >
              <div style={{ paddingBottom: 72 }}>
                {fileViewMode === "tree" ? (
                  <ProjectTreeView
                    onSelectFile={(path) => {
                      // Use onSelectAnyFile if provided, otherwise simulate file selection
                      if (onSelectAnyFile) {
                        onSelectAnyFile(path);
                      } else {
                        // Find if file is in changes list
                        const isStaged = files?.staged?.includes(path) ?? false;
                        onSelectFile(path, isStaged, { metaKey: false, ctrlKey: false, shiftKey: false } as React.MouseEvent<HTMLLIElement>);
                      }
                    }}
                    selectedFile={selectedFiles?.[0]?.path}
                    gitStatuses={files?.statuses}
                    scrollToFile={scrollToFile}
                    onScrollComplete={onScrollComplete}
                    onDeletePath={onDeletePath}
                    onBlameFile={onBlameFile}
                    repoId={repoId}
                  />
                ) : groupingActive ? (
                  groupedSections.map((group) => {
                    const stageable = [
                      ...group.files.unstaged,
                      ...group.files.untracked,
                      ...group.files.conflicts,
                    ];
                    const discardTracked = group.files.unstaged;
                    const discardUntracked = group.files.untracked;
                    const discardCount =
                      discardTracked.length + discardUntracked.length;
                    const groupCount =
                      group.files.conflicts.length +
                      group.files.staged.length +
                      group.files.unstaged.length +
                      group.files.untracked.length;
                    const isGroupCollapsed = collapsedGroups.has(group.id);

                    return (
                      <div
                        key={group.id}
                        className="mb-4 rounded-lg border border-slate-800/80 bg-slate-950/40"
                        data-testid={`file-group-${group.id}`}
                      >
                        <div
                          className={`flex flex-wrap items-center justify-between gap-2 px-3 py-2 ${isGroupCollapsed ? "" : "border-b border-slate-800/70"}`}
                        >
                          <button
                            type="button"
                            className="flex items-center gap-2 min-w-0 hover:bg-slate-800/30 rounded px-1 -ml-1 transition-colors"
                            onClick={() => toggleGroupCollapse(group.id)}
                            data-testid={`file-group-toggle-${group.id}`}
                          >
                            {isGroupCollapsed ? (
                              <ChevronRight className="h-3.5 w-3.5 text-slate-500 flex-shrink-0" />
                            ) : (
                              <ChevronDown className="h-3.5 w-3.5 text-slate-500 flex-shrink-0" />
                            )}
                            <div className="min-w-0 text-left">
                              <div className="text-xs font-semibold uppercase tracking-wider text-slate-300">
                                {group.label}
                              </div>
                              {group.displayPrefix && (
                                <div className="text-[11px] text-slate-500">
                                  {group.displayPrefix}
                                </div>
                              )}
                            </div>
                          </button>
                          <div className="flex items-center gap-2 text-xs text-slate-500">
                            {onOpenReview && group.displayPrefix && /^(scenarios|apps|services)\//.test(group.displayPrefix) ? (
                              <button
                                type="button"
                                className="h-7 px-2 inline-flex items-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 transition-colors"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  onOpenReview(group.displayPrefix?.split("/")[1] ?? "");
                                }}
                                title="Open scenario review"
                              >
                                {groupCount !== undefined && <span>{groupCount} files</span>}
                                <ClipboardCheck className="h-3.5 w-3.5" />
                              </button>
                            ) : (
                              <button
                                type="button"
                                className="hover:underline decoration-slate-600 cursor-pointer"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  openGroupMetrics(group.files, group.label);
                                }}
                              >
                                {groupCount} files
                              </button>
                            )}
                            {!isGroupCollapsed &&
                              stageable.length > 0 &&
                              onStagePaths && (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => onStagePaths(stageable)}
                                  disabled={isStaging}
                                  className={compactHeader ? "h-7 w-7 p-0" : "h-7 px-2"}
                                  title="Stage All"
                                >
                                  {compactHeader ? <Plus className="h-3 w-3" /> : "Stage All"}
                                </Button>
                              )}
                            {!isGroupCollapsed &&
                              discardCount > 0 &&
                              onDiscardPaths && (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => setConfirmingGroup(group.id)}
                                  disabled={isDiscarding}
                                  className={`border-red-400/40 text-red-200 hover:bg-red-900/20 ${compactHeader ? "h-7 w-7 p-0" : "h-7 px-2"}`}
                                  title="Discard All"
                                >
                                  {compactHeader ? <Trash2 className="h-3 w-3" /> : "Discard All"}
                                </Button>
                              )}
                          </div>
                        </div>
                        {!isGroupCollapsed && (
                          <>
                            {confirmingGroup === group.id &&
                              discardCount > 0 && (
                                <div className="flex items-center justify-between gap-2 px-3 py-2 text-xs text-red-200 bg-red-950/30 border-b border-red-900/40">
                                  <span>
                                    Discard {discardCount} changes in this
                                    group?
                                  </span>
                                  <div className="flex items-center gap-2">
                                    <button
                                      type="button"
                                      className="px-2 py-1 rounded border border-red-400/40 text-red-100 hover:bg-red-900/30"
                                      onClick={() => {
                                        if (discardTracked.length > 0) {
                                          onDiscardPaths?.(
                                            discardTracked,
                                            false,
                                          );
                                        }
                                        if (discardUntracked.length > 0) {
                                          onDiscardPaths?.(
                                            discardUntracked,
                                            true,
                                          );
                                        }
                                        setConfirmingGroup(null);
                                      }}
                                    >
                                      Discard
                                    </button>
                                    <button
                                      type="button"
                                      className="px-2 py-1 rounded border border-slate-600 text-slate-200 hover:bg-slate-800/50"
                                      onClick={() => setConfirmingGroup(null)}
                                    >
                                      Cancel
                                    </button>
                                  </div>
                                </div>
                              )}
                            <div className="px-2 py-2">
                              <FileSection
                                key={`${group.id}-conflicts`}
                                title="Conflicts"
                                category="conflicts"
                                files={group.files.conflicts}
                                fileStatuses={files?.statuses}
                                binaryFiles={binarySet}
                                approvedFiles={approvedPaths}
                                maxPathChars={maxPathChars}
                                icon={
                                  <AlertTriangle className="h-3.5 w-3.5 text-red-500" />
                                }
                                selectedFiles={selectedFiles}
                                selectedKeySet={selectedKeySet}
                                selectionKey={selectionKey}
                                onSelectFile={onSelectFile}
                                onAction={onStageFile}
                                actionIcon={
                                  <Plus className="h-3 w-3 text-slate-400" />
                                }
                                actionLabel="Stage file"
                                isLoading={isStaging}
                                changeStats={summarizeFileStats(
                                  group.files.conflicts,
                                  fileStats?.unstaged,
                                )}
                                onIgnore={handleIgnoreFile}
                                isIgnoring={isIgnoring}
                                confirmingIgnore={confirmingIgnore}
                                onConfirmIgnore={onConfirmIgnore}
                                groupingRules={groupingRules}
                                onOpenMobileActions={handleOpenMobileActions}
                                onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                                mobileSelectionMode={mobileSelectionMode}
                                onLongPress={handleLongPress}
                                onMobileTap={handleMobileTap}
                                onStatsClick={() => openGroupCategoryMetrics(group.files.conflicts, "unstaged", group.label)}
                                onViewMetrics={openFileMetrics}
                              />
                              <FileSection
                                key={`${group.id}-staged`}
                                title="Staged"
                                category="staged"
                                files={group.files.staged}
                                fileStatuses={files?.statuses}
                                binaryFiles={binarySet}
                                approvedFiles={approvedPaths}
                                maxPathChars={maxPathChars}
                                icon={
                                  <FilePlus className="h-3.5 w-3.5 text-emerald-500" />
                                }
                                selectedFiles={selectedFiles}
                                selectedKeySet={selectedKeySet}
                                selectionKey={selectionKey}
                                onSelectFile={onSelectFile}
                                onAction={onUnstageFile}
                                actionIcon={
                                  <Minus className="h-3 w-3 text-slate-400" />
                                }
                                actionLabel="Unstage file"
                                isLoading={isStaging}
                                changeStats={summarizeFileStats(
                                  group.files.staged,
                                  fileStats?.staged,
                                )}
                                onIgnore={handleIgnoreFile}
                                isIgnoring={isIgnoring}
                                confirmingIgnore={confirmingIgnore}
                                onConfirmIgnore={onConfirmIgnore}
                                groupingRules={groupingRules}
                                onOpenMobileActions={handleOpenMobileActions}
                                onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                                mobileSelectionMode={mobileSelectionMode}
                                onLongPress={handleLongPress}
                                onMobileTap={handleMobileTap}
                                onStatsClick={() => openGroupCategoryMetrics(group.files.staged, "staged", group.label)}
                                onViewMetrics={openFileMetrics}
                              />
                              <FileSection
                                key={`${group.id}-unstaged`}
                                title="Modified"
                                category="unstaged"
                                files={group.files.unstaged}
                                fileStatuses={files?.statuses}
                                binaryFiles={binarySet}
                                approvedFiles={approvedPaths}
                                maxPathChars={maxPathChars}
                                icon={
                                  <FileX className="h-3.5 w-3.5 text-amber-500" />
                                }
                                selectedFiles={selectedFiles}
                                selectedKeySet={selectedKeySet}
                                selectionKey={selectionKey}
                                onSelectFile={onSelectFile}
                                onAction={onStageFile}
                                actionIcon={
                                  <Plus className="h-3 w-3 text-slate-400" />
                                }
                                actionLabel="Stage file"
                                isLoading={isStaging}
                                changeStats={summarizeFileStats(
                                  group.files.unstaged,
                                  fileStats?.unstaged,
                                )}
                                onDiscard={handleDiscardUnstaged}
                                isDiscarding={isDiscarding}
                                confirmingDiscard={confirmingDiscard}
                                onConfirmDiscard={onConfirmDiscard}
                                onIgnore={handleIgnoreFile}
                                isIgnoring={isIgnoring}
                                confirmingIgnore={confirmingIgnore}
                                onConfirmIgnore={onConfirmIgnore}
                                groupingRules={groupingRules}
                                onOpenMobileActions={handleOpenMobileActions}
                                onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                                mobileSelectionMode={mobileSelectionMode}
                                onLongPress={handleLongPress}
                                onMobileTap={handleMobileTap}
                                onStatsClick={() => openGroupCategoryMetrics(group.files.unstaged, "unstaged", group.label)}
                                onViewMetrics={openFileMetrics}
                              />
                              <FileSection
                                key={`${group.id}-untracked`}
                                title="Untracked"
                                category="untracked"
                                files={group.files.untracked}
                                fileStatuses={files?.statuses}
                                binaryFiles={binarySet}
                                approvedFiles={approvedPaths}
                                maxPathChars={maxPathChars}
                                icon={
                                  <File className="h-3.5 w-3.5 text-slate-500" />
                                }
                                selectedFiles={selectedFiles}
                                selectedKeySet={selectedKeySet}
                                selectionKey={selectionKey}
                                onSelectFile={onSelectFile}
                                onAction={onStageFile}
                                actionIcon={
                                  <Plus className="h-3 w-3 text-slate-400" />
                                }
                                actionLabel="Stage file"
                                isLoading={isStaging}
                                changeStats={summarizeFileStats(
                                  group.files.untracked,
                                  fileStats?.untracked,
                                )}
                                defaultExpanded={false}
                                onDiscard={handleDiscardUntracked}
                                isDiscarding={isDiscarding}
                                confirmingDiscard={confirmingDiscard}
                                onConfirmDiscard={onConfirmDiscard}
                                onIgnore={handleIgnoreFile}
                                isIgnoring={isIgnoring}
                                confirmingIgnore={confirmingIgnore}
                                onConfirmIgnore={onConfirmIgnore}
                                groupingRules={groupingRules}
                                onOpenMobileActions={handleOpenMobileActions}
                                onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                                mobileSelectionMode={mobileSelectionMode}
                                onLongPress={handleLongPress}
                                onMobileTap={handleMobileTap}
                                onStatsClick={() => openGroupCategoryMetrics(group.files.untracked, "untracked", group.label)}
                                onViewMetrics={openFileMetrics}
                              />
                            </div>
                          </>
                        )}
                      </div>
                    );
                  })
                ) : (
                  <>
                    {/* Conflicts - Always show first if any */}
                    <FileSection
                      title="Conflicts"
                      category="conflicts"
                      files={files?.conflicts ?? []}
                      fileStatuses={files?.statuses}
                      binaryFiles={binarySet}
                      approvedFiles={approvedPaths}
                      maxPathChars={maxPathChars}
                      icon={
                        <AlertTriangle className="h-3.5 w-3.5 text-red-500" />
                      }
                      selectedFiles={selectedFiles}
                      selectedKeySet={selectedKeySet}
                      selectionKey={selectionKey}
                      onSelectFile={onSelectFile}
                      onAction={onStageFile}
                      actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
                      actionLabel="Stage file"
                      isLoading={isStaging}
                      changeStats={summarizeFileStats(
                        files?.conflicts ?? [],
                        fileStats?.unstaged,
                      )}
                      onIgnore={handleIgnoreFile}
                      isIgnoring={isIgnoring}
                      confirmingIgnore={confirmingIgnore}
                      onConfirmIgnore={onConfirmIgnore}
                      groupingRules={groupingRules}
                      onOpenMobileActions={handleOpenMobileActions}
                      onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onStatsClick={openAggregateMetrics}
                      onViewMetrics={openFileMetrics}
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
                      icon={
                        <FilePlus className="h-3.5 w-3.5 text-emerald-500" />
                      }
                      selectedFiles={selectedFiles}
                      selectedKeySet={selectedKeySet}
                      selectionKey={selectionKey}
                      onSelectFile={onSelectFile}
                      onAction={onUnstageFile}
                      actionIcon={<Minus className="h-3 w-3 text-slate-400" />}
                      actionLabel="Unstage file"
                      isLoading={isStaging}
                      changeStats={summarizeFileStats(
                        files?.staged ?? [],
                        fileStats?.staged,
                      )}
                      onIgnore={handleIgnoreFile}
                      isIgnoring={isIgnoring}
                      confirmingIgnore={confirmingIgnore}
                      onConfirmIgnore={onConfirmIgnore}
                      groupingRules={groupingRules}
                      onOpenMobileActions={handleOpenMobileActions}
                      onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onStatsClick={openAggregateMetrics}
                      onViewMetrics={openFileMetrics}
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
                      changeStats={summarizeFileStats(
                        files?.unstaged ?? [],
                        fileStats?.unstaged,
                      )}
                      onDiscard={handleDiscardUnstaged}
                      isDiscarding={isDiscarding}
                      confirmingDiscard={confirmingDiscard}
                      onConfirmDiscard={onConfirmDiscard}
                      onIgnore={handleIgnoreFile}
                      isIgnoring={isIgnoring}
                      confirmingIgnore={confirmingIgnore}
                      onConfirmIgnore={onConfirmIgnore}
                      groupingRules={groupingRules}
                      onOpenMobileActions={handleOpenMobileActions}
                      onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onStatsClick={openAggregateMetrics}
                      onViewMetrics={openFileMetrics}
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
                      changeStats={summarizeFileStats(
                        files?.untracked ?? [],
                        fileStats?.untracked,
                      )}
                      defaultExpanded={false}
                      onDiscard={handleDiscardUntracked}
                      isDiscarding={isDiscarding}
                      confirmingDiscard={confirmingDiscard}
                      onConfirmDiscard={onConfirmDiscard}
                      onIgnore={handleIgnoreFile}
                      isIgnoring={isIgnoring}
                      confirmingIgnore={confirmingIgnore}
                      onConfirmIgnore={onConfirmIgnore}
                      groupingRules={groupingRules}
                      onOpenMobileActions={handleOpenMobileActions}
                      onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onStatsClick={openAggregateMetrics}
                      onViewMetrics={openFileMetrics}
                    />
                  </>
                )}

                {/* Empty State - only show for flat/grouped views */}
                {fileViewMode !== "tree" && files && totalFilesCount === 0 && (
                  <div
                    className="flex flex-col items-center justify-center py-12 text-center"
                    data-testid="empty-state"
                  >
                    <File className="h-8 w-8 text-slate-700 mb-3" />
                    <p className="text-sm text-slate-500">
                      No changes detected
                    </p>
                    <p className="text-xs text-slate-600 mt-1">
                      Working directory is clean
                    </p>
                  </div>
                )}
              </div>
            </ScrollArea>
          </CardContent>
        )}
      </Card>

      {/* Mobile file action bottom sheet (suppressed in selection mode) */}
      {isMobile && !mobileSelectionMode && mobileActionFileInfo && (
        <BottomSheet
          isOpen={Boolean(mobileActionFile)}
          onClose={() => setMobileActionFile(null)}
          title={
            mobileActionFileInfo.path.split("/").pop() ||
            mobileActionFileInfo.path
          }
        >
          <div className="space-y-1">
            {/* View metrics */}
            <BottomSheetAction
              icon={<BarChart3 className="h-5 w-5 text-slate-300" />}
              label="View Metrics"
              description="View change metrics for this file"
              onClick={() => {
                const cat: FileCategory = mobileActionFileInfo.isStaged ? "staged"
                  : mobileActionFileInfo.isUntracked ? "untracked"
                  : "unstaged";
                openFileMetrics(mobileActionFileInfo.path, cat);
                setMobileActionFile(null);
              }}
            />

            {/* Stage/Unstage action */}
            {mobileActionFileInfo.isStaged && (
              <BottomSheetAction
                icon={<Minus className="h-5 w-5 text-slate-300" />}
                label="Unstage"
                description="Remove from staged changes"
                onClick={() => {
                  onUnstageFile(mobileActionFileInfo.path);
                  setMobileActionFile(null);
                }}
              />
            )}
            {(mobileActionFileInfo.isUnstaged ||
              mobileActionFileInfo.isUntracked ||
              mobileActionFileInfo.isConflict) && (
              <BottomSheetAction
                icon={<Plus className="h-5 w-5 text-emerald-300" />}
                label="Stage"
                description="Add to staged changes"
                onClick={() => {
                  onStageFile(mobileActionFileInfo.path);
                  setMobileActionFile(null);
                }}
              />
            )}

            {/* Ignore action */}
            {(() => {
              const group = groupingRules ? resolveGroupForFile(mobileActionFileInfo.path, groupingRules) : null;
              if (group) {
                return (
                  <>
                    <BottomSheetAction
                      icon={<EyeOff className="h-5 w-5 text-blue-400" />}
                      label="Ignore (Project)"
                      description="Add to root .gitignore"
                      onClick={() => {
                        onIgnoreFile(mobileActionFileInfo.path, "project");
                        setMobileActionFile(null);
                      }}
                    />
                    <BottomSheetAction
                      icon={<EyeOff className="h-5 w-5 text-amber-300" />}
                      label={`Ignore (${group.groupLabel})`}
                      description={`Add to ${group.groupDir}.gitignore`}
                      onClick={() => {
                        onIgnoreFile(mobileActionFileInfo.path, "group", group.groupDir);
                        setMobileActionFile(null);
                      }}
                    />
                  </>
                );
              }
              return (
                <BottomSheetAction
                  icon={<EyeOff className="h-5 w-5 text-amber-300" />}
                  label="Ignore"
                  description="Add to .gitignore"
                  onClick={() => {
                    onIgnoreFile(mobileActionFileInfo.path);
                    setMobileActionFile(null);
                  }}
                />
              );
            })()}

            {/* Discard action - only for unstaged/untracked */}
            {(mobileActionFileInfo.isUnstaged ||
              mobileActionFileInfo.isUntracked) && (
              <BottomSheetAction
                icon={<Trash2 className="h-5 w-5 text-red-400" />}
                label="Discard Changes"
                description={
                  mobileActionFileInfo.isUntracked
                    ? "Delete this file"
                    : "Revert to last commit"
                }
                variant="danger"
                onClick={() => {
                  onDiscardFile(
                    mobileActionFileInfo.path,
                    mobileActionFileInfo.isUntracked,
                  );
                  setMobileActionFile(null);
                }}
              />
            )}
          </div>
        </BottomSheet>
      )}

      {/* Context menu for right-click blame action */}
      <ContextMenu
        isOpen={Boolean(contextMenu)}
        position={contextMenu ?? { x: 0, y: 0 }}
        items={contextMenuItems}
        onClose={handleCloseContextMenu}
      />

      {/* Change metrics modal */}
      <ChangeMetricsModal
        isOpen={metricsModal !== null}
        onClose={() => setMetricsModal(null)}
        mode={metricsModal?.mode ?? "aggregate"}
        stats={metricsModal?.stats}
        filePath={metricsModal?.path}
        fileStats={metricsModal?.filteredFileStats ?? fileStats}
        title={metricsModal?.title}
        enhancedStats={enhancedQuery.stats}
        enhancedLoading={enhancedQuery.isLoading}
        isUntracked={metricsModal?.category === "untracked"}
        fileHotspots={fileHotspots}
      />
    </MobileContext.Provider>
  );
}
