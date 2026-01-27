import { Bot, Loader2, CheckCircle2, XCircle, AlertCircle, StopCircle } from "lucide-react";
import type { AgentRunStatus } from "../../../lib/api";

interface AgentStatusIndicatorProps {
  /** Current run status */
  status: AgentRunStatus | undefined;
  /** Current execution phase */
  phase?: string;
  /** Progress percentage (0-100) */
  progress?: number;
  /** Error message if failed */
  errorMsg?: string;
  /** Callback to stop the run */
  onStop?: () => void;
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
 * Displays the current status of an agent run with progress indicator.
 */
export function AgentStatusIndicator({
  status,
  phase,
  progress = 0,
  errorMsg,
  onStop
}: AgentStatusIndicatorProps) {
  if (!status) return null;

  const config = STATUS_CONFIG[status] || STATUS_CONFIG.pending;
  const isRunning = status === "running" || status === "starting" || status === "pending";

  return (
    <div className="flex items-center gap-3 px-4 py-2 bg-zinc-800/50 border-b border-zinc-700">
      {/* Status icon and label */}
      <div className={`flex items-center gap-2 ${config.color}`}>
        {config.icon}
        <span className="text-sm font-medium">{config.label}</span>
      </div>

      {/* Phase */}
      {phase && isRunning && (
        <span className="text-xs text-zinc-500">
          {phase}
        </span>
      )}

      {/* Progress bar */}
      {config.showProgress && progress > 0 && (
        <div className="flex-1 max-w-xs">
          <div className="h-1.5 bg-zinc-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 transition-all duration-300"
              style={{ width: `${Math.min(progress, 100)}%` }}
            />
          </div>
        </div>
      )}

      {/* Error message */}
      {errorMsg && status === "failed" && (
        <span className="text-xs text-red-400 truncate max-w-xs" title={errorMsg}>
          {errorMsg}
        </span>
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

export default AgentStatusIndicator;
