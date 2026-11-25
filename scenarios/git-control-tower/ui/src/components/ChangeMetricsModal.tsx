import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import {
  memo,
  useEffect,
  useMemo,
  useCallback,
  useState,
  useRef,
  type ReactNode,
} from "react";
import {
  X,
  BarChart3,
  Loader2,
  Info,
  Copy,
  Check,
  AlertTriangle,
  ArrowRight,
  Layers,
  FilePlus2,
  FlaskConical,
  Target,
  Gauge,
  Activity,
  ChevronDown,
  FileText,
  GitBranch,
  ShieldCheck,
} from "lucide-react";
import { useIsMobile } from "../hooks";
import { cn } from "../lib/utils";
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
  type AggregateMetrics,
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
  fileStatsState?: "loading" | "refreshing" | "ready" | "unavailable";
  onRetryFileStats?: () => void;
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

type Tone = "success" | "info" | "warning" | "neutral";

interface ReviewSignal {
  tone: Tone;
  pill: string;
  body: string;
}

interface ToneStyle {
  card: string;
  icon: string;
  pill: string;
  text: string;
}

const toneStyles: Record<Tone, ToneStyle> = {
  success: {
    card: "border-emerald-400/25 bg-emerald-400/[0.06]",
    icon: "bg-emerald-400/15 text-emerald-300",
    pill: "border-emerald-400/35 bg-emerald-400/10 text-emerald-300",
    text: "text-emerald-300",
  },
  info: {
    card: "border-cyan-400/25 bg-cyan-400/[0.06]",
    icon: "bg-cyan-400/15 text-cyan-300",
    pill: "border-cyan-400/35 bg-cyan-400/10 text-cyan-300",
    text: "text-cyan-300",
  },
  warning: {
    card: "border-amber-400/25 bg-amber-400/[0.06]",
    icon: "bg-amber-400/15 text-amber-300",
    pill: "border-amber-400/35 bg-amber-400/10 text-amber-300",
    text: "text-amber-300",
  },
  neutral: {
    card: "border-slate-700/70 bg-slate-900/40",
    icon: "bg-slate-800 text-slate-300",
    pill: "border-slate-700 bg-slate-800/70 text-slate-300",
    text: "text-slate-300",
  },
};

/** Density bar: visual representation of change scatter. */
function DensityBar({ density }: { density?: number }) {
  const [showHelp, setShowHelp] = useState(false);
  if (!density || density === 0) return null;
  const pct = Math.min(density * 100, 100);
  const label = densityLabel(density);
  return (
    <div className="space-y-1 pt-1" data-testid="density-bar">
      <div className="flex items-center justify-between text-[11px] text-slate-400">
        <span className="flex items-center gap-1">
          Density
          <button
            type="button"
            className="inline-flex items-center justify-center text-slate-500 transition-colors hover:text-slate-200"
            onClick={() => setShowHelp((v) => !v)}
            aria-label="What is density?"
            data-testid="density-help-toggle"
          >
            <Info className="h-3 w-3" />
          </button>
        </span>
        <span className="font-mono tabular-nums text-slate-300">
          {density.toFixed(2)}
          {label ? ` (${label})` : ""}
        </span>
      </div>
      {showHelp && (
        <div
          className="rounded-lg border border-slate-800/80 bg-slate-950/70 px-2.5 py-2 text-[10.5px] leading-relaxed text-slate-400"
          data-testid="density-help-text"
        >
          <p>
            Density = hunks &divide; lines changed. It measures how scattered
            your edits are within the file.
          </p>
          <ul className="mt-1.5 space-y-0.5 list-none">
            <li>
              <span className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" style={{ backgroundColor: "#34d399" }} />
              <strong className="text-slate-300">&le; 0.10 &mdash; Focused:</strong>{" "}
              changes are in a few contiguous blocks.
            </li>
            <li>
              <span className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" style={{ backgroundColor: "#fbbf24" }} />
              <strong className="text-slate-300">&le; 0.30 &mdash; Moderate:</strong>{" "}
              edits touch several areas of the file.
            </li>
            <li>
              <span className="mr-1.5 inline-block h-2 w-2 rounded-full align-middle" style={{ backgroundColor: "#fb7185" }} />
              <strong className="text-slate-300">&gt; 0.30 &mdash; Scattered:</strong>{" "}
              many small edits spread across the file.
            </li>
          </ul>
        </div>
      )}
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-800">
        <div
          className="h-full rounded-full transition-all"
          style={{
            width: `${pct}%`,
            backgroundColor:
              density <= 0.1
                ? "#34d399"
                : density <= 0.3
                  ? "#fbbf24"
                  : "#fb7185",
          }}
        />
      </div>
    </div>
  );
}

export function MetricRow({
  label,
  value,
  testId,
}: {
  label: string;
  value: string | number;
  testId?: string;
}) {
  return (
    <div className="flex items-center justify-between gap-3 py-1">
      <span className="text-[11px] text-slate-400">{label}</span>
      <span className="text-right text-[11px] font-medium text-slate-200" data-testid={testId}>
        {value}
      </span>
    </div>
  );
}

export function CategoryRow({
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
    <div className="flex items-center justify-between gap-2 py-1.5">
      <span className="text-[11px] text-slate-400">
        {label} ({count})
      </span>
      <div className="flex items-center gap-2.5 font-mono text-[10.5px] tabular-nums">
        <span className="text-emerald-300">+{additions}</span>
        <span className="text-rose-300">-{deletions}</span>
        <span className="font-medium text-cyan-300">{formatNetLines(netLines)}</span>
      </div>
    </div>
  );
}

function compactNumber(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return String(value);
}

function formatCount(value: number): string {
  return value.toLocaleString();
}

function buildAggregateSignal(aggregate: AggregateMetrics): ReviewSignal {
  if (aggregate.totalFiles === 0) {
    return {
      tone: "neutral",
      pill: "No changes",
      body: "There are no changed files to measure.",
    };
  }

  const parts: string[] = [];
  let tone: Tone = "success";
  const totalLines = aggregate.totalAdditions + aggregate.totalDeletions;

  if (aggregate.totalFiles >= 3 && aggregate.paretoPercent >= 80) {
    parts.push(
      `${aggregate.paretoPercent}% of changed lines sit in ${formatCount(aggregate.paretoTopN)} files. Start with the largest.`,
    );
    tone = "info";
  }

  if (aggregate.testFileCount > 0 && aggregate.testToCodeRatio < 0.1) {
    parts.push(
      `Tests are ${Math.round(aggregate.testToCodeRatio * 100)}% of changed lines.`,
    );
    tone = "warning";
  } else if (aggregate.testFileCount === 0 && aggregate.totalFiles >= 5) {
    parts.push("No test files changed.");
    tone = "warning";
  }

  if (
    aggregate.binaryCount > 0 &&
    aggregate.binaryCount / aggregate.totalFiles >= 0.15
  ) {
    parts.push(`${formatCount(aggregate.binaryCount)} binary files are included.`);
    if (tone !== "warning") tone = "info";
  }

  if (aggregate.churnRatio >= 0.7) {
    parts.push(`Mostly rewriting (${Math.round(aggregate.churnRatio * 100)}% churn).`);
    if (tone === "success") tone = "info";
  }

  if (parts.length === 0) {
    if (aggregate.newFileCount / aggregate.totalFiles >= 0.5) {
      parts.push(
        `${Math.round((aggregate.newFileCount / aggregate.totalFiles) * 100)}% of changed files are new.`,
      );
    } else if (totalLines > 0 && aggregate.totalAdditions / totalLines >= 0.75) {
      parts.push("Mostly additive; few deletions.");
    } else {
      parts.push("No unusual signals. Review the diff normally.");
    }
  }

  return {
    tone,
    pill: tone === "warning" ? "Needs a plan" : tone === "info" ? "Worth a look" : "Low attention",
    body: parts.slice(0, 2).join(" "),
  };
}

function buildFileSignal({
  stats,
  hunkSrc,
  isUntracked,
  enhancedLoading,
}: {
  stats?: DiffStats;
  hunkSrc?: DiffStats;
  isUntracked?: boolean;
  enhancedLoading?: boolean;
}): ReviewSignal {
  if (isUntracked) {
    return {
      tone: "info",
      pill: "New file",
      body: "No committed version to compare. Review the new content directly.",
    };
  }
  if (enhancedLoading) {
    return {
      tone: "info",
      pill: "Loading",
      body: "Detailed hunk and density metrics are loading.",
    };
  }
  if (stats?.is_binary) {
    return {
      tone: "info",
      pill: "Binary",
      body: "Binary file. Line metrics are not available.",
    };
  }

  const hunks = hunkSrc?.hunk_count ?? 0;
  if (hunks === 0) {
    return {
      tone: "success",
      pill: "Low attention",
      body: "No hunks to review.",
    };
  }

  const largest = hunkSrc?.largest_hunk ?? 0;
  const density = hunkSrc?.density;
  const risk = hunks * largest;

  if (risk >= 200 || (density != null && density > 0.3)) {
    return {
      tone: "warning",
      pill: "Needs a close read",
      body: `${hunks} hunk${hunks === 1 ? "" : "s"}, largest ${largest} lines. The edits are ${densityLabel(density) || "spread out"}.`,
    };
  }
  if (risk >= 50) {
    return {
      tone: "info",
      pill: "Worth a look",
      body: `${hunks} hunk${hunks === 1 ? "" : "s"}, largest ${largest} lines. Moderate review weight.`,
    };
  }
  return {
    tone: "success",
    pill: "Low attention",
    body: `A focused edit: ${hunks} hunk${hunks === 1 ? "" : "s"}, largest ${largest} lines.`,
  };
}

function buildAggregateSummary(aggregate: AggregateMetrics): string {
  const lines = [
    "Change metrics — working tree",
    `+${formatCount(aggregate.totalAdditions)} / -${formatCount(aggregate.totalDeletions)} (net ${formatNetLines(aggregate.totalNetLines)})`,
    `${formatCount(aggregate.totalFiles)} files${aggregate.binaryCount ? ` (${formatCount(aggregate.binaryCount)} binary)` : ""}${aggregate.newFileCount ? `, ${formatCount(aggregate.newFileCount)} new` : ""}`,
  ];
  if (aggregate.testFileCount > 0) {
    lines.push(
      `${formatCount(aggregate.testFileCount)} test files, ${Math.round(aggregate.testToCodeRatio * 100)}% test-to-code`,
    );
  } else {
    lines.push("No test files changed");
  }
  if (aggregate.totalFiles >= 3) {
    lines.push(
      `${aggregate.paretoPercent}% of changes in ${formatCount(aggregate.paretoTopN)} files`,
    );
  }
  return lines.join("\n");
}

function buildFileSummary({
  stats,
  hunkSrc,
  filePath,
  isUntracked,
}: {
  stats?: DiffStats;
  hunkSrc?: DiffStats;
  filePath?: string;
  isUntracked?: boolean;
}) {
  const additions = stats?.additions ?? 0;
  const deletions = stats?.deletions ?? 0;
  const net = stats?.net_lines ?? additions - deletions;
  const lines = [
    `File metrics — ${filePath ?? "selected file"}`,
    `+${formatCount(additions)} / -${formatCount(deletions)} (net ${formatNetLines(net)})`,
  ];
  if (isUntracked) {
    lines.push("New file: no committed version to compare");
  } else if (hunkSrc?.hunk_count) {
    lines.push(
      `${hunkSrc.hunk_count} hunks, largest ${hunkSrc.largest_hunk ?? 0} lines`,
    );
    if (hunkSrc.density != null) {
      lines.push(`Density ${hunkSrc.density.toFixed(2)} (${densityLabel(hunkSrc.density)})`);
    }
    const risk = changeRiskScore(hunkSrc);
    lines.push(`Risk ${risk}${riskLabel(risk) ? ` (${riskLabel(risk)})` : ""}`);
  }
  return lines.join("\n");
}

function topChangedFiles(fileStats?: RepoFileStats, limit = 20) {
  if (!fileStats) return [];
  const seen = new Set<string>();
  const rows: Array<{ path: string; additions: number; deletions: number }> = [];
  for (const category of [fileStats.staged, fileStats.unstaged, fileStats.untracked]) {
    if (!category) continue;
    for (const [path, stats] of Object.entries(category)) {
      if (seen.has(path)) continue;
      seen.add(path);
      rows.push({
        path,
        additions: stats.additions ?? 0,
        deletions: stats.deletions ?? 0,
      });
    }
  }
  return rows
    .sort(
      (a, b) =>
        b.additions + b.deletions - (a.additions + a.deletions),
    )
    .slice(0, limit);
}

async function copyText(text: string): Promise<boolean> {
  try {
    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch {
    /* fall through to the legacy path */
  }
  try {
    if (typeof document === "undefined") return false;
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "fixed";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.select();
    const ok = document.execCommand("copy");
    textarea.remove();
    return ok;
  } catch {
    return false;
  }
}

function CopyActionButton({
  text,
  label,
  variant = "secondary",
  icon,
}: {
  text: string;
  label: string;
  variant?: "primary" | "secondary";
  icon?: ReactNode;
}) {
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (timeoutRef.current != null) window.clearTimeout(timeoutRef.current);
    };
  }, []);

  const handleCopy = async () => {
    if (!(await copyText(text))) return;
    setCopied(true);
    if (timeoutRef.current != null) window.clearTimeout(timeoutRef.current);
    timeoutRef.current = window.setTimeout(() => setCopied(false), 1600);
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={cn(
        "inline-flex min-h-11 flex-1 items-center justify-center gap-1.5 rounded-lg border px-3 text-[11.5px] font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400/60 sm:min-h-9 sm:flex-none",
        variant === "primary"
          ? "border-cyan-400/60 bg-cyan-400 text-slate-950 hover:bg-cyan-300"
          : "border-slate-700 bg-slate-900/40 text-slate-200 hover:bg-slate-800",
      )}
    >
      {copied ? <Check className="h-3.5 w-3.5" /> : icon ?? <Copy className="h-3.5 w-3.5" />}
      {copied ? "Copied" : label}
    </button>
  );
}

function ChangeHero({
  additions,
  deletions,
  net,
  netTestId,
  pill,
  label = "Net change",
}: {
  additions: number;
  deletions: number;
  net: number;
  netTestId?: string;
  pill?: string;
  label?: string;
}) {
  const total = additions + deletions;
  const addPct = total > 0 ? (additions / total) * 100 : 0;
  const delPct = total > 0 ? (deletions / total) * 100 : 0;
  const netTone =
    net > 0 ? "text-cyan-300" : net < 0 ? "text-rose-300" : "text-slate-200";
  const defaultPill =
    total === 0
      ? "No line changes"
      : addPct >= 75
        ? "Mostly additive"
        : delPct >= 75
          ? "Mostly deletions"
          : "Mixed";

  return (
    <section className="relative overflow-hidden rounded-xl border border-slate-800/80 bg-gradient-to-br from-slate-900/90 to-slate-950/90 p-4">
      <div className="pointer-events-none absolute -right-16 -top-24 h-56 w-56 rounded-full bg-cyan-400/10 blur-2xl" />
      <div className="relative flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[9.5px] font-semibold uppercase tracking-[0.14em] text-slate-500">
            {label}
          </p>
          <div className="mt-1 flex items-baseline gap-2" data-testid={netTestId}>
            <span className="text-[11px] font-medium text-slate-500">net</span>{" "}
            <span
              className={cn(
                "font-mono text-3xl font-semibold leading-none tracking-tight tabular-nums",
                netTone,
              )}
            >
              {formatNetLines(net)}
            </span>
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-[10.5px] tabular-nums">
            <span className="text-emerald-300">+{formatCount(additions)} added</span>
            <span className="text-rose-300">−{formatCount(deletions)} removed</span>
          </div>
        </div>
        <span
          className={cn(
            "inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[9.5px] font-semibold",
            defaultPill === "Mostly deletions"
              ? toneStyles.warning.pill
              : defaultPill === "Mixed"
                ? toneStyles.info.pill
                : toneStyles.success.pill,
          )}
        >
          {pill ?? defaultPill}
        </span>
      </div>
      <div
        className="mt-3 flex h-2 overflow-hidden rounded-full bg-slate-950 ring-1 ring-inset ring-slate-800/80"
        role="img"
        aria-label={`${addPct.toFixed(1)} percent additions, ${delPct.toFixed(1)} percent deletions`}
      >
        <div
          className="h-full bg-gradient-to-r from-emerald-500 to-emerald-300"
          style={{ width: `${addPct}%` }}
        />
        <div
          className="h-full bg-gradient-to-r from-rose-500 to-rose-300"
          style={{ width: `${delPct}%` }}
        />
      </div>
      <div className="mt-1 flex justify-between text-[9.5px] text-slate-500">
        <span>{addPct.toFixed(1)}% added</span>
        <span>{delPct.toFixed(1)}% removed</span>
      </div>
    </section>
  );
}

function StatTile({
  icon,
  label,
  value,
  detail,
  tone = "neutral",
}: {
  icon: ReactNode;
  label: string;
  value: string;
  detail: string;
  tone?: Tone;
}) {
  return (
    <div className="min-w-0 rounded-xl border border-slate-800/70 bg-slate-900/35 p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-[8px] font-semibold uppercase tracking-[0.06em] text-slate-500 min-[380px]:text-[9px] min-[380px]:tracking-[0.08em]">
          {label}
        </span>
        <span
          className={cn(
            "grid h-6 w-6 shrink-0 place-items-center rounded-md",
            toneStyles[tone].icon,
          )}
        >
          {icon}
        </span>
      </div>
      <div className="mt-1 truncate font-mono text-xl font-semibold leading-none tracking-tight text-slate-100 tabular-nums">
        {value}
      </div>
      <div className="mt-1 truncate text-[10px] text-slate-400">{detail}</div>
    </div>
  );
}

function CompositionRow({
  label,
  count,
  additions,
  deletions,
  maxTotal,
  barClass,
}: {
  label: string;
  count: number;
  additions: number;
  deletions: number;
  maxTotal: number;
  barClass: string;
}) {
  const total = additions + deletions;
  const width =
    count > 0 && maxTotal > 0
      ? Math.max((total / maxTotal) * 100, total > 0 ? 2 : 0)
      : 0;

  if (count === 0) {
    return (
      <div className="flex items-center justify-between gap-2 text-[11px]">
        <span className="truncate text-slate-300">
          {label} <span className="text-slate-500">· 0</span>
        </span>
        <span className="shrink-0 text-[10px] text-slate-500">none</span>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-baseline justify-between gap-2 text-[11px]">
        <span className="min-w-0 truncate text-slate-300">
          {label} <span className="text-slate-500">· {formatCount(count)}</span>
        </span>
        <span className="shrink-0 font-mono text-[10px] tabular-nums">
          <span className="text-emerald-300">+{formatCount(additions)}</span>{" "}
          <span className="text-rose-300">−{formatCount(deletions)}</span>
        </span>
      </div>
      <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-slate-950 ring-1 ring-inset ring-slate-800/70">
        <div
          className={cn("h-full rounded-full bg-gradient-to-r", barClass)}
          style={{ width: `${width}%` }}
        />
      </div>
    </div>
  );
}

function TypeRow({
  extension,
  count,
  maxCount,
  barClass,
}: {
  extension: string;
  count: number;
  maxCount: number;
  barClass: string;
}) {
  const width = maxCount > 0 ? Math.max((count / maxCount) * 100, 2) : 0;
  return (
    <div>
      <div className="flex items-baseline justify-between gap-2 text-[11px]">
        <span className="font-mono text-slate-300">{extension}</span>
        <span className="font-mono text-[10px] tabular-nums text-slate-400">
          {formatCount(count)}
        </span>
      </div>
      <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-slate-950 ring-1 ring-inset ring-slate-800/70">
        <div
          className={cn("h-full rounded-full bg-gradient-to-r", barClass)}
          style={{ width: `${width}%` }}
        />
      </div>
    </div>
  );
}

function ReviewSignalCard({ signal }: { signal: ReviewSignal }) {
  const styles = toneStyles[signal.tone];
  const Icon =
    signal.tone === "warning"
      ? AlertTriangle
      : signal.tone === "success"
        ? Check
        : signal.tone === "info"
          ? Info
          : Activity;
  return (
    <section className={cn("mt-3 flex items-start gap-2.5 rounded-xl border px-3 py-2.5", styles.card)}>
      <span className={cn("grid h-6 w-6 shrink-0 place-items-center rounded-md", styles.icon)}>
        <Icon className="h-3.5 w-3.5" />
      </span>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs font-semibold text-slate-100">Review signal</span>
          <span className={cn("inline-flex rounded-full border px-2 py-0.5 text-[9.5px] font-semibold", styles.pill)}>
            {signal.pill}
          </span>
        </div>
        <p className="mt-0.5 text-[11px] leading-relaxed text-slate-300">{signal.body}</p>
      </div>
    </section>
  );
}

function AllMetricsDetails({ children }: { children: ReactNode }) {
  return (
    <details className="group mt-3 rounded-xl border border-slate-800/70 bg-slate-900/25">
      <summary className="flex cursor-pointer list-none items-center justify-between gap-2 px-3 py-2.5 text-[11px] font-medium text-slate-300 [&::-webkit-details-marker]:hidden">
        <span className="flex items-center gap-2">
          <Layers className="h-3.5 w-3.5 text-slate-500" />
          All metrics
        </span>
        <ChevronDown className="h-3.5 w-3.5 text-slate-500 transition-transform group-open:rotate-180" />
      </summary>
      <div className="border-t border-slate-800/60 px-3 py-3">{children}</div>
    </details>
  );
}

function EmptyMetrics() {
  return (
    <div
      className="flex flex-col items-center justify-center rounded-xl border border-dashed border-slate-700/80 bg-slate-900/20 px-6 py-10 text-center"
      data-testid="metrics-empty"
    >
      <span className="grid h-10 w-10 place-items-center rounded-full bg-slate-800/80 text-slate-400">
        <BarChart3 className="h-5 w-5" />
      </span>
      <p className="mt-3 text-sm font-medium text-slate-200">No changes to measure</p>
      <p className="mt-1 max-w-sm text-[11.5px] text-slate-500">
        The working tree has no line changes yet. Metrics appear when a file changes.
      </p>
    </div>
  );
}

function LoadingState() {
  return (
    <div className="space-y-3 py-1" data-testid="metrics-loading" role="status">
      <div className="h-28 animate-pulse rounded-xl bg-slate-900/70" />
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <div key={index} className="h-20 animate-pulse rounded-xl bg-slate-900/50" />
        ))}
      </div>
      <div className="h-24 animate-pulse rounded-xl bg-slate-900/40" />
      <p className="text-[11px] text-slate-500">Loading change metrics…</p>
    </div>
  );
}

function UnavailableState({ onRetry }: { onRetry?: () => void }) {
  return (
    <div
      className="space-y-3 rounded-xl border border-amber-400/20 bg-amber-400/[0.04] p-4"
      data-testid="metrics-unavailable"
      role="alert"
    >
      <div className="flex items-start gap-2.5">
        <span className="grid h-6 w-6 shrink-0 place-items-center rounded-md bg-amber-400/15 text-amber-300">
          <AlertTriangle className="h-3.5 w-3.5" />
        </span>
        <div>
          <p className="text-sm font-medium text-slate-200">Change metrics could not be loaded.</p>
          <p className="mt-1 text-[11.5px] text-slate-400">
            The file list is still current; retry to load its line totals.
          </p>
        </div>
      </div>
      {onRetry && (
        <button
          type="button"
          className="min-h-11 rounded-lg border border-slate-700 px-3 text-xs font-semibold text-slate-200 hover:bg-slate-800"
          onClick={onRetry}
        >
          Retry metrics
        </button>
      )}
    </div>
  );
}

function RefreshingPill() {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-cyan-400/20 bg-cyan-400/[0.05] px-2.5 py-1.5 text-[10.5px] text-cyan-200">
      <Loader2 className="h-3 w-3 animate-spin" />
      Updating current change metrics…
    </div>
  );
}

function AggregateFlightDeck({ fileStats }: { fileStats?: RepoFileStats }) {
  const aggregate = useMemo(() => aggregateFileStats(fileStats), [fileStats]);
  const stagedCat = useMemo(() => categoryStats(fileStats?.staged), [fileStats]);
  const unstagedCat = useMemo(() => categoryStats(fileStats?.unstaged), [fileStats]);
  const untrackedCat = useMemo(() => categoryStats(fileStats?.untracked), [fileStats]);
  const signal = useMemo(() => buildAggregateSignal(aggregate), [aggregate]);

  const maxCategoryTotal = Math.max(
    stagedCat.additions + stagedCat.deletions,
    unstagedCat.additions + unstagedCat.deletions,
    untrackedCat.additions + untrackedCat.deletions,
    0,
  );

  const topTypes = useMemo(
    () =>
      Object.entries(aggregate.fileTypeBreakdown)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 5),
    [aggregate.fileTypeBreakdown],
  );
  const maxTypeCount = topTypes[0]?.[1] ?? 0;

  const typeColors = [
    "from-cyan-500 to-cyan-300",
    "from-violet-500 to-violet-300",
    "from-amber-500 to-amber-300",
    "from-emerald-500 to-emerald-300",
    "from-blue-500 to-blue-300",
  ];

  if (aggregate.totalFiles === 0) {
    return (
      <div data-testid="aggregate-metrics-content">
        <EmptyMetrics />
      </div>
    );
  }

  const net = aggregate.totalNetLines;
  const total = aggregate.totalAdditions + aggregate.totalDeletions;

  return (
    <div className="space-y-3" data-testid="aggregate-metrics-content">
      <ChangeHero
        additions={aggregate.totalAdditions}
        deletions={aggregate.totalDeletions}
        net={net}
        netTestId="metric-net-lines"
      />

      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <StatTile
          icon={<Layers className="h-3.5 w-3.5" />}
          label="Files"
          value={formatCount(aggregate.totalFiles)}
          detail={`${formatCount(aggregate.binaryCount)} binary`}
          tone="info"
        />
        <StatTile
          icon={<FilePlus2 className="h-3.5 w-3.5" />}
          label="New files"
          value={formatCount(aggregate.newFileCount)}
          detail={`${Math.round((aggregate.newFileCount / aggregate.totalFiles) * 100)}% of files`}
          tone="success"
        />
        <StatTile
          icon={<FlaskConical className="h-3.5 w-3.5" />}
          label="Test ratio"
          value={`${Math.round(aggregate.testToCodeRatio * 100)}%`}
          detail={`${formatCount(aggregate.testFileCount)} test files`}
          tone={aggregate.testToCodeRatio < 0.1 ? "warning" : "neutral"}
        />
        <StatTile
          icon={<Target className="h-3.5 w-3.5" />}
          label="Concentration"
          value={`${aggregate.paretoPercent}%`}
          detail={
            aggregate.paretoTopN > 0
              ? `in ${formatCount(aggregate.paretoTopN)} files`
              : "not enough files"
          }
          tone="info"
        />
      </div>

      <div className="grid gap-2 md:grid-cols-2">
        <section className="flex h-full flex-col rounded-xl border border-slate-800/70 bg-slate-900/25 p-3">
          <div className="mb-2 flex items-baseline justify-between gap-2">
            <h3 className="text-[10px] font-semibold uppercase tracking-[0.09em] text-slate-400">
              By state
            </h3>
            <span className="text-[9.5px] text-slate-500">
              {[stagedCat.count, unstagedCat.count, untrackedCat.count].filter(Boolean).length} categories
            </span>
          </div>
          <div className="flex flex-1 flex-col justify-between gap-2.5">
            <CompositionRow
              label="In the index"
              count={stagedCat.count}
              additions={stagedCat.additions}
              deletions={stagedCat.deletions}
              maxTotal={maxCategoryTotal}
              barClass="from-sky-500 to-cyan-300"
            />
            <CompositionRow
              label="In the working tree"
              count={unstagedCat.count}
              additions={unstagedCat.additions}
              deletions={unstagedCat.deletions}
              maxTotal={maxCategoryTotal}
              barClass="from-violet-500 to-violet-300"
            />
            <CompositionRow
              label="New, not tracked"
              count={untrackedCat.count}
              additions={untrackedCat.additions}
              deletions={untrackedCat.deletions}
              maxTotal={maxCategoryTotal}
              barClass="from-emerald-500 to-emerald-300"
            />
          </div>
        </section>

        <section className="rounded-xl border border-slate-800/70 bg-slate-900/25 p-3">
          <div className="mb-2 flex items-baseline justify-between gap-2">
            <h3 className="text-[10px] font-semibold uppercase tracking-[0.09em] text-slate-400">
              By file type
            </h3>
            <span className="text-[9.5px] text-slate-500">
              top {Math.min(5, Object.keys(aggregate.fileTypeBreakdown).length)} of{" "}
              {Object.keys(aggregate.fileTypeBreakdown).length}
            </span>
          </div>
          <div className="space-y-2.5">
            {topTypes.map(([extension, count], index) => (
              <TypeRow
                key={extension}
                extension={extension}
                count={count}
                maxCount={maxTypeCount}
                barClass={typeColors[index % typeColors.length] ?? "from-cyan-500 to-cyan-300"}
              />
            ))}
          </div>
        </section>
      </div>

      <ReviewSignalCard signal={signal} />

      <AllMetricsDetails>
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
        <div className="mt-2 space-y-0.5 border-t border-slate-800/60 pt-2">
          <CategoryRow label="Staged" {...stagedCat} />
          <CategoryRow label="Unstaged" {...unstagedCat} />
          <CategoryRow label="Untracked" {...untrackedCat} />
        </div>
      </AllMetricsDetails>

      {total === 0 && (
        <p className="text-[10.5px] text-slate-500">
          Line totals are zero; file counts still reflect the working tree.
        </p>
      )}
    </div>
  );
}

/** Reusable aggregate metrics content for embedding in other modals. */
export function AggregateMetricsContent({ fileStats }: { fileStats: RepoFileStats }) {
  return <AggregateFlightDeck fileStats={fileStats} />;
}

function RiskRing({ score, label }: { score: number; label: string }) {
  const clamped = Math.max(0, Math.min(score, 100));
  const radius = 26;
  const circumference = 2 * Math.PI * radius;
  const dash = (clamped / 100) * circumference;
  const color =
    score >= 200 ? "text-rose-400" : score >= 50 ? "text-amber-400" : "text-emerald-400";

  return (
    <div className="relative h-20 w-20 shrink-0">
      <svg viewBox="0 0 64 64" className="h-20 w-20 -rotate-90">
        <circle
          cx="32"
          cy="32"
          r={radius}
          fill="none"
          strokeWidth="6"
          className="text-slate-700/70"
          stroke="currentColor"
        />
        <circle
          cx="32"
          cy="32"
          r={radius}
          fill="none"
          strokeWidth="6"
          strokeLinecap="round"
          strokeDasharray={`${dash} ${circumference}`}
          className={color}
          stroke="currentColor"
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="font-mono text-lg font-semibold leading-none text-slate-100 tabular-nums">
          {score}
        </span>
        <span className={cn("mt-0.5 text-[7.5px] font-bold uppercase tracking-wider", color)}>
          {label || "—"}
        </span>
      </div>
    </div>
  );
}

function ShapeRow({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between gap-2 py-1 text-[11px]">
      <span className="text-slate-400">{label}</span>
      <span className="font-mono text-[10.5px] tabular-nums text-slate-200">{value}</span>
    </div>
  );
}

function FileFlightDeck({
  stats,
  enhancedStats,
  enhancedLoading,
  isUntracked,
  filePath,
  fileHotspots,
}: {
  stats?: DiffStats;
  enhancedStats?: DiffStats;
  enhancedLoading?: boolean;
  isUntracked?: boolean;
  filePath?: string;
  fileHotspots?: Record<string, number>;
}) {
  const hunkSrc = enhancedStats ?? stats;
  const densitySrc = enhancedStats ?? stats;
  const commentSrc = enhancedStats ?? stats;
  const detailSrc = enhancedStats ?? stats;
  const additions = stats?.additions ?? 0;
  const deletions = stats?.deletions ?? 0;
  const net = stats?.net_lines ?? additions - deletions;
  const hasHunks = Boolean(hunkSrc?.hunk_count && hunkSrc.hunk_count > 0);
  const riskScore = hasHunks ? changeRiskScore(hunkSrc as DiffStats) : 0;
  const risk = riskLabel(riskScore);
  const churn = stats ? fileChurnRatio(stats) : 0;
  const avg = hasHunks && hunkSrc ? avgLinesPerHunk(hunkSrc) : 0;
  const commentLines =
    (commentSrc?.comment_additions ?? 0) + (commentSrc?.comment_deletions ?? 0);
  const signal = useMemo(
    () => buildFileSignal({ stats, hunkSrc, isUntracked, enhancedLoading }),
    [stats, hunkSrc, isUntracked, enhancedLoading],
  );
  const hotspotCount =
    filePath && fileHotspots ? fileHotspots[filePath] : undefined;

  const showEnhanced = !isUntracked;
  const scattered =
    densitySrc?.density != null && densitySrc.density > 0.3;

  return (
    <div className="space-y-3">
      <ChangeHero
        additions={additions}
        deletions={deletions}
        net={net}
        netTestId="metric-net-lines"
        pill={
          isUntracked
            ? "New file"
            : stats?.is_binary
              ? "Binary"
              : undefined
        }
      />

      {enhancedLoading && showEnhanced && (
        <div
          className="flex items-center gap-2 rounded-lg border border-slate-800/70 bg-slate-900/40 px-2.5 py-1.5 text-[10.5px] text-slate-400"
          data-testid="enhanced-loading"
        >
          <Loader2 className="h-3 w-3 animate-spin text-slate-500" />
          Loading detailed metrics…
        </div>
      )}

      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <StatTile
          icon={<GitBranch className="h-3.5 w-3.5" />}
          label="Hunks"
          value={showEnhanced && hasHunks ? formatCount(hunkSrc?.hunk_count ?? 0) : "—"}
          detail={showEnhanced ? (hasHunks ? "contiguous blocks" : "no hunks") : "not available"}
          tone="info"
        />
        <StatTile
          icon={<ArrowRight className="h-3.5 w-3.5" />}
          label="Largest hunk"
          value={showEnhanced && hasHunks ? formatCount(hunkSrc?.largest_hunk ?? 0) : "—"}
          detail={showEnhanced ? "lines" : "not available"}
          tone="info"
        />
        <StatTile
          icon={<Gauge className="h-3.5 w-3.5" />}
          label="Density"
          value={
            showEnhanced && densitySrc?.density
              ? densitySrc.density.toFixed(2)
              : "—"
          }
          detail={
            showEnhanced
              ? densityLabel(densitySrc?.density) || "no scatter"
              : "not available"
          }
          tone={
            densitySrc?.density != null && densitySrc.density > 0.3
              ? "warning"
              : "success"
          }
        />
        <StatTile
          icon={<ShieldCheck className="h-3.5 w-3.5" />}
          label="Risk"
          value={showEnhanced && hasHunks ? formatCount(riskScore) : "—"}
          detail={showEnhanced ? risk || "no hunks" : "not available"}
          tone={
            riskScore >= 200 ? "warning" : riskScore >= 50 ? "info" : "success"
          }
        />
      </div>

      <div className="grid gap-2 md:grid-cols-[1.1fr_1fr]">
        <section className="rounded-xl border border-slate-800/70 bg-slate-900/25 p-3">
          <div className="mb-1.5 flex items-baseline justify-between gap-2">
            <h3 className="text-[10px] font-semibold uppercase tracking-[0.09em] text-slate-400">
              Risk weight
            </h3>
            <span className="text-[9.5px] text-slate-500">hunks × largest hunk</span>
          </div>
          <div className="flex items-center gap-4 py-0.5">
            <RiskRing score={riskScore} label={risk} />
            <div className="min-w-0">
              <p className="text-[11.5px] font-semibold text-slate-100">
                {isUntracked
                  ? "New file"
                  : enhancedLoading
                    ? "Loading"
                    : scattered
                      ? "Scattered edits"
                      : riskScore >= 200
                        ? "Needs a close read"
                        : riskScore >= 50
                          ? "Moderate review weight"
                          : "Nothing unusual"}
              </p>
              <p className="mt-0.5 text-[10.5px] leading-relaxed text-slate-400">
                {isUntracked
                  ? "No committed version to compare."
                  : enhancedLoading
                    ? "Detailed metrics are still loading."
                    : hasHunks
                      ? scattered
                        ? `${hunkSrc?.hunk_count} hunks, largest ${hunkSrc?.largest_hunk ?? 0} lines. Density ${densitySrc?.density?.toFixed(2)} means many small edits.`
                        : `${hunkSrc?.hunk_count} hunk${hunkSrc?.hunk_count === 1 ? "" : "s"}, largest ${hunkSrc?.largest_hunk ?? 0} lines, ${densityLabel(densitySrc?.density) || "no scatter"}.`
                      : "No hunks to measure."}
              </p>
            </div>
          </div>
        </section>

        <section className="rounded-xl border border-slate-800/70 bg-slate-900/25 p-3">
          <div className="mb-1 flex items-baseline justify-between gap-2">
            <h3 className="text-[10px] font-semibold uppercase tracking-[0.09em] text-slate-400">
              Change shape
            </h3>
            <span className="text-[9.5px] text-slate-500">this file</span>
          </div>
          <ShapeRow label="Hunk count" value={showEnhanced && hasHunks ? hunkSrc?.hunk_count ?? 0 : "—"} />
          <ShapeRow label="Avg lines / hunk" value={showEnhanced && hasHunks ? avg : "—"} />
          <ShapeRow
            label="Churn ratio"
            value={churn > 0 ? `${(churn * 100).toFixed(0)}% ${churnLabel(churn)}` : "—"}
          />
          <ShapeRow
            label="Comment lines"
            value={
              showEnhanced && !enhancedLoading && commentLines > 0
                ? `+${commentSrc?.comment_additions ?? 0} / -${commentSrc?.comment_deletions ?? 0}`
                : "—"
            }
          />
          {showEnhanced && !enhancedLoading && densitySrc?.density ? (
            <DensityBar density={densitySrc.density} />
          ) : null}
        </section>
      </div>

      <ReviewSignalCard signal={signal} />

      <AllMetricsDetails>
        {showEnhanced && !enhancedLoading && hasHunks && (
          <>
            <MetricRow
              label="Hunks"
              value={hunkSrc?.hunk_count ?? 0}
              testId="metric-hunk-count"
            />
            <MetricRow
              label="Largest hunk"
              value={`${hunkSrc?.largest_hunk ?? 0} lines`}
              testId="metric-largest-hunk"
            />
          </>
        )}
        {showEnhanced && !enhancedLoading && commentLines > 0 && (
          <MetricRow
            label="Comment lines"
            value={`+${commentSrc?.comment_additions ?? 0} / -${commentSrc?.comment_deletions ?? 0}`}
            testId="metric-comment-lines"
          />
        )}
        {churn > 0 && (
          <MetricRow
            label="Churn ratio"
            value={`${(churn * 100).toFixed(0)}% ${churnLabel(churn)}`}
            testId="metric-file-churn"
          />
        )}
        {showEnhanced && !enhancedLoading && hasHunks && (
          <>
            <MetricRow
              label="Avg lines/hunk"
              value={avg}
              testId="metric-avg-lines-per-hunk"
            />
            <MetricRow
              label="Risk score"
              value={`${riskScore}${risk ? ` (${risk})` : ""}`}
              testId="metric-risk-score"
            />
          </>
        )}
        {hotspotCount && hotspotCount > 1 && (
          <MetricRow
            label="Hotspot"
            value={`${hotspotCount} commits in last 50`}
            testId="metric-hotspot"
          />
        )}
        <div className="mt-1 space-y-0.5 border-t border-slate-800/60 pt-2">
          {!isUntracked && detailSrc?.is_rename && detailSrc?.old_path && (
            <div
              className="flex items-center gap-2 text-[11px] text-slate-300"
              data-testid="metric-rename"
            >
              <span className="text-slate-500">Rename:</span>
              <span className="truncate">{detailSrc.old_path}</span>
              <ArrowRight className="h-3 w-3 shrink-0 text-slate-500" />
              <span className="truncate">{filePath}</span>
            </div>
          )}
          {stats?.is_binary && (
            <div className="text-[11px] text-amber-300" data-testid="metric-binary">
              Binary file
            </div>
          )}
          {filePath && isTestFile(filePath) && (
            <div className="text-[11px] text-amber-300" data-testid="metric-is-test-file">
              Test file
            </div>
          )}
          {(stats?.is_new_file || isUntracked) && (
            <div className="text-[11px] text-emerald-300" data-testid="metric-new-file">
              New file
            </div>
          )}
          {stats?.is_deleted_file && (
            <div className="text-[11px] text-rose-300" data-testid="metric-deleted-file">
              Deleted file
            </div>
          )}
        </div>
      </AllMetricsDetails>
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
  fileStatsState = "ready",
  onRetryFileStats,
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
    if (!isOpen || isMobile) return;
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, isMobile, handleKeyDown]);

  const aggregate = useMemo(
    () => (mode === "aggregate" ? aggregateFileStats(fileStats) : null),
    [mode, fileStats],
  );
  const hunkSrc = enhancedStats ?? stats;

  const displayTitle =
    title ?? (mode === "file" ? "File pulse" : "Change pulse");
  const overline = mode === "file" ? "File metrics" : "Change metrics";
  const subtitle =
    mode === "file"
      ? filePath ?? "Selected file"
      : isMobile
        ? `${formatCount(aggregate?.totalFiles ?? 0)} files${
            aggregate
              ? ` · ${compactNumber(aggregate.totalAdditions + aggregate.totalDeletions)} lines`
              : ""
          }`
        : `Working tree · ${formatCount(aggregate?.totalFiles ?? 0)} files${
            aggregate
              ? ` · ${compactNumber(aggregate.totalAdditions + aggregate.totalDeletions)} lines touched`
              : ""
          }`;

  const summary =
    mode === "aggregate" && aggregate
      ? buildAggregateSummary(aggregate)
      : buildFileSummary({ stats, hunkSrc, filePath, isUntracked });

  const topFiles = useMemo(
    () => (mode === "aggregate" ? topChangedFiles(fileStats) : []),
    [mode, fileStats],
  );
  const topFilesText = topFiles
    .map((row) => `${row.path}\t+${row.additions}\t-${row.deletions}`)
    .join("\n");

  const actions =
    mode === "aggregate" ? (
      <div className="flex w-full items-center justify-between gap-2 sm:w-auto">
        <span className="hidden text-[10px] text-slate-500 lg:block">
          Read-only · local metrics
        </span>
        <div className="flex w-full gap-2 sm:w-auto">
          <CopyActionButton
            text={summary}
            label="Copy summary"
            variant={topFiles.length > 0 ? "secondary" : "primary"}
          />
          {topFiles.length > 0 && (
            <CopyActionButton
              text={topFilesText}
              label={`Copy top ${topFiles.length}`}
              variant="primary"
            />
          )}
        </div>
      </div>
    ) : (
      <div className="flex w-full items-center justify-between gap-2 sm:w-auto">
        <span className="hidden text-[10px] text-slate-500 lg:block">
          Read-only · local metrics
        </span>
        <div className="flex w-full gap-2 sm:w-auto">
          <CopyActionButton text={summary} label="Copy summary" />
          {filePath && (
            <CopyActionButton
              text={filePath}
              label="Copy path"
              variant="primary"
            />
          )}
        </div>
      </div>
    );

  const aggregateReady =
    mode === "aggregate" && Boolean(aggregate) && (aggregate?.totalFiles ?? 0) > 0;
  const aggregateBlocked =
    mode === "aggregate" &&
    (fileStatsState === "loading" ||
      fileStatsState === "unavailable" ||
      (fileStatsState === "refreshing" && !fileStats));
  const actionBar =
    mode === "file" || (aggregateReady && !aggregateBlocked) ? actions : undefined;

  if (!isOpen) return null;

  const refreshing =
    mode === "aggregate" &&
    fileStatsState === "refreshing" &&
    Boolean(fileStats);

  const body =
    mode === "aggregate" &&
    (fileStatsState === "loading" ||
      (fileStatsState === "refreshing" && !fileStats)) ? (
      <LoadingState />
    ) : mode === "aggregate" && fileStatsState === "unavailable" ? (
      <UnavailableState onRetry={onRetryFileStats} />
    ) : mode === "aggregate" ? (
      <AggregateFlightDeck fileStats={fileStats} />
    ) : (
      <FileFlightDeck
        stats={stats}
        enhancedStats={enhancedStats}
        enhancedLoading={enhancedLoading}
        isUntracked={isUntracked}
        filePath={filePath}
        fileHotspots={fileHotspots}
      />
    );

  // Mobile: full-screen drawer with the shared RCL surface.
  if (isMobile) {
    return (
      <FullPageDrawer
        open={isOpen}
        onClose={onClose}
        title={displayTitle}
        ariaLabel="Change metrics"
        closeLabel="Close"
        dismissAffordance="close"
        contentClassName="px-3 py-3"
        headerExtra={
          <span className="mt-0.5 block max-w-[64vw] truncate text-[11px] font-normal text-slate-400">
            {subtitle}
          </span>
        }
        footer={actionBar}
      >
        <div
          className="space-y-3"
          data-testid="metrics-modal"
          aria-busy={refreshing || undefined}
        >
          {refreshing && <RefreshingPill />}
          {body}
        </div>
      </FullPageDrawer>
    );
  }

  // Desktop: centered overlay with a wider Flight Deck surface.
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4 py-6"
      role="dialog"
      aria-modal="true"
      aria-label="Change metrics"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="flex max-h-[88vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border border-slate-800 bg-slate-950 shadow-2xl">
        <header className="flex items-start gap-3 border-b border-slate-800/80 px-5 py-4">
          <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl border border-cyan-400/25 bg-cyan-400/10 text-cyan-300">
            {mode === "file" ? (
              <FileText className="h-5 w-5" />
            ) : (
              <BarChart3 className="h-5 w-5" />
            )}
          </div>
          <div className="min-w-0 flex-1">
            <p className="text-[9.5px] font-semibold uppercase tracking-[0.14em] text-slate-500">
              {overline}
            </p>
            <h2 className="mt-0.5 truncate text-lg font-semibold text-slate-50">
              {displayTitle}
            </h2>
            <p className="mt-0.5 truncate text-[11px] text-slate-400">{subtitle}</p>
          </div>
          <button
            type="button"
            className="grid h-8 w-8 shrink-0 place-items-center rounded-full text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400/60"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </header>

        <div
          className="min-h-0 flex-1 space-y-3 overflow-y-auto px-5 py-4"
          data-testid="metrics-modal"
          aria-busy={refreshing || undefined}
        >
          {refreshing && <RefreshingPill />}
          {body}
        </div>

        {actionBar ? (
          <footer className="border-t border-slate-800/80 px-5 py-3">{actionBar}</footer>
        ) : null}
      </div>
    </div>
  );
});
