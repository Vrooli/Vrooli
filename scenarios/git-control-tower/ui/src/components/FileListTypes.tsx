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
  loading?: boolean;
  loadError?: string | null;
  onRetryLoad?: () => void;
  files?: RepoFilesStatus;
  fileStats?: RepoFileStats;
  previewFileStats?: RepoFileStats;
  fileStatsState?: "loading" | "refreshing" | "ready" | "unavailable";
  onRetryFileStats?: () => void;
  selectedFiles?: SelectedFileEntry[];
  selectedKeySet?: Set<string>;
  selectionKey: (entry: SelectedFileEntry) => string;
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
  onStageFilesWithSameName?: (path: string) => void;
  repoId?: string | null;
  mobileSelectionMode?: boolean;
  onOpenReview?: (scenarioSlug: string) => void;
  onEnterSelectionMode?: (path: string, staged: boolean) => void;
  onExitSelectionMode?: () => void;
  onMobileSelectFile?: (path: string, staged: boolean, mode: "toggle" | "range") => void;
  fileHotspots?: Record<string, number>;
  runIndex?: Map<string, import("../lib/runAttribution").RunAttribution>;
  onRevealInTree?: (file: string) => void;
  onOpenRun?: (runId: string) => void;
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
  runAttribution?: import("../lib/runAttribution").RunAttribution;
  onOpenRun?: (runId: string) => void;
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

/** Summarize the line stats that belong to a grouped change header. */
export function summarizeGroupFileStats(
  files: Partial<Record<FileCategory, string[]>>,
  fileStats?: RepoFileStats,
) {
  const summary = { additions: 0, deletions: 0, files: 0 };
  let hasStats = false;
  for (const category of ["staged", "unstaged", "untracked"] as const) {
    const categoryStats = fileStats?.[category];
    if (!categoryStats) continue;
    for (const path of files[category] ?? []) {
      const stats = categoryStats[path];
      if (!stats) continue;
      hasStats = true;
      summary.additions += stats.additions ?? 0;
      summary.deletions += stats.deletions ?? 0;
      summary.files += stats.files || 1;
    }
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
      <span className="flex items-center gap-1 font-mono tabular-nums text-emerald-400">
        <Plus className={iconSize} />
        {stats?.additions ?? 0}
      </span>
      <span className="flex items-center gap-1 font-mono tabular-nums text-rose-400">
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

/** Compact collapsed-group summary: a fixed-width bar encodes additions/deletions. */
export function ChangeStatsRibbon({
  stats,
  label,
  state = "ready",
  onClick,
}: {
  stats?: DiffStats;
  label: string;
  state?: "loading" | "refreshing" | "ready" | "unavailable";
  onClick: () => void;
}) {
  const additions = stats?.additions ?? 0;
  const deletions = stats?.deletions ?? 0;
  const hasStats = Boolean(stats);
  const total = additions + deletions;
  const addedPercent = total > 0 ? (additions / total) * 100 : 0;
  const deletedPercent = total > 0 ? (deletions / total) * 100 : 0;
  const accessibleLabel = state === "loading"
    ? `Change metrics for ${label} are loading`
    : state === "refreshing"
      ? `Change metrics for ${label} are updating; showing the previous snapshot`
    : state === "unavailable"
      ? `Change metrics for ${label} are unavailable`
      : `View change metrics for ${label}`;

  return (
    <button
      type="button"
      className="flex min-h-11 items-center rounded px-1.5 hover:bg-slate-800/60 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sky-400"
      onClick={(event) => {
        event.stopPropagation();
        onClick();
      }}
      aria-label={accessibleLabel}
      title={accessibleLabel}
      data-testid={`change-stats-ribbon-${label}`}
    >
      {state === "loading" ? (
        <span aria-hidden="true" className="h-2 w-12 animate-pulse rounded-sm bg-slate-700" />
      ) : state === "refreshing" ? (
        <span aria-hidden="true" className="flex items-center gap-1.5">
          <span className="flex h-1.5 w-12 overflow-hidden rounded-full bg-slate-700">
            {hasStats && total > 0 ? <><span className="h-full bg-emerald-400" style={{ width: `${addedPercent}%` }} /><span className="h-full bg-rose-400" style={{ width: `${deletedPercent}%` }} /></> : <span className="h-full w-full bg-slate-500" />}
          </span>
          <span className="h-2 w-2 animate-spin rounded-full border border-slate-500 border-t-sky-400" />
        </span>
      ) : state === "unavailable" ? (
        <span aria-hidden="true" className="grid h-4 w-4 place-items-center rounded-full border border-amber-400/50 text-[10px] font-semibold text-amber-300">!</span>
      ) : hasStats ? (
        <span aria-hidden="true" className="flex h-1.5 w-12 overflow-hidden rounded-full bg-slate-700">
          {total > 0
            ? <><span className="h-full bg-emerald-400" style={{ width: `${addedPercent}%` }} /><span className="h-full bg-rose-400" style={{ width: `${deletedPercent}%` }} /></>
            : <span className="h-full w-full bg-slate-500" />}
        </span>
      ) : (
        <span aria-hidden="true" className="h-1.5 w-12 rounded-full bg-slate-700" />
      )}
    </button>
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
