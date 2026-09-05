// DOC: docs/internal/ERROR_SEMANTICS.md#client-side-failure-handling
// DOC: docs/internal/SEAMS.md#axis-3-error-codes-recovery-api-ui
import { AlertTriangle, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { ErrorInfo } from "../lib/errors";
import { cn } from "../lib/classnames";
import { strings } from "../consts/strings";
import { IconButton } from "@vrooli/react-component-library/IconButton";

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
  const { t } = useTranslation();
  return (
    <div
      data-testid="create-error-banner"
      className={cn("wc-stable-theme rounded-md border border-wc-error bg-wc-error-surface py-2 ps-[max(1rem,var(--wc-safe-left,0px))] pe-[max(1rem,var(--wc-safe-right,0px))] text-sm text-wc-error-text", className)}
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
            {t(strings.errorBanner.retry)}
          </button>
        )}
        <IconButton
          onClick={onDismiss}
          aria-label={t(strings.errorBanner.dismiss)}
          size="sm"
          className="shrink-0"
        >
          <X />
        </IconButton>
      </div>
      {error.recovery && (
        <p data-testid="error-recovery-hint" className="mt-1 text-xs text-wc-error-detail/70 ps-6">
          {error.recovery}
        </p>
      )}
    </div>
  );
}
