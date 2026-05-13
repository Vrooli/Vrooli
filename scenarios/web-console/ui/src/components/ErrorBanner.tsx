// DOC: docs/internal/ERROR_SEMANTICS.md#client-side-failure-handling
// DOC: docs/internal/SEAMS.md#axis-3-error-codes--recovery-api--ui
import { AlertTriangle, X } from "lucide-react";
import type { ErrorInfo } from "../lib/errors";
import { cn } from "../lib/classnames";

// Re-export so existing importers of ErrorInfo from this file continue to work.
export type { ErrorInfo } from "../lib/errors";

/**
 * ── VOLATILE: Error display component. ──
 * When adding new error categories, recovery actions, or display
 * variations, changes land here. The parent only needs to pass the
 * error payload; all rendering decisions are owned by this component.
 */

interface ErrorBannerProps {
  error: ErrorInfo;
  onDismiss: () => void;
  onRetry?: () => void;
  /** Additional CSS classes for the container. */
  className?: string;
}

// [REQ:P0-001b] Independent Pane Session Lifecycle — error feedback
export default function ErrorBanner({
  error,
  onDismiss,
  onRetry,
  className = "",
}: ErrorBannerProps) {
  return (
    <div
      data-testid="create-error-banner"
      className={cn("rounded-md border border-wc-error bg-wc-error-surface px-4 py-2 text-sm text-wc-error-text", className)}
    >
      <div className="flex items-center gap-2">
        <AlertTriangle className="h-4 w-4 shrink-0" />
        <span className="flex-1">{error.message}</span>
        {error.retry && onRetry && (
          <button
            data-testid="error-retry-button"
            onClick={onRetry}
            className="shrink-0 text-xs underline hover:text-red-100"
          >
            Retry
          </button>
        )}
        <button onClick={onDismiss} className="shrink-0 p-0.5 hover:text-red-100">
          <X className="h-3 w-3" />
        </button>
      </div>
      {error.recovery && (
        <p data-testid="error-recovery-hint" className="mt-1 text-xs text-wc-error-detail/70 pl-6">
          {error.recovery}
        </p>
      )}
    </div>
  );
}
