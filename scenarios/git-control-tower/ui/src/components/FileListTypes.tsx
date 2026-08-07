import { createContext, useContext } from "react";
import { Plus, Minus } from "lucide-react";
import type { ChangeGroupAPI, DiffStats, RepoFilesStatus, RepoFileStats, FileViewMode } from "../lib/api";

export const MobileContext = createContext(false);

export type FileCategory = "staged" | "unstaged" | "untracked" | "conflicts";

export type SelectedFileEntry = { path: string; staged: boolean };

export type GroupingRule = {
  id: string;
  label: string;
  prefix?: string;
  prefixes?: string[];
  mode?: "prefix" | "segment";
};

export interface FileListProps {
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
  /** Paths with an in-flight stage/unstage/discard op, for per-row spinners. */
  pendingPaths?: ReadonlySet<string>;
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
  resolvedGroups?: ChangeGroupAPI[];
  groupingAvailable?: boolean;
  onCycleViewMode?: () => void;
  onStagePaths?: (paths: string[]) => void;
  onDiscardPaths?: (paths: string[], untracked: boolean) => void;
  onSelectAnyFile?: (path: string) => void;
  scrollToFile?: string;
  onScrollComplete?: () => void;
  /**
   * Persisted scroll position store (mobile only). When supplied, FileList saves
   * the Changes list scrollTop on scroll and restores it on mount, surviving the
   * panel unmount that happens when switching mobile tabs. Desktop keeps all
   * panels mounted and does not pass this.
   */
  scrollTopStore?: React.MutableRefObject<number>;
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

export interface FileSectionProps {
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
  /** Paths with an in-flight op; the matching row shows a spinner. */
  pendingPaths?: ReadonlySet<string>;
  /** Section-wide loading fallback when pendingPaths is not supplied. */
  isLoading?: boolean;
  changeStats?: DiffStats;
  defaultExpanded?: boolean;
  /** Controlled expand state (with onToggle); falls back to internal state when omitted. */
  expanded?: boolean;
  onToggle?: () => void;
  onDiscard?: (path: string) => void;
  isDiscarding?: boolean;
  confirmingDiscard?: string | null;
  onConfirmDiscard?: (path: string | null) => void;
  onIgnore?: (path: string, level?: "project" | "group", groupDir?: string) => void;
  isIgnoring?: boolean;
  confirmingIgnore?: string | null;
  onConfirmIgnore?: (path: string | null) => void;
  resolvedGroups?: ChangeGroupAPI[];
  onOpenMobileActions?: (file: string) => void;
  onContextMenu?: (file: string, event: React.MouseEvent) => void;
  mobileSelectionMode?: boolean;
  onLongPress?: (file: string, staged: boolean) => void;
  onMobileTap?: (file: string, staged: boolean, mode: "toggle" | "range") => void;
  onStatsClick?: () => void;
  onViewMetrics?: (file: string, category: FileCategory) => void;
}

export interface FileRowProps {
  file: string;
  displayPath?: string;
  badge?: { label: string; style: string };
  isSelected?: boolean;
  isStaged: boolean;
  canDiscard?: boolean;
  isLoading?: boolean;
  isDiscarding?: boolean;
  isIgnoring?: boolean;
  isBinary?: boolean;
  isApproved?: boolean;
  itemTestId?: string;
  actionTestId?: string;
  discardTestId?: string;
  ignoreTestId?: string;
  actionIcon?: React.ReactNode;
  actionLabel?: string;
  onSelectFile: (path: string, staged: boolean, event: React.MouseEvent<HTMLLIElement>) => void;
  onAction: (path: string) => void;
  onDiscard?: (path: string) => void;
  onConfirmDiscard?: (path: string | null) => void;
  onIgnore?: (path: string, level?: "project" | "group", groupDir?: string) => void;
  onConfirmIgnore?: (path: string | null) => void;
  confirmingDiscard?: string | null;
  confirmingIgnore?: string | null;
  resolvedGroups?: ChangeGroupAPI[];
  onOpenMobileActions?: (file: string) => void;
  onContextMenu?: (file: string, event: React.MouseEvent) => void;
  mobileSelectionMode?: boolean;
  onLongPress?: (file: string, staged: boolean) => void;
  onMobileTap?: (file: string, staged: boolean, mode: "toggle" | "range") => void;
  onViewMetrics?: (file: string, category: FileCategory) => void;
  category?: FileCategory;
}

export const statusStyleMap = {
  D: "text-red-400 border-red-500/40 bg-red-500/10",
  M: "text-amber-300 border-amber-500/40 bg-amber-500/10",
  A: "text-emerald-300 border-emerald-500/40 bg-emerald-500/10",
  R: "text-cyan-300 border-cyan-500/40 bg-cyan-500/10",
  U: "text-red-300 border-red-500/40 bg-red-500/10",
  "?": "text-slate-300 border-slate-500/40 bg-slate-500/10",
};

export function summarizeFileStats(
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

export function LineStats({
  stats,
  compact = false,
  onClick,
}: {
  stats?: DiffStats;
  compact?: boolean;
  onClick?: (e: React.MouseEvent) => void;
}) {
  const isMobile = useContext(MobileContext);
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
        className={`flex items-center gap-2 ${textSize} hover:underline decoration-slate-600 cursor-pointer ${isMobile ? "min-h-11" : ""}`}
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

export function getStatusBadge(code: string | undefined, category: FileCategory) {
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
