import { Bot, Loader2, CheckCircle2, XCircle, AlertCircle, StopCircle, Copy, Check, Zap } from "lucide-react";
import { useState, useCallback } from "react";
import type { AgentRunStatus } from "../../../lib/api";
import type { AgentMetric } from "./AgentEventList";

interface AgentStatusIndicatorProps {
  /** Current run status */
  status: AgentRunStatus | undefined;
  /** Current execution phase */
  phase?: string;
  /** Progress percentage (0-100) */
  progress?: number;
  /** Error message if failed */
  errorMsg?: string;
  /** Aggregated metrics from metric events */
  metrics?: AgentMetric[];
  /** Callback to stop the run */
  onStop?: () => void;
  /** Render as inline metadata inside an existing row */
  inline?: boolean;
}

const STATUS_CONFIG: Record<AgentRunStatus, {
  label: string;
  color: string;
  icon: React.ReactNode;
  showProgress: boolean;
}> = {
  pending: {
    label: "Pending",
    color: "text-zinc-400",
    icon: <Loader2 className="h-4 w-4 animate-spin" />,
    showProgress: false
  },
  starting: {
    label: "Starting",
    color: "text-blue-400",
    icon: <Loader2 className="h-4 w-4 animate-spin" />,
    showProgress: false
  },
  running: {
    label: "Running",
    color: "text-blue-400",
    icon: <Bot className="h-4 w-4 animate-pulse" />,
    showProgress: true
  },
  needs_review: {
    label: "Needs Review",
    color: "text-yellow-400",
    icon: <AlertCircle className="h-4 w-4" />,
    showProgress: false
  },
  complete: {
    label: "Completed",
    color: "text-green-400",
    icon: <CheckCircle2 className="h-4 w-4" />,
    showProgress: false
  },
  failed: {
    label: "Failed",
    color: "text-red-400",
    icon: <XCircle className="h-4 w-4" />,
    showProgress: false
  },
  cancelled: {
    label: "Stopped",
    color: "text-zinc-400",
    icon: <StopCircle className="h-4 w-4" />,
    showProgress: false
  }
};

/**
 * Displays the current status of an agent run with progress indicator
 * and aggregated metrics (tokens, cost, API calls).
 */
export function AgentStatusIndicator({
  status,
  phase,
  progress = 0,
  errorMsg,
  metrics,
  onStop,
  inline = false,
}: AgentStatusIndicatorProps) {
  const [copied, setCopied] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const handleCopy = useCallback(async () => {
    if (!errorMsg) return;
    try {
      await navigator.clipboard.writeText(errorMsg);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback: select the text
      const el = document.querySelector("[data-agent-error]");
      if (el) {
        const range = document.createRange();
        range.selectNodeContents(el);
        window.getSelection()?.removeAllRanges();
        window.getSelection()?.addRange(range);
      }
    }
  }, [errorMsg]);

  if (!status) return null;

  const config = STATUS_CONFIG[status];
  const isRunning = status === "running" || status === "starting" || status === "pending";

  // Aggregate metrics for display
  const metricChips = aggregateMetrics(metrics);

  return (
    <div
      className={inline
        ? "min-w-0 flex-1 flex items-center gap-3"
        : "flex items-center gap-3 px-4 py-2 bg-zinc-800/50 border-b border-zinc-700"
      }
    >
      {/* Status icon and label */}
      <div className={`flex items-center gap-2 ${config.color}`}>
        {config.icon}
        <span className={inline ? "text-xs font-medium" : "text-sm font-medium"}>{config.label}</span>
      </div>

      {/* Phase */}
      {phase && isRunning && (
        <span className="text-xs text-zinc-500">
          {phase}
        </span>
      )}

      {/* Progress bar */}
      {config.showProgress && progress > 0 && (
        <div className={inline ? "flex-1 max-w-[10rem]" : "flex-1 max-w-xs"}>
          <div className="h-1.5 bg-zinc-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 transition-all duration-300"
              style={{ width: `${Math.min(progress, 100)}%` }}
            />
          </div>
        </div>
      )}

      {/* Aggregated metrics */}
      {metricChips.length > 0 && (
        <div className="flex items-center gap-2 min-w-0">
          <Zap className="h-3 w-3 text-zinc-500 flex-shrink-0" />
          {metricChips.map((chip) => (
            <span
              key={chip.label}
              className="px-1.5 py-0.5 text-xs rounded bg-zinc-700/50 text-zinc-400"
              title={chip.tooltip}
            >
              {chip.label}
            </span>
          ))}
        </div>
      )}

      {/* Error message */}
      {errorMsg && status === "failed" && (
        <div className="flex items-center gap-1.5 min-w-0 flex-1">
          <span
            data-agent-error
            className={`text-xs text-red-400 select-text cursor-text ${expanded ? "whitespace-pre-wrap break-all" : "truncate"}`}
            onClick={() => setExpanded(!expanded)}
            title={expanded ? undefined : errorMsg}
          >
            {errorMsg}
          </span>
          <button
            onClick={() => { void handleCopy(); }}
            className="flex-shrink-0 p-0.5 rounded text-zinc-500 hover:text-zinc-300 transition-colors"
            title="Copy error message"
          >
            {copied ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
          </button>
        </div>
      )}

      {/* Spacer */}
      <div className="flex-1" />

      {/* Stop button */}
      {isRunning && onStop && (
        <button
          onClick={onStop}
          className="
            flex items-center gap-1 px-2 py-1 rounded
            text-xs text-red-400 hover:text-red-300 hover:bg-red-500/10
            transition-colors
          "
        >
          <StopCircle className="h-3 w-3" />
          Stop
        </button>
      )}
    </div>
  );
}

interface MetricChip {
  label: string;
  tooltip: string;
}

/** Aggregate raw metric events into displayable chips. */
function aggregateMetrics(metrics?: AgentMetric[]): MetricChip[] {
  if (!metrics || metrics.length === 0) return [];

  const chips: MetricChip[] = [];
  const totals = new Map<string, { value: number; unit: string }>();

  for (const m of metrics) {
    const existing = totals.get(m.name);
    if (existing) {
      existing.value += m.value;
    } else {
      totals.set(m.name, { value: m.value, unit: m.unit });
    }
  }

  for (const [name, { value, unit }] of totals) {
    const formatted = formatMetricValue(value, unit);
    chips.push({
      label: `${formatted} ${unit || name}`,
      tooltip: `${name}: ${value} ${unit}`,
    });
  }

  return chips;
}

function formatMetricValue(value: number, unit: string): string {
  if (unit === "tokens" || unit === "bytes") {
    if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
    if (value >= 1_000) return `${(value / 1_000).toFixed(1)}k`;
  }
  if (unit === "ms") {
    if (value >= 1_000) return `${(value / 1_000).toFixed(1)}s`;
  }
  if (unit === "usd" || unit === "USD" || unit === "$") {
    return `$${value.toFixed(4)}`;
  }
  return value % 1 === 0 ? String(value) : value.toFixed(2);
}

export default AgentStatusIndicator;
