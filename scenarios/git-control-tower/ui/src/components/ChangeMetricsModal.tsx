import { memo, useMemo, useEffect, useCallback } from "react";
import { X, BarChart3, Plus, Minus, ArrowRight } from "lucide-react";
import { useIsMobile } from "../hooks";
import type { DiffStats, RepoFileStats } from "../lib/api";
import {
  aggregateFileStats,
  formatNetLines,
  densityLabel,
  categoryStats,
  isTestFile,
  churnLabel,
  formatFileTypeBreakdown,
} from "../lib/metrics";

interface ChangeMetricsModalProps {
  isOpen: boolean;
  onClose: () => void;
  mode: "file" | "aggregate";
  /** File mode: stats for a single file */
  stats?: DiffStats;
  /** File mode: path of the file */
  filePath?: string;
  /** Aggregate mode: all file stats by category */
  fileStats?: RepoFileStats;
  /** Custom title override */
  title?: string;
}

/** Density bar: visual representation of change scatter. */
function DensityBar({ density }: { density?: number }) {
  if (!density || density === 0) return null;
  const pct = Math.min(density * 100, 100);
  const label = densityLabel(density);
  return (
    <div className="space-y-1" data-testid="density-bar">
      <div className="flex items-center justify-between text-xs text-slate-400">
        <span>Density</span>
        <span>
          {density.toFixed(2)}
          {label ? ` (${label})` : ""}
        </span>
      </div>
      <div className="h-1.5 w-full rounded-full bg-slate-800 overflow-hidden">
        <div
          className="h-full rounded-full transition-all"
          style={{
            width: `${pct}%`,
            backgroundColor:
              density <= 0.1
                ? "#10b981"
                : density <= 0.3
                  ? "#f59e0b"
                  : "#ef4444",
          }}
        />
      </div>
    </div>
  );
}

function MetricRow({
  label,
  value,
  testId,
}: {
  label: string;
  value: string | number;
  testId?: string;
}) {
  return (
    <div className="flex items-center justify-between py-1">
      <span className="text-xs text-slate-400">{label}</span>
      <span className="text-xs font-medium text-slate-200" data-testid={testId}>
        {value}
      </span>
    </div>
  );
}

function CategoryRow({
  label,
  additions,
  deletions,
  netLines,
  count,
}: {
  label: string;
  additions: number;
  deletions: number;
  netLines: number;
  count: number;
}) {
  if (count === 0) return null;
  return (
    <div className="flex items-center justify-between py-1.5">
      <span className="text-xs text-slate-400">
        {label} ({count})
      </span>
      <div className="flex items-center gap-3 text-xs">
        <span className="text-emerald-500">+{additions}</span>
        <span className="text-red-500">-{deletions}</span>
        <span className="text-blue-400 font-medium">{formatNetLines(netLines)}</span>
      </div>
    </div>
  );
}

export const ChangeMetricsModal = memo(function ChangeMetricsModal({
  isOpen,
  onClose,
  mode,
  stats,
  filePath,
  fileStats,
  title,
}: ChangeMetricsModalProps) {
  const isMobile = useIsMobile();

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    },
    [onClose],
  );

  useEffect(() => {
    if (!isOpen) return;
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, handleKeyDown]);

  const aggregate = useMemo(
    () => (mode === "aggregate" ? aggregateFileStats(fileStats) : null),
    [mode, fileStats],
  );

  const stagedCat = useMemo(
    () => (mode === "aggregate" ? categoryStats(fileStats?.staged) : null),
    [mode, fileStats],
  );
  const unstagedCat = useMemo(
    () => (mode === "aggregate" ? categoryStats(fileStats?.unstaged) : null),
    [mode, fileStats],
  );
  const untrackedCat = useMemo(
    () => (mode === "aggregate" ? categoryStats(fileStats?.untracked) : null),
    [mode, fileStats],
  );

  if (!isOpen) return null;

  const s = mode === "file" ? stats : undefined;
  const additions = s?.additions ?? aggregate?.totalAdditions ?? 0;
  const deletions = s?.deletions ?? aggregate?.totalDeletions ?? 0;
  const netLines = s?.net_lines ?? aggregate?.totalNetLines ?? additions - deletions;

  const content = (
    <div className="space-y-4" data-testid="metrics-modal">
      {/* Summary */}
      <div className="space-y-3">
        <h3 className="text-xs font-medium text-slate-500 uppercase tracking-wider">
          Summary
        </h3>
        <div className="flex items-center gap-4">
          <span className="flex items-center gap-1 text-emerald-500 text-sm font-medium">
            <Plus className="h-3.5 w-3.5" />
            {additions}
          </span>
          <span className="flex items-center gap-1 text-red-500 text-sm font-medium">
            <Minus className="h-3.5 w-3.5" />
            {deletions}
          </span>
          <span
            className="text-sm font-medium text-blue-400"
            data-testid="metric-net-lines"
          >
            net {formatNetLines(netLines)}
          </span>
        </div>

        {/* Hunk info (file mode only — aggregate doesn't have hunk data) */}
        {mode === "file" && s && (s.hunk_count ?? 0) > 0 && (
          <div className="space-y-0.5">
            <MetricRow
              label="Hunks"
              value={s.hunk_count ?? 0}
              testId="metric-hunk-count"
            />
            <MetricRow
              label="Largest hunk"
              value={`${s.largest_hunk ?? 0} lines`}
              testId="metric-largest-hunk"
            />
          </div>
        )}

        {/* Comment lines (file mode only) */}
        {mode === "file" && s && (s.comment_additions || s.comment_deletions) ? (
          <MetricRow
            label="Comment lines"
            value={`+${s.comment_additions ?? 0} / -${s.comment_deletions ?? 0}`}
            testId="metric-comment-lines"
          />
        ) : null}

        {/* Density bar (file mode only) */}
        {mode === "file" && s && <DensityBar density={s.density} />}
      </div>

      {/* Details: rename / binary / test file (file mode) */}
      {mode === "file" && s && (s.is_rename || s.is_binary || (filePath && isTestFile(filePath))) && (
        <div className="space-y-2 border-t border-slate-800 pt-3">
          <h3 className="text-xs font-medium text-slate-500 uppercase tracking-wider">
            Details
          </h3>
          {s.is_rename && s.old_path && (
            <div
              className="flex items-center gap-2 text-xs text-slate-300"
              data-testid="metric-rename"
            >
              <span className="text-slate-500">Rename:</span>
              <span className="truncate">{s.old_path}</span>
              <ArrowRight className="h-3 w-3 text-slate-500 flex-shrink-0" />
              <span className="truncate">{filePath}</span>
            </div>
          )}
          {s.is_binary && (
            <div className="text-xs text-amber-400" data-testid="metric-binary">
              Binary file
            </div>
          )}
          {filePath && isTestFile(filePath) && (
            <div className="text-xs text-amber-400" data-testid="metric-is-test-file">
              Test file
            </div>
          )}
        </div>
      )}

      {/* Aggregate breakdown */}
      {mode === "aggregate" && aggregate && (
        <div className="space-y-2 border-t border-slate-800 pt-3">
          <h3 className="text-xs font-medium text-slate-500 uppercase tracking-wider">
            Breakdown
          </h3>
          <MetricRow
            label="Files"
            value={`${aggregate.totalFiles}${aggregate.binaryCount > 0 ? ` (${aggregate.binaryCount} binary)` : ""}${aggregate.renameCount > 0 ? ` (${aggregate.renameCount} renamed)` : ""}`}
            testId="metric-total-files"
          />
          {Object.keys(aggregate.fileTypeBreakdown).length > 0 && (
            <MetricRow
              label="File types"
              value={formatFileTypeBreakdown(aggregate.fileTypeBreakdown)}
              testId="metric-file-types"
            />
          )}
          {aggregate.testFileCount > 0 && (
            <MetricRow
              label="Test files"
              value={aggregate.testFileCount}
              testId="metric-test-files"
            />
          )}
          {aggregate.churnRatio > 0 && (
            <MetricRow
              label="Churn"
              value={`${(aggregate.churnRatio * 100).toFixed(0)}% ${churnLabel(aggregate.churnRatio)}`}
              testId="metric-churn"
            />
          )}
          {aggregate.totalFiles >= 3 && (
            <MetricRow
              label={`Top ${aggregate.paretoTopN} file(s)`}
              value={`${aggregate.paretoPercent}% of changes`}
              testId="metric-concentration"
            />
          )}
          <div className="space-y-0.5 border-t border-slate-800/60 pt-2 mt-2">
            {stagedCat && (
              <CategoryRow label="Staged" {...stagedCat} />
            )}
            {unstagedCat && (
              <CategoryRow label="Unstaged" {...unstagedCat} />
            )}
            {untrackedCat && (
              <CategoryRow label="Untracked" {...untrackedCat} />
            )}
          </div>
        </div>
      )}
    </div>
  );

  // Mobile: full-screen overlay
  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Change metrics"
      >
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <div className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5 text-slate-400" />
            <h2 className="text-base font-semibold text-slate-100">
              {title ?? (mode === "file" ? "File Metrics" : "Change Metrics")}
            </h2>
          </div>
          <button
            type="button"
            className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-6">
          {filePath && mode === "file" && (
            <p className="text-xs text-slate-500 truncate mb-4">{filePath}</p>
          )}
          {content}
        </div>
      </div>
    );
  }

  // Desktop: centered overlay
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="Change metrics"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="w-full max-w-sm rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <div className="flex items-center gap-2">
            <BarChart3 className="h-4 w-4 text-slate-400" />
            <h2 className="text-sm font-semibold text-slate-100">
              {title ?? (mode === "file" ? "File Metrics" : "Change Metrics")}
            </h2>
          </div>
          <button
            type="button"
            className="h-7 w-7 inline-flex items-center justify-center rounded-full text-slate-400 hover:bg-slate-800 hover:text-slate-200"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="px-4 py-4 max-h-[60vh] overflow-y-auto">
          {filePath && mode === "file" && (
            <p className="text-xs text-slate-500 truncate mb-3">{filePath}</p>
          )}
          {content}
        </div>
      </div>
    </div>
  );
});
