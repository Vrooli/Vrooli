// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
// Shared error fallback UI used by both ErrorBoundary (global) and
// PanelErrorBoundary (per-panel). Consolidates the duplicated alert
// layout so styling and messaging are consistent.
import { AlertTriangle, RefreshCw } from "lucide-react";

interface Props {
  /** Main message shown to the user */
  message: string;
  /** Optional sub-text shown below the message */
  detail?: string;
  /** Callback to attempt recovery */
  onRetry: () => void;
  /** Optional label for the retry button (default: "Try again") */
  retryLabel?: string;
  /** Optional extra action (e.g., "Reload page") */
  secondaryAction?: { label: string; onClick: () => void };
  /** Icon size class (default: "h-8 w-8") */
  iconSize?: string;
}

export function ErrorFallback({
  message,
  detail,
  onRetry,
  retryLabel = "Try again",
  secondaryAction,
  iconSize = "h-8 w-8",
}: Props) {
  return (
    <div className="text-center" role="alert">
      <AlertTriangle className={`${iconSize} text-red-400 mx-auto mb-3`} aria-hidden="true" />
      <p className="text-sm text-slate-400 mb-3">{message}</p>
      {detail && <p className="text-xs text-slate-500 mb-4">{detail}</p>}
      <div className="flex items-center justify-center gap-3">
        <button
          onClick={onRetry}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/10 text-sm text-white hover:bg-white/20"
        >
          <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
          {retryLabel}
        </button>
        {secondaryAction && (
          <button
            onClick={secondaryAction.onClick}
            className="px-3 py-1.5 rounded-lg border border-white/10 text-slate-400 hover:text-white hover:border-white/20 text-sm"
          >
            {secondaryAction.label}
          </button>
        )}
      </div>
    </div>
  );
}
