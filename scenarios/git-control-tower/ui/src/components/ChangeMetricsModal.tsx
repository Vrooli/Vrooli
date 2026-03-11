import { memo, useMemo, useEffect, useCallback, useState } from "react";
import { X, BarChart3, Plus, Minus, ArrowRight, Loader2, Info } from "lucide-react";
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
  fileChurnRatio,
  avgLinesPerHunk,
  changeRiskScore,
  riskLabel,
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
  /** Enhanced stats fetched on-demand from /repo/diff (hunks, density, comments, rename) */
  enhancedStats?: DiffStats;
  /** Whether enhanced stats are currently loading */
  enhancedLoading?: boolean;
  /** Whether the file is untracked (no enhanced metrics available) */
  isUntracked?: boolean;
  fileHotspots?: Record<string, number>;
}

/** Density bar: visual representation of change scatter. */
function DensityBar({ density }: { density?: number }) {
  const [showHelp, setShowHelp] = useState(false);
  if (!density || density === 0) return null;
  const pct = Math.min(density * 100, 100);
  const label = densityLabel(density);
  return (
    <div className="space-y-1" data-testid="density-bar">
      <div className="flex items-center justify-between text-xs text-slate-400">
        <span className="flex items-center gap-1">
          Density
          <button
            type="button"
            className="inline-flex items-center justify-center text-slate-500 hover:text-slate-300 transition-colors"
            onClick={() => setShowHelp((v) => !v)}
            aria-label="What is density?"
            data-testid="density-help-toggle"
          >
            <Info className="h-3 w-3" />
          </button>
        </span>
        <span>
          {density.toFixed(2)}
          {label ? ` (${label})` : ""}
        </span>
      </div>
      {showHelp && (
        <div
          className="text-[11px] leading-relaxed text-slate-400 bg-slate-900/60 rounded-md px-2.5 py-2 border border-slate-800/60"
          data-testid="density-help-text"
        >
          <p>
            Density = hunks &divide; lines changed. It measures how scattered
            your edits are within the file.
          </p>
          <ul className="mt-1.5 space-y-0.5 list-none">
            <li>
              <span className="inline-block w-2 h-2 rounded-full mr-1.5 align-middle" style={{ backgroundColor: "#10b981" }} />
              <strong className="text-slate-300">&le; 0.10 &mdash; Focused:</strong>{" "}
              changes are in a few contiguous blocks.
            </li>
            <li>
              <span className="inline-block w-2 h-2 rounded-full mr-1.5 align-middle" style={{ backgroundColor: "#f59e0b" }} />
              <strong className="text-slate-300">&le; 0.30 &mdash; Moderate:</strong>{" "}
              edits touch several areas of the file.
            </li>
            <li>
              <span className="inline-block w-2 h-2 rounded-full mr-1.5 align-middle" style={{ backgroundColor: "#ef4444" }} />
              <strong className="text-slate-300">&gt; 0.30 &mdash; Scattered:</strong>{" "}
              many small edits spread across the file.
            </li>
          </ul>
        </div>
      )}
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

function EnhancedLoadingRow() {
  return (
    <div className="flex items-center gap-2 py-1" data-testid="enhanced-loading">
      <Loader2 className="h-3 w-3 animate-spin text-slate-500" />
      <span className="text-xs text-slate-500">Loading detailed metrics…</span>
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
  enhancedStats,
  enhancedLoading,
  isUntracked,
  fileHotspots,
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
        {mode === "file" && !isUntracked && (() => {
          const es = enhancedStats;
          const hunkSrc = es ?? s;
          if (enhancedLoading) return <EnhancedLoadingRow />;
          if (hunkSrc && (hunkSrc.hunk_count ?? 0) > 0) return (
            <div className="space-y-0.5">
              <MetricRow
                label="Hunks"
                value={hunkSrc.hunk_count ?? 0}
                testId="metric-hunk-count"
              />
              <MetricRow
                label="Largest hunk"
                value={`${hunkSrc.largest_hunk ?? 0} lines`}
                testId="metric-largest-hunk"
              />
            </div>
          );
          return null;
        })()}

        {/* Comment lines (file mode only) */}
        {mode === "file" && !isUntracked && (() => {
          const es = enhancedStats;
          const commentSrc = es ?? s;
          if (!enhancedLoading && commentSrc && (commentSrc.comment_additions || commentSrc.comment_deletions)) return (
            <MetricRow
              label="Comment lines"
              value={`+${commentSrc.comment_additions ?? 0} / -${commentSrc.comment_deletions ?? 0}`}
              testId="metric-comment-lines"
            />
          );
          return null;
        })()}

        {/* Density bar (file mode only) */}
        {mode === "file" && !isUntracked && (() => {
          const densitySrc = enhancedStats ?? s;
          if (!enhancedLoading && densitySrc) return <DensityBar density={densitySrc.density} />;
          return null;
        })()}
        {/* Per-file churn ratio (file mode only) */}
        {mode === "file" && (() => {
          if (!s) return null;
          const ratio = fileChurnRatio(s);
          if (ratio === 0) return null;
          return (
            <MetricRow
              label="Churn ratio"
              value={`${(ratio * 100).toFixed(0)}% ${churnLabel(ratio)}`}
              testId="metric-file-churn"
            />
          );
        })()}

        {/* Avg lines per hunk (file mode only) */}
        {mode === "file" && !isUntracked && (() => {
          const hunkSrc = enhancedStats ?? s;
          if (!hunkSrc || !hunkSrc.hunk_count || hunkSrc.hunk_count === 0) return null;
          return (
            <MetricRow
              label="Avg lines/hunk"
              value={avgLinesPerHunk(hunkSrc)}
              testId="metric-avg-lines-per-hunk"
            />
          );
        })()}

        {/* Change risk score (file mode only) */}
        {mode === "file" && !isUntracked && (() => {
          const hunkSrc = enhancedStats ?? s;
          if (!hunkSrc || !hunkSrc.hunk_count || hunkSrc.hunk_count === 0) return null;
          const score = changeRiskScore(hunkSrc);
          const label = riskLabel(score);
          return (
            <MetricRow
              label="Risk score"
              value={`${score}${label ? ` (${label})` : ""}`}
              testId="metric-risk-score"
            />
          );
        })()}

        {/* File hotspot (file mode only) */}
        {mode === "file" && (() => {
          const count = filePath && fileHotspots?.[filePath];
          if (!count || count <= 1) return null;
          return (
            <MetricRow
              label="Hotspot"
              value={`${count} commits in last 50`}
              testId="metric-hotspot"
            />
          );
        })()}
      </div>

      {/* Details: rename / binary / test file (file mode) */}
      {mode === "file" && (() => {
        const detailSrc = enhancedStats ?? s;
        const showRename = !isUntracked && detailSrc?.is_rename && detailSrc?.old_path;
        const showBinary = s?.is_binary;
        const showTest = filePath && isTestFile(filePath);
        const showNewFile = s?.is_new_file || isUntracked;
        const showDeletedFile = s?.is_deleted_file;
        if (!showRename && !showBinary && !showTest && !showNewFile && !showDeletedFile) return null;
        return (
          <div className="space-y-2 border-t border-slate-800 pt-3">
            <h3 className="text-xs font-medium text-slate-500 uppercase tracking-wider">
              Details
            </h3>
            {showRename && (
              <div
                className="flex items-center gap-2 text-xs text-slate-300"
                data-testid="metric-rename"
              >
                <span className="text-slate-500">Rename:</span>
                <span className="truncate">{detailSrc?.old_path}</span>
                <ArrowRight className="h-3 w-3 text-slate-500 flex-shrink-0" />
                <span className="truncate">{filePath}</span>
              </div>
            )}
            {showBinary && (
              <div className="text-xs text-amber-400" data-testid="metric-binary">
                Binary file
              </div>
            )}
            {showTest && (
              <div className="text-xs text-amber-400" data-testid="metric-is-test-file">
                Test file
              </div>
            )}
            {showNewFile && (
              <div className="text-xs text-emerald-400" data-testid="metric-new-file">
                New file
              </div>
            )}
            {showDeletedFile && (
              <div className="text-xs text-red-400" data-testid="metric-deleted-file">
                Deleted file
              </div>
            )}
          </div>
        );
      })()}

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
          {aggregate.testToCodeRatio > 0 && (
            <MetricRow
              label="Test-to-code ratio"
              value={`${(aggregate.testToCodeRatio * 100).toFixed(0)}%`}
              testId="metric-test-to-code-ratio"
            />
          )}
          {aggregate.newFileCount > 0 && (
            <MetricRow
              label="New files"
              value={aggregate.newFileCount}
              testId="metric-new-file-count"
            />
          )}
          {aggregate.deletedFileCount > 0 && (
            <MetricRow
              label="Deleted files"
              value={aggregate.deletedFileCount}
              testId="metric-deleted-file-count"
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
